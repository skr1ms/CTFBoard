package challenge

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTagUseCase_Create_Success(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	name, color := "tag1", "#ff0000"

	deps.tagRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, tag *entity.Tag) {
		assert.Equal(t, name, tag.Name)
		assert.Equal(t, color, tag.Color)
	})

	uc := h.CreateTagUseCase()
	got, err := uc.Create(ctx, name, color)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, name, got.Name)
	assert.Equal(t, color, got.Color)
}

func TestTagUseCase_Create_Error(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	name, color := "tag1", "#ff0000"

	deps.tagRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := h.CreateTagUseCase()
	got, err := uc.Create(ctx, name, color)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestTagUseCase_GetByID_Success(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()
	tag := h.NewTag("t", "#ccc")
	tag.ID = id

	deps.tagRepo.EXPECT().GetByID(mock.Anything, id).Return(tag, nil)

	uc := h.CreateTagUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, tag.Name, got.Name)
}

func TestTagUseCase_GetByID_Error(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.tagRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := h.CreateTagUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestTagUseCase_GetAll_Success(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	list := []*entity.Tag{h.NewTag("t1", "#aaa"), h.NewTag("t2", "#bbb")}

	deps.tagRepo.EXPECT().GetAll(mock.Anything).Return(list, nil)

	uc := h.CreateTagUseCase()
	got, err := uc.GetAll(ctx)

	assert.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestTagUseCase_GetAll_Error(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()

	deps.tagRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := h.CreateTagUseCase()
	got, err := uc.GetAll(ctx)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestTagUseCase_Update_Success(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()
	tag := h.NewTag("old", "#000")
	tag.ID = id
	name, color := "new", "#fff"

	deps.tagRepo.EXPECT().GetByID(mock.Anything, id).Return(tag, nil)
	deps.tagRepo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, tag *entity.Tag) {
		assert.Equal(t, name, tag.Name)
		assert.Equal(t, color, tag.Color)
	})

	uc := h.CreateTagUseCase()
	got, err := uc.Update(ctx, id, name, color)

	assert.NoError(t, err)
	assert.Equal(t, name, got.Name)
}

func TestTagUseCase_Update_Error(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.tagRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := h.CreateTagUseCase()
	got, err := uc.Update(ctx, id, "name", "color")

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestTagUseCase_Delete_Success(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.tagRepo.EXPECT().Delete(mock.Anything, id).Return(nil)

	uc := h.CreateTagUseCase()
	err := uc.Delete(ctx, id)

	assert.NoError(t, err)
}

func TestTagUseCase_Delete_Error(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.tagRepo.EXPECT().Delete(mock.Anything, id).Return(assert.AnError)

	uc := h.CreateTagUseCase()
	err := uc.Delete(ctx, id)

	assert.Error(t, err)
}
