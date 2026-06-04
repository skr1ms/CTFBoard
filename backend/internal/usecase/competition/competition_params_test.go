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

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

type fakeKeyValueStore struct {
	mu    sync.Mutex
	store map[string][]byte
}

func (f *fakeKeyValueStore) Get(ctx context.Context, key string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if v, ok := f.store[key]; ok {
		return v, nil
	}

	return nil, errors.New("key not found")
}

func (f *fakeKeyValueStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.store == nil {
		f.store = make(map[string][]byte)
	}

	f.store[key] = value

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

type fakePubSubStore struct {
	subscribeCh  <-chan string
	subscribeErr error
	publishErr   error
	publishCalls []struct{ Channel, Message string }
	mu           sync.Mutex
}

func (f *fakePubSubStore) Subscribe(_ context.Context, _ string) (<-chan string, error) {
	if f.subscribeErr != nil {
		return nil, f.subscribeErr
	}

	if f.subscribeCh != nil {
		return f.subscribeCh, nil
	}

	return make(chan string), nil
}

func (f *fakePubSubStore) Publish(_ context.Context, channel, message string) error {
	f.mu.Lock()
	f.publishCalls = append(f.publishCalls, struct{ Channel, Message string }{channel, message})
	f.mu.Unlock()

	return f.publishErr
}

func TestCompetitionParamUseCase_Get_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	p := newTestCompetitionParam(key, "v", "desc", domain.CompetitionParamTypeString)

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{p}, nil)

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
	list := []*domain.CompetitionParam{newTestCompetitionParam("k1", "v1", "", domain.CompetitionParamTypeString)}

	d.configRepo.EXPECT().GetAll(mock.Anything).Return(list, nil)

	uc := d.createCompetitionParamUseCase()
	_, _ = uc.Get(ctx, "k1") //nolint:errcheck // setup call
	got, err := uc.GetAll(ctx)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(got), domain.ConfigRegistryCount())

	var k1 *domain.CompetitionParam

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
	valueType := domain.CompetitionParamTypeString
	actorID := uuid.New()
	clientIP := "127.0.0.1"

	d.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, p *domain.CompetitionParam) {
		assert.Equal(t, key, p.Key)
		assert.Equal(t, value, p.Value)
		assert.Equal(t, valueType, p.ValueType)
	})
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	uc := d.createCompetitionParamUseCase()
	err := uc.Set(ctx, key, value, desc, valueType, "", actorID, clientIP)

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
	err := uc.Set(ctx, key, value, "", domain.CompetitionParamTypeString, "", actorID, "")

	assert.Error(t, err)
}

func TestCompetitionParamUseCase_Set_InvalidVisibility_ReturnsError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()

	uc := d.createCompetitionParamUseCase()
	err := uc.Set(ctx, "score_visibility", "garbage", "", domain.CompetitionParamTypeString, "", uuid.New(), "")

	assert.Error(t, err)

	var ve *apperr.ValidationError
	assert.ErrorAs(t, err, &ve)
}

func TestCompetitionParamUseCase_Delete_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	p := newTestCompetitionParam(key, "v", "", domain.CompetitionParamTypeString)
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
	p := newTestCompetitionParam(key, "val", "", domain.CompetitionParamTypeString)

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{p}, nil)

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

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{}, nil)
	d.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(nil, apperr.ErrCompetitionParamNotFound)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetString(ctx, key, defaultVal)

	assert.Equal(t, defaultVal, got)
}

func TestCompetitionParamUseCase_GetInt_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	p := newTestCompetitionParam(key, "42", "", domain.CompetitionParamTypeInt)

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{p}, nil)

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

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{}, nil)
	d.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(nil, apperr.ErrCompetitionParamNotFound)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetInt(ctx, key, defaultVal)

	assert.Equal(t, defaultVal, got)
}

func TestCompetitionParamUseCase_GetBool_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "k"
	p := newTestCompetitionParam(key, "true", "", domain.CompetitionParamTypeBool)

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{p}, nil)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetBool(ctx, key, false)

	assert.True(t, got)
}

func TestCompetitionParamUseCase_GetBool_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "missing"

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{}, nil)
	d.configRepo.EXPECT().GetByKey(mock.Anything, key).Return(nil, apperr.ErrCompetitionParamNotFound)

	uc := d.createCompetitionParamUseCase()
	got := uc.GetBool(ctx, key, true)

	assert.True(t, got)
}

func TestCompetitionParamUseCase_GetByCategory_ReturnsOnlyCategoryTheme(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	theme1 := newTestCompetitionParam("theme_color_primary", "#fff", "", domain.CompetitionParamTypeString)
	theme1.Category = "theme"
	theme2 := newTestCompetitionParam("theme_dark_mode", "true", "", domain.CompetitionParamTypeBool)
	theme2.Category = "theme"

	d.configRepo.EXPECT().GetByCategory(mock.Anything, "theme").Return([]*domain.CompetitionParam{theme1, theme2}, nil)

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

	var ve *apperr.ValidationError
	assert.ErrorAs(t, err, &ve)
}

