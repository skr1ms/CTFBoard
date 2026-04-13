package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

type CommentRepo struct {
	BaseRepo
}

var _ repo.CommentRepository = (*CommentRepo)(nil)

func NewCommentRepo(pool *pgxpool.Pool) *CommentRepo {
	return &CommentRepo{BaseRepo: BaseRepo{pool: pool}}
}

func (r *CommentRepo) Create(ctx context.Context, comment *domain.Comment) error {
	EnsureID(&comment.ID)

	now := time.Now()

	row, err := r.Q(ctx).CreateComment(ctx, sqlc.CreateCommentParams{
		ID:          comment.ID,
		UserID:      comment.UserID,
		ChallengeID: comment.ChallengeID,
		Content:     comment.Content,
		CreatedAt:   pgutil.TimeToTimestamptz(&now),
		UpdatedAt:   pgutil.TimeToTimestamptz(&now),
	})
	if err != nil {
		return fmt.Errorf("CommentRepo - Create: %w", err)
	}

	comment.CreatedAt = pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt))
	comment.UpdatedAt = pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.UpdatedAt))

	return nil
}

func (r *CommentRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Comment, error) {
	row, err := GetOrNotFound(func() (sqlc.GetCommentByIDRow, error) { return r.Q(ctx).GetCommentByID(ctx, ID) },
		apperr.ErrCommentNotFound, "CommentRepo - GetByID")
	if err != nil {
		return nil, err
	}

	return &domain.Comment{
		ID:          row.ID,
		UserID:      row.UserID,
		Username:    row.Username,
		ChallengeID: row.ChallengeID,
		Content:     row.Content,
		CreatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
		UpdatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.UpdatedAt)),
	}, nil
}

func (r *CommentRepo) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Comment, error) {
	rows, err := r.Q(ctx).GetCommentsByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("CommentRepo - GetByChallengeID: %w", err)
	}

	out := make([]*domain.Comment, len(rows))
	for i, row := range rows {
		out[i] = &domain.Comment{
			ID:          row.ID,
			UserID:      row.UserID,
			Username:    row.Username,
			ChallengeID: row.ChallengeID,
			Content:     row.Content,
			CreatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
			UpdatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.UpdatedAt)),
		}
	}

	return out, nil
}

func (r *CommentRepo) GetAll(ctx context.Context) ([]*domain.Comment, error) {
	rows, err := r.Q(ctx).GetAllComments(ctx)
	if err != nil {
		return nil, fmt.Errorf("CommentRepo - GetAll: %w", err)
	}

	out := make([]*domain.Comment, len(rows))
	for i, row := range rows {
		out[i] = &domain.Comment{
			ID:          row.ID,
			UserID:      row.UserID,
			Username:    row.Username,
			ChallengeID: row.ChallengeID,
			Content:     row.Content,
			CreatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
			UpdatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.UpdatedAt)),
		}
	}

	return out, nil
}

func (r *CommentRepo) Update(ctx context.Context, comment *domain.Comment) error {
	now := time.Now()

	err := r.Q(ctx).UpdateComment(ctx, sqlc.UpdateCommentParams{
		ID:        comment.ID,
		Content:   comment.Content,
		UpdatedAt: pgutil.TimeToTimestamptz(&now),
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return apperr.ErrCommentNotFound
		}

		return fmt.Errorf("CommentRepo - Update: %w", err)
	}

	return nil
}

func (r *CommentRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	err := r.Q(ctx).DeleteComment(ctx, ID)
	if err != nil {
		return fmt.Errorf("CommentRepo - Delete: %w", err)
	}

	return nil
}
