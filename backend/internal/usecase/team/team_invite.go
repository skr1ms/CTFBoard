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

// UpdateMyTeam validates the competition-level team-switch guard, runs updateMyTeamTx
// in a transaction, then evicts both the team-specific cache entry and the full
// scoreboard cache (name change affects public display).
func (uc *TeamUseCase) UpdateMyTeam(ctx context.Context, captainID uuid.UUID, name string) (*domain.Team, error) {
	if _, err := uc.deps.Guard.RequireTeamSwitch(ctx); err != nil {
		return nil, fmt.Errorf("TeamUseCase - UpdateMyTeam - Guard.RequireTeamSwitch: %w", err)
	}

	var team *domain.Team

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var err error

		team, err = uc.updateMyTeamTx(ctx, captainID, name)
		if err != nil {
			return fmt.Errorf("TeamUseCase - UpdateMyTeam - updateMyTeamTx: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - UpdateMyTeam - TM.Run: %w", err)
	}

	if team != nil {
		cacheutil.InvalidateTeam(ctx, uc.deps.TeamCache, uc.deps.Logger, team.ID)
		cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, team.ID)
	}

	return team, nil
}

// updateMyTeamTx performs the team name update inside a transaction: locks the team row,
// validates name uniqueness, recomputes the slug, and persists the change.
func (uc *TeamUseCase) updateMyTeamTx(ctx context.Context, captainID uuid.UUID, name string) (*domain.Team, error) {
	comp, err := uc.deps.CompRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - CompetitionRepo.Get: %w", err)
	}

	if err := guard.ValidateTeamSwitchState(comp); err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - ValidateTeamSwitchState: %w", err)
	}

	if err := uc.deps.UserRepo.Lock(ctx, captainID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - UserRepo.Lock: %w", err)
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - UserRepo.GetByID: %w", err)
	}

	if user.TeamID == nil {
		return nil, apperr.ErrTeamNotFound
	}

	if user.IsBanned {
		return nil, apperr.ErrUserBanned
	}

	if err := uc.deps.TeamRepo.Lock(ctx, *user.TeamID); err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - TeamRepo.Lock: %w", err)
	}

	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - TeamRepo.GetByID: %w", err)
	}

	if team.IsBanned {
		return nil, apperr.ErrTeamBanned
	}

	if team.CaptainID != captainID {
		return nil, apperr.ErrNotCaptain
	}

	if team.Name != name {
		err := uc.validateTeamNameAvailable(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - validateTeamNameAvailable: %w", err)
		}
	}

	if err := uc.deps.TeamRepo.UpdateName(ctx, team.ID, name); err != nil {
		return nil, fmt.Errorf("TeamUseCase - updateMyTeamTx - TeamRepo.UpdateName: %w", err)
	}

	team.Name = name

	return team, nil
}

func (uc *TeamUseCase) GetInviteToken(ctx context.Context, captainID uuid.UUID) (*domain.Team, error) {
	if _, err := uc.deps.Guard.RequireTeamSwitch(ctx); err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetInviteToken - Guard.RequireTeamSwitch: %w", err)
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetInviteToken - UserRepo.GetByID: %w", err)
	}

	if user.IsBanned {
		return nil, apperr.ErrUserBanned
	}

	if user.TeamID == nil {
		return nil, apperr.ErrTeamNotFound
	}

	team, err := uc.deps.TeamRepo.GetByID(ctx, *user.TeamID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - GetInviteToken - TeamRepo.GetByID: %w", err)
	}

	if team.CaptainID != captainID {
		return nil, apperr.ErrNotCaptain
	}

	if team.IsBanned {
		return nil, apperr.ErrTeamBanned
	}

	return team, nil
}

const defaultInviteTokenTTL = 7 * 24 * time.Hour

// RegenerateInviteToken rotates the team invite token inside a transaction and returns the updated team.
func (uc *TeamUseCase) RegenerateInviteToken(ctx context.Context, captainID uuid.UUID) (*domain.Team, error) {
	var team *domain.Team

	err := uc.deps.TM.Run(ctx, func(txCtx context.Context) error {
		comp, err := uc.deps.CompRepo.Get(txCtx)
		if err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - CompetitionRepo.Get: %w", err)
		}

		if err := guard.ValidateTeamSwitchState(comp); err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - ValidateTeamSwitchState: %w", err)
		}

		if err := uc.deps.UserRepo.Lock(txCtx, captainID); err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - UserRepo.Lock: %w", err)
		}

		user, err := uc.deps.UserRepo.GetByID(txCtx, captainID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - UserRepo.GetByID: %w", err)
		}

		if user.IsBanned {
			return apperr.ErrUserBanned
		}

		if user.TeamID == nil {
			return apperr.ErrTeamNotFound
		}

		if err := uc.deps.TeamRepo.Lock(txCtx, *user.TeamID); err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - TeamRepo.Lock: %w", err)
		}

		var errTeam error

		team, errTeam = uc.deps.TeamRepo.GetByID(txCtx, *user.TeamID)
		if errTeam != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - TeamRepo.GetByID: %w", errTeam)
		}

		if team.CaptainID != captainID {
			return apperr.ErrNotCaptain
		}

		if team.IsSolo {
			return apperr.ErrTeamNotFound
		}

		if team.IsBanned {
			return apperr.ErrTeamBanned
		}

		newToken := uuid.New()

		expiresAt := time.Now().Add(defaultInviteTokenTTL)
		if err := uc.deps.TeamRepo.UpdateInviteToken(txCtx, team.ID, newToken, &expiresAt); err != nil {
			return fmt.Errorf("TeamUseCase - RegenerateInviteToken - TeamRepo.UpdateInviteToken: %w", err)
		}

		team.InviteToken = newToken
		team.InviteTokenExpiresAt = &expiresAt

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - RegenerateInviteToken - TM.Run: %w", err)
	}

	return team, nil
}
