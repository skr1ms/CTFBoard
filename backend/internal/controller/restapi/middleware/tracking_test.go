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
	"github.com/wahrwelt-kit/go-logkit"

	midMock "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware/mock"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
)

func TestIPTracking_ValidUser_TracksIP(t *testing.T) {
	t.Parallel()

	trackingRepo := midMock.NewMockTrackingRepository(t)
	done := make(chan struct{})

	trackingRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ *domain.TrackingEntry) error {
			close(done)

			return nil
		}).
		Once()

	trackingUC := user.NewTrackingUseCase(user.TrackingDeps{TrackingRepo: trackingRepo})
	userID := uuid.New()
	u := &domain.User{ID: userID, Role: domain.RoleUser}

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), userContextKey, u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Use(IPTracking(context.Background(), trackingUC, logkit.Noop()))
	r.Get("/", okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
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

	trackingRepo := midMock.NewMockTrackingRepository(t)

	trackingUC := user.NewTrackingUseCase(user.TrackingDeps{TrackingRepo: trackingRepo})

	r := chi.NewRouter()
	r.Use(IPTracking(context.Background(), trackingUC, logkit.Noop()))
	r.Get("/", okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
