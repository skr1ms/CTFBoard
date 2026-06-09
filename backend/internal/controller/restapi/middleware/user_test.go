package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
	userMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mock"
)

func TestInjectUser_ValidUserID_InjectsUser(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	expectedUser := &domain.User{ID: userID, Role: domain.RoleUser}

	userRepo := userMock.NewMockUserRepository(t)
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(expectedUser, nil)

	userUC := user.NewUserUseCase(user.UserDeps{UserRepo: userRepo})

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), httputil.UserIDKey, userID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(InjectUser(userUC, nil, nil))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		u, ok := GetUser(r.Context())
		assert.True(t, ok)
		assert.Equal(t, userID, u.ID)
		w.WriteHeader(http.StatusOK)
	})

	req := newRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestInjectUser_DoesNotReuseCachedUserSnapshot(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	teamID := uuid.New()

	userRepo := userMock.NewMockUserRepository(t)
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&domain.User{ID: userID, Role: domain.RoleUser}, nil).Once()
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&domain.User{ID: userID, Role: domain.RoleUser, TeamID: &teamID}, nil).Once()

	userUC := user.NewUserUseCase(user.UserDeps{UserRepo: userRepo})
	redisClient, _ := redismock.NewClientMock()
	userCache := cachekit.New(redisClient)
	seenTeamIDs := make([]*uuid.UUID, 0, 2)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), httputil.UserIDKey, userID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(InjectUser(userUC, userCache, nil))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		u, ok := GetUser(r.Context())
		require.True(t, ok)

		seenTeamIDs = append(seenTeamIDs, u.TeamID)

		w.WriteHeader(http.StatusOK)
	})

	for range 2 {
		req := newRequest(http.MethodGet, "/", http.NoBody)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}

	require.Len(t, seenTeamIDs, 2)
	assert.Nil(t, seenTeamIDs[0])
	require.NotNil(t, seenTeamIDs[1])
	assert.Equal(t, teamID, *seenTeamIDs[1])
}

func TestInjectUser_UserNotFound_Returns404(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	userRepo := userMock.NewMockUserRepository(t)
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, apperr.ErrUserNotFound)

	userUC := user.NewUserUseCase(user.UserDeps{UserRepo: userRepo})

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), httputil.UserIDKey, userID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(InjectUser(userUC, nil, nil))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := newRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}
