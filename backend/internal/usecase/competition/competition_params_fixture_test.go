package competition

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func competitionParamSetParams(key, value, description string, valueType domain.CompetitionParamValueType, category string, actorID uuid.UUID, clientIP string) usecase.CompetitionParamSetParams {
	return usecase.CompetitionParamSetParams{
		Key:         key,
		Value:       value,
		Description: description,
		ValueType:   valueType,
		Category:    category,
		ActorID:     actorID,
		ClientIP:    clientIP,
	}
}

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
