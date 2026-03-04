package competition

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/google/uuid"
)

func (h *CompetitionTestHelper) CreateBracketUseCase() *BracketUseCase {
	h.t.Helper()
	return NewBracketUseCase(BracketDeps{BracketRepo: h.deps.bracketRepo, TM: h.deps.tm})
}

func (h *CompetitionTestHelper) NewBracket(name, description string, isDefault bool) *entity.Bracket {
	h.t.Helper()
	return &entity.Bracket{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		IsDefault:   isDefault,
	}
}
