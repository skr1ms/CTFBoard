package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestRequireAuth_NoUser_Returns401(t *testing.T) {
	t.Parallel()

	r := buildRouter(RequireAuth(func(*domain.User) error { return nil }))
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireAuth_Admin_SkipsCheck(t *testing.T) {
	t.Parallel()

	// check always returns an error, but admin bypasses it
	checkCalled := false
	mw := RequireAuth(func(*domain.User) error {
		checkCalled = true

		return apperr.ErrUserBanned
	})

	admin := &domain.User{Role: domain.RoleAdmin}
	r := buildRouter(injectUser(admin), mw)
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.False(t, checkCalled, "check must not be called for admin")
}

func TestRequireAuth_CheckPasses_AllowsThrough(t *testing.T) {
	t.Parallel()

	mw := RequireAuth(func(*domain.User) error { return nil })
	user := &domain.User{Role: domain.RoleUser}
	r := buildRouter(injectUser(user), mw)
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireAuth_CheckFails_ReturnsError(t *testing.T) {
	t.Parallel()

	mw := RequireAuth(func(*domain.User) error { return apperr.ErrUserBanned })
	user := &domain.User{Role: domain.RoleUser}
	r := buildRouter(injectUser(user), mw)
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
