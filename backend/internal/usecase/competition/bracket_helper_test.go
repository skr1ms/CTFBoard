package competition

import (
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
)

func (h *CompetitionTestHelper) CreateBracketUseCase() *BracketUseCase {
	h.t.Helper()
	return NewBracketUseCase(h.deps.bracketRepo)
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
