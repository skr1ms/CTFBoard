package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type TagRepo struct {
	BaseRepo
}

var _ repo.TagRepository = (*TagRepo)(nil)

func NewTagRepo(pool *pgxpool.Pool) *TagRepo {
	return &TagRepo{BaseRepo: BaseRepo{pool: pool}}
}

func (r *TagRepo) Create(ctx context.Context, tag *domain.Tag) error {
	EnsureID(&tag.ID)
	color := lo.EmptyableToPtr(tag.Color)
	if tag.Color == "" {
		def := "#6b7280"
		color = &def
	}
	if err := r.Q(ctx).CreateTag(ctx, sqlc.CreateTagParams{
		ID:    tag.ID,
		Name:  tag.Name,
		Color: color,
	}); err != nil {
		return fmt.Errorf("TagRepo - Create: %w", err)
	}
	return nil
}

func (r *TagRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Tag, error) {
	row, err := GetOrNotFound(func() (sqlc.Tag, error) { return r.Q(ctx).GetTagByID(ctx, ID) }, httperr.ErrTagNotFound, "TagRepo - GetByID")
	if err != nil {
		return nil, err
	}
	return &domain.Tag{ID: row.ID, Name: row.Name, Color: lo.FromPtr(row.Color)}, nil
}

func (r *TagRepo) GetByName(ctx context.Context, name string) (*domain.Tag, error) {
	row, err := GetOrNotFound(func() (sqlc.Tag, error) { return r.Q(ctx).GetTagByName(ctx, name) }, httperr.ErrTagNotFound, "TagRepo - GetByName")
	if err != nil {
		return nil, err
	}
	return &domain.Tag{ID: row.ID, Name: row.Name, Color: lo.FromPtr(row.Color)}, nil
}

func (r *TagRepo) GetAll(ctx context.Context) ([]*domain.Tag, error) {
	rows, err := r.Q(ctx).GetAllTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("TagRepo - GetAll: %w", err)
	}
	out := make([]*domain.Tag, len(rows))
	for i, row := range rows {
		out[i] = &domain.Tag{
			ID:    row.ID,
			Name:  row.Name,
			Color: lo.FromPtr(row.Color),
		}
	}
	return out, nil
}

func (r *TagRepo) Update(ctx context.Context, tag *domain.Tag) error {
	color := lo.EmptyableToPtr(tag.Color)
	if tag.Color == "" {
		def := "#6b7280"
		color = &def
	}
	if err := r.Q(ctx).UpdateTag(ctx, sqlc.UpdateTagParams{
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
	if err := r.Q(ctx).DeleteTag(ctx, ID); err != nil {
		return fmt.Errorf("TagRepo - Delete: %w", err)
	}
	return nil
}

func (r *TagRepo) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Tag, error) {
	rows, err := r.Q(ctx).GetTagsByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("TagRepo - GetByChallengeID: %w", err)
	}
	out := make([]*domain.Tag, len(rows))
	for i, row := range rows {
		out[i] = &domain.Tag{
			ID:    row.ID,
			Name:  row.Name,
			Color: lo.FromPtr(row.Color),
		}
	}
	return out, nil
}

func (r *TagRepo) GetByChallengeIDs(ctx context.Context, challengeIDs []uuid.UUID) (map[uuid.UUID][]*domain.Tag, error) {
	if len(challengeIDs) == 0 {
		return map[uuid.UUID][]*domain.Tag{}, nil
	}
	rows, err := r.Q(ctx).GetTagsByChallengeIDs(ctx, challengeIDs)
	if err != nil {
		return nil, fmt.Errorf("TagRepo - GetByChallengeIDs: %w", err)
	}
	out := make(map[uuid.UUID][]*domain.Tag)
	for _, row := range rows {
		tag := &domain.Tag{
			ID:    row.ID,
			Name:  row.Name,
			Color: lo.FromPtr(row.Color),
		}
		out[row.ChallengeID] = append(out[row.ChallengeID], tag)
	}
	return out, nil
}
