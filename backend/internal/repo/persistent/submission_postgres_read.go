package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

func (r *SubmissionRepo) GetByChallenge(ctx context.Context, challengeID uuid.UUID, freezeTime *time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByChallenge: %w", err)
	}

	rows, err := r.Q(ctx).GetSubmissionsByChallenge(ctx, sqlc.GetSubmissionsByChallengeParams{
		ChallengeID: challengeID,
		FreezeTime:  pgutil.TimeToTimestamptz(freezeTime),
		Limit:       limit32,
		Offset:      offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByChallenge - scan: %w", err)
	}

	result := make([]*domain.SubmissionWithDetails, len(rows))
	for i, row := range rows {
		result[i] = &domain.SubmissionWithDetails{
			Submission: toBaseSubmission(newSubRow(
				row.ID, row.UserID, row.TeamID, row.ChallengeID,
				row.SubmittedFlag, row.IsCorrect, row.SubmissionType,
				row.IP, row.CreatedAt, row.BannedUserID, nil,
			)),
			Username: row.Username,
			TeamName: row.TeamName,
		}
	}

	return result, nil
}

func (r *SubmissionRepo) GetByUser(ctx context.Context, userID uuid.UUID, freezeTime *time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByUser: %w", err)
	}

	rows, err := r.Q(ctx).GetSubmissionsByUser(ctx, sqlc.GetSubmissionsByUserParams{
		UserID:     userID,
		FreezeTime: pgutil.TimeToTimestamptz(freezeTime),
		Limit:      limit32,
		Offset:     offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByUser - scan: %w", err)
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

func (r *SubmissionRepo) GetByTeam(ctx context.Context, teamID uuid.UUID, freezeTime *time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByTeam: %w", err)
	}

	rows, err := r.Q(ctx).GetSubmissionsByTeam(ctx, sqlc.GetSubmissionsByTeamParams{
		TeamID:     &teamID,
		FreezeTime: pgutil.TimeToTimestamptz(freezeTime),
		Limit:      limit32,
		Offset:     offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByTeam - scan: %w", err)
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

func (r *SubmissionRepo) GetAll(ctx context.Context, freezeTime *time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetAll: %w", err)
	}

	rows, err := r.Q(ctx).GetAllSubmissions(ctx, sqlc.GetAllSubmissionsParams{
		FreezeTime: pgutil.TimeToTimestamptz(freezeTime),
		Limit:      limit32,
		Offset:     offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetAll - scan: %w", err)
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
			TeamName:          row.TeamName,
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: row.ChallengeCategory,
		}
	}

	return result, nil
}

func (r *SubmissionRepo) CountByChallenge(ctx context.Context, challengeID uuid.UUID, freezeTime *time.Time) (int64, error) {
	n, err := r.Q(ctx).CountSubmissionsByChallenge(ctx, sqlc.CountSubmissionsByChallengeParams{
		ChallengeID: challengeID,
		FreezeTime:  pgutil.TimeToTimestamptz(freezeTime),
	})
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountByChallenge: %w", err)
	}

	return n, nil
}

func (r *SubmissionRepo) CountByUser(ctx context.Context, userID uuid.UUID, freezeTime *time.Time) (int64, error) {
	n, err := r.Q(ctx).CountSubmissionsByUser(ctx, sqlc.CountSubmissionsByUserParams{
		UserID:     userID,
		FreezeTime: pgutil.TimeToTimestamptz(freezeTime),
	})
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountByUser: %w", err)
	}

	return n, nil
}

func (r *SubmissionRepo) CountByTeam(ctx context.Context, teamID uuid.UUID, freezeTime *time.Time) (int64, error) {
	n, err := r.Q(ctx).CountSubmissionsByTeam(ctx, sqlc.CountSubmissionsByTeamParams{
		TeamID:     &teamID,
		FreezeTime: pgutil.TimeToTimestamptz(freezeTime),
	})
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountByTeam: %w", err)
	}

	return n, nil
}

func (r *SubmissionRepo) CountAll(ctx context.Context, freezeTime *time.Time) (int64, error) {
	n, err := r.Q(ctx).CountAllSubmissions(ctx, pgutil.TimeToTimestamptz(freezeTime))
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountAll: %w", err)
	}

	return n, nil
}

func (r *SubmissionRepo) GetStats(ctx context.Context, challengeID uuid.UUID, freezeTime *time.Time) (*domain.SubmissionStats, error) {
	row, err := r.Q(ctx).GetSubmissionStats(ctx, sqlc.GetSubmissionStatsParams{
		ChallengeID: challengeID,
		FreezeTime:  pgutil.TimeToTimestamptz(freezeTime),
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetStats: %w", err)
	}

	return &domain.SubmissionStats{
		Total:     int(row.Total),
		Correct:   int(row.Correct),
		Incorrect: int(row.Incorrect),
	}, nil
}

func (r *SubmissionRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.SubmissionWithDetails, error) {
	row, err := r.Q(ctx).GetSubmissionByID(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, apperr.ErrSubmissionNotFound
		}

		return nil, fmt.Errorf("SubmissionRepo - GetByID: %w", err)
	}

	return &domain.SubmissionWithDetails{
		Submission: toBaseSubmission(newSubRow(
			row.ID, row.UserID, row.TeamID, row.ChallengeID,
			row.SubmittedFlag, row.IsCorrect, row.SubmissionType,
			row.IP, row.CreatedAt, row.BannedUserID, nil,
		)),
		Username:          row.Username,
		TeamName:          row.TeamName,
		ChallengeTitle:    row.ChallengeTitle,
		ChallengeCategory: row.ChallengeCategory,
	}, nil
}

func (r *SubmissionRepo) GetByIDForUpdate(ctx context.Context, ID uuid.UUID) (*domain.Submission, error) {
	row, err := r.Q(ctx).GetSubmissionByIDForUpdate(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, apperr.ErrSubmissionNotFound
		}

		return nil, fmt.Errorf("SubmissionRepo - GetByIDForUpdate: %w", err)
	}

	s := toBaseSubmission(newSubRow(
		row.ID, row.UserID, row.TeamID, row.ChallengeID,
		row.SubmittedFlag, row.IsCorrect, row.SubmissionType,
		row.IP, row.CreatedAt, row.BannedUserID, row.BannedTeamID,
	))

	return &s, nil
}
