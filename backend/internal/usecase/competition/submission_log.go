package competition

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func (uc *SubmissionUseCase) LogSubmission(ctx context.Context, sub *domain.Submission) error {
	if err := uc.deps.SubmissionRepo.Create(ctx, sub); err != nil {
		return fmt.Errorf("SubmissionUseCase - LogSubmission - SubmissionRepo.Create: %w", err)
	}

	return nil
}

func (uc *SubmissionUseCase) LogRateLimited(ctx context.Context, userID, teamID, challengeID uuid.UUID, ip string) error {
	sub := &domain.Submission{
		UserID:        userID,
		TeamID:        &teamID,
		ChallengeID:   challengeID,
		SubmittedFlag: "",
		IsCorrect:     false,
		Type:          domain.SubmissionTypeRatelimited,
		IP:            ip,
		CreatedAt:     time.Now(),
	}

	if err := uc.deps.SubmissionRepo.Create(ctx, sub); err != nil {
		return fmt.Errorf("SubmissionUseCase - LogRateLimited - SubmissionRepo.Create: %w", err)
	}

	return nil
}
