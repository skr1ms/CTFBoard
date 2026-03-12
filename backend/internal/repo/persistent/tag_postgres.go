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

type TagRepo struct {
	pool *pgxpool.Pool
}

var _ repo.TagRepository = (*TagRepo)(nil)

func NewTagRepo(pool *pgxpool.Pool) *TagRepo {
	return &TagRepo{pool: pool}
}

func (r *TagRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
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
	if err := r.q(ctx).CreateTag(ctx, sqlc.CreateTagParams{
		ID:    tag.ID,
		Name:  tag.Name,
		Color: color,
	}); err != nil {
		return fmt.Errorf("TagRepo - Create: %w", err)
	}
	return nil
}

func (r *TagRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Tag, error) {
	row, err := r.q(ctx).GetTagByID(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrTagNotFound
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
	row, err := r.q(ctx).GetTagByName(ctx, name)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrTagNotFound
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
	rows, err := r.q(ctx).GetAllTags(ctx)
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
	if err := r.q(ctx).UpdateTag(ctx, sqlc.UpdateTagParams{
		ID:    tag.ID,
		Name:  tag.Name,
		Color: color,
	}); err != nil {
		return fmt.Errorf("TagRepo - Update: %w", err)
	}
	return nil
}

// Delete removes a tag by ID. Idempotent: returns nil if the tag does not exist.
func (r *TagRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := r.q(ctx).DeleteTag(ctx, ID); err != nil {
		return fmt.Errorf("TagRepo - Delete: %w", err)
	}
	return nil
}

func (r *TagRepo) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Tag, error) {
	rows, err := r.q(ctx).GetTagsByChallengeID(ctx, challengeID)
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
	rows, err := r.q(ctx).GetTagsByChallengeIDs(ctx, challengeIDs)
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
