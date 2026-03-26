package challenge

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/scoring"
)

const maxSolutionContentLen = 524288

func (uc *ChallengeUseCase) AdminUpsertSolution(ctx context.Context, challengeID uuid.UUID, content string) (*domain.ChallengeSolution, error) {
	if utf8.RuneCountInString(content) > maxSolutionContentLen {
		return nil, httperr.NewValidationErrorf("solution content exceeds maximum length")
	}

	if _, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - AdminUpsertSolution - ChallengeRepo.GetByID: %w", err)
	}

	solution, err := uc.deps.ChallengeRepo.UpsertSolution(ctx, challengeID, content)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - AdminUpsertSolution - ChallengeRepo.UpsertSolution: %w", err)
	}

	return solution, nil
}

func (uc *ChallengeUseCase) AdminDeleteSolution(ctx context.Context, challengeID uuid.UUID) error {
	err := uc.deps.ChallengeRepo.DeleteSolution(ctx, challengeID)
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - AdminDeleteSolution - ChallengeRepo.DeleteSolution: %w", err)
	}

	return nil
}

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
				return httperr.ErrTeamBanned
			}
		}

		if uc.deps.CompRepo != nil {
			comp, err := uc.deps.CompRepo.Get(ctx)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - CompetitionRepo.Get: %w", err)
			}

			if comp != nil {
				if !skipCompetitionCheck && !comp.IsSubmissionAllowed() {
					return httperr.ErrSubmissionNotAllowed
				}

				if team != nil {
					if comp.Mode == domain.ModeTeamsOnly && team.IsSolo {
						return httperr.ErrTeamModeRequired
					}

					if comp.Mode == domain.ModeSoloOnly && !team.IsSolo {
						return httperr.ErrSoloModeRequired
					}

					if comp.MinTeamSize > 0 && !team.IsSolo {
						count, err := uc.deps.TeamRepo.CountTeamMembers(ctx, teamID)
						if err != nil {
							return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - TeamRepo.CountTeamMembers: %w", err)
						}

						if count < comp.MinTeamSize {
							return httperr.ErrTeamBelowMinSize
						}
					}
				}
			}
		}

		solvedChallenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - ChallengeRepo.GetByID: %w", err)
		}

		if uc.deps.UserRepo != nil {
			user, err := uc.deps.UserRepo.GetByID(ctx, userID)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - UserRepo.GetByID: %w", err)
			}

			if user.TeamID == nil || *user.TeamID != teamID {
				return httperr.ErrUserNotInTeam
			}

			if user.IsBanned {
				return httperr.ErrUserBanned
			}
		}

		if _, err := uc.deps.SolveRepo.GetByTeamAndChallengeForUpdate(ctx, teamID, challengeID); err == nil {
			return nil
		} else if !errors.Is(err, httperr.ErrSolveNotFound) {
			return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - SolveRepo.GetByTeamAndChallengeForUpdate: %w", err)
		}

		solveCount, err := uc.deps.ChallengeRepo.IncrementSolveCount(ctx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - ChallengeRepo.IncrementSolveCount: %w", err)
		}

		pointsAtSolve, err := scoring.ApplySolveScore(ctx,
			solvedChallenge.InitialValue, solvedChallenge.MinValue, solvedChallenge.Decay, solvedChallenge.Points, solveCount,
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

	return nil
}

func (uc *ChallengeUseCase) AdminDeleteSolve(ctx context.Context, teamID, challengeID uuid.UUID) error {
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		_, err := uc.deps.SolveRepo.GetByTeamAndChallengeForUpdate(ctx, teamID, challengeID)
		if err != nil {
			if errors.Is(err, httperr.ErrSolveNotFound) {
				return err
			}

			return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - SolveRepo.GetByTeamAndChallengeForUpdate: %w", err)
		}

		solvedChallenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - ChallengeRepo.GetByID: %w", err)
		}

		if err = uc.deps.SolveRepo.DeleteByTeamAndChallenge(ctx, teamID, challengeID); err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - SolveRepo.DeleteByTeamAndChallenge: %w", err)
		}

		solveCount, err := uc.deps.ChallengeRepo.DecrementSolveCount(ctx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - ChallengeRepo.DecrementSolveCount: %w", err)
		}

		if err := uc.submitRecordSolveUpdatePointsIfDecay(ctx, challengeID, solvedChallenge, solveCount); err != nil {
			return err
		}

		if solvedChallenge.InitialValue > 0 && solvedChallenge.Decay > 0 && uc.deps.SolveRepo != nil {
			rows, err := uc.deps.SolveRepo.GetSolvesForPointsRecalc(ctx, []uuid.UUID{challengeID})
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - SolveRepo.GetSolvesForPointsRecalc: %w", err)
			}

			recalcRows := make([]*scoring.SolveRowForPointsRecalc, 0, len(rows))
			for _, r := range rows {
				recalcRows = append(recalcRows, &scoring.SolveRowForPointsRecalc{
					ID: r.ID, ChallengeID: r.ChallengeID,
					InitialValue: r.InitialValue, MinValue: r.MinValue, Decay: r.Decay,
				})
			}

			solveIDs, newPoints := scoring.RecalculatePointsAtSolveRows(recalcRows)
			if len(solveIDs) > 0 {
				err := uc.deps.SolveRepo.BatchUpdateSolvePoints(ctx, solveIDs, newPoints)
				if err != nil {
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

	return nil
}
