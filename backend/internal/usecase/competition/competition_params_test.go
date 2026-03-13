package competition

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type fakeKeyValueStore struct {
	mu    sync.Mutex
	store map[string]string
}

func (f *fakeKeyValueStore) Get(ctx context.Context, key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v, ok := f.store[key]; ok {
		return v, nil
	}
	return "", nil
}

func (f *fakeKeyValueStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.store == nil {
		f.store = make(map[string]string)
	}
	f.store[key] = string(value)
	return nil
}

func (f *fakeKeyValueStore) Del(ctx context.Context, keys ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, k := range keys {
		delete(f.store, k)
	}
	return nil
}

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
	assert.GreaterOrEqual(t, len(got), len(entity.ConfigRegistry))
	var k1 *entity.CompetitionParam
	for _, p := range got {
		if p.Key == "k1" {
			k1 = p
			break
		}
	}
	require.NotNil(t, k1)
	assert.Equal(t, "v1", k1.Value)
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

func TestCompetitionParamUseCase_GetByCategory_ReturnsOnlyCategoryTheme(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	generalParam := newTestCompetitionParam("ctf_name", "v", "", entity.CompetitionParamTypeString)
	generalParam.Category = "general"
	theme1 := newTestCompetitionParam("theme_color_primary", "#fff", "", entity.CompetitionParamTypeString)
	theme1.Category = "theme"
	theme2 := newTestCompetitionParam("theme_dark_mode", "true", "", entity.CompetitionParamTypeBool)
	theme2.Category = "theme"
	all := []*entity.CompetitionParam{generalParam, theme1, theme2}

	d.configRepo.EXPECT().GetAll(mock.Anything).Return(all, nil)

	uc := d.createCompetitionParamUseCase()
	got, err := uc.GetByCategory(ctx, "theme")

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(got), 2)
	for _, p := range got {
		assert.Equal(t, "theme", p.Category)
	}
}

func TestCompetitionParamUseCase_GetByCategory_InvalidCategory_ReturnsError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()

	uc := d.createCompetitionParamUseCase()
	got, err := uc.GetByCategory(ctx, "invalid")

	assert.Error(t, err)
	assert.Nil(t, got)
	var he *httperr.HTTPError
	assert.True(t, assert.ErrorAs(t, err, &he) && he.Code == "VALIDATION_ERROR")
}

func TestCompetitionParamUseCase_SetBatch_InvalidCategory_ReturnsError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	params := []*entity.CompetitionParam{
		{Key: "k", Value: "v", ValueType: entity.CompetitionParamTypeString, Category: "invalid"},
	}
	actorID := uuid.New()

	uc := d.createCompetitionParamUseCase()
	err := uc.SetBatch(ctx, params, actorID, "")

	assert.Error(t, err)
	var he *httperr.HTTPError
	assert.True(t, assert.ErrorAs(t, err, &he) && he.Code == "VALIDATION_ERROR")
}

func TestCompetitionParamUseCase_GetAfterSet_ReturnsValue(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key, value := "my_key", "my_value"
	actorID := uuid.New()
	afterSet := newTestCompetitionParam(key, value, "desc", entity.CompetitionParamTypeString)

	d.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{afterSet}, nil)

	uc := d.createCompetitionParamUseCase()
	err := uc.Set(ctx, key, value, "desc", entity.CompetitionParamTypeString, actorID, "")
	assert.NoError(t, err)
	got, err := uc.Get(ctx, key)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, key, got.Key)
	assert.Equal(t, value, got.Value)
}

func TestCompetitionParamUseCase_GetAll_IncludesDefaults(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{}, nil)

	uc := d.createCompetitionParamUseCase()
	got, err := uc.GetAll(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(got), len(entity.ConfigRegistry))
	seen := make(map[string]struct{})
	for _, p := range got {
		seen[p.Key] = struct{}{}
	}
	for k := range entity.ConfigRegistry {
		assert.Contains(t, seen, k, "GetAll should include registry key %q", k)
	}
}

func TestCompetitionParamUseCase_SetBatch_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	params := []*entity.CompetitionParam{
		{Key: "ctf_name", Value: "MyCTF", ValueType: entity.CompetitionParamTypeString, Category: "general"},
		{Key: "theme_color_primary", Value: "#ff0000", ValueType: entity.CompetitionParamTypeString, Category: "theme"},
	}
	actorID := uuid.New()

	d.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil).Times(2)
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := d.createCompetitionParamUseCase()
	err := uc.SetBatch(ctx, params, actorID, "")
	assert.NoError(t, err)
}

