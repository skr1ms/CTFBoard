package competition

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
)

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
	err := uc.Set(ctx, competitionParamSetParams(key, value, "", domain.CompetitionParamTypeString, "", actorID, ""))

	assert.NoError(t, err)

	_, ok := kv.store[configsCacheKey]
	assert.False(t, ok, "invalidate should have deleted configs cache key")
	require.Len(t, pubsub.publishCalls, 1)
	assert.Equal(t, configsInvChannel, pubsub.publishCalls[0].Channel)
	assert.Equal(t, "1", pubsub.publishCalls[0].Message)
}

func TestCompetitionParamUseCase_Set_DefersInvalidationInTransaction(t *testing.T) {
	t.Parallel()

	d := newCompetitionTestDeps(t)
	collector := txctx.NewCollector()
	ctx := txctx.WithCollector(context.Background(), collector)
	key, value := "k", "v"
	actorID := uuid.New()

	d.configRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	kv := &fakeKeyValueStore{store: map[string][]byte{configsCacheKey: []byte("stale")}}
	pubsub := &fakePubSubStore{}

	uc := d.createCompetitionParamUseCaseWithCache(kv, pubsub)
	err := uc.Set(ctx, competitionParamSetParams(key, value, "", domain.CompetitionParamTypeString, "", actorID, ""))

	assert.NoError(t, err)

	_, ok := kv.store[configsCacheKey]
	assert.True(t, ok, "invalidate should wait for outer commit")
	assert.Empty(t, pubsub.publishCalls)

	collector.Run(context.Background())

	_, ok = kv.store[configsCacheKey]
	assert.False(t, ok)
	require.Len(t, pubsub.publishCalls, 1)
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