func TestCompetitionParamUseCase_SetBatch_InvalidCategory_ReturnsError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	params := []*domain.CompetitionParam{
		{Key: "k", Value: "v", ValueType: domain.CompetitionParamTypeString, Category: "invalid"},
	}
	actorID := uuid.New()

	uc := d.createCompetitionParamUseCase()
	err := uc.SetBatch(ctx, params, actorID, "")

	assert.Error(t, err)

	var ve2 *apperr.ValidationError
	assert.ErrorAs(t, err, &ve2)
}

func TestCompetitionParamUseCase_SetBatch_InvalidVisibility_ReturnsError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	params := []*domain.CompetitionParam{
		{Key: "challenge_visibility", Value: "invalid-private", ValueType: domain.CompetitionParamTypeString, Category: "visibility"},
	}

	uc := d.createCompetitionParamUseCase()
	err := uc.SetBatch(ctx, params, uuid.New(), "")

	assert.Error(t, err)

	var ve *apperr.ValidationError
	assert.ErrorAs(t, err, &ve)
}

func TestCompetitionParamUseCase_GetAfterSet_ReturnsValue(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key, value := "my_key", "my_value"
	actorID := uuid.New()
	afterSet := newTestCompetitionParam(key, value, "desc", domain.CompetitionParamTypeString)

	d.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{afterSet}, nil)

	uc := d.createCompetitionParamUseCase()
	err := uc.Set(ctx, key, value, "desc", domain.CompetitionParamTypeString, "", actorID, "")
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

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{}, nil)

	uc := d.createCompetitionParamUseCase()
	got, err := uc.GetAll(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(got), domain.ConfigRegistryCount())

	seen := make(map[string]struct{})

	for _, p := range got {
		seen[p.Key] = struct{}{}
	}

	domain.RangeConfigRegistry(func(k string, _ domain.ConfigDef) bool {
		assert.Contains(t, seen, k, "GetAll should include registry key %q", k)

		return true
	})
}

func TestCompetitionParamUseCase_SetBatch_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	params := []*domain.CompetitionParam{
		{Key: "ctf_name", Value: "MyCTF", ValueType: domain.CompetitionParamTypeString, Category: "general"},
		{Key: "theme_color_primary", Value: "#ff0000", ValueType: domain.CompetitionParamTypeString, Category: "theme"},
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
	cached := []*domain.CompetitionParam{
		{Key: key, Value: "FromRedis", ValueType: domain.CompetitionParamTypeString, Category: "general"},
	}
	payload, err := json.Marshal(cached)
	require.NoError(t, err)

	kv := &fakeKeyValueStore{store: map[string][]byte{configsCacheKey: payload}}

	ch := make(chan string, 1)
	ch <- "1"

	close(ch)
	pubsub := &fakePubSubStore{subscribeCh: ch}

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

	kv := &fakeKeyValueStore{store: map[string][]byte{configsCacheKey: []byte("stale")}}
	pubsub := &fakePubSubStore{}

	uc := d.createCompetitionParamUseCaseWithCache(kv, pubsub)
	err := uc.Set(ctx, key, value, "", domain.CompetitionParamTypeString, "", actorID, "")

	assert.NoError(t, err)

	_, ok := kv.store[configsCacheKey]
	assert.False(t, ok, "invalidate should have deleted configs cache key")
	require.Len(t, pubsub.publishCalls, 1)
	assert.Equal(t, configsInvChannel, pubsub.publishCalls[0].Channel)
	assert.Equal(t, "1", pubsub.publishCalls[0].Message)
}

func TestCompetitionParamUseCase_Get_WhenRedisReturnsInvalidJSON_FallsBackToDB(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "ctf_name"
	fromDB := newTestCompetitionParam(key, "FromDB", "", domain.CompetitionParamTypeString)
	fromDB.Category = "general"

	kv := &fakeKeyValueStore{store: map[string][]byte{configsCacheKey: []byte("not-valid-json")}}
	pubsub := &fakePubSubStore{}

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{fromDB}, nil)

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
	err := uc.Set(ctx, key, "not valid json", "", domain.CompetitionParamTypeJSON, "", actorID, "")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrCompetitionParamInvalidValueType) ||
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
	err := uc.Set(ctx, key, `{"a":1}`, "", domain.CompetitionParamTypeJSON, "", actorID, "")

	assert.NoError(t, err)
}

func TestCompetitionParamUseCase_Get_WhenPubSubSubscribeFails_StillLoadsFromDB(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	ctx := context.Background()
	key := "ctf_name"
	fromDB := newTestCompetitionParam(key, "FromDB", "", domain.CompetitionParamTypeString)
	fromDB.Category = "general"

	pubsub := &fakePubSubStore{subscribeErr: assert.AnError}

	d.configRepo.EXPECT().GetAll(mock.Anything).Return([]*domain.CompetitionParam{fromDB}, nil)

	uc := d.createCompetitionParamUseCaseWithCache(nil, pubsub)
	got, err := uc.Get(ctx, key)

	assert.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "FromDB", got.Value)
}
