package challenge

import (
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
)

func (h *ChallengeTestHelper) CreateTagUseCase() *TagUseCase {
	h.t.Helper()
	return NewTagUseCase(h.deps.tagRepo)
}

func (h *ChallengeTestHelper) NewTag(name, color string) *entity.Tag {
	h.t.Helper()
	return &entity.Tag{
		ID:    uuid.New(),
		Name:  name,
		Color: color,
	}
}
