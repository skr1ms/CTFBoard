package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireVerified_Disabled_Success(t *testing.T) {
	r := chi.NewRouter()
	r.Use(RequireVerified(false))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireVerified_NoUser_Error(t *testing.T) {
	r := chi.NewRouter()
	r.Use(RequireVerified(true))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireVerified_Admin_Success(t *testing.T) {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := &entity.User{ID: uuid.New(), Role: entity.RoleAdmin, IsVerified: false}
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(RequireVerified(true))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireVerified_Unverified_Error(t *testing.T) {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := &entity.User{ID: uuid.New(), Role: entity.RoleUser, IsVerified: false}
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(RequireVerified(true))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireVerified_Verified_Success(t *testing.T) {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := &entity.User{ID: uuid.New(), Role: entity.RoleUser, IsVerified: true}
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(RequireVerified(true))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func withUser(ctx context.Context, u *entity.User) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}
