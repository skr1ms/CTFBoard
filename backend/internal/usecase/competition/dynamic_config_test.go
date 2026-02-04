package competition

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDynamicConfigUseCase_Get_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "k"
	cfg := h.NewConfig(key, "v", "desc", entity.ConfigTypeString)

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.Config{cfg}, nil)

	uc := h.CreateDynamicConfigUseCase()
	got, err := uc.Get(ctx, key)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, key, got.Key)
	assert.Equal(t, "v", got.Value)
}

func TestDynamicConfigUseCase_Get_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "k"

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := h.CreateDynamicConfigUseCase()
	got, err := uc.Get(ctx, key)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestDynamicConfigUseCase_GetAll_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	list := []*entity.Config{h.NewConfig("k1", "v1", "", entity.ConfigTypeString)}

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return(list, nil)

	uc := h.CreateDynamicConfigUseCase()
	_, _ = uc.Get(ctx, "k1") //nolint:errcheck // setup call
	got, err := uc.GetAll(ctx)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestDynamicConfigUseCase_GetAll_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := h.CreateDynamicConfigUseCase()
	got, err := uc.GetAll(ctx)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestDynamicConfigUseCase_Set_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key, value, desc := "k", "v", "d"
	valueType := entity.ConfigTypeString
	actorID := uuid.New()
	clientIP := "127.0.0.1"

	deps.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, cfg *entity.Config) {
		assert.Equal(t, key, cfg.Key)
		assert.Equal(t, value, cfg.Value)
		assert.Equal(t, valueType, cfg.ValueType)
	})
	deps.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := h.CreateDynamicConfigUseCase()
	err := uc.Set(ctx, key, value, desc, valueType, actorID, clientIP)

	assert.NoError(t, err)
}

func TestDynamicConfigUseCase_Set_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key, value := "k", "v"
	actorID := uuid.New()

	deps.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := h.CreateDynamicConfigUseCase()
	err := uc.Set(ctx, key, value, "", entity.ConfigTypeString, actorID, "")

	assert.Error(t, err)
}

func TestDynamicConfigUseCase_Delete_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "k"
	cfg := h.NewConfig(key, "v", "", entity.ConfigTypeString)
	actorID := uuid.New()
	clientIP := "127.0.0.1"

	deps.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(cfg, nil)
	deps.configRepo.EXPECT().Delete(mock.Anything, key).Return(nil)
	deps.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := h.CreateDynamicConfigUseCase()
	err := uc.Delete(ctx, key, actorID, clientIP)

	assert.NoError(t, err)
}

func TestDynamicConfigUseCase_Delete_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "k"
	actorID := uuid.New()

	deps.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(nil, assert.AnError)

	uc := h.CreateDynamicConfigUseCase()
	err := uc.Delete(ctx, key, actorID, "")

	assert.Error(t, err)
}

func TestDynamicConfigUseCase_GetString_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "k"
	cfg := h.NewConfig(key, "val", "", entity.ConfigTypeString)

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.Config{cfg}, nil)

	uc := h.CreateDynamicConfigUseCase()
	got := uc.GetString(ctx, key, "default")

	assert.Equal(t, "val", got)
}

func TestDynamicConfigUseCase_GetString_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "missing"
	defaultVal := "def"

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.Config{}, nil)

	uc := h.CreateDynamicConfigUseCase()
	got := uc.GetString(ctx, key, defaultVal)

	assert.Equal(t, defaultVal, got)
}

func TestDynamicConfigUseCase_GetInt_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "k"
	cfg := h.NewConfig(key, "42", "", entity.ConfigTypeInt)

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.Config{cfg}, nil)

	uc := h.CreateDynamicConfigUseCase()
	got := uc.GetInt(ctx, key, 0)

	assert.Equal(t, 42, got)
}

func TestDynamicConfigUseCase_GetInt_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "missing"
	defaultVal := 10

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := h.CreateDynamicConfigUseCase()
	got := uc.GetInt(ctx, key, defaultVal)

	assert.Equal(t, defaultVal, got)
}

func TestDynamicConfigUseCase_GetBool_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "k"
	cfg := h.NewConfig(key, "true", "", entity.ConfigTypeBool)

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.Config{cfg}, nil)

	uc := h.CreateDynamicConfigUseCase()
	got := uc.GetBool(ctx, key, false)

	assert.True(t, got)
}

func TestDynamicConfigUseCase_GetBool_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	key := "missing"
	defaultVal := true

	deps.configRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := h.CreateDynamicConfigUseCase()
	got := uc.GetBool(ctx, key, defaultVal)

	assert.Equal(t, defaultVal, got)
}
