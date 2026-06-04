package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

type staticVisibilityGetter struct {
	value string
}

func (g staticVisibilityGetter) GetString(context.Context, string, string) string {
	return g.value
}

func visibilityGuardStatus(value string, user *domain.User) int {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if user != nil {
		req = req.WithContext(context.WithValue(req.Context(), userContextKey, user))
	}

	rec := httptest.NewRecorder()
	VisibilityGuard(staticVisibilityGetter{value: value}, "score_visibility")(next).ServeHTTP(rec, req)

	return rec.Code
}

func TestVisibilityGuard_PublicAllowsGuest(t *testing.T) {
	t.Parallel()

	assert.Equal(t, http.StatusNoContent, visibilityGuardStatus("public", nil))
}

func TestVisibilityGuard_PrivateRequiresUser(t *testing.T) {
	t.Parallel()

	assert.Equal(t, http.StatusUnauthorized, visibilityGuardStatus("private", nil))
	assert.Equal(t, http.StatusNoContent, visibilityGuardStatus("private", &domain.User{Role: domain.RoleUser}))
}

func TestVisibilityGuard_HiddenBlocksNonAdmin(t *testing.T) {
	t.Parallel()

	assert.Equal(t, http.StatusNotFound, visibilityGuardStatus("hidden", &domain.User{Role: domain.RoleUser}))
	assert.Equal(t, http.StatusNoContent, visibilityGuardStatus("hidden", &domain.User{Role: domain.RoleAdmin}))
}

func TestVisibilityGuard_UnknownValueFailsClosed(t *testing.T) {
	t.Parallel()

	assert.Equal(t, http.StatusNotFound, visibilityGuardStatus("garbage", nil))
	assert.Equal(t, http.StatusNoContent, visibilityGuardStatus("garbage", &domain.User{Role: domain.RoleAdmin}))
}
