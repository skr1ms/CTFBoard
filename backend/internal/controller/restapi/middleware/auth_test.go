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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
)

func newTestJWTService(t *testing.T) *jwt.JWTService {
	t.Helper()
	svc, err := jwt.NewJWTService(
		[]jwt.KeyEntry{{Kid: "0", Secret: "access-secret-min-32-chars-long!"}},
		[]jwt.KeyEntry{{Kid: "0", Secret: "refresh-secret-min-32-chars-long"}},
		time.Hour, time.Hour, nil, nil)
	require.NoError(t, err)
	return svc
}

func TestAuth_NoHeader_Error(t *testing.T) {
	t.Parallel()
	svc := newTestJWTService(t)
	r := chi.NewRouter()
	r.Use(Auth(svc, nil, nil, logger.Noop()))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_BearerSuccess(t *testing.T) {
	t.Parallel()
	svc := newTestJWTService(t)
	userID := uuid.New()
	token, err := svc.GenerateTokenPair(userID, "a@b.c", "Name", string(entity.RoleAdmin))
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Use(Auth(svc, nil, nil, logger.Noop()))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, userID.String(), GetUserID(r.Context()))
		assert.Equal(t, string(entity.RoleAdmin), GetUserRole(r.Context()))
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuth_BearerInvalid_Error(t *testing.T) {
	t.Parallel()
	svc := newTestJWTService(t)
	r := chi.NewRouter()
	r.Use(Auth(svc, nil, nil, logger.Noop()))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_InvalidFormat_Error(t *testing.T) {
	t.Parallel()
	svc := newTestJWTService(t)
	r := chi.NewRouter()
	r.Use(Auth(svc, nil, nil, logger.Noop()))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "InvalidScheme token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAdmin_Success(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			adminUser := &entity.User{ID: uuid.New(), Role: entity.RoleAdmin}
			ctx := context.WithValue(r.Context(), userContextKey, adminUser)
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
	t.Parallel()
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), UserRoleKey, string(entity.RoleUser))
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

func TestAuth_TokenSuccess(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	tokenID := uuid.New()
	apiToken := &entity.APIToken{ID: tokenID, UserID: userID}
	user := &entity.User{ID: userID, Role: entity.RoleUser}

	apiAuth := mocks.NewMockAPITokenAuther(t)
	apiAuth.On("GetByTokenHash", mock.Anything, mock.AnythingOfType("string")).Return(apiToken, nil)
	apiAuth.On("ValidateToken", apiToken).Return(true)
	apiAuth.On("UpdateLastUsedAt", mock.Anything, tokenID).Return(nil)

	userGet := mocks.NewMockUserByIDGetter(t)
	userGet.On("GetByID", mock.Anything, userID).Return(user, nil)

	r := chi.NewRouter()
	r.Use(Auth(nil, apiAuth, userGet, logger.Noop()))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, userID.String(), GetUserID(r.Context()))
		assert.Equal(t, string(entity.RoleUser), GetUserRole(r.Context()))
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token my-api-token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuth_TokenError(t *testing.T) {
	t.Parallel()
	apiAuth := mocks.NewMockAPITokenAuther(t)
	apiAuth.On("GetByTokenHash", mock.Anything, mock.AnythingOfType("string")).Return((*entity.APIToken)(nil), errors.New("token not found"))

	userGet := mocks.NewMockUserByIDGetter(t)

	r := chi.NewRouter()
	r.Use(Auth(nil, apiAuth, userGet, logger.Noop()))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token bad-token")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}
