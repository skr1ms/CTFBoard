package team

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

// Leave validates the competition guard, runs leaveTx (prepare -> validate -> execute) in
// a transaction, then invalidates the user cache and full scoreboard cache.
func (uc *TeamUseCase) Leave(ctx context.Context, userID uuid.UUID) error {
	_, err := uc.deps.Guard.RequireTeamSwitch(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - Leave - Guard: %w", err)
	}

	var leftTeamID uuid.UUID

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var errLeave error

		leftTeamID, errLeave = uc.leaveTx(ctx, userID)
		if errLeave != nil {
			return fmt.Errorf("TeamUseCase - Leave - leaveTx: %w", errLeave)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("TeamUseCase - Leave - TM.Run: %w", err)
	}

	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, userID)
	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, leftTeamID)

	return nil
}

// leaveTx orchestrates the leave flow inside a transaction:
// prepare (validate + lock) -> determine outcome (transfer captain or disband) -> execute.
func (uc *TeamUseCase) leaveTx(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	comp, err := uc.deps.CompRepo.GetForUpdate(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - leaveTx - CompetitionRepo.GetForUpdate: %w", err)
	}

	if err := guard.ValidateTeamSwitchState(comp); err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - leaveTx - ValidateTeamSwitchState: %w", err)
	}

	user, team, members, err := uc.leavePrepare(ctx, userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - leaveTx - leavePrepare: %w", err)
	}

	if err := uc.leaveValidate(user, team, members, comp); err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - leaveTx - leaveValidate: %w", err)
	}

	return team.ID, uc.leaveExecute(ctx, userID, team)
}

// leavePrepare acquires advisory locks on the user row then the team row (in that order),
// fetches both entities and the current member list for use in leaveValidate.
func (uc *TeamUseCase) leavePrepare(ctx context.Context, userID uuid.UUID) (*domain.User, *domain.Team, []*domain.User, error) {
	if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - leavePrepare - UserRepo.Lock: %w", err)
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - leavePrepare - UserRepo.GetByID: %w", err)
	}

	if user.TeamID == nil {
		return nil, nil, nil, apperr.ErrTeamNotFound
	}

	if err := uc.deps.TeamRepo.Lock(ctx, *user.TeamID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - leavePrepare - TeamRepo.Lock: %w", err)
	}

	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - leavePrepare - TeamRepo.GetByID: %w", err)
	}

	members, err := uc.deps.UserRepo.GetByTeamID(ctx, *user.TeamID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - leavePrepare - UserRepo.GetByTeamID: %w", err)
	}

	return user, team, members, nil
}

// leaveValidate checks all preconditions for a leave operation: user/team ban status,
// solo-only mode restriction, single-member team guard, captain restriction, and
// minimum team size constraint from competition settings.
func (uc *TeamUseCase) leaveValidate(user *domain.User, team *domain.Team, members []*domain.User, comp *domain.Competition) error {
	if user.IsBanned {
		return apperr.ErrUserBanned
	}

	if team.IsBanned {
		return apperr.ErrTeamBanned
	}

	if team.IsSolo && comp.Mode == domain.ModeSoloOnly {
		return apperr.ErrCannotLeaveSoloTeam
	}

	if len(members) == 1 {
		return apperr.ErrCannotLeaveAsOnlyMember
	}

	if team.CaptainID == user.ID {
		return apperr.ErrCaptainCannotLeave
	}

	minSize := comp.MinTeamSize
	if minSize > 0 && len(members)-1 < minSize {
		return apperr.ErrTeamBelowMinSize
	}

	return nil
}

// leaveExecute clears the user's team membership and writes a TeamActionLeft audit log
// entry. Cache invalidation is handled by the caller (Leave) after the transaction.
func (uc *TeamUseCase) leaveExecute(ctx context.Context, userID uuid.UUID, team *domain.Team) error {
	err := uc.deps.UserRepo.UpdateTeamID(ctx, userID, nil)
	if err != nil {
		return fmt.Errorf("TeamUseCase - leaveExecute - UserRepo.UpdateTeamID: %w", err)
	}

	auditLog := &domain.TeamAuditLog{TeamID: team.ID, UserID: &userID, Action: domain.TeamActionLeft}

	err = uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog)
	if err != nil {
		return fmt.Errorf("TeamUseCase - leaveExecute - TeamRepo.CreateAuditLog: %w", err)
	}

	return nil
}
