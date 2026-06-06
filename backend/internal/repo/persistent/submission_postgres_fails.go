package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

func (r *SubmissionRepo) CountFailedByIP(ctx context.Context, ip string, since time.Time) (int64, error) {
	n, err := r.Q(ctx).CountFailedSubmissionsByIP(ctx, sqlc.CountFailedSubmissionsByIPParams{
		IP:        ip,
		CreatedAt: pgutil.TimeToTimestamptz(&since),
	})
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountFailedByIP: %w", err)
	}

	return n, nil
}

func (r *SubmissionRepo) GetFailsByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetFailsByUser: %w", err)
	}

	rows, err := r.Q(ctx).GetFailsByUserID(ctx, sqlc.GetFailsByUserIDParams{
		UserID: userID,
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetFailsByUser - scan: %w", err)
	}

	result := make([]*domain.SubmissionWithDetails, len(rows))
	for i, row := range rows {
		result[i] = &domain.SubmissionWithDetails{
			Submission: toBaseSubmission(newSubRow(
				row.ID, row.UserID, row.TeamID, row.ChallengeID,
				row.SubmittedFlag, row.IsCorrect, row.SubmissionType,
				row.IP, row.CreatedAt, row.BannedUserID, nil,
			)),
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: row.ChallengeCategory,
		}
	}

	return result, nil
}

func (r *SubmissionRepo) CountFailsByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	n, err := r.Q(ctx).CountFailsByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountFailsByUser: %w", err)
	}

	return n, nil
}

func (r *SubmissionRepo) GetFailsByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetFailsByTeam: %w", err)
	}

	rows, err := r.Q(ctx).GetFailsByTeamID(ctx, sqlc.GetFailsByTeamIDParams{
		TeamID: &teamID,
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetFailsByTeam - scan: %w", err)
	}

	result := make([]*domain.SubmissionWithDetails, len(rows))
	for i, row := range rows {
		result[i] = &domain.SubmissionWithDetails{
			Submission: toBaseSubmission(newSubRow(
				row.ID, row.UserID, row.TeamID, row.ChallengeID,
				row.SubmittedFlag, row.IsCorrect, row.SubmissionType,
				row.IP, row.CreatedAt, row.BannedUserID, nil,
			)),
			Username:          row.Username,
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: row.ChallengeCategory,
		}
	}

	return result, nil
}

func (r *SubmissionRepo) CountFailsByTeam(ctx context.Context, teamID uuid.UUID) (int64, error) {
	n, err := r.Q(ctx).CountFailsByTeamID(ctx, &teamID)
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountFailsByTeam: %w", err)
	}

	return n, nil
}
