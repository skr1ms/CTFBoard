package team

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

// CreateSoloTeam creates a solo wrapper team for userID. Delegates to createSoloTeamTx
// inside a transaction; invalidates the user cache on success.
func (uc *TeamUseCase) CreateSoloTeam(ctx context.Context, userID uuid.UUID, confirmReset bool) (*domain.Team, error) {
	var team *domain.Team

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var errCreate error

		team, errCreate = uc.createSoloTeamTx(ctx, userID, confirmReset, false, false)
		if errCreate != nil {
			return fmt.Errorf("TeamUseCase - CreateSoloTeam - createSoloTeamTx: %w", errCreate)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - CreateSoloTeam - TM.Run: %w", err)
	}

	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, userID)
	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, team.ID)

	return team, nil
}

func (uc *TeamUseCase) requireSoloModeOnly(ctx context.Context) error {
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - requireSoloModeOnly - CompetitionRepo.Get: %w", err)
	}

	if !comp.Mode.AllowsSolo() {
		return apperr.ErrSoloModeNotAllowed
	}

	return nil
}

// CreateSoloTeamForNewUser creates a solo team for a freshly registered user.
// Unlike CreateSoloTeam it skips the team-switch guard (the user has no team yet)
// and passes skipTeamSwitchCheck=true to createSoloTeamTx.
func (uc *TeamUseCase) CreateSoloTeamForNewUser(ctx context.Context, userID uuid.UUID) (*domain.Team, error) {
	if err := uc.requireSoloModeOnly(ctx); err != nil {
		return nil, fmt.Errorf("TeamUseCase - CreateSoloTeamForNewUser - requireSoloModeOnly: %w", err)
	}

	var team *domain.Team

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var errCreate error

		team, errCreate = uc.createSoloTeamTx(ctx, userID, false, true, true)
		if errCreate != nil {
			return fmt.Errorf("TeamUseCase - CreateSoloTeamForNewUser - createSoloTeamTx: %w", errCreate)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - CreateSoloTeamForNewUser - TM.Run: %w", err)
	}

	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, userID)
	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, team.ID)

	return team, nil
}

// createSoloTeamTx creates a solo wrapper team for userID inside an existing transaction
// Name resolution prefers the username, falls back to "<username> (Solo)", then to a
// UUID-suffixed variant. If the final insert still hits a unique-violation it retries up
// to maxSoloNameRetries times - each iteration generates a fresh placeholder token for
// the suffix. Any existing solo or auto-created team is cleaned up via
// handleSoloTeamCleanup before the new team is created. After a successful insert the
// invite token is replaced with the team's own ID (making it a stable, never-expiring
// token), and the user is assigned to the team.
func (uc *TeamUseCase) createSoloTeamTx(ctx context.Context, userID uuid.UUID, confirmReset, isAutoCreated, skipTeamSwitchCheck bool) (*domain.Team, error) {
	comp, err := uc.deps.CompRepo.GetForUpdate(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - CompetitionRepo.GetForUpdate: %w", err)
	}

	if !skipTeamSwitchCheck {
		if err := guard.ValidateTeamSwitchState(comp); err != nil {
			return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - ValidateTeamSwitchState: %w", err)
		}
	}

	if !comp.Mode.AllowsSolo() {
		return nil, apperr.ErrSoloModeNotAllowed
	}

	if err := uc.checkMaxTeams(ctx); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - checkMaxTeams: %w", err)
	}

	if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - UserRepo.Lock: %w", err)
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - UserRepo.GetByID: %w", err)
	}

	if user.IsBanned {
		return nil, apperr.ErrUserBanned
	}

	if user.WasInBannedTeam {
		return nil, apperr.ErrUserWasInBannedTeam
	}

	if user.TeamID != nil {
		err := uc.handleSoloTeamCleanup(ctx, user, userID, confirmReset, nil)
		if err != nil {
			return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - handleSoloTeamCleanup: %w", err)
		}
	}

	const maxSoloNameRetries = 15

	var team *domain.Team

	for attempt := range maxSoloNameRetries {
		placeholderToken := uuid.New()
		expiresAt := time.Now().Add(defaultInviteTokenTTL)

		team = &domain.Team{
			Name:                 user.Username,
			InviteToken:          placeholderToken,
			CaptainID:            userID,
			IsSolo:               true,
			IsAutoCreated:        isAutoCreated,
			InviteTokenExpiresAt: &expiresAt,
		}
		if _, err := uc.deps.TeamRepo.GetByName(ctx, team.Name); err == nil {
			fallback := fmt.Sprintf("%s (Solo)", user.Username)
			if _, err := uc.deps.TeamRepo.GetByName(ctx, fallback); err == nil {
				fallback = fmt.Sprintf("%s-%s", user.Username, placeholderToken.String())
			}

			team.Name = fallback
		}

		err := uc.deps.TeamRepo.Create(ctx, team)
		if err == nil {
			break
		}

		if !errors.Is(err, apperr.ErrTeamAlreadyExists) || attempt == maxSoloNameRetries-1 {
			return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - TeamRepo.Create: %w", err)
		}
	}

	if err := uc.deps.TeamRepo.UpdateInviteToken(ctx, team.ID, team.ID, nil); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - TeamRepo.UpdateInviteToken: %w", err)
	}

	team.InviteToken = team.ID

	team.InviteTokenExpiresAt = nil
	if err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, &team.ID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - UserRepo.UpdateTeamID: %w", err)
	}

	auditLog := &domain.TeamAuditLog{
		TeamID: team.ID, UserID: &userID, Action: domain.TeamActionCreated,
		Details: map[string]any{"mode": "solo"},
	}
	if err := uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog); err != nil {
		return nil, fmt.Errorf("TeamUseCase - createSoloTeamTx - TeamRepo.CreateAuditLog: %w", err)
	}

	return team, nil
}
