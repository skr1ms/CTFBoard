package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubmissionRepo struct {
	pool *pgxpool.Pool
}

var _ repo.SubmissionRepository = (*SubmissionRepo)(nil)

func NewSubmissionRepo(pool *pgxpool.Pool) *SubmissionRepo {
	return &SubmissionRepo{pool: pool}
}

func (r *SubmissionRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
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

	if err := r.q(ctx).CreateSubmission(ctx, sqlc.CreateSubmissionParams{
		ID:            sub.ID,
		UserID:        sub.UserID,
		TeamID:        sub.TeamID,
		ChallengeID:   sub.ChallengeID,
		SubmittedFlag: sub.SubmittedFlag,
		IsCorrect:     sub.IsCorrect,
		IP:            ip,
		CreatedAt:     &sub.CreatedAt,
	}); err != nil {
		return fmt.Errorf("SubmissionRepo - Create: %w", err)
	}
	return nil
}

func (r *SubmissionRepo) CreateBatch(ctx context.Context, subs []*entity.Submission) error {
	if len(subs) == 0 {
		return nil
	}
	now := time.Now()
	rows := make([][]any, 0, len(subs))
	for _, sub := range subs {
		if sub.ID == uuid.Nil {
			sub.ID = uuid.New()
		}
		if sub.CreatedAt.IsZero() {
			sub.CreatedAt = now
		}
		var ip *string
		if sub.IP != "" {
			ip = &sub.IP
		}
		rows = append(rows, []any{sub.ID, sub.UserID, sub.TeamID, sub.ChallengeID, sub.SubmittedFlag, sub.IsCorrect, ip, sub.CreatedAt})
	}
	conn := ExtractDB(ctx, r.pool)
	if copier, ok := conn.(interface {
		CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	}); ok {
		_, err := copier.CopyFrom(ctx,
			pgx.Identifier{"submissions"},
			[]string{"id", "user_id", "team_id", "challenge_id", "submitted_flag", "is_correct", "ip", "created_at"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("SubmissionRepo - CreateBatch - CopyFrom: %w", err)
		}
		return nil
	}
	for _, sub := range subs {
		if err := r.Create(ctx, sub); err != nil {
			return fmt.Errorf("SubmissionRepo - CreateBatch: %w", err)
		}
	}
	return nil
}

func (r *SubmissionRepo) GetByChallenge(ctx context.Context, challengeID uuid.UUID, limit, offset int) ([]*entity.SubmissionWithDetails, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByChallenge - limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByChallenge - offset: %w", err)
	}
	rows, err := r.q(ctx).GetSubmissionsByChallenge(ctx, sqlc.GetSubmissionsByChallengeParams{
		ChallengeID: challengeID,
		Limit:       limit32,
		Offset:      offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByChallenge: %w", err)
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
				IP:            ptrStrToStr(row.IP),
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
		return nil, fmt.Errorf("SubmissionRepo - GetByUser - limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByUser - offset: %w", err)
	}
	rows, err := r.q(ctx).GetSubmissionsByUser(ctx, sqlc.GetSubmissionsByUserParams{
		UserID: userID,
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByUser: %w", err)
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
				IP:            ptrStrToStr(row.IP),
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
		return nil, fmt.Errorf("SubmissionRepo - GetByTeam - limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByTeam - offset: %w", err)
	}
	rows, err := r.q(ctx).GetSubmissionsByTeam(ctx, sqlc.GetSubmissionsByTeamParams{
		TeamID: &teamID,
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByTeam: %w", err)
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
				IP:            ptrStrToStr(row.IP),
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
		return nil, fmt.Errorf("SubmissionRepo - GetAll - limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetAll - offset: %w", err)
	}
	rows, err := r.q(ctx).GetAllSubmissions(ctx, sqlc.GetAllSubmissionsParams{
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetAll: %w", err)
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
				IP:            ptrStrToStr(row.IP),
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
	n, err := r.q(ctx).CountSubmissionsByChallenge(ctx, challengeID)
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountByChallenge: %w", err)
	}
	return n, nil
}

func (r *SubmissionRepo) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	n, err := r.q(ctx).CountSubmissionsByUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountByUser: %w", err)
	}
	return n, nil
}

func (r *SubmissionRepo) CountByTeam(ctx context.Context, teamID uuid.UUID) (int64, error) {
	n, err := r.q(ctx).CountSubmissionsByTeam(ctx, &teamID)
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountByTeam: %w", err)
	}
	return n, nil
}

func (r *SubmissionRepo) CountAll(ctx context.Context) (int64, error) {
	n, err := r.q(ctx).CountAllSubmissions(ctx)
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountAll: %w", err)
	}
	return n, nil
}

func (r *SubmissionRepo) CountFailedByIP(ctx context.Context, ip string, since time.Time) (int64, error) {
	n, err := r.q(ctx).CountFailedSubmissionsByIP(ctx, sqlc.CountFailedSubmissionsByIPParams{
		IP:        &ip,
		CreatedAt: &since,
	})
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountFailedByIP: %w", err)
	}
	return n, nil
}

