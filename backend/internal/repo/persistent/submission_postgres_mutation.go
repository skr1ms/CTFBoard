package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

func (r *SubmissionRepo) Update(ctx context.Context, ID uuid.UUID, isCorrect bool) error {
	err := r.Q(ctx).UpdateSubmission(ctx, sqlc.UpdateSubmissionParams{
		ID:        ID,
		IsCorrect: isCorrect,
	})
	if err != nil {
		return fmt.Errorf("SubmissionRepo - Update: %w", err)
	}

	return nil
}

// Delete removes a submission by ID. Idempotent: returns nil if the submission does not exist.
func (r *SubmissionRepo) Discard(ctx context.Context, ID uuid.UUID) error {
	err := r.Q(ctx).DiscardSubmission(ctx, ID)
	if err != nil {
		return fmt.Errorf("SubmissionRepo - Discard: %w", err)
	}

	return nil
}

func (r *SubmissionRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	err := r.Q(ctx).DeleteSubmission(ctx, ID)
	if err != nil {
		return fmt.Errorf("SubmissionRepo - Delete: %w", err)
	}

	return nil
}

func (r *SubmissionRepo) DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error {
	err := r.Q(ctx).DeleteSubmissionsByTeamID(ctx, &teamID)
	if err != nil {
		return fmt.Errorf("SubmissionRepo - DeleteByTeamID: %w", err)
	}

	return nil
}

func (r *SubmissionRepo) SoftBanByTeamID(ctx context.Context, teamID uuid.UUID) error {
	err := r.Q(ctx).SoftBanSubmissionsByTeamID(ctx, &teamID)
	if err != nil {
		return fmt.Errorf("SubmissionRepo - SoftBanByTeamID: %w", err)
	}

	return nil
}

func (r *SubmissionRepo) RestoreByBannedTeamID(ctx context.Context, teamID uuid.UUID) error {
	err := r.Q(ctx).RestoreSubmissionsByBannedTeamID(ctx, &teamID)
	if err != nil {
		return fmt.Errorf("SubmissionRepo - RestoreByBannedTeamID: %w", err)
	}

	return nil
}

func (r *SubmissionRepo) SoftBanByUserID(ctx context.Context, userID uuid.UUID) error {
	err := r.Q(ctx).SoftBanSubmissionsByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("SubmissionRepo - SoftBanByUserID: %w", err)
	}

	return nil
}

func (r *SubmissionRepo) RestoreByBannedUserID(ctx context.Context, userID uuid.UUID) error {
	err := r.Q(ctx).RestoreSubmissionsByBannedUserID(ctx, &userID)
	if err != nil {
		return fmt.Errorf("SubmissionRepo - RestoreByBannedUserID: %w", err)
	}

	return nil
}
