package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_NoHeader_Error(t *testing.T) {
	svc := jwt.NewJWTService("access-secret-min-32-chars-long", "refresh-secret-min-32-chars-long", time.Hour, time.Hour)
	r := chi.NewRouter()
	r.Use(Auth(svc, nil, nil))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_BearerSuccess(t *testing.T) {
	svc := jwt.NewJWTService("access-secret-min-32-chars-long", "refresh-secret-min-32-chars-long", time.Hour, time.Hour)
	userID := uuid.New()
	token, err := svc.GenerateTokenPair(userID, "a@b.c", "Name", entity.RoleAdmin)
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Use(Auth(svc, nil, nil))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, userID.String(), GetUserID(r.Context()))
		assert.Equal(t, entity.RoleAdmin, GetUserRole(r.Context()))
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuth_BearerInvalid_Error(t *testing.T) {
	svc := jwt.NewJWTService("access-secret-min-32-chars-long", "refresh-secret-min-32-chars-long", time.Hour, time.Hour)
	r := chi.NewRouter()
	r.Use(Auth(svc, nil, nil))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_InvalidFormat_Error(t *testing.T) {
	svc := jwt.NewJWTService("access-secret-min-32-chars-long", "refresh-secret-min-32-chars-long", time.Hour, time.Hour)
	r := chi.NewRouter()
	r.Use(Auth(svc, nil, nil))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "InvalidScheme token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAdmin_Success(t *testing.T) {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), UserRoleKey, entity.RoleAdmin)
			ctx = context.WithValue(ctx, httputil.UserIDKey, uuid.New().String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(Admin)
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAdmin_Error(t *testing.T) {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), UserRoleKey, entity.RoleUser)
			ctx = context.WithValue(ctx, httputil.UserIDKey, uuid.New().String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(Admin)
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
}

type mockAPITokenAuther struct {
	token *entity.APIToken
	err   error
	valid bool
}

func (m *mockAPITokenAuther) GetByTokenHash(_ context.Context, _ string) (*entity.APIToken, error) {
	return m.token, m.err
}

func (m *mockAPITokenAuther) UpdateLastUsedAt(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockAPITokenAuther) ValidateToken(t *entity.APIToken) bool {
	if t == nil {
		return false
	}
	return m.valid
}

type mockUserByIDGetter struct {
	user *entity.User
	err  error
}

func (m *mockUserByIDGetter) GetByID(_ context.Context, _ uuid.UUID) (*entity.User, error) {
	return m.user, m.err
}

func TestAuth_TokenSuccess(t *testing.T) {
	userID := uuid.New()
	tokenID := uuid.New()
	apiToken := &entity.APIToken{ID: tokenID, UserID: userID}
	user := &entity.User{ID: userID, Role: entity.RoleUser}

	apiAuth := &mockAPITokenAuther{token: apiToken, err: nil, valid: true}
	userGet := &mockUserByIDGetter{user: user, err: nil}

	r := chi.NewRouter()
	r.Use(Auth(nil, apiAuth, userGet))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, userID.String(), GetUserID(r.Context()))
		assert.Equal(t, entity.RoleUser, GetUserRole(r.Context()))
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token my-api-token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuth_TokenError(t *testing.T) {
	apiAuth := &mockAPITokenAuther{token: nil, err: errors.New("token not found"), valid: false}
	userGet := &mockUserByIDGetter{user: nil, err: nil}

	r := chi.NewRouter()
	r.Use(Auth(nil, apiAuth, userGet))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token bad-token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}
