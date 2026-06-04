package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestRequireVerified_Disabled_Success(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Use(RequireVerified(false))
	r.Get("/", okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireVerified_NoUser_Error(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Use(RequireVerified(true))
	r.Get("/", okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireVerified_Admin_Success(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := &domain.User{ID: uuid.New(), Role: domain.RoleAdmin, IsVerified: false}
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(RequireVerified(true))
	r.Get("/", okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireVerified_Unverified_Error(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := &domain.User{ID: uuid.New(), Role: domain.RoleUser, IsVerified: false}
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(RequireVerified(true))
	r.Get("/", okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireVerified_Verified_Success(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := &domain.User{ID: uuid.New(), Role: domain.RoleUser, IsVerified: true}
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(RequireVerified(true))
	r.Get("/", okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireVerifiedFromSettings_UsesLiveDisabledSetting(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := &domain.User{ID: uuid.New(), Role: domain.RoleUser, IsVerified: false}
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(RequireVerifiedFromSettings(true, &staticSettingsGetter{settings: &domain.Settings{VerifyEmails: false}}, logkit.Noop()))
	r.Get("/", okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireVerifiedFromSettings_UsesLiveEnabledSettingWhenFallbackEnabled(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := &domain.User{ID: uuid.New(), Role: domain.RoleUser, IsVerified: false}
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(RequireVerifiedFromSettings(true, &staticSettingsGetter{settings: &domain.Settings{VerifyEmails: true}}, logkit.Noop()))
	r.Get("/", okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireVerifiedFromSettings_FallbackDisabledBypassesLiveEnabledSetting(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := &domain.User{ID: uuid.New(), Role: domain.RoleUser, IsVerified: false}
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(RequireVerifiedFromSettings(false, &staticSettingsGetter{settings: &domain.Settings{VerifyEmails: true}}, logkit.Noop()))
	r.Get("/", okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func withUser(ctx context.Context, u *domain.User) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}
