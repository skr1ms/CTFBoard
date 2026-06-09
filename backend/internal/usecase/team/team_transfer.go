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

// TransferCaptain validates the guard, rejects a self-transfer immediately, then runs
// transferCaptainTx in a transaction. On success it invalidates the new captain's user
// cache, the team cache, and the full scoreboard cache.
func (uc *TeamUseCase) TransferCaptain(ctx context.Context, captainID, newCaptainID uuid.UUID) error {
	_, err := uc.deps.Guard.RequireTeamSwitch(ctx)
	if err != nil {
		return fmt.Errorf("TeamUseCase - TransferCaptain - Guard: %w", err)
	}

	if captainID == newCaptainID {
		return apperr.ErrCannotTransferToSelf
	}

	var teamID uuid.UUID

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var txErr error

		teamID, txErr = uc.transferCaptainTx(ctx, captainID, newCaptainID)

		return txErr
	}); err != nil {
		return fmt.Errorf("TeamUseCase - TransferCaptain - TM.Run: %w", err)
	}

	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, captainID)
	cacheutil.InvalidateUser(ctx, uc.deps.UserCache, newCaptainID)
	cacheutil.InvalidateTeam(ctx, uc.deps.TeamCache, uc.deps.Logger, teamID)
	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, teamID)

	return nil
}

// transferCaptainTx executes the captaincy transfer inside a transaction after acquiring per-user locks.
func (uc *TeamUseCase) transferCaptainTx(ctx context.Context, captainID, newCaptainID uuid.UUID) (uuid.UUID, error) {
	comp, err := uc.deps.CompRepo.GetForUpdate(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - transferCaptainTx - CompetitionRepo.GetForUpdate: %w", err)
	}

	if err := guard.ValidateTeamSwitchState(comp); err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - transferCaptainTx - ValidateTeamSwitchState: %w", err)
	}

	captain, team, newCaptain, err := uc.transferCaptainPrepare(ctx, captainID, newCaptainID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - transferCaptainTx - transferCaptainPrepare: %w", err)
	}

	if err := uc.transferCaptainValidate(captain, team, newCaptain, captainID); err != nil {
		return uuid.Nil, fmt.Errorf("TeamUseCase - transferCaptainTx - transferCaptainValidate: %w", err)
	}

	return team.ID, uc.transferCaptainExecute(ctx, captainID, newCaptainID, team)
}

// transferCaptainPrepare acquires per-user locks in lexicographic UUID order to prevent
// deadlocks, then locks the team row and loads all three entities required for the transfer.
func (uc *TeamUseCase) transferCaptainPrepare(ctx context.Context, captainID, newCaptainID uuid.UUID) (*domain.User, *domain.Team, *domain.User, error) {
	firstID, secondID := captainID, newCaptainID
	if captainID.String() > newCaptainID.String() {
		firstID, secondID = newCaptainID, captainID
	}

	if err := uc.deps.UserRepo.Lock(ctx, firstID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - transferCaptainPrepare - UserRepo.Lock(first): %w", err)
	}
	if err := uc.deps.UserRepo.Lock(ctx, secondID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - transferCaptainPrepare - UserRepo.Lock(second): %w", err)
	}

	captain, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - transferCaptainPrepare - UserRepo.GetByID: %w", err)
	}

	if captain.TeamID == nil {
		return nil, nil, nil, apperr.ErrTeamNotFound
	}

	if err := uc.deps.TeamRepo.Lock(ctx, *captain.TeamID); err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - transferCaptainPrepare - TeamRepo.Lock: %w", err)
	}

	team, err := uc.deps.TeamRepo.GetByID(ctx, *captain.TeamID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - transferCaptainPrepare - TeamRepo.GetByID: %w", err)
	}

	newCaptain, err := uc.deps.UserRepo.GetByID(ctx, newCaptainID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("TeamUseCase - transferCaptainPrepare - UserRepo.GetByID(newCaptain): %w", err)
	}

	return captain, team, newCaptain, nil
}

func (uc *TeamUseCase) transferCaptainValidate(captain *domain.User, team *domain.Team, newCaptain *domain.User, captainID uuid.UUID) error {
	if team.CaptainID != captainID {
		return apperr.ErrNotCaptain
	}

	if captain.IsBanned {
		return apperr.ErrUserBanned
	}

	if team.IsBanned {
		return apperr.ErrTeamBanned
	}

	if newCaptain.TeamID == nil || *newCaptain.TeamID != team.ID {
		return apperr.ErrNewCaptainNotInTeam
	}

	if newCaptain.IsBanned {
		return apperr.ErrUserBanned
	}

	return nil
}

func (uc *TeamUseCase) transferCaptainExecute(ctx context.Context, captainID, newCaptainID uuid.UUID, team *domain.Team) error {
	err := uc.deps.TeamRepo.UpdateCaptain(ctx, team.ID, newCaptainID)
	if err != nil {
		return fmt.Errorf("TeamUseCase - transferCaptainExecute - TeamRepo.UpdateCaptain: %w", err)
	}

	auditLog := &domain.TeamAuditLog{
		TeamID: team.ID, UserID: &captainID, Action: domain.TeamActionCaptainTransfer,
		Details: map[string]any{"from": captainID.String(), "to": newCaptainID.String()},
	}

	err = uc.deps.TeamRepo.CreateAuditLog(ctx, auditLog)
	if err != nil {
		return fmt.Errorf("TeamUseCase - transferCaptainExecute - TeamRepo.CreateAuditLog: %w", err)
	}

	return nil
}
