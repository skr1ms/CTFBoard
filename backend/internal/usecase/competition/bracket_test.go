package competition

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBracketUseCase_Create_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	name, desc := "bracket1", "desc"
	isDefault := true

	deps.bracketRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, b *entity.Bracket) {
		assert.Equal(t, name, b.Name)
		assert.Equal(t, desc, b.Description)
		assert.Equal(t, isDefault, b.IsDefault)
	})

	uc := h.CreateBracketUseCase()
	got, err := uc.Create(ctx, name, desc, isDefault)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, name, got.Name)
}

func TestBracketUseCase_Create_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	name, desc := "bracket1", "desc"

	deps.bracketRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := h.CreateBracketUseCase()
	got, err := uc.Create(ctx, name, desc, false)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestBracketUseCase_GetByID_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()
	bracket := h.NewBracket("b", "d", false)
	bracket.ID = id

	deps.bracketRepo.EXPECT().GetByID(mock.Anything, id).Return(bracket, nil)

	uc := h.CreateBracketUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestBracketUseCase_GetByID_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.bracketRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := h.CreateBracketUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestBracketUseCase_GetAll_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	list := []*entity.Bracket{h.NewBracket("b1", "d", false)}

	deps.bracketRepo.EXPECT().GetAll(mock.Anything).Return(list, nil)

	uc := h.CreateBracketUseCase()
	got, err := uc.GetAll(ctx)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestBracketUseCase_GetAll_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()

	deps.bracketRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := h.CreateBracketUseCase()
	got, err := uc.GetAll(ctx)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestBracketUseCase_Update_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()
	bracket := h.NewBracket("old", "oldd", false)
	bracket.ID = id
	name, desc := "new", "newd"
	isDefault := true

	deps.bracketRepo.EXPECT().GetByID(mock.Anything, id).Return(bracket, nil)
	deps.bracketRepo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, b *entity.Bracket) {
		assert.Equal(t, name, b.Name)
		assert.Equal(t, desc, b.Description)
		assert.Equal(t, isDefault, b.IsDefault)
	})

	uc := h.CreateBracketUseCase()
	got, err := uc.Update(ctx, id, name, desc, isDefault)

	assert.NoError(t, err)
	assert.Equal(t, name, got.Name)
}

func TestBracketUseCase_Update_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.bracketRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := h.CreateBracketUseCase()
	got, err := uc.Update(ctx, id, "name", "desc", false)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestBracketUseCase_Delete_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.bracketRepo.EXPECT().Delete(mock.Anything, id).Return(nil)

	uc := h.CreateBracketUseCase()
	err := uc.Delete(ctx, id)

	assert.NoError(t, err)
}

func TestBracketUseCase_Delete_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.bracketRepo.EXPECT().Delete(mock.Anything, id).Return(assert.AnError)

	uc := h.CreateBracketUseCase()
	err := uc.Delete(ctx, id)

	assert.Error(t, err)
}
