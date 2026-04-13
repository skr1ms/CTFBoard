package computil

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// stubGetter is a minimal CompetitionGetter for testing.
type stubGetter struct {
	comp *domain.Competition
	err  error
	// called tracks whether Get was invoked
	called bool
}

func (s *stubGetter) Get(_ context.Context) (*domain.Competition, error) {
	s.called = true

	return s.comp, s.err
}

var errDB = errors.New("db error")

// ---- Cached ----

func TestCached_UCSucceeds_RepoNotCalled(t *testing.T) {
	t.Parallel()

	comp := &domain.Competition{Name: "CTF"}
	uc := &stubGetter{comp: comp}
	repo := &stubGetter{}

	got := Cached(context.Background(), uc, repo)

	require.NotNil(t, got)
	assert.Equal(t, "CTF", got.Name)
	assert.True(t, uc.called)
	assert.False(t, repo.called, "repo should not be called when uc succeeds")
}

func TestCached_UCFails_RepoSucceeds(t *testing.T) {
	t.Parallel()

	comp := &domain.Competition{Name: "fallback"}
	uc := &stubGetter{err: errDB}
	repo := &stubGetter{comp: comp}

	got := Cached(context.Background(), uc, repo)

	require.NotNil(t, got)
	assert.Equal(t, "fallback", got.Name)
	assert.True(t, uc.called)
	assert.True(t, repo.called)
}

func TestCached_BothFail_ReturnsNil(t *testing.T) {
	t.Parallel()

	uc := &stubGetter{err: errDB}
	repo := &stubGetter{err: errDB}

	got := Cached(context.Background(), uc, repo)

	assert.Nil(t, got)
}

func TestCached_BothNil_ReturnsNil(t *testing.T) {
	t.Parallel()

	got := Cached(context.Background(), nil, nil)

	assert.Nil(t, got)
}

func TestCached_NilUC_RepoSucceeds(t *testing.T) {
	t.Parallel()

	comp := &domain.Competition{Name: "direct"}
	repo := &stubGetter{comp: comp}

	got := Cached(context.Background(), nil, repo)

	require.NotNil(t, got)
	assert.Equal(t, "direct", got.Name)
}

// ---- Fresh ----

func TestFresh_RepoSucceeds_UCNotCalled(t *testing.T) {
	t.Parallel()

	comp := &domain.Competition{Name: "fresh"}
	repo := &stubGetter{comp: comp}
	uc := &stubGetter{}

	got := Fresh(context.Background(), repo, uc)

	require.NotNil(t, got)
	assert.Equal(t, "fresh", got.Name)
	assert.True(t, repo.called)
	assert.False(t, uc.called, "uc should not be called when repo succeeds")
}

func TestFresh_RepoFails_UCSucceeds(t *testing.T) {
	t.Parallel()

	comp := &domain.Competition{Name: "cached-fallback"}
	repo := &stubGetter{err: errDB}
	uc := &stubGetter{comp: comp}

	got := Fresh(context.Background(), repo, uc)

	require.NotNil(t, got)
	assert.Equal(t, "cached-fallback", got.Name)
	assert.True(t, repo.called)
	assert.True(t, uc.called)
}

func TestFresh_BothFail_ReturnsNil(t *testing.T) {
	t.Parallel()

	repo := &stubGetter{err: errDB}
	uc := &stubGetter{err: errDB}

	got := Fresh(context.Background(), repo, uc)

	assert.Nil(t, got)
}

func TestFresh_NilRepo_UCSucceeds(t *testing.T) {
	t.Parallel()

	comp := &domain.Competition{Name: "uc"}
	uc := &stubGetter{comp: comp}

	got := Fresh(context.Background(), nil, uc)

	require.NotNil(t, got)
	assert.Equal(t, "uc", got.Name)
}
