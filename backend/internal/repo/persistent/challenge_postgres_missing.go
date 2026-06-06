package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func (r *ChallengeRepo) GetMissingChallengesByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.Challenge, error) {
	rows, err := r.Q(ctx).GetMissingChallengesByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetMissingChallengesByTeamID: %w", err)
	}

	out := make([]*domain.Challenge, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainChallenge(fieldsFromGetMissingByTeamID(row).toChallengeRow()))
	}

	return out, nil
}

func (r *ChallengeRepo) GetMissingChallengesByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Challenge, error) {
	rows, err := r.Q(ctx).GetMissingChallengesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetMissingChallengesByUserID: %w", err)
	}

	out := make([]*domain.Challenge, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainChallenge(fieldsFromGetMissingByUserID(row).toChallengeRow()))
	}

	return out, nil
}
