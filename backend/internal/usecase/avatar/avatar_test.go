package avatar

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	avatarMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/avatar/mock"
)

func makePNG(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			img.Set(x, y, color.RGBA{R: 128, G: 64, B: 32, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

type fakeCache struct {
	mu    sync.Mutex
	store map[string][]byte
}

func (f *fakeCache) Get(_ context.Context, key string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if v, ok := f.store[key]; ok {
		return v, nil
	}

	return nil, errors.New("cache miss")
}

func (f *fakeCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.store == nil {
		f.store = make(map[string][]byte)
	}

	f.store[key] = value

	return nil
}

func (f *fakeCache) Del(_ context.Context, keys ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, k := range keys {
		delete(f.store, k)
	}

	return nil
}

type avatarTestDeps struct {
	userRepo *avatarMock.MockUserRepository
	teamRepo *avatarMock.MockTeamRepository
	storage  *avatarMock.MockAvatarStorage
	cache    *fakeCache
	logger   logkit.Logger
}

func newAvatarTestDeps(t *testing.T) *avatarTestDeps {
	t.Helper()

	l, err := logkit.New(logkit.WithLevel(logkit.ErrorLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, err)

	return &avatarTestDeps{
		userRepo: avatarMock.NewMockUserRepository(t),
		teamRepo: avatarMock.NewMockTeamRepository(t),
		storage:  avatarMock.NewMockAvatarStorage(t),
		cache:    &fakeCache{},
		logger:   l,
	}
}

func (d *avatarTestDeps) newUseCase() *AvatarUseCase {
	return NewAvatarUseCase(AvatarDeps{
		UserRepo: d.userRepo,
		TeamRepo: d.teamRepo,
		Storage:  d.storage,
		Cache:    d.cache,
		Config:   domain.GetDefaultAvatarConfig(),
		Logger:   d.logger,
	})
}
