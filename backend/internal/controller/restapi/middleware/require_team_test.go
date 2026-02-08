package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireTeam_NoUser_Error(t *testing.T) {
	r := chi.NewRouter()
	r.Use(RequireTeam(string(entity.ModeFlexible)))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireTeam_Admin_Success(t *testing.T) {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := &entity.User{ID: uuid.New(), Role: entity.RoleAdmin, TeamID: nil}
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(RequireTeam(string(entity.ModeFlexible)))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireTeam_NoTeam_Error(t *testing.T) {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := &entity.User{ID: uuid.New(), Role: entity.RoleUser, TeamID: nil}
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(RequireTeam(string(entity.ModeFlexible)))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireTeam_HasTeam_Success(t *testing.T) {
	teamID := uuid.New()
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := &entity.User{ID: uuid.New(), Role: entity.RoleUser, TeamID: &teamID}
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(RequireTeam(string(entity.ModeFlexible)))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
