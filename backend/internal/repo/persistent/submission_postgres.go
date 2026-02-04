package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type SubmissionRepo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewSubmissionRepo(pool *pgxpool.Pool) *SubmissionRepo {
	return &SubmissionRepo{
		pool: pool,
		q:    sqlc.New(pool),
	}
}

func (r *SubmissionRepo) Create(ctx context.Context, sub *entity.Submission) error {
	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = time.Now()
	}

	var ip *string
	if sub.IP != "" {
		ip = &sub.IP
	}

	return r.q.CreateSubmission(ctx, sqlc.CreateSubmissionParams{
		ID:            sub.ID,
		UserID:        sub.UserID,
		TeamID:        sub.TeamID,
		ChallengeID:   sub.ChallengeID,
		SubmittedFlag: sub.SubmittedFlag,
		IsCorrect:     sub.IsCorrect,
		Ip:            ip,
		CreatedAt:     &sub.CreatedAt,
	})
}

func (r *SubmissionRepo) GetByChallenge(ctx context.Context, challengeID uuid.UUID, limit, offset int) ([]*entity.SubmissionWithDetails, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByChallenge limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByChallenge offset: %w", err)
	}
	rows, err := r.q.GetSubmissionsByChallenge(ctx, sqlc.GetSubmissionsByChallengeParams{
		ChallengeID: challengeID,
		Limit:       limit32,
		Offset:      offset32,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*entity.SubmissionWithDetails, len(rows))
	for i, row := range rows {
		result[i] = &entity.SubmissionWithDetails{
			Submission: entity.Submission{
				ID:            row.ID,
				UserID:        row.UserID,
				TeamID:        row.TeamID,
				ChallengeID:   row.ChallengeID,
				SubmittedFlag: row.SubmittedFlag,
				IsCorrect:     row.IsCorrect,
				IP:            ptrStrToStr(row.Ip),
				CreatedAt:     ptrTimeToTime(row.CreatedAt),
			},
			Username: row.Username,
			TeamName: row.TeamName,
		}
	}
	return result, nil
}

func (r *SubmissionRepo) GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.SubmissionWithDetails, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByUser limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByUser offset: %w", err)
	}
	rows, err := r.q.GetSubmissionsByUser(ctx, sqlc.GetSubmissionsByUserParams{
		UserID: userID,
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*entity.SubmissionWithDetails, len(rows))
	for i, row := range rows {
		result[i] = &entity.SubmissionWithDetails{
			Submission: entity.Submission{
				ID:            row.ID,
				UserID:        row.UserID,
				TeamID:        row.TeamID,
				ChallengeID:   row.ChallengeID,
				SubmittedFlag: row.SubmittedFlag,
				IsCorrect:     row.IsCorrect,
				IP:            ptrStrToStr(row.Ip),
				CreatedAt:     ptrTimeToTime(row.CreatedAt),
			},
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: ptrStrToStr(row.ChallengeCategory),
		}
	}
	return result, nil
}

func (r *SubmissionRepo) GetByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*entity.SubmissionWithDetails, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByTeam limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByTeam offset: %w", err)
	}
	rows, err := r.q.GetSubmissionsByTeam(ctx, sqlc.GetSubmissionsByTeamParams{
		TeamID: &teamID,
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*entity.SubmissionWithDetails, len(rows))
	for i, row := range rows {
		result[i] = &entity.SubmissionWithDetails{
			Submission: entity.Submission{
				ID:            row.ID,
				UserID:        row.UserID,
				TeamID:        row.TeamID,
				ChallengeID:   row.ChallengeID,
				SubmittedFlag: row.SubmittedFlag,
				IsCorrect:     row.IsCorrect,
				IP:            ptrStrToStr(row.Ip),
				CreatedAt:     ptrTimeToTime(row.CreatedAt),
			},
			Username:          row.Username,
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: ptrStrToStr(row.ChallengeCategory),
		}
	}
	return result, nil
}

func (r *SubmissionRepo) GetAll(ctx context.Context, limit, offset int) ([]*entity.SubmissionWithDetails, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetAll limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetAll offset: %w", err)
	}
	rows, err := r.q.GetAllSubmissions(ctx, sqlc.GetAllSubmissionsParams{
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*entity.SubmissionWithDetails, len(rows))
	for i, row := range rows {
		result[i] = &entity.SubmissionWithDetails{
			Submission: entity.Submission{
				ID:            row.ID,
				UserID:        row.UserID,
				TeamID:        row.TeamID,
				ChallengeID:   row.ChallengeID,
				SubmittedFlag: row.SubmittedFlag,
				IsCorrect:     row.IsCorrect,
				IP:            ptrStrToStr(row.Ip),
				CreatedAt:     ptrTimeToTime(row.CreatedAt),
			},
			Username:          row.Username,
			TeamName:          row.TeamName,
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: ptrStrToStr(row.ChallengeCategory),
		}
	}
	return result, nil
}

func (r *SubmissionRepo) CountByChallenge(ctx context.Context, challengeID uuid.UUID) (int64, error) {
	return r.q.CountSubmissionsByChallenge(ctx, challengeID)
}

func (r *SubmissionRepo) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	return r.q.CountSubmissionsByUser(ctx, userID)
}

func (r *SubmissionRepo) CountByTeam(ctx context.Context, teamID uuid.UUID) (int64, error) {
	return r.q.CountSubmissionsByTeam(ctx, &teamID)
}

func (r *SubmissionRepo) CountAll(ctx context.Context) (int64, error) {
	return r.q.CountAllSubmissions(ctx)
}

func (r *SubmissionRepo) CountFailedByIP(ctx context.Context, ip string, since time.Time) (int64, error) {
	return r.q.CountFailedSubmissionsByIP(ctx, sqlc.CountFailedSubmissionsByIPParams{
		Ip:        &ip,
		CreatedAt: &since,
	})
}

func (r *SubmissionRepo) GetStats(ctx context.Context, challengeID uuid.UUID) (*entity.SubmissionStats, error) {
	row, err := r.q.GetSubmissionStats(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	return &entity.SubmissionStats{
		Total:     int(row.Total),
		Correct:   int(row.Correct),
		Incorrect: int(row.Incorrect),
	}, nil
}
