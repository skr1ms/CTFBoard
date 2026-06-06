package team

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

// TryCreate attempts to create a team for captainID. If the user is not yet in any
// team the creation proceeds immediately and the result contains the new team. If the
// user already belongs to a solo or auto-created team, the function returns a result
// with RequiresConfirm=true and a summary of the data that would be lost, giving the
// caller the opportunity to present a confirmation step before calling ConfirmCreate
// Hard errors (banned user, competition state, etc.) are still returned as errors.
func (uc *TeamUseCase) TryCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*usecase.TeamCreateResult, error) {
	comp, err := uc.deps.Guard.RequireTeamSwitchAndTeamsMode(ctx)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - TryCreate - Guard: %w", err)
	}

	if isSolo && !comp.Mode.AllowsSolo() {
		return nil, apperr.ErrSoloModeNotAllowed
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, captainID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - TryCreate - UserRepo.GetByID: %w", err)
	}

	if user.TeamID == nil {
		team, err := uc.Create(ctx, name, captainID, isSolo, false)
		if err != nil {
			return nil, fmt.Errorf("TeamUseCase - TryCreate - createTx: %w", err)
		}

		return &usecase.TeamCreateResult{Team: team}, nil
	}

	opResult, err := uc.tryCreateWhenInTeam(ctx, captainID)
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - TryCreate - tryCreateWhenInTeam: %w", err)
	}

	return opResult, nil
}

// tryCreateWhenInTeam handles the case where the user is already a member of a
// team when TryCreate is called. It locks the user row inside a transaction,
// re-fetches the membership state to close TOCTOU races, then inspects the old
// team. If the old team is a multi-member or regular (non-solo, non-auto) team
// the function returns ErrUserAlreadyInTeam immediately. For an eligible
// solo/auto-created single-member team it collects the affected data summary
// (points, solve count, awards, hint unlocks) and returns RequiresConfirm=true
// so the caller can present a confirmation step before proceeding to ConfirmCreate.
// No data is deleted here - actual cleanup happens in createTx -> handleSoloTeamCleanup.
func (uc *TeamUseCase) tryCreateWhenInTeam(ctx context.Context, captainID uuid.UUID) (*usecase.TeamCreateResult, error) {
	var result *usecase.TeamCreateResult

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if _, err := uc.deps.Guard.RequireTeamSwitchAndTeamsMode(ctx); err != nil {
			return fmt.Errorf("TeamUseCase - tryCreateWhenInTeam - Guard.RequireTeamSwitchAndTeamsMode: %w", err)
		}

		if err := uc.deps.UserRepo.Lock(ctx, captainID); err != nil {
			return fmt.Errorf("TeamUseCase - tryCreateWhenInTeam - UserRepo.Lock: %w", err)
		}

		freshUser, err := uc.deps.UserRepo.GetByID(ctx, captainID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - tryCreateWhenInTeam - UserRepo.GetByID: %w", err)
		}

		if freshUser.IsBanned {
			return apperr.ErrUserBanned
		}

		if freshUser.TeamID == nil {
			return apperr.ErrTeamNotFound
		}

		oldTeam, err := uc.deps.TeamRepo.GetByID(ctx, *freshUser.TeamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - tryCreateWhenInTeam - TeamRepo.GetByID: %w", err)
		}

		members, err := uc.deps.UserRepo.GetByTeamID(ctx, *freshUser.TeamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - tryCreateWhenInTeam - UserRepo.GetByTeamID: %w", err)
		}

		if !uc.shouldCleanupSoloTeam(freshUser, members, oldTeam) {
			return apperr.ErrUserAlreadyInTeam
		}

		points, err := uc.deps.SolveRepo.GetTeamScore(ctx, *freshUser.TeamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - tryCreateWhenInTeam - SolveRepo.GetTeamScore: %w", err)
		}

		solveCount := 0

		if uc.deps.SolveRepo != nil {
			if solves, err := uc.deps.SolveRepo.GetByTeamIDWithDetails(ctx, *freshUser.TeamID); err == nil {
				solveCount = len(solves)
			}
		}

		awardsTotal := 0

		if uc.deps.AwardRepo != nil {
			if total, err := uc.deps.AwardRepo.GetTeamTotalAwards(ctx, *freshUser.TeamID); err == nil {
				awardsTotal = total
			}
		}

		hintUnlockCount := 0

		if uc.deps.HintRepo != nil {
			if n, err := uc.deps.HintRepo.CountByTeamID(ctx, *freshUser.TeamID); err == nil {
				hintUnlockCount = n
			}
		}

		result = &usecase.TeamCreateResult{
			RequiresConfirm:    true,
			ConfirmationReason: usecase.ConfirmReasonSoloTeamReset,
			AffectedData:       &usecase.TeamCreateAffectedData{Points: points, SolveCount: solveCount, AwardsTotal: awardsTotal, HintUnlockCount: hintUnlockCount},
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - TryCreate - TM.Run: %w", err)
	}

	return result, nil
}

func (uc *TeamUseCase) ConfirmCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*domain.Team, error) {
	return uc.Create(ctx, name, captainID, isSolo, true)
}
