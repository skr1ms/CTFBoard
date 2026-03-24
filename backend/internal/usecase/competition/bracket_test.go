package competition

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestBracketUseCase_Create_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	name, desc := "bracket1", "desc"
	isDefault := true

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.bracketRepo.EXPECT().ClearAllDefaults(mock.Anything).Return(nil).Once()
	d.bracketRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, b *domain.Bracket) {
		assert.Equal(t, name, b.Name)
		assert.Equal(t, desc, b.Description)
		assert.Equal(t, isDefault, b.IsDefault)
	})

	uc := d.createBracketUseCase()
	got, err := uc.Create(ctx, name, desc, true)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, name, got.Name)
}

func TestBracketUseCase_Create_NotDefault_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	name, desc := "bracket1", "desc"

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.bracketRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := d.createBracketUseCase()
	got, err := uc.Create(ctx, name, desc, false)

	assert.NoError(t, err)
	assert.NotNil(t, got)
}

func TestBracketUseCase_Create_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	name, desc := "bracket1", "desc"

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.bracketRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := d.createBracketUseCase()
	got, err := uc.Create(ctx, name, desc, false)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestBracketUseCase_GetByID_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	id := uuid.New()
	bracket := newTestBracket("b", "d", false)
	bracket.ID = id

	d.bracketRepo.EXPECT().GetByID(mock.Anything, id).Return(bracket, nil)

	uc := d.createBracketUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestBracketUseCase_GetByID_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.bracketRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := d.createBracketUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestBracketUseCase_GetAll_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	list := []*domain.Bracket{newTestBracket("b1", "d", false)}

	d.bracketRepo.EXPECT().GetAll(mock.Anything).Return(list, nil)

	uc := d.createBracketUseCase()
	got, err := uc.GetAll(ctx)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestBracketUseCase_GetAll_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()

	d.bracketRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := d.createBracketUseCase()
	got, err := uc.GetAll(ctx)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestBracketUseCase_Update_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	id := uuid.New()
	bracket := newTestBracket("old", "oldd", false)
	bracket.ID = id
	name, desc := "new", "newd"
	isDefault := true

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.bracketRepo.EXPECT().GetByID(mock.Anything, id).Return(bracket, nil)
	d.bracketRepo.EXPECT().ClearAllDefaults(mock.Anything).Return(nil).Once()
	d.bracketRepo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, b *domain.Bracket) {
		assert.Equal(t, name, b.Name)
		assert.Equal(t, desc, b.Description)
		assert.Equal(t, isDefault, b.IsDefault)
	})

	uc := d.createBracketUseCase()
	got, err := uc.Update(ctx, id, name, desc, true)

	assert.NoError(t, err)
	assert.Equal(t, name, got.Name)
}

func TestBracketUseCase_Update_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.bracketRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := d.createBracketUseCase()
	got, err := uc.Update(ctx, id, "name", "desc", false)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestBracketUseCase_Delete_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.bracketRepo.EXPECT().Delete(mock.Anything, id).Return(nil)

	uc := d.createBracketUseCase()
	err := uc.Delete(ctx, id)

	assert.NoError(t, err)
}

func TestBracketUseCase_Delete_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.bracketRepo.EXPECT().Delete(mock.Anything, id).Return(assert.AnError)

	uc := d.createBracketUseCase()
	err := uc.Delete(ctx, id)

	assert.Error(t, err)
}
