package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestRequireTeam_NoUser_Error(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Use(RequireTeam())
	r.Get("/", okHandler())

	req := newRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireTeam_Admin_Success(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := &domain.User{ID: uuid.New(), Role: domain.RoleAdmin, TeamID: nil}
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(RequireTeam())
	r.Get("/", okHandler())

	req := newRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireTeam_NoTeam_Error(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := &domain.User{ID: uuid.New(), Role: domain.RoleUser, TeamID: nil}
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(RequireTeam())
	r.Get("/", okHandler())

	req := newRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestRequireTeam_HasTeam_Success(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := &domain.User{ID: uuid.New(), Role: domain.RoleUser, TeamID: &teamID}
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(RequireTeam())
	r.Get("/", okHandler())

	req := newRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
