package challenge

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

const maxSolutionContentLen = 524288

func (uc *ChallengeUseCase) AdminUpsertSolution(ctx context.Context, challengeID uuid.UUID, params usecase.ChallengeSolutionUpsertParams) (*domain.ChallengeSolution, error) {
	if utf8.RuneCountInString(params.Content) > maxSolutionContentLen {
		return nil, apperr.NewValidationErrorf("solution content exceeds maximum length")
	}

	state, err := parseRequestedSolutionState(params.State)
	if err != nil {
		return nil, err
	}

	if _, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - AdminUpsertSolution - ChallengeRepo.GetByID: %w", err)
	}

	if params.State == "" {
		state, err = uc.resolveExistingSolutionState(ctx, challengeID)
		if err != nil {
			return nil, err
		}
	}

	solution, err := uc.deps.ChallengeRepo.UpsertSolution(ctx, challengeID, params.Content, state)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - AdminUpsertSolution - ChallengeRepo.UpsertSolution: %w", err)
	}

	return solution, nil
}

func parseRequestedSolutionState(requestedState string) (string, error) {
	state := domain.SolutionStateOrDefault(requestedState)
	if requestedState != "" {
		if state != requestedState {
			return "", apperr.NewValidationErrorf("solution state must be hidden, solved_only, after_event, or admin_only")
		}

		return state, nil
	}

	return state, nil
}

func (uc *ChallengeUseCase) resolveExistingSolutionState(ctx context.Context, challengeID uuid.UUID) (string, error) {
	existing, err := uc.deps.ChallengeRepo.GetSolution(ctx, challengeID)
	if err != nil {
		if errors.Is(err, apperr.ErrChallengeNotFound) {
			return domain.SolutionStateSolvedOnly, nil
		}

		return "", fmt.Errorf("ChallengeUseCase - AdminUpsertSolution - ChallengeRepo.GetSolution: %w", err)
	}

	return domain.SolutionStateOrDefault(existing.State), nil
}

func (uc *ChallengeUseCase) AdminDeleteSolution(ctx context.Context, challengeID uuid.UUID) error {
	err := uc.deps.ChallengeRepo.DeleteSolution(ctx, challengeID)
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - AdminDeleteSolution - ChallengeRepo.DeleteSolution: %w", err)
	}

	return nil
}

// AdminCreateSolve records a solve on behalf of an admin inside a transaction
// with row-level locks on the user, team, solve, and challenge rows. It checks team and user ban
// status, rejects non-admin users marked as former members of banned teams,
// enforces competition mode constraints (solo/teams/min-size), and optionally
// skips the submission-window check when skipCompetitionCheck is true. Before inserting it
// reads the solve row with FOR UPDATE to provide idempotency: if the solve already exists
// the transaction returns nil without duplicating it. Dynamic score decay is applied via
// ApplySolveScore and PointsAtSolve is stored on the new solve record.
func (uc *ChallengeUseCase) AdminCreateSolve(ctx context.Context, userID, teamID, challengeID uuid.UUID, skipCompetitionCheck bool) error {
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if uc.deps.UserRepo != nil {
			err := uc.deps.UserRepo.Lock(ctx, userID)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - UserRepo.Lock: %w", err)
			}
		}

		var team *domain.Team

		if uc.deps.TeamRepo != nil {
			if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
				return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - TeamRepo.Lock: %w", err)
			}

			var err error

			team, err = uc.deps.TeamRepo.GetByID(ctx, teamID)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - TeamRepo.GetByID: %w", err)
			}

			if team.IsBanned {
				return apperr.ErrTeamBanned
			}
		}

		solvedChallenge, err := uc.deps.ChallengeRepo.GetByIDForUpdate(ctx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - ChallengeRepo.GetByIDForUpdate: %w", err)
		}

		var user *domain.User

		if uc.deps.UserRepo != nil {
			user, err = uc.deps.UserRepo.GetByID(ctx, userID)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - UserRepo.GetByID: %w", err)
			}

			if user.TeamID == nil || *user.TeamID != teamID {
				return apperr.ErrUserNotInTeam
			}

			if user.IsBanned {
				return apperr.ErrUserBanned
			}

			if user.WasInBannedTeam && user.Role != domain.RoleAdmin {
				return apperr.ErrUserWasInBannedTeam
			}
		}

		if uc.deps.CompRepo != nil {
			comp, err := uc.deps.CompRepo.Get(ctx)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - CompetitionRepo.Get: %w", err)
			}

			if comp != nil {
				if !skipCompetitionCheck && !comp.IsSubmissionAllowed() {
					return apperr.ErrSubmissionNotAllowed
				}

				if err := guard.ValidateSubmissionEligibility(ctx, user, team, comp, uc.deps.TeamRepo); err != nil {
					return err
				}
			}
		}

		if _, err := uc.deps.SolveRepo.GetByTeamAndChallengeForUpdate(ctx, teamID, challengeID); err == nil {
			return nil
		} else if !errors.Is(err, apperr.ErrSolveNotFound) {
			return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - SolveRepo.GetByTeamAndChallengeForUpdate: %w", err)
		}

		solveCount, err := uc.deps.ChallengeRepo.IncrementSolveCount(ctx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - ChallengeRepo.IncrementSolveCount: %w", err)
		}

		pointsAtSolve, err := scoring.ApplySolveScore(ctx,
			solvedChallenge.InitialValue, solvedChallenge.MinValue, solvedChallenge.Decay, solvedChallenge.Points, solveCount,
			scoring.GetDecayFn(ctx, uc.deps.CompParamUC),
			func(ctx context.Context, pts int) error {
				err := uc.deps.ChallengeRepo.UpdatePoints(ctx, challengeID, pts)
				if err != nil {
					return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - ChallengeRepo.UpdatePoints: %w", err)
				}

				solvedChallenge.Points = pts

				return nil
			},
		)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - ApplySolveScore: %w", err)
		}

		solve := &domain.Solve{UserID: userID, TeamID: teamID, ChallengeID: challengeID, PointsAtSolve: pointsAtSolve}
		if err = uc.deps.SolveRepo.Create(ctx, solve); err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - SolveRepo.Create: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - TM.Run: %w", err)
	}

	uc.submitInvalidateCache(ctx, teamID)
	uc.invalidateStatisticsCache(ctx, "AdminCreateSolve")

	return nil
}

