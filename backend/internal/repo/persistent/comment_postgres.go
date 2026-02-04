package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type CommentRepo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewCommentRepo(pool *pgxpool.Pool) *CommentRepo {
	return &CommentRepo{
		pool: pool,
		q:    sqlc.New(pool),
	}
}

func (r *CommentRepo) Create(ctx context.Context, comment *entity.Comment) error {
	if comment.ID == uuid.Nil {
		comment.ID = uuid.New()
	}
	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = time.Now()
	}
	if comment.UpdatedAt.IsZero() {
		comment.UpdatedAt = comment.CreatedAt
	}
	createdAt := &comment.CreatedAt
	updatedAt := &comment.UpdatedAt
	_, err := r.q.CreateComment(ctx, sqlc.CreateCommentParams{
		ID:          comment.ID,
		UserID:      comment.UserID,
		ChallengeID: comment.ChallengeID,
		Content:     comment.Content,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	})
	if err != nil {
		return fmt.Errorf("CommentRepo - Create: %w", err)
	}
	return nil
}

func (r *CommentRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Comment, error) {
	row, err := r.q.GetCommentByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrCommentNotFound
		}
		return nil, fmt.Errorf("CommentRepo - GetByID: %w", err)
	}
	return commentRowToEntity(row), nil
}

func (r *CommentRepo) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Comment, error) {
	rows, err := r.q.GetCommentsByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("CommentRepo - GetByChallengeID: %w", err)
	}
	out := make([]*entity.Comment, len(rows))
	for i, row := range rows {
		out[i] = commentRowToEntity(row)
	}
	return out, nil
}

func (r *CommentRepo) Update(ctx context.Context, comment *entity.Comment) error {
	comment.UpdatedAt = time.Now()
	updatedAt := &comment.UpdatedAt
	return r.q.UpdateComment(ctx, sqlc.UpdateCommentParams{
		ID:        comment.ID,
		Content:   comment.Content,
		UpdatedAt: updatedAt,
	})
}

func (r *CommentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteComment(ctx, id)
}

func commentRowToEntity(row sqlc.Comment) *entity.Comment {
	return &entity.Comment{
		ID:          row.ID,
		UserID:      row.UserID,
		ChallengeID: row.ChallengeID,
		Content:     row.Content,
		CreatedAt:   ptrTimeToTime(row.CreatedAt),
		UpdatedAt:   ptrTimeToTime(row.UpdatedAt),
	}
}
