package challenge

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/google/uuid"
)

func (h *ChallengeTestHelper) CreateTagUseCase() *TagUseCase {
	h.t.Helper()
	return NewTagUseCase(TagDeps{TagRepo: h.deps.tagRepo, ChallengeRepo: h.deps.challengeRepo})
}

func (h *ChallengeTestHelper) NewTag(name, color string) *entity.Tag {
	h.t.Helper()
	return &entity.Tag{
		ID:    uuid.New(),
		Name:  name,
		Color: color,
	}
}
