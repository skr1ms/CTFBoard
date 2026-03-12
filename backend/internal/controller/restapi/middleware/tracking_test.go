package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
)

func TestIPTracking_ValidUser_TracksIP(t *testing.T) {
	t.Parallel()

	trackingRepo := mocks.NewMockTrackingRepository(t)
	done := make(chan struct{})
	trackingRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ *entity.TrackingEntry) error {
			close(done)
			return nil
		}).
		Once()

	trackingUC := user.NewTrackingUseCase(user.TrackingDeps{TrackingRepo: trackingRepo})
	userID := uuid.New()
	u := &entity.User{ID: userID, Role: entity.RoleUser}

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), userContextKey, u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(IPTracking(context.Background(), trackingUC, nil, logger.Noop()))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "test-agent")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("tracking Create was not called within timeout")
	}
}

func TestIPTracking_NoUser_Skips(t *testing.T) {
	t.Parallel()

	trackingRepo := mocks.NewMockTrackingRepository(t)

	trackingUC := user.NewTrackingUseCase(user.TrackingDeps{TrackingRepo: trackingRepo})

	r := chi.NewRouter()
	r.Use(IPTracking(context.Background(), trackingUC, nil, logger.Noop()))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
