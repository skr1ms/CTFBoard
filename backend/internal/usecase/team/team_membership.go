package team

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

// Join validates the competition guard, delegates the full join flow to joinTx inside a
// transaction, then invalidates the user cache and full scoreboard cache on success.
func (uc *TeamUseCase) Join(ctx context.Context, inviteToken, userID uuid.UUID, confirmReset bool) (*domain.Team, error) {
	_, err := uc.deps.Guard.RequireTeamSwitch(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Join - Guard.RequireTeamSwitch: %w", err)
	}

	var team *domain.Team

	err = uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var errJoin error

		team, errJoin = uc.joinTx(ctx, inviteToken, userID, confirmReset)
		if errJoin != nil {
			return fmt.Errorf("TeamUseCase - Join - joinTx: %w", errJoin)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - Join - TM.Run: %w", err)
	}

	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, userID)
	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, team.ID)

	return team, nil
}

// joinTx processes a join-via-invite-token request inside a transaction. It delegates
// initial validation and pre-lock reads to joinTxPrepare, then acquires advisory locks
// in a deadlock-safe order: when the user is already in a team, both the old and new
// team rows are locked using orderTeamLockIDs (lexicographic on UUID); otherwise only
// the target team is locked. After re-fetching the team under the lock it checks for a
// ban, verifies the team is not a solo team, confirms the invite token has not expired
// or been rotated, and enforces MaxTeamSize. If the user holds a solo or auto-created
// team it is cleaned up via handleSoloTeamCleanup (requires confirmReset=true).
func (uc *TeamUseCase) joinTx(ctx context.Context, inviteToken, userID uuid.UUID, confirmReset bool) (*domain.Team, error) {
	team, user, comp, err := uc.joinTxPrepare(ctx, inviteToken, userID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - joinTxPrepare: %w", err)
	}

	if user.TeamID != nil {
		firstID, secondID := orderTeamLockIDs(*user.TeamID, &team.ID)
		err := uc.deps.TeamRepo.Lock(ctx, firstID)
		if err != nil {
			return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.Lock(first): %w", err)
		}

		if secondID != uuid.Nil {
			err := uc.deps.TeamRepo.Lock(ctx, secondID)
			if err != nil {
				return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.Lock(second): %w", err)
			}
		}
	} else {
		err := uc.deps.TeamRepo.Lock(ctx, team.ID)
		if err != nil {
			return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.Lock: %w", err)
		}
	}

	team, err = uc.deps.TeamRepo.GetByID(ctx, team.ID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.GetByID: %w", err)
	}

	if team.IsBanned {
		return nil, apperr.ErrTeamBanned
	}

	if team.IsSolo {
		return nil, apperr.ErrTeamNotFound
	}

	if team.InviteTokenExpiresAt != nil && time.Now().After(*team.InviteTokenExpiresAt) {
		return nil, apperr.ErrInviteExpired
	}

	if team.InviteToken != inviteToken {
		return nil, apperr.ErrInviteExpired
	}

	maxSize := resolveMaxTeamSize(comp, uc.deps.DefaultMaxTeamSize)

	members, err := uc.deps.UserRepo.GetByTeamID(ctx, team.ID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.GetByTeamID: %w", err)
	}

	if len(members) >= maxSize {
		return nil, apperr.ErrTeamFull
	}

	if user.TeamID != nil {
		err := uc.handleSoloTeamCleanup(ctx, user, userID, confirmReset, &team.ID)
		if err != nil {
			return nil, fmt.Errorf("TeamUseCase - joinTx - handleSoloTeamCleanup: %w", err)
		}
	}

	if err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, &team.ID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.UpdateTeamID: %w", err)
	}

	auditLog := &domain.TeamAuditLog{TeamID: team.ID, UserID: &userID, Action: domain.TeamActionJoined}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.CreateAuditLog: %w", err)
	}

	return team, nil
}

// joinTxPrepare loads and validates the team and user before acquiring locks inside the join transaction.
// Checks invite token validity, competition mode, team capacity, and ban status.
func (uc *TeamUseCase) joinTxPrepare(ctx context.Context, inviteToken, userID uuid.UUID) (*domain.Team, *domain.User, *domain.Competition, error) {
	comp, err := uc.deps.CompRepo.GetForUpdate(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - joinTx - CompetitionRepo.GetForUpdate: %w", err)
	}

	if err := guard.ValidateTeamSwitchState(comp); err != nil {
		return nil, nil, nil, err
	}

	if !comp.Mode.AllowsTeams() {
		return nil, nil, nil, apperr.ErrTeamsNotAllowed
	}

	if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.Lock: %w", err)
	}

	team, err := uc.deps.TeamRepo.GetByInviteToken(ctx, inviteToken)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - joinTx - TeamRepo.GetByInviteToken: %w", err)
	}

	if team.IsBanned {
		return nil, nil, nil, apperr.ErrTeamBanned
	}

	if team.IsSolo {
		return nil, nil, nil, apperr.ErrTeamNotFound
	}

	members, err := uc.deps.UserRepo.GetByTeamID(ctx, team.ID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.GetByTeamID: %w", err)
	}

	maxSize := resolveMaxTeamSize(comp, uc.deps.DefaultMaxTeamSize)

	if len(members) >= maxSize {
		return nil, nil, nil, apperr.ErrTeamFull
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - joinTx - UserRepo.GetByID: %w", err)
	}

	if user.IsBanned {
		return nil, nil, nil, apperr.ErrUserBanned
	}

	if user.WasInBannedTeam {
		return nil, nil, nil, apperr.ErrUserWasInBannedTeam
	}

	return team, user, comp, nil
}
