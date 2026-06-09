package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

const challengeRequirementsGraphLockKey int64 = 0x415354524f435452

func (r *ChallengeRepo) GetRequirements(ctx context.Context, ID uuid.UUID) ([]*domain.ChallengeRequirement, error) {
	rows, err := r.Q(ctx).GetChallengeRequirements(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetRequirements: %w", err)
	}

	out := make([]*domain.ChallengeRequirement, len(rows))
	for i, row := range rows {
		out[i] = &domain.ChallengeRequirement{
			ChallengeID:    row.ID,
			ChallengeTitle: row.Title,
			Category:       lo.EmptyableToPtr(row.Category),
		}
	}

	return out, nil
}

func (r *ChallengeRepo) GetRequirementsForEnforcement(ctx context.Context, ID uuid.UUID) ([]*domain.ChallengeRequirement, error) {
	rows, err := r.Q(ctx).GetChallengeRequirementsForEnforcement(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetRequirementsForEnforcement: %w", err)
	}

	out := make([]*domain.ChallengeRequirement, len(rows))
	for i, row := range rows {
		out[i] = &domain.ChallengeRequirement{
			ChallengeID:    row.ID,
			ChallengeTitle: row.Title,
			Category:       lo.EmptyableToPtr(row.Category),
		}
	}

	return out, nil
}

func (r *ChallengeRepo) GetAllRequirementPairs(ctx context.Context) ([]*domain.ChallengeRequirementPair, error) {
	rows, err := r.Q(ctx).GetAllChallengeRequirements(ctx)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetAllRequirementPairs: %w", err)
	}

	out := make([]*domain.ChallengeRequirementPair, len(rows))
	for i, row := range rows {
		out[i] = &domain.ChallengeRequirementPair{
			ChallengeID:         row.ChallengeID,
			RequiredChallengeID: row.RequiredChallengeID,
		}
	}

	return out, nil
}

func (r *ChallengeRepo) AcquireRequirementsLock(ctx context.Context) error {
	return r.AcquireAdvisoryLock(ctx, challengeRequirementsGraphLockKey)
}

// SetRequirements replaces the full prerequisite set for a challenge atomically:
// deletes all existing rows for challengeID, then inserts the new requirementIDs
// with ON CONFLICT DO NOTHING. Must be called inside a transaction to avoid gaps
// if the caller holds a lock on the challenge row.
func (r *ChallengeRepo) SetRequirements(ctx context.Context, challengeID uuid.UUID, requirementIDs []uuid.UUID) error {
	if err := requireTx(ctx, "ChallengeRepo - SetRequirements"); err != nil {
		return err
	}

	if err := r.Q(ctx).DeleteChallengeRequirements(ctx, challengeID); err != nil {
		return fmt.Errorf("ChallengeRepo - SetRequirements - Delete: %w", err)
	}

	if len(requirementIDs) == 0 {
		return nil
	}

	if err := r.Q(ctx).InsertChallengeRequirements(ctx, sqlc.InsertChallengeRequirementsParams{
		ChallengeID:          challengeID,
		RequiredChallengeIds: requirementIDs,
	}); err != nil {
		return fmt.Errorf("ChallengeRepo - SetRequirements - Insert: %w", err)
	}

	return nil
}
