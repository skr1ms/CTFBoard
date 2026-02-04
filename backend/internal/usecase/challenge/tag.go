package challenge

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

type TagUseCase struct {
	tagRepo repo.TagRepository
}

func NewTagUseCase(
	tagRepo repo.TagRepository,
) *TagUseCase {
	return &TagUseCase{tagRepo: tagRepo}
}

func (uc *TagUseCase) Create(ctx context.Context, name, color string) (*entity.Tag, error) {
	if name == "" {
		return nil, fmt.Errorf("TagUseCase - Create: name is required")
	}
	tag := &entity.Tag{
		ID:    uuid.New(),
		Name:  name,
		Color: color,
	}
	if tag.Color == "" {
		tag.Color = "#6b7280"
	}
	if err := uc.tagRepo.Create(ctx, tag); err != nil {
		return nil, fmt.Errorf("TagUseCase - Create: %w", err)
	}
	return tag, nil
}

func (uc *TagUseCase) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tag, error) {
	tag, err := uc.tagRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("TagUseCase - GetByID: %w", err)
	}
	return tag, nil
}

func (uc *TagUseCase) GetAll(ctx context.Context) ([]*entity.Tag, error) {
	tags, err := uc.tagRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("TagUseCase - GetAll: %w", err)
	}
	return tags, nil
}

func (uc *TagUseCase) Update(ctx context.Context, id uuid.UUID, name, color string) (*entity.Tag, error) {
	tag, err := uc.tagRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("TagUseCase - Update - GetByID: %w", err)
	}
	tag.Name = name
	if color != "" {
		tag.Color = color
	} else {
		tag.Color = "#6b7280"
	}
	if err := uc.tagRepo.Update(ctx, tag); err != nil {
		return nil, fmt.Errorf("TagUseCase - Update: %w", err)
	}
	return tag, nil
}

func (uc *TagUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	if err := uc.tagRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("TagUseCase - Delete: %w", err)
	}
	return nil
}
