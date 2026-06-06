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

// KickMember validates the guard, rejects a self-kick immediately, then runs
// kickMemberTx in a transaction. On success it invalidates the target user's cache
// and the full scoreboard cache.
func (uc *TeamUseCase) KickMember(ctx context.Context, captainID, targetUserID uuid.UUID) error {
	_, err := uc.deps.Guard.RequireTeamSwitch(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - KickMember - Guard: %w", err)
	}

	if captainID == targetUserID {
		return apperr.ErrCannotKickSelf
	}

	var kickedFromTeamID uuid.UUID

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var errKick error

		kickedFromTeamID, errKick = uc.kickMemberTx(ctx, captainID, targetUserID)

		return errKick
	}); err != nil {
		return fmt.Errorf("TeamUseCase - KickMember - TM.Run: %w", err)
	}

	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, targetUserID)
	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, kickedFromTeamID)

	return nil
}

// kickMemberTx runs the kick operation inside a transaction: checks competition guard,
// acquires locks via kickMemberPrepare, validates constraints, then executes the kick.
// Returns the team ID so the caller can perform targeted cache invalidation.
func (uc *TeamUseCase) kickMemberTx(ctx context.Context, captainID, targetUserID uuid.UUID) (uuid.UUID, error) {
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - kickMemberTx - CompRepo.Get: %w", err)
	}

	if err := guard.ValidateTeamSwitchState(comp); err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - kickMemberTx - ValidateTeamSwitchState: %w", err)
	}

	captain, team, targetUser, err := uc.kickMemberPrepare(ctx, captainID, targetUserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - kickMemberTx - kickMemberPrepare: %w", err)
	}

	members, err := uc.deps.UserRepo.GetByTeamID(ctx, team.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - kickMemberTx - UserRepo.GetByTeamID: %w", err)
	}

	if err := uc.kickMemberValidate(captain, team, targetUser, captainID, targetUserID, len(members), comp.MinTeamSize); err != nil {
		return uuid.Nil, err
	}

	return team.ID, uc.kickMemberExecute(ctx, team.ID, captainID, targetUserID)
}

// kickMemberPrepare acquires per-user locks in lexicographic UUID order (captain and
// target), then locks the team row, and loads all three entities needed for validation.
func (uc *TeamUseCase) kickMemberPrepare(ctx context.Context, captainID, targetUserID uuid.UUID) (*domain.User, *domain.Team, *domain.User, error) {
	firstID, secondID := captainID, targetUserID
	if captainID.String() > targetUserID.String() {
		firstID, secondID = targetUserID, captainID
	}

	if err := uc.deps.UserRepo.Lock(ctx, firstID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - kickMemberPrepare - UserRepo.Lock (first): %w", err)
	}
	if err := uc.deps.UserRepo.Lock(ctx, secondID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - kickMemberPrepare - UserRepo.Lock (second): %w", err)
	}

	captain, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - kickMemberPrepare - UserRepo.GetByID (captain): %w", err)
	}

	if captain.TeamID == nil {
		return nil, nil, nil, apperr.ErrTeamNotFound
	}

	if err := uc.deps.TeamRepo.Lock(ctx, *captain.TeamID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - kickMemberPrepare - TeamRepo.Lock: %w", err)
	}

	team, err := uc.deps.TeamRepo.GetByID(ctx, *captain.TeamID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - kickMemberPrepare - TeamRepo.GetByID: %w", err)
	}

	targetUser, err := uc.deps.UserRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - kickMemberPrepare - UserRepo.GetByID (target): %w", err)
	}

	return captain, team, targetUser, nil
}

// kickMemberValidate enforces kick preconditions: captain ban, captaincy ownership,
// anti-self-kick guard on captain row, team ban, target membership in team, and
// minimum team size constraint.
func (uc *TeamUseCase) kickMemberValidate(captain *domain.User, team *domain.Team, targetUser *domain.User, captainID, _ uuid.UUID, memberCount, minTeamSize int) error {
	if captain.IsBanned {
		return apperr.ErrUserBanned
	}

	if team.CaptainID != captainID {
		return apperr.ErrNotCaptain
	}

	if targetUser.ID == team.CaptainID {
		return apperr.ErrCannotKickCaptain
	}

	if team.IsBanned {
		return apperr.ErrTeamBanned
	}

	if targetUser.TeamID == nil || *targetUser.TeamID != team.ID {
		return apperr.ErrUserNotFound
	}

	if minTeamSize > 0 && memberCount-1 < minTeamSize {
		return apperr.ErrTeamBelowMinSize
	}

	return nil
}

// kickMemberExecute clears the target's team membership and writes a TeamActionMemberKicked
// audit log with the target_user_id recorded in Details for traceability.
func (uc *TeamUseCase) kickMemberExecute(ctx context.Context, teamID, captainID, targetUserID uuid.UUID) error {
	err := uc.deps.UserRepo.UpdateTeamID(ctx, targetUserID, nil)
	if err != nil {
		return fmt.Errorf("TeamUseCase - kickMemberExecute - UserRepo.UpdateTeamID: %w", err)
	}

	auditLog := &domain.TeamAuditLog{
		TeamID: teamID, UserID: &captainID, Action: domain.TeamActionMemberKicked,
		Details: map[string]any{"target_user_id": targetUserID.String()},
	}

	err = uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog)
	if err != nil {
		return fmt.Errorf("TeamUseCase - kickMemberExecute - TeamRepo.CreateAuditLog: %w", err)
	}

	return nil
}
