package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
	usermocks "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInjectUser_ValidUserID_InjectsUser(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	expectedUser := &entity.User{ID: userID, Role: entity.RoleUser}

	userRepo := usermocks.NewMockUserRepository(t)
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(expectedUser, nil)

	userUC := user.NewUserUseCase(user.UserDeps{UserRepo: userRepo})

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), httputil.UserIDKey, userID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(InjectUser(userUC, nil))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		u, ok := GetUser(r.Context())
		assert.True(t, ok)
		assert.Equal(t, userID, u.ID)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestInjectUser_UserNotFound_Returns404(t *testing.T) {
	t.Parallel()
	userID := uuid.New()

	userRepo := usermocks.NewMockUserRepository(t)
	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, httperr.ErrUserNotFound)

	userUC := user.NewUserUseCase(user.UserDeps{UserRepo: userRepo})

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), httputil.UserIDKey, userID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(InjectUser(userUC, nil))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}