// RecalcAllDynamicPoints recalculates solve counts and points for every challenge that
// uses dynamic scoring (initial_value > 0 and decay > 0). Safe to run at any time:
// it reads the current solve table and derives the correct point values from scratch.
// Useful to heal inconsistent state left by old code or manual data imports.
func (uc *ChallengeUseCase) RecalcAllDynamicPoints(ctx context.Context) error {
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		ids, err := uc.deps.ChallengeRepo.GetAllDynamicIDs(ctx)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - RecalcAllDynamicPoints - ChallengeRepo.GetAllDynamicIDs: %w", err)
		}

		if len(ids) == 0 {
			return nil
		}

		var (
			getSolves   func(context.Context, []uuid.UUID) ([]*scoring.SolveRowForPointsRecalc, error)
			batchUpdate func(context.Context, []uuid.UUID, []int) error
		)

		if uc.deps.SolveRepo != nil {
			getSolves = scoring.MapSolvesForRecalcFn(
				uc.deps.SolveRepo.GetSolvesForPointsRecalc,
				scoring.DefaultSolveMapper,
				"ChallengeUseCase - RecalcAllDynamicPoints",
			)
			batchUpdate = uc.deps.SolveRepo.BatchUpdateSolvePoints
		}

		return scoring.AdjustDynamicScores(
			ctx, ids, uc.deps.ChallengeRepo,
			getSolves, batchUpdate,
			scoring.GetDecayFn(ctx, uc.deps.CompParamUC),
		)
	})
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - RecalcAllDynamicPoints - TM.Run: %w", err)
	}

	uc.InvalidateChallengeListCache(ctx)
	uc.InvalidateScoreboardCache(ctx)
	uc.invalidateStatisticsCache(ctx, "RecalcAllDynamicPoints")

	return nil
}

// AdminDeleteSolve removes a solve record inside a transaction. It reads the
// solve and challenge with FOR UPDATE to lock them, then deletes the solve and decrements the challenge's
// solve count. If the challenge uses dynamic scoring (InitialValue > 0 and Decay > 0) it
// recalculates the current challenge points via submitRecordSolveUpdatePointsIfDecay, then
// fetches all remaining solves and calls scoring.RecalculatePointsAtSolveRows to recompute
// the historical PointsAtSolve values for each previous solver, persisting the new values
// in a single BatchUpdateSolvePoints call.
func (uc *ChallengeUseCase) AdminDeleteSolve(ctx context.Context, teamID, challengeID uuid.UUID) error {
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		_, err := uc.deps.SolveRepo.GetByTeamAndChallengeForUpdate(ctx, teamID, challengeID)
		if err != nil {
			if errors.Is(err, apperr.ErrSolveNotFound) {
				return err
			}

			return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - SolveRepo.GetByTeamAndChallengeForUpdate: %w", err)
		}

		solvedChallenge, err := uc.deps.ChallengeRepo.GetByIDForUpdate(ctx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - ChallengeRepo.GetByIDForUpdate: %w", err)
		}

		if err = uc.deps.SolveRepo.DeleteByTeamAndChallenge(ctx, teamID, challengeID); err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - SolveRepo.DeleteByTeamAndChallenge: %w", err)
		}

		solveCount, err := uc.deps.ChallengeRepo.DecrementSolveCount(ctx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - ChallengeRepo.DecrementSolveCount: %w", err)
		}

		if err := uc.submitRecordSolveUpdatePointsIfDecay(ctx, challengeID, solvedChallenge, solveCount); err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - submitRecordSolveUpdatePointsIfDecay: %w", err)
		}

		if solvedChallenge.InitialValue > 0 && solvedChallenge.Decay > 0 && uc.deps.SolveRepo != nil {
			getSolves := scoring.MapSolvesForRecalcFn(
				uc.deps.SolveRepo.GetSolvesForPointsRecalc,
				scoring.DefaultSolveMapper,
				"ChallengeUseCase - AdminDeleteSolve",
			)

			recalcRows, err := getSolves(ctx, []uuid.UUID{challengeID})
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - GetSolvesForPointsRecalc: %w", err)
			}

			solveIDs, newPoints := scoring.RecalculatePointsAtSolveRows(recalcRows, scoring.GetDecayFn(ctx, uc.deps.CompParamUC))
			if len(solveIDs) > 0 {
				if err := uc.deps.SolveRepo.BatchUpdateSolvePoints(ctx, solveIDs, newPoints); err != nil {
					return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - SolveRepo.BatchUpdateSolvePoints: %w", err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - TM.Run: %w", err)
	}

	uc.submitInvalidateCache(ctx, teamID)
	uc.invalidateStatisticsCache(ctx, "AdminDeleteSolve")

	return nil
}
