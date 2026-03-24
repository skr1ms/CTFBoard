package challenge

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestTagUseCase_Create_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	name, color := "tag1", "#ff0000"

	d.tagRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, tag *domain.Tag) {
		assert.Equal(t, name, tag.Name)
		assert.Equal(t, color, tag.Color)
	})

	uc := d.createTagUseCase()
	got, err := uc.Create(ctx, name, color)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, name, got.Name)
	assert.Equal(t, color, got.Color)
}

func TestTagUseCase_Create_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	name, color := "tag1", "#ff0000"

	d.tagRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := d.createTagUseCase()
	got, err := uc.Create(ctx, name, color)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestTagUseCase_GetByID_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	id := uuid.New()
	tag := newTestTag("t", "#ccc")
	tag.ID = id

	d.tagRepo.EXPECT().GetByID(mock.Anything, id).Return(tag, nil)

	uc := d.createTagUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, tag.Name, got.Name)
}

func TestTagUseCase_GetByID_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.tagRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := d.createTagUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestTagUseCase_GetAll_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	list := []*domain.Tag{newTestTag("t1", "#aaa"), newTestTag("t2", "#bbb")}

	d.tagRepo.EXPECT().GetAll(mock.Anything).Return(list, nil)

	uc := d.createTagUseCase()
	got, err := uc.GetAll(ctx)

	assert.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestTagUseCase_GetAll_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()

	d.tagRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := d.createTagUseCase()
	got, err := uc.GetAll(ctx)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestTagUseCase_Update_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	id := uuid.New()
	tag := newTestTag("old", "#000")
	tag.ID = id
	name, color := "new", "#fff"

	d.tagRepo.EXPECT().GetByID(mock.Anything, id).Return(tag, nil)
	d.tagRepo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, tag *domain.Tag) {
		assert.Equal(t, name, tag.Name)
		assert.Equal(t, color, tag.Color)
	})

	uc := d.createTagUseCase()
	got, err := uc.Update(ctx, id, name, color)

	assert.NoError(t, err)
	assert.Equal(t, name, got.Name)
}

func TestTagUseCase_Update_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.tagRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := d.createTagUseCase()
	got, err := uc.Update(ctx, id, "name", "color")

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestTagUseCase_Delete_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.tagRepo.EXPECT().Delete(mock.Anything, id).Return(nil)

	uc := d.createTagUseCase()
	err := uc.Delete(ctx, id)

	assert.NoError(t, err)
}

func TestTagUseCase_Delete_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.tagRepo.EXPECT().Delete(mock.Anything, id).Return(assert.AnError)

	uc := d.createTagUseCase()
	err := uc.Delete(ctx, id)

	assert.Error(t, err)
}
