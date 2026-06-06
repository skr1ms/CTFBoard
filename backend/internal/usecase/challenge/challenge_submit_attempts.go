package challenge

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// submitCheckRequirementsInTx verifies that the team has solved all prerequisite challenges.
// Called inside the submit transaction so the check is consistent with the advisory lock.
func (uc *ChallengeUseCase) submitCheckRequirementsInTx(ctx context.Context, challengeID, teamID uuid.UUID) error {
	requirements, err := uc.deps.ChallengeRepo.GetRequirements(ctx, challengeID)
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - submitRecordSolve - GetRequirements: %w", err)
	}

	if uc.deps.SolveRepo == nil || len(requirements) == 0 {
		return nil
	}

	requirementIDs := make([]uuid.UUID, 0, len(requirements))
	for _, req := range requirements {
		requirementIDs = append(requirementIDs, req.ChallengeID)
	}

	solvedIDs, err := uc.deps.SolveRepo.GetSolvedChallengeIDsByTeam(ctx, teamID, requirementIDs)
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - submitRecordSolve - GetSolvedChallengeIDsByTeam: %w", err)
	}

	solvedSet := make(map[uuid.UUID]struct{}, len(solvedIDs))
	for _, id := range solvedIDs {
		solvedSet[id] = struct{}{}
	}

	for _, req := range requirements {
		if _, ok := solvedSet[req.ChallengeID]; !ok {
			return apperr.ErrRequirementsNotMet
		}
	}

	return nil
}

// countAttempts returns the number of submissions by the team for the challenge.
// When window > 0 only submissions within the rolling time window are counted
// (rate-limiting semantics); otherwise the total lifetime count is returned.
func (uc *ChallengeUseCase) countAttempts(ctx context.Context, teamID, challengeID uuid.UUID, window time.Duration) (int64, error) {
	if window > 0 {
		windowStart := time.Now().Add(-window)

		return uc.deps.SubmissionRepo.CountSubmissionsByTeamAndChallengeInWindow(ctx, teamID, challengeID, windowStart)
	}

	return uc.deps.SubmissionRepo.CountSubmissionsByTeamAndChallenge(ctx, teamID, challengeID)
}

// submitLogIncorrectAndEnforceMaxAttempts records an incorrect submission and enforces the
// per-team, per-challenge attempt limit. When MaxAttempts is configured it wraps both
// operations in a serializable transaction protected by an advisory lock so that the
// count read and the submission insert are atomic: a concurrent submission cannot sneak in
// between the read and the write. If the count already equals or exceeds MaxAttempts the
// transaction returns ErrMaxAttemptsReached without inserting. When MaxAttempts is not
// configured the submission is written directly without a transaction.
func (uc *ChallengeUseCase) submitLogIncorrectAndEnforceMaxAttempts(sc *submitContext, challenge *domain.Challenge) error {
	sub := &domain.Submission{
		UserID:        sc.userID,
		TeamID:        &sc.teamID,
		ChallengeID:   sc.challengeID,
		SubmittedFlag: sc.flag,
		IsCorrect:     false,
		Type:          domain.SubmissionTypeIncorrect,
		IP:            sc.clientIP,
		CreatedAt:     time.Now(),
	}

	if challenge.MaxAttempts > 0 && uc.deps.SubmissionRepo != nil && uc.deps.TM != nil {
		err := uc.deps.TM.Run(sc.ctx, func(ctx context.Context) error {
			if err := uc.deps.SubmissionRepo.AcquireAdvisoryLockForSubmit(ctx, sc.teamID, sc.challengeID); err != nil {
				return fmt.Errorf("ChallengeUseCase - submitLogIncorrectAndEnforceMaxAttempts - AcquireAdvisoryLockForSubmit: %w", err)
			}

			count, err := uc.countAttempts(ctx, sc.teamID, sc.challengeID, challenge.MaxAttemptsWindow)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - submitLogIncorrectAndEnforceMaxAttempts - countAttempts: %w", err)
			}

			if count >= int64(challenge.MaxAttempts) {
				return apperr.ErrMaxAttemptsReached
			}

			return uc.deps.SubmissionRepo.Create(ctx, sub)
		})
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - submitLogIncorrectAndEnforceMaxAttempts - TM.Run: %w", err)
		}

		return nil
	}

	if uc.deps.SubmissionRepo != nil {
		return uc.deps.SubmissionRepo.Create(sc.ctx, sub)
	}

	return nil
}
