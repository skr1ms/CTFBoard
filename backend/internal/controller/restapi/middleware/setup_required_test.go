package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type setupRequiredStatusStub struct {
	complete bool
	calls    int
	err      error
}

func (s *setupRequiredStatusStub) IsComplete(context.Context) (bool, error) {
	s.calls++

	return s.complete, s.err
}

func TestSetupRequiredDoesNotCacheIncompleteStatus(t *testing.T) {
	t.Parallel()

	status := &setupRequiredStatusStub{}
	handler := SetupRequired(status, SetupAllowlist{})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", http.NoBody)
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, req)

	status.complete = true
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, req)

	assert.Equal(t, http.StatusServiceUnavailable, first.Code)
	assert.Equal(t, http.StatusOK, second.Code)
	assert.Equal(t, 2, status.calls)
}

func TestSetupRequiredCachesCompleteStatus(t *testing.T) {
	t.Parallel()

	status := &setupRequiredStatusStub{complete: true}
	handler := SetupRequired(status, SetupAllowlist{})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", http.NoBody)
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, req)

	status.complete = false
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, req)

	assert.Equal(t, http.StatusOK, first.Code)
	assert.Equal(t, http.StatusOK, second.Code)
	assert.Equal(t, 1, status.calls)
}

func TestSetupRequiredFailsClosedOnStatusError(t *testing.T) {
	t.Parallel()

	status := &setupRequiredStatusStub{err: errors.New("setup store unavailable")}
	nextCalled := false
	handler := SetupRequired(status, SetupAllowlist{})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.False(t, nextCalled)
	assert.Equal(t, 1, status.calls)
}

func TestSetupRequiredAllowlistExactAndPrefixes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want int
	}{
		{name: "setup exact", path: "/api/v1/setup", want: http.StatusOK},
		{name: "setup prefix rejected", path: "/api/v1/setup-status", want: http.StatusServiceUnavailable},
		{name: "avatar prefix allowed", path: "/api/v1/avatars/users/u/hash.webp", want: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status := &setupRequiredStatusStub{}
			handler := SetupRequired(status, SetupAllowlist{
				Exact:    []string{"/api/v1/setup"},
				Prefixes: []string{"/api/v1/avatars/"},
			})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.want, rec.Code)
		})
	}
}
