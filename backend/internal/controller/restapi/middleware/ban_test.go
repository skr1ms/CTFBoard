package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

type mockTeamGetter struct {
	team *domain.Team
	err  error
}

func (m *mockTeamGetter) GetByID(_ context.Context, _ uuid.UUID) (*domain.Team, error) {
	return m.team, m.err
}

func injectUser(u *domain.User) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := withUser(r.Context(), u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func buildRouter(middlewares ...func(http.Handler) http.Handler) *chi.Mux {
	r := chi.NewRouter()
	for _, mw := range middlewares {
		r.Use(mw)
	}
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return r
}

func TestRequireUserNotBanned_NoUser_PassesThrough(t *testing.T) {
	t.Parallel()
	r := buildRouter(RequireUserNotBanned())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireUserNotBanned_Admin_PassesThrough(t *testing.T) {
	t.Parallel()
	admin := &domain.User{ID: uuid.New(), Role: domain.RoleAdmin, IsBanned: true}
	r := buildRouter(injectUser(admin), RequireUserNotBanned())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireUserNotBanned_NotBanned_PassesThrough(t *testing.T) {
	t.Parallel()
	user := &domain.User{ID: uuid.New(), Role: domain.RoleUser, IsBanned: false}
	r := buildRouter(injectUser(user), RequireUserNotBanned())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireUserNotBanned_Banned_Returns403(t *testing.T) {
	t.Parallel()
	teamID := uuid.New()
	user := &domain.User{ID: uuid.New(), Role: domain.RoleUser, IsBanned: true, TeamID: &teamID}
	r := buildRouter(injectUser(user), RequireUserNotBanned())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireTeamNotBanned_NoUser_PassesThrough(t *testing.T) {
	t.Parallel()
	getter := &mockTeamGetter{}
	r := buildRouter(RequireTeamNotBanned(getter, nil))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireTeamNotBanned_Admin_PassesThrough(t *testing.T) {
	t.Parallel()
	teamID := uuid.New()
	admin := &domain.User{ID: uuid.New(), Role: domain.RoleAdmin, TeamID: &teamID}
	getter := &mockTeamGetter{team: &domain.Team{IsBanned: true}}
	r := buildRouter(injectUser(admin), RequireTeamNotBanned(getter, nil))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireTeamNotBanned_NoTeam_PassesThrough(t *testing.T) {
	t.Parallel()
	user := &domain.User{ID: uuid.New(), Role: domain.RoleUser, TeamID: nil}
	getter := &mockTeamGetter{}
	r := buildRouter(injectUser(user), RequireTeamNotBanned(getter, nil))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireTeamNotBanned_TeamNotBanned_PassesThrough(t *testing.T) {
	t.Parallel()
	teamID := uuid.New()
	user := &domain.User{ID: uuid.New(), Role: domain.RoleUser, TeamID: &teamID}
	getter := &mockTeamGetter{team: &domain.Team{IsBanned: false}}
	r := buildRouter(injectUser(user), RequireTeamNotBanned(getter, nil))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireTeamNotBanned_TeamBanned_Returns403(t *testing.T) {
	t.Parallel()
	teamID := uuid.New()
	user := &domain.User{ID: uuid.New(), Role: domain.RoleUser, TeamID: &teamID}
	getter := &mockTeamGetter{team: &domain.Team{IsBanned: true}}
	r := buildRouter(injectUser(user), RequireTeamNotBanned(getter, nil))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireTeamNotBanned_GetterError_Returns500(t *testing.T) {
	t.Parallel()
	teamID := uuid.New()
	user := &domain.User{ID: uuid.New(), Role: domain.RoleUser, TeamID: &teamID}
	getter := &mockTeamGetter{err: errors.New("db error")}
	r := buildRouter(injectUser(user), RequireTeamNotBanned(getter, nil))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}
