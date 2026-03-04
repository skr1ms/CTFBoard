package competition

import (
	"context"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCompetitionParamUseCase_Get_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "k"
	p := h.NewCompetitionParam(key, "v", "desc", entity.CompetitionParamTypeString)

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{p}, nil)

	uc := h.CreateCompetitionParamUseCase()
	got, err := uc.Get(ctx, key)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, key, got.Key)
	assert.Equal(t, "v", got.Value)
}

func TestCompetitionParamUseCase_Get_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "k"

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := h.CreateCompetitionParamUseCase()
	got, err := uc.Get(ctx, key)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestCompetitionParamUseCase_GetAll_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	list := []*entity.CompetitionParam{h.NewCompetitionParam("k1", "v1", "", entity.CompetitionParamTypeString)}

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return(list, nil)

	uc := h.CreateCompetitionParamUseCase()
	_, _ = uc.Get(ctx, "k1") //nolint:errcheck // setup call
	got, err := uc.GetAll(ctx)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestCompetitionParamUseCase_GetAll_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := h.CreateCompetitionParamUseCase()
	got, err := uc.GetAll(ctx)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestCompetitionParamUseCase_Set_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key, value, desc := "k", "v", "d"
	valueType := entity.CompetitionParamTypeString
	actorID := uuid.New()
	clientIP := "127.0.0.1"

	deps.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, p *entity.CompetitionParam) {
		assert.Equal(t, key, p.Key)
		assert.Equal(t, value, p.Value)
		assert.Equal(t, valueType, p.ValueType)
	})
	deps.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := h.CreateCompetitionParamUseCase()
	err := uc.Set(ctx, key, value, desc, valueType, actorID, clientIP)

	assert.NoError(t, err)
}

func TestCompetitionParamUseCase_Set_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key, value := "k", "v"
	actorID := uuid.New()

	deps.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := h.CreateCompetitionParamUseCase()
	err := uc.Set(ctx, key, value, "", entity.CompetitionParamTypeString, actorID, "")

	assert.Error(t, err)
}

func TestCompetitionParamUseCase_Delete_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "k"
	p := h.NewCompetitionParam(key, "v", "", entity.CompetitionParamTypeString)
	actorID := uuid.New()
	clientIP := "127.0.0.1"

	deps.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(p, nil)
	deps.configRepo.EXPECT().Delete(mock.Anything, key).Return(nil)
	deps.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := h.CreateCompetitionParamUseCase()
	err := uc.Delete(ctx, key, actorID, clientIP)

	assert.NoError(t, err)
}

func TestCompetitionParamUseCase_Delete_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "k"
	actorID := uuid.New()

	deps.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(nil, assert.AnError)

	uc := h.CreateCompetitionParamUseCase()
	err := uc.Delete(ctx, key, actorID, "")

	assert.Error(t, err)
}

func TestCompetitionParamUseCase_GetString_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "k"
	p := h.NewCompetitionParam(key, "val", "", entity.CompetitionParamTypeString)

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{p}, nil)

	uc := h.CreateCompetitionParamUseCase()
	got := uc.GetString(ctx, key, "default")

	assert.Equal(t, "val", got)
}

func TestCompetitionParamUseCase_GetString_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "missing"
	defaultVal := "def"

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{}, nil)
	deps.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(nil, httperr.ErrCompetitionParamNotFound)

	uc := h.CreateCompetitionParamUseCase()
	got := uc.GetString(ctx, key, defaultVal)

	assert.Equal(t, defaultVal, got)
}

func TestCompetitionParamUseCase_GetInt_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "k"
	p := h.NewCompetitionParam(key, "42", "", entity.CompetitionParamTypeInt)

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{p}, nil)

	uc := h.CreateCompetitionParamUseCase()
	got := uc.GetInt(ctx, key, 0)

	assert.Equal(t, 42, got)
}

func TestCompetitionParamUseCase_GetInt_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "missing"
	defaultVal := 10

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{}, nil)
	deps.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(nil, httperr.ErrCompetitionParamNotFound)

	uc := h.CreateCompetitionParamUseCase()
	got := uc.GetInt(ctx, key, defaultVal)

	assert.Equal(t, defaultVal, got)
}

func TestCompetitionParamUseCase_GetBool_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "k"
	p := h.NewCompetitionParam(key, "true", "", entity.CompetitionParamTypeBool)

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{p}, nil)

	uc := h.CreateCompetitionParamUseCase()
	got := uc.GetBool(ctx, key, false)

	assert.True(t, got)
}

func TestCompetitionParamUseCase_GetBool_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "missing"
	defaultVal := true

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{}, nil)
	deps.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(nil, httperr.ErrCompetitionParamNotFound)

	uc := h.CreateCompetitionParamUseCase()
	got := uc.GetBool(ctx, key, defaultVal)

	assert.Equal(t, defaultVal, got)
}
