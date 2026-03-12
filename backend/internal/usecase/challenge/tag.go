package challenge

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

const defaultTagColor = "#6b7280"

type TagDeps struct {
	TagRepo       repo.TagRepository
	ChallengeRepo repo.ChallengeRepository
}

type TagUseCase struct {
	deps TagDeps
}

var _ usecase.TagUseCase = (*TagUseCase)(nil)

func NewTagUseCase(deps TagDeps) *TagUseCase {
	return &TagUseCase{deps: deps}
}

func (uc *TagUseCase) Create(ctx context.Context, name, color string) (*entity.Tag, error) {
	if name == "" {
		return nil, httperr.ErrTagNameRequired
	}
	tag := &entity.Tag{
		ID:    uuid.New(),
		Name:  name,
		Color: color,
	}
	if tag.Color == "" {
		tag.Color = defaultTagColor
	}
	if err := uc.deps.TagRepo.Create(ctx, tag); err != nil {
		return nil, fmt.Errorf("TagUseCase - Create - TagRepo.Create: %w", err)
	}
	return tag, nil
}

func (uc *TagUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Tag, error) {
	tag, err := uc.deps.TagRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("TagUseCase - GetByID - TagRepo.GetByID: %w", err)
	}
	return tag, nil
}

func (uc *TagUseCase) GetAll(ctx context.Context) ([]*entity.Tag, error) {
	tags, err := uc.deps.TagRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("TagUseCase - GetAll - TagRepo.GetAll: %w", err)
	}
	return tags, nil
}

func (uc *TagUseCase) Update(ctx context.Context, ID uuid.UUID, name, color string) (*entity.Tag, error) {
	if name == "" {
		return nil, httperr.ErrTagNameRequired
	}
	tag, err := uc.deps.TagRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("TagUseCase - Update - TagRepo.GetByID: %w", err)
	}
	tag.Name = name
	if color != "" {
		tag.Color = color
	} else {
		tag.Color = defaultTagColor
	}
	if err := uc.deps.TagRepo.Update(ctx, tag); err != nil {
		return nil, fmt.Errorf("TagUseCase - Update - TagRepo.Update: %w", err)
	}
	return tag, nil
}

func (uc *TagUseCase) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := uc.deps.TagRepo.Delete(ctx, ID); err != nil {
		return fmt.Errorf("TagUseCase - Delete - TagRepo.Delete: %w", err)
	}
	return nil
}

func (uc *TagUseCase) GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Tag, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("TagUseCase - GetByChallengeID - ChallengeRepo.GetByID: %w", err)
	}
	if challenge.IsHidden {
		return nil, httperr.ErrChallengeNotFound
	}
	tags, err := uc.deps.TagRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("TagUseCase - GetByChallengeID - TagRepo.GetByChallengeID: %w", err)
	}
	return tags, nil
}
