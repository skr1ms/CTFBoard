package challenge

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

// submitRecordSolve atomically records a correct solve inside a SERIALIZABLE transaction
// The strategy inside the transaction is
//  1. Re-read the competition state (GetForUpdate) to detect freeze and re-check the
//     submission window; returns ErrSubmissionNotAllowed if the window has just closed
//  2. Lock and re-validate user (still in the team, not banned) and team (not banned,
//     mode/min-size constraints) rows with SELECT … FOR UPDATE to prevent concurrent
//     membership changes from producing an inconsistent solve record
//  3. Lock the challenge row (GetByIDForUpdate) and re-check hidden/locked state and
//     prerequisites - all inside the same transaction for idempotency
//  4. If MaxAttempts is set, acquires a second advisory lock keyed on (teamID, challengeID)
//     to prevent two concurrent correct submissions from both slipping under the cap
//     if the cap is exceeded and a solve already exists the call is treated as idempotent
//     (alreadySolved = true)
//  5. Inserts the correct submission record, then delegates to deps.SolveRecord
//     which handles idempotency (returns alreadySolved), dynamic score decay, and
//     first-blood detection
//
// Returns the locked challenge, the new solve count, an alreadySolved flag, a wasFrozen
// flag (used by the caller to decide which cache keys to invalidate), and any error.
func (uc *ChallengeUseCase) submitRecordSolve(sc *submitContext, _ *domain.Challenge) (*domain.Challenge, int, bool, bool, error) {
	var (
		solvedChallenge *domain.Challenge
		solveCount      int
		alreadySolved   bool
		wasFrozen       bool
	)

	err := uc.deps.TM.Run(sc.ctx, func(ctx context.Context) error {
		var (
			comp      *domain.Competition
			freshUser *domain.User
		)

		if uc.deps.CompRepo != nil {
			c, err := uc.deps.CompRepo.GetForUpdate(ctx)
			if err != nil && !errors.Is(err, apperr.ErrCompetitionNotFound) {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - CompRepo.GetForUpdate: %w", err)
			}

			comp = c
			if comp != nil {
				wasFrozen = comp.IsFreezeActive()
			}
		}

		if comp != nil && !comp.IsSubmissionAllowed() {
			return apperr.ErrSubmissionNotAllowed
		}

		if uc.deps.UserRepo != nil {
			if err := uc.deps.UserRepo.Lock(ctx, sc.userID); err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - UserRepo.Lock: %w", err)
			}

			freshUser, err := uc.deps.UserRepo.GetByID(ctx, sc.userID)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - UserRepo.GetByID: %w", err)
			}

			if freshUser.TeamID == nil || *freshUser.TeamID != sc.teamID {
				return apperr.ErrTeamMemberNotFound
			}

			if freshUser.IsBanned {
				return apperr.ErrUserBanned
			}

			if freshUser.WasInBannedTeam && freshUser.Role != domain.RoleAdmin {
				return apperr.ErrUserWasInBannedTeam
			}
		}

		if uc.deps.TeamRepo != nil {
			if err := uc.deps.TeamRepo.Lock(ctx, sc.teamID); err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - TeamRepo.Lock: %w", err)
			}

			freshTeam, err := uc.deps.TeamRepo.GetByID(ctx, sc.teamID)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - TeamRepo.GetByID: %w", err)
			}

			if err := guard.ValidateSubmissionEligibility(ctx, freshUser, freshTeam, comp, uc.deps.TeamRepo); err != nil {
				return err
			}
		}

		freshChallenge, err := uc.deps.ChallengeRepo.GetByIDForUpdate(ctx, sc.challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - submitRecordSolve - ChallengeRepo.GetByIDForUpdate: %w", err)
		}

		if err := guard.EnsureChallengeVisible(freshChallenge); err != nil {
			return err
		}

		if freshChallenge.State == domain.ChallengeStateLocked {
			return apperr.ErrChallengeLocked
		}

		if err := uc.submitCheckRequirementsInTx(ctx, sc.challengeID, sc.teamID); err != nil {
			return err
		}

		if freshChallenge.MaxAttempts > 0 && uc.deps.SubmissionRepo != nil {
			if err := uc.deps.SubmissionRepo.AcquireAdvisoryLockForSubmit(ctx, sc.teamID, sc.challengeID); err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - AcquireAdvisoryLockForSubmit: %w", err)
			}

			count, err := uc.countAttempts(ctx, sc.teamID, sc.challengeID, freshChallenge.MaxAttemptsWindow)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - countAttempts: %w", err)
			}

			if count >= int64(freshChallenge.MaxAttempts) {
				if uc.deps.SolveRepo != nil {
					_, solveErr := uc.deps.SolveRepo.GetByTeamAndChallenge(ctx, sc.teamID, sc.challengeID)
					if solveErr == nil {
						solvedChallenge = freshChallenge
						alreadySolved = true

						return nil
					}
				}

				return apperr.ErrMaxAttemptsReached
			}
		}

		correctSub := &domain.Submission{
			UserID:        sc.userID,
			TeamID:        &sc.teamID,
			ChallengeID:   sc.challengeID,
			SubmittedFlag: sc.flag,
			IsCorrect:     true,
			Type:          domain.SubmissionTypeCorrect,
			IP:            sc.clientIP,
			CreatedAt:     time.Now(),
		}

		if uc.deps.SubmissionRepo != nil {
			err := uc.deps.SubmissionRepo.Create(ctx, correctSub)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - SubmissionRepo.Create: %w", err)
			}
		}

		solvedChallenge = freshChallenge
		solve := &domain.Solve{UserID: sc.userID, TeamID: sc.teamID, ChallengeID: sc.challengeID}

		if uc.deps.SolveRecord == nil {
			return fmt.Errorf("ChallengeUseCase - submitRecordSolve: SolveRecord not configured")
		}

		solveCount, err = uc.deps.SolveRecord(ctx, solve, freshChallenge, uc.deps.ChallengeRepo, uc.deps.SolveRepo, scoring.GetDecayFn(ctx, uc.deps.CompParamUC))
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - submitRecordSolve - RecordSolveInTx: %w", err)
		}

		return nil
	})
	if err != nil {
		var ve *apperr.ValidationError
		if errors.As(err, &ve) || errors.Is(err, apperr.ErrMaxAttemptsReached) || errors.Is(err, apperr.ErrAlreadySolved) {
			return nil, 0, false, false, err
		}

		return nil, 0, false, false, fmt.Errorf("ChallengeUseCase - submitRecordSolve - TM.Run: %w", err)
	}

	return solvedChallenge, solveCount, alreadySolved, wasFrozen, nil
}

// submitRecordSolveUpdatePointsIfDecay applies dynamic score decay after recording a solve.
// When the new score differs from the current score, updates the challenge points in the DB.
func (uc *ChallengeUseCase) submitRecordSolveUpdatePointsIfDecay(ctx context.Context, challengeID uuid.UUID, solvedChallenge *domain.Challenge, solveCount int) error {
	_, err := scoring.ApplySolveScore(ctx,
		solvedChallenge.InitialValue, solvedChallenge.MinValue, solvedChallenge.Decay, solvedChallenge.Points, solveCount,
		scoring.GetDecayFn(ctx, uc.deps.CompParamUC),
		func(ctx context.Context, pts int) error {
			err := uc.deps.ChallengeRepo.UpdatePoints(ctx, challengeID, pts)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolveUpdatePointsIfDecay - ChallengeRepo.UpdatePoints: %w", err)
			}

			solvedChallenge.Points = pts

			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - submitRecordSolveUpdatePointsIfDecay - ApplySolveScore: %w", err)
	}

	return nil
}