func (r *SubmissionRepo) GetStats(ctx context.Context, challengeID uuid.UUID) (*entity.SubmissionStats, error) {
	row, err := r.q(ctx).GetSubmissionStats(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetStats: %w", err)
	}
	return &entity.SubmissionStats{
		Total:     int(row.Total),
		Correct:   int(row.Correct),
		Incorrect: int(row.Incorrect),
	}, nil
}

func (r *SubmissionRepo) GetFailsByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.SubmissionWithDetails, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetFailsByUser - limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetFailsByUser - offset: %w", err)
	}
	rows, err := r.q(ctx).GetFailsByUserID(ctx, sqlc.GetFailsByUserIDParams{
		UserID: userID,
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetFailsByUser: %w", err)
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
				IP:            ptrStrToStr(row.IP),
				CreatedAt:     ptrTimeToTime(row.CreatedAt),
			},
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: ptrStrToStr(row.ChallengeCategory),
		}
	}
	return result, nil
}

func (r *SubmissionRepo) CountFailsByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	n, err := r.q(ctx).CountFailsByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountFailsByUser: %w", err)
	}
	return n, nil
}

func (r *SubmissionRepo) GetFailsByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*entity.SubmissionWithDetails, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetFailsByTeam - limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetFailsByTeam - offset: %w", err)
	}
	rows, err := r.q(ctx).GetFailsByTeamID(ctx, sqlc.GetFailsByTeamIDParams{
		TeamID: &teamID,
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetFailsByTeam: %w", err)
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
				IP:            ptrStrToStr(row.IP),
				CreatedAt:     ptrTimeToTime(row.CreatedAt),
			},
			Username:          row.Username,
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: ptrStrToStr(row.ChallengeCategory),
		}
	}
	return result, nil
}

func (r *SubmissionRepo) CountFailsByTeam(ctx context.Context, teamID uuid.UUID) (int64, error) {
	n, err := r.q(ctx).CountFailsByTeamID(ctx, &teamID)
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountFailsByTeam: %w", err)
	}
	return n, nil
}

func (r *SubmissionRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.SubmissionWithDetails, error) {
	row, err := r.q(ctx).GetSubmissionByID(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrSubmissionNotFound
		}
		return nil, fmt.Errorf("SubmissionRepo - GetByID: %w", err)
	}
	return &entity.SubmissionWithDetails{
		Submission: entity.Submission{
			ID:            row.ID,
			UserID:        row.UserID,
			TeamID:        row.TeamID,
			ChallengeID:   row.ChallengeID,
			SubmittedFlag: row.SubmittedFlag,
			IsCorrect:     row.IsCorrect,
			IP:            ptrStrToStr(row.IP),
			CreatedAt:     ptrTimeToTime(row.CreatedAt),
		},
		Username:          row.Username,
		TeamName:          row.TeamName,
		ChallengeTitle:    row.ChallengeTitle,
		ChallengeCategory: ptrStrToStr(row.ChallengeCategory),
	}, nil
}

func (r *SubmissionRepo) Update(ctx context.Context, ID uuid.UUID, isCorrect bool) error {
	if err := r.q(ctx).UpdateSubmission(ctx, sqlc.UpdateSubmissionParams{
		ID:        ID,
		IsCorrect: isCorrect,
	}); err != nil {
		return fmt.Errorf("SubmissionRepo - Update: %w", err)
	}
	return nil
}

// Delete removes a submission by ID. Idempotent: returns nil if the submission does not exist.
func (r *SubmissionRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := r.q(ctx).DeleteSubmission(ctx, ID); err != nil {
		return fmt.Errorf("SubmissionRepo - Delete: %w", err)
	}
	return nil
}

func (r *SubmissionRepo) DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error {
	if err := r.q(ctx).DeleteSubmissionsByTeamID(ctx, &teamID); err != nil {
		return fmt.Errorf("SubmissionRepo - DeleteByTeamID: %w", err)
	}
	return nil
}

func (r *SubmissionRepo) GetByIDForUpdate(ctx context.Context, ID uuid.UUID) (*entity.Submission, error) {
	row, err := r.q(ctx).GetSubmissionByIDForUpdate(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrSubmissionNotFound
		}
		return nil, fmt.Errorf("SubmissionRepo - GetByIDForUpdate: %w", err)
	}
	s := &entity.Submission{
		ID:            row.ID,
		UserID:        row.UserID,
		TeamID:        row.TeamID,
		ChallengeID:   row.ChallengeID,
		SubmittedFlag: row.SubmittedFlag,
		IsCorrect:     row.IsCorrect,
	}
	if row.IP != nil {
		s.IP = *row.IP
	}
	if row.CreatedAt != nil {
		s.CreatedAt = *row.CreatedAt
	}
	return s, nil
}
