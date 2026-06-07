package challenge

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

func ensureSolutionPolicyAccess(
	ctx context.Context,
	state string,
	challengeID uuid.UUID,
	teamID *uuid.UUID,
	isAdmin bool,
	settingsRepo repo.SettingsRepository,
	compRepo repo.CompetitionRepository,
	solveRepo repo.SolveRepository,
	deniedErr error,
) error {
	if isAdmin {
		return nil
	}

	writeupEnabled, err := loadWriteupEnabled(ctx, settingsRepo)
	if err != nil {
		return err
	}

	if !writeupEnabled {
		return apperr.ErrWriteupsDisabled
	}

	normalizedState := domain.SolutionStateOrDefault(state)
	switch normalizedState {
	case domain.SolutionStateHidden, domain.SolutionStateAdminOnly:
		return deniedErr
	}

	eventEnded := false

	if normalizedState == domain.SolutionStateAfterEvent {
		var err error

		eventEnded, err = loadCompetitionEnded(ctx, compRepo)
		if err != nil {
			return err
		}
	}

	solved, err := hasTeamSolvedChallenge(ctx, solveRepo, teamID, challengeID)
	if err != nil {
		return err
	}

	if domain.CanViewSolution(normalizedState, solved, eventEnded, writeupEnabled, isAdmin) {
		return nil
	}

	return deniedErr
}

func loadWriteupEnabled(ctx context.Context, settingsRepo repo.SettingsRepository) (bool, error) {
	if settingsRepo == nil {
		return true, nil
	}

	settings, err := settingsRepo.Get(ctx)
	if err != nil {
		return false, fmt.Errorf("SettingsRepo.Get: %w", err)
	}

	return settings.WriteupEnabled, nil
}

func loadCompetitionEnded(ctx context.Context, compRepo repo.CompetitionRepository) (bool, error) {
	if compRepo == nil {
		return false, nil
	}

	comp, err := compRepo.Get(ctx)
	if err != nil {
		return false, fmt.Errorf("CompetitionRepo.Get: %w", err)
	}

	if comp == nil {
		return false, nil
	}

	return comp.GetStatus() == domain.CompetitionStatusEnded, nil
}

func hasTeamSolvedChallenge(ctx context.Context, solveRepo repo.SolveRepository, teamID *uuid.UUID, challengeID uuid.UUID) (bool, error) {
	if teamID == nil || solveRepo == nil {
		return false, nil
	}

	_, err := solveRepo.GetByTeamAndChallenge(ctx, *teamID, challengeID)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, apperr.ErrSolveNotFound) {
		return false, nil
	}

	return false, fmt.Errorf("SolveRepo.GetByTeamAndChallenge: %w", err)
}
