package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type TagRepo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewTagRepo(pool *pgxpool.Pool) *TagRepo {
	return &TagRepo{
		pool: pool,
		q:    sqlc.New(pool),
	}
}

func (r *TagRepo) Create(ctx context.Context, tag *entity.Tag) error {
	if tag.ID == uuid.Nil {
		tag.ID = uuid.New()
	}
	color := strPtrOrNil(tag.Color)
	if tag.Color == "" {
		def := "#6b7280"
		color = &def
	}
	return r.q.CreateTag(ctx, sqlc.CreateTagParams{
		ID:    tag.ID,
		Name:  tag.Name,
		Color: color,
	})
}

func (r *TagRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tag, error) {
	row, err := r.q.GetTagByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrTagNotFound
		}
		return nil, fmt.Errorf("TagRepo - GetByID: %w", err)
	}
	return &entity.Tag{
		ID:    row.ID,
		Name:  row.Name,
		Color: ptrStrToStr(row.Color),
	}, nil
}

func (r *TagRepo) GetByName(ctx context.Context, name string) (*entity.Tag, error) {
	row, err := r.q.GetTagByName(ctx, name)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrTagNotFound
		}
		return nil, fmt.Errorf("TagRepo - GetByName: %w", err)
	}
	return &entity.Tag{
		ID:    row.ID,
		Name:  row.Name,
		Color: ptrStrToStr(row.Color),
	}, nil
}

func (r *TagRepo) GetAll(ctx context.Context) ([]*entity.Tag, error) {
	rows, err := r.q.GetAllTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("TagRepo - GetAll: %w", err)
	}
	out := make([]*entity.Tag, len(rows))
	for i, row := range rows {
		out[i] = &entity.Tag{
			ID:    row.ID,
			Name:  row.Name,
			Color: ptrStrToStr(row.Color),
		}
	}
	return out, nil
}

func (r *TagRepo) Update(ctx context.Context, tag *entity.Tag) error {
	color := strPtrOrNil(tag.Color)
	if tag.Color == "" {
		def := "#6b7280"
		color = &def
	}
	return r.q.UpdateTag(ctx, sqlc.UpdateTagParams{
		ID:    tag.ID,
		Name:  tag.Name,
		Color: color,
	})
}

func (r *TagRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTag(ctx, id)
}

func (r *TagRepo) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Tag, error) {
	rows, err := r.q.GetTagsByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("TagRepo - GetByChallengeID: %w", err)
	}
	out := make([]*entity.Tag, len(rows))
	for i, row := range rows {
		out[i] = &entity.Tag{
			ID:    row.ID,
			Name:  row.Name,
			Color: ptrStrToStr(row.Color),
		}
	}
	return out, nil
}

func (r *TagRepo) GetByChallengeIDs(ctx context.Context, challengeIDs []uuid.UUID) (map[uuid.UUID][]*entity.Tag, error) {
	if len(challengeIDs) == 0 {
		return map[uuid.UUID][]*entity.Tag{}, nil
	}
	rows, err := r.q.GetTagsByChallengeIDs(ctx, challengeIDs)
	if err != nil {
		return nil, fmt.Errorf("TagRepo - GetByChallengeIDs: %w", err)
	}
	out := make(map[uuid.UUID][]*entity.Tag)
	for _, row := range rows {
		tag := &entity.Tag{
			ID:    row.ID,
			Name:  row.Name,
			Color: ptrStrToStr(row.Color),
		}
		out[row.ChallengeID] = append(out[row.ChallengeID], tag)
	}
	return out, nil
}

func (r *TagRepo) SetChallengeTags(ctx context.Context, challengeID uuid.UUID, tagIDs []uuid.UUID) error {
	if err := r.q.DeleteChallengeTags(ctx, challengeID); err != nil {
		return fmt.Errorf("TagRepo - SetChallengeTags - Delete: %w", err)
	}
	for _, tagID := range tagIDs {
		if err := r.q.AddChallengeTag(ctx, sqlc.AddChallengeTagParams{
			ChallengeID: challengeID,
			TagID:       tagID,
		}); err != nil {
			return fmt.Errorf("TagRepo - SetChallengeTags - Add: %w", err)
		}
	}
	return nil
}
