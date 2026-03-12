package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type CommentRepo struct {
	pool *pgxpool.Pool
}

var _ repo.CommentRepository = (*CommentRepo)(nil)

func NewCommentRepo(pool *pgxpool.Pool) *CommentRepo {
	return &CommentRepo{pool: pool}
}

func (r *CommentRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func (r *CommentRepo) Create(ctx context.Context, comment *entity.Comment) error {
	if comment.ID == uuid.Nil {
		comment.ID = uuid.New()
	}
	row, err := r.q(ctx).CreateComment(ctx, sqlc.CreateCommentParams{
		ID:          comment.ID,
		UserID:      comment.UserID,
		ChallengeID: comment.ChallengeID,
		Content:     comment.Content,
	})
	if err != nil {
		return fmt.Errorf("CommentRepo - Create: %w", err)
	}
	comment.CreatedAt = ptrTimeToTime(timestamptzToTime(row.CreatedAt))
	comment.UpdatedAt = ptrTimeToTime(timestamptzToTime(row.UpdatedAt))
	return nil
}

func (r *CommentRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Comment, error) {
	row, err := r.q(ctx).GetCommentByID(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrCommentNotFound
		}
		return nil, fmt.Errorf("CommentRepo - GetByID: %w", err)
	}
	return toEntityComment(row), nil
}

func (r *CommentRepo) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Comment, error) {
	rows, err := r.q(ctx).GetCommentsByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("CommentRepo - GetByChallengeID: %w", err)
	}
	out := make([]*entity.Comment, len(rows))
	for i, row := range rows {
		out[i] = toEntityComment(row)
	}
	return out, nil
}

func (r *CommentRepo) GetAll(ctx context.Context) ([]*entity.Comment, error) {
	rows, err := r.q(ctx).GetAllComments(ctx)
	if err != nil {
		return nil, fmt.Errorf("CommentRepo - GetAll: %w", err)
	}
	out := make([]*entity.Comment, len(rows))
	for i, row := range rows {
		out[i] = toEntityComment(row)
	}
	return out, nil
}

func (r *CommentRepo) Update(ctx context.Context, comment *entity.Comment) error {
	if err := r.q(ctx).UpdateComment(ctx, sqlc.UpdateCommentParams{
		ID:      comment.ID,
		Content: comment.Content,
	}); err != nil {
		if isNoRows(err) {
			return httperr.ErrCommentNotFound
		}
		return fmt.Errorf("CommentRepo - Update: %w", err)
	}
	return nil
}

func (r *CommentRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := r.q(ctx).DeleteComment(ctx, ID); err != nil {
		return fmt.Errorf("CommentRepo - Delete: %w", err)
	}
	return nil
}

func toEntityComment(row sqlc.Comment) *entity.Comment {
	return &entity.Comment{
		ID:          row.ID,
		UserID:      row.UserID,
		ChallengeID: row.ChallengeID,
		Content:     row.Content,
		CreatedAt:   ptrTimeToTime(timestamptzToTime(row.CreatedAt)),
		UpdatedAt:   ptrTimeToTime(timestamptzToTime(row.UpdatedAt)),
	}
}
