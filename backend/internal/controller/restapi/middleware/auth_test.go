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
	"github.com/wahrwelt-kit/go-httpkit/httputil"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"

	midMock "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware/mock"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func newTestJWTService(t *testing.T) *jwtkit.JWTService {
	t.Helper()

	svc, err := jwtkit.NewJWTService(jwtkit.Config{
		AccessKeys:  []jwtkit.KeyEntry{{Kid: "0", Secret: []byte("access-secret-min-32-chars-long!")}},
		RefreshKeys: []jwtkit.KeyEntry{{Kid: "0", Secret: []byte("refresh-secret-min-32-chars-long")}},
		AccessTTL:   time.Hour,
		RefreshTTL:  time.Hour,
		Issuer:      "test-issuer",
	})
	require.NoError(t, err)

	return svc
}

func TestAuth_NoHeader_Error(t *testing.T) {
	t.Parallel()
	svc := newTestJWTService(t)
	r := chi.NewRouter()
	r.Use(Auth(svc, nil, nil, logkit.Noop()))
	r.Get("/", okHandler())
	ServeAndExpect(t, r, http.MethodGet, "/", nil, http.StatusUnauthorized)
}

func TestAuth_BearerSuccess(t *testing.T) {
	t.Parallel()
	svc := newTestJWTService(t)
	userID := uuid.New()
	token, err := svc.GenerateTokenPair(context.Background(), userID, string(domain.RoleAdmin))
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Use(Auth(svc, nil, nil, logkit.Noop()))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, userID.String(), GetUserID(r.Context()))
		assert.Equal(t, string(domain.RoleAdmin), GetUserRole(r.Context()))
		w.WriteHeader(http.StatusOK)
	})

	ServeAndExpect(t, r, http.MethodGet, "/", map[string]string{"Authorization": "Bearer " + token.AccessToken}, http.StatusOK)
}

func TestAuth_BearerInvalid_Error(t *testing.T) {
	t.Parallel()
	svc := newTestJWTService(t)
	r := chi.NewRouter()
	r.Use(Auth(svc, nil, nil, logkit.Noop()))
	r.Get("/", okHandler())

	ServeAndExpect(t, r, http.MethodGet, "/", map[string]string{"Authorization": "Bearer invalid-token"}, http.StatusUnauthorized)
}

func TestAuth_InvalidFormat_Error(t *testing.T) {
	t.Parallel()
	svc := newTestJWTService(t)
	r := chi.NewRouter()
	r.Use(Auth(svc, nil, nil, logkit.Noop()))
	r.Get("/", okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
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
			adminUser := &domain.User{ID: uuid.New(), Role: domain.RoleAdmin}
			ctx := context.WithValue(r.Context(), userContextKey, adminUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(Admin)
	r.Get("/", okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAdmin_Error(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), UserRoleKey, string(domain.RoleUser))
			ctx = context.WithValue(ctx, httputil.UserIDKey, uuid.New().String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(Admin)
	r.Get("/", okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAuth_TokenSuccess(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	tokenID := uuid.New()
	apiToken := &domain.APIToken{ID: tokenID, UserID: userID}
	user := &domain.User{ID: userID, Role: domain.RoleUser}

	apiAuth := midMock.NewMockAPITokenAuther(t)
	apiAuth.On("GetByTokenHash", mock.Anything, mock.AnythingOfType("string")).Return(apiToken, nil)
	apiAuth.On("ValidateToken", apiToken).Return(true)
	apiAuth.On("UpdateLastUsedAt", mock.Anything, tokenID).Return(nil)

	userGet := midMock.NewMockUserByIDGetter(t)
	userGet.On("GetByID", mock.Anything, userID).Return(user, nil)

	r := chi.NewRouter()
	r.Use(Auth(nil, apiAuth, userGet, logkit.Noop()))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, userID.String(), GetUserID(r.Context()))
		assert.Equal(t, string(domain.RoleUser), GetUserRole(r.Context()))
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Authorization", "Token my-api-token")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// UpdateLastUsedAt is called asynchronously; give the goroutine time to run.
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuth_TokenError(t *testing.T) {
	t.Parallel()
	apiAuth := midMock.NewMockAPITokenAuther(t)
	apiAuth.On("GetByTokenHash", mock.Anything, mock.AnythingOfType("string")).Return((*domain.APIToken)(nil), errors.New("token not found"))

	userGet := midMock.NewMockUserByIDGetter(t)

	r := chi.NewRouter()
	r.Use(Auth(nil, apiAuth, userGet, logkit.Noop()))
	r.Get("/", okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Authorization", "Token bad-token")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}
