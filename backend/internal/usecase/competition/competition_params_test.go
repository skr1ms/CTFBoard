package competition

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func TestCompetitionParamUseCase_Get_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	p := newTestCompetitionParam(key, "v", "desc", entity.CompetitionParamTypeString)

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{p}, nil)

	uc := d.createCompetitionParamUseCase()
	got, err := uc.Get(ctx, key)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, key, got.Key)
	assert.Equal(t, "v", got.Value)
}

func TestCompetitionParamUseCase_Get_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"

	d.configRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := d.createCompetitionParamUseCase()
	got, err := uc.Get(ctx, key)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestCompetitionParamUseCase_GetAll_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	list := []*entity.CompetitionParam{newTestCompetitionParam("k1", "v1", "", entity.CompetitionParamTypeString)}

	d.configRepo.EXPECT().GetAll(mock.Anything).Return(list, nil)

	uc := d.createCompetitionParamUseCase()
	_, _ = uc.Get(ctx, "k1") //nolint:errcheck // setup call
	got, err := uc.GetAll(ctx)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestCompetitionParamUseCase_GetAll_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()

	d.configRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := d.createCompetitionParamUseCase()
	got, err := uc.GetAll(ctx)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestCompetitionParamUseCase_Set_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key, value, desc := "k", "v", "d"
	valueType := entity.CompetitionParamTypeString
	actorID := uuid.New()
	clientIP := "127.0.0.1"

	d.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, p *entity.CompetitionParam) {
		assert.Equal(t, key, p.Key)
		assert.Equal(t, value, p.Value)
		assert.Equal(t, valueType, p.ValueType)
	})
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := d.createCompetitionParamUseCase()
	err := uc.Set(ctx, key, value, desc, valueType, actorID, clientIP)

	assert.NoError(t, err)
}

func TestCompetitionParamUseCase_Set_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key, value := "k", "v"
	actorID := uuid.New()

	d.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := d.createCompetitionParamUseCase()
	err := uc.Set(ctx, key, value, "", entity.CompetitionParamTypeString, actorID, "")

	assert.Error(t, err)
}

func TestCompetitionParamUseCase_Delete_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	p := newTestCompetitionParam(key, "v", "", entity.CompetitionParamTypeString)
	actorID := uuid.New()
	clientIP := "127.0.0.1"

	d.configRepo.EXPECT().GetByKeyForUpdate(mock.Anything, key).Return(p, nil)
	d.configRepo.EXPECT().Delete(mock.Anything, key).Return(nil)
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := d.createCompetitionParamUseCase()
	err := uc.Delete(ctx, key, actorID, clientIP)

	assert.NoError(t, err)
}

func TestCompetitionParamUseCase_Delete_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	actorID := uuid.New()

	d.configRepo.EXPECT().GetByKeyForUpdate(mock.Anything, key).Return(nil, assert.AnError)

	uc := d.createCompetitionParamUseCase()
	err := uc.Delete(ctx, key, actorID, "")

	assert.Error(t, err)
}

func TestCompetitionParamUseCase_GetString_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	p := newTestCompetitionParam(key, "val", "", entity.CompetitionParamTypeString)

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{p}, nil)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetString(ctx, key, "default")

	assert.Equal(t, "val", got)
}

func TestCompetitionParamUseCase_GetString_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "missing"
	defaultVal := "def"

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{}, nil)
	d.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(nil, httperr.ErrCompetitionParamNotFound)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetString(ctx, key, defaultVal)

	assert.Equal(t, defaultVal, got)
}

func TestCompetitionParamUseCase_GetInt_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	p := newTestCompetitionParam(key, "42", "", entity.CompetitionParamTypeInt)

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{p}, nil)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetInt(ctx, key, 0)

	assert.Equal(t, 42, got)
}

func TestCompetitionParamUseCase_GetInt_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "missing"
	defaultVal := 10

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{}, nil)
	d.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(nil, httperr.ErrCompetitionParamNotFound)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetInt(ctx, key, defaultVal)

	assert.Equal(t, defaultVal, got)
}

func TestCompetitionParamUseCase_GetBool_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	p := newTestCompetitionParam(key, "true", "", entity.CompetitionParamTypeBool)

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{p}, nil)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetBool(ctx, key, false)

	assert.True(t, got)
}

func TestCompetitionParamUseCase_GetBool_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "missing"

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{}, nil)
	d.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(nil, httperr.ErrCompetitionParamNotFound)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetBool(ctx, key, true)

	assert.Equal(t, true, got)
}