func TestCompetitionParamUseCase_Get_WhenCacheHit_ReturnsFromRedis(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "ctf_name"
	cached := []*entity.CompetitionParam{
		{Key: key, Value: "FromRedis", ValueType: entity.CompetitionParamTypeString, Category: "general"},
	}
	payload, err := json.Marshal(cached)
	require.NoError(t, err)
	kv := &fakeKeyValueStore{store: map[string]string{configsCacheKey: string(payload)}}
	ch := make(chan string, 1)
	ch <- "1"
	close(ch)
	pubsub := mocks.NewMockPubSubStore(t)
	pubsub.EXPECT().Subscribe(mock.Anything, configsInvChannel).Return((<-chan string)(ch), nil).Maybe()

	uc := d.createCompetitionParamUseCaseWithCache(kv, pubsub)
	got, err := uc.Get(ctx, key)

	assert.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, key, got.Key)
	assert.Equal(t, "FromRedis", got.Value)
	d.configRepo.AssertNotCalled(t, "GetAll", mock.Anything)
}

func TestCompetitionParamUseCase_Set_CallsCacheDelAndPubSubPublish(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key, value := "k", "v"
	actorID := uuid.New()

	d.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	kv := &fakeKeyValueStore{store: map[string]string{configsCacheKey: "stale"}}
	pubsub := mocks.NewMockPubSubStore(t)
	pubsub.EXPECT().Subscribe(mock.Anything, configsInvChannel).Return((<-chan string)(make(chan string)), nil).Maybe()
	pubsub.EXPECT().Publish(mock.Anything, configsInvChannel, "1").Return(nil).Once()

	uc := d.createCompetitionParamUseCaseWithCache(kv, pubsub)
	err := uc.Set(ctx, key, value, "", entity.CompetitionParamTypeString, actorID, "")

	assert.NoError(t, err)
	_, ok := kv.store[configsCacheKey]
	assert.False(t, ok, "invalidate should have deleted configs cache key")
}

func TestCompetitionParamUseCase_Get_WhenRedisReturnsInvalidJSON_FallsBackToDB(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "ctf_name"
	fromDB := newTestCompetitionParam(key, "FromDB", "", entity.CompetitionParamTypeString)
	fromDB.Category = "general"

	kv := &fakeKeyValueStore{store: map[string]string{configsCacheKey: "not-valid-json"}}
	pubsub := mocks.NewMockPubSubStore(t)
	pubsub.EXPECT().Subscribe(mock.Anything, configsInvChannel).Return((<-chan string)(make(chan string)), nil).Maybe()

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{fromDB}, nil)

	uc := d.createCompetitionParamUseCaseWithCache(kv, pubsub)
	got, err := uc.Get(ctx, key)

	assert.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, key, got.Key)
	assert.Equal(t, "FromDB", got.Value)
}

func TestCompetitionParamUseCase_Set_JSONValueType_InvalidReturnsError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "some_json_key"
	actorID := uuid.New()

	uc := d.createCompetitionParamUseCase()
	err := uc.Set(ctx, key, "not valid json", "", entity.CompetitionParamTypeJSON, actorID, "")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrCompetitionParamInvalidValueType) ||
		strings.Contains(err.Error(), "validateValueType"))
}

func TestCompetitionParamUseCase_Set_JSONValueType_ValidSucceeds(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "some_json_key"
	actorID := uuid.New()

	d.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := d.createCompetitionParamUseCase()
	err := uc.Set(ctx, key, `{"a":1}`, "", entity.CompetitionParamTypeJSON, actorID, "")

	assert.NoError(t, err)
}

func TestCompetitionParamUseCase_Get_WhenPubSubSubscribeFails_StillLoadsFromDB(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "ctf_name"
	fromDB := newTestCompetitionParam(key, "FromDB", "", entity.CompetitionParamTypeString)
	fromDB.Category = "general"

	pubsub := mocks.NewMockPubSubStore(t)
	pubsub.EXPECT().Subscribe(mock.Anything, configsInvChannel).Return((<-chan string)(nil), assert.AnError).Maybe()

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*entity.CompetitionParam{fromDB}, nil)

	uc := d.createCompetitionParamUseCaseWithCache(nil, pubsub)
	got, err := uc.Get(ctx, key)

	assert.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "FromDB", got.Value)
}
