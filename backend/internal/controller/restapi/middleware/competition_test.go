package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	compmocks "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mocks"
)

func newActiveCompetition() *entity.Competition {
	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(24 * time.Hour)
	return &entity.Competition{
		ID:        1,
		Name:      "test",
		StartTime: &past,
		EndTime:   &future,
		IsPaused:  false,
	}
}

func newEndedCompetition() *entity.Competition {
	past := time.Now().Add(-48 * time.Hour)
	pastEnd := time.Now().Add(-1 * time.Hour)
	return &entity.Competition{
		ID:        1,
		Name:      "test",
		StartTime: &past,
		EndTime:   &pastEnd,
		IsPaused:  false,
	}
}

func newNotStartedCompetition() *entity.Competition {
	future := time.Now().Add(1 * time.Hour)
	return &entity.Competition{
		ID:        1,
		Name:      "test",
		StartTime: &future,
	}
}

func newPausedButEndedCompetition() *entity.Competition {
	startPast := time.Now().Add(-48 * time.Hour)
	endPast := time.Now().Add(-1 * time.Hour)
	pausedAt := time.Now().Add(-30 * time.Minute)
	return &entity.Competition{
		ID:        1,
		Name:      "test",
		StartTime: &startPast,
		EndTime:   &endPast,
		IsPaused:  true,
		PausedAt:  &pausedAt,
	}
}

func TestCompetitionActive_ActiveCompetition_Passes(t *testing.T) {
	t.Parallel()
	compRepo := compmocks.NewMockCompetitionRepository(t)
	compRepo.EXPECT().Get(mock.Anything).Return(newActiveCompetition(), nil)

	compUC := competition.NewCompetitionUseCase(competition.CompetitionDeps{CompetitionRepo: compRepo})

	r := chi.NewRouter()
	r.Use(CompetitionActive(compUC))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCompetitionActive_NotActive_Returns403(t *testing.T) {
	t.Parallel()
	compRepo := compmocks.NewMockCompetitionRepository(t)
	compRepo.EXPECT().Get(mock.Anything).Return(newNotStartedCompetition(), nil)

	compUC := competition.NewCompetitionUseCase(competition.CompetitionDeps{CompetitionRepo: compRepo})

	r := chi.NewRouter()
	r.Use(CompetitionActive(compUC))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestCompetitionEnded_Ended_Passes(t *testing.T) {
	t.Parallel()
	compRepo := compmocks.NewMockCompetitionRepository(t)
	compRepo.EXPECT().Get(mock.Anything).Return(newEndedCompetition(), nil)

	compUC := competition.NewCompetitionUseCase(competition.CompetitionDeps{CompetitionRepo: compRepo})

	r := chi.NewRouter()
	r.Use(CompetitionEnded(compUC))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCompetitionEnded_NotEnded_Returns403(t *testing.T) {
	t.Parallel()
	compRepo := compmocks.NewMockCompetitionRepository(t)
	compRepo.EXPECT().Get(mock.Anything).Return(newActiveCompetition(), nil)

	compUC := competition.NewCompetitionUseCase(competition.CompetitionDeps{CompetitionRepo: compRepo})

	r := chi.NewRouter()
	r.Use(CompetitionEnded(compUC))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestCompetitionEnded_PausedButEndTimePassed_Passes(t *testing.T) {
	t.Parallel()
	compRepo := compmocks.NewMockCompetitionRepository(t)
	compRepo.EXPECT().Get(mock.Anything).Return(newPausedButEndedCompetition(), nil)

	compUC := competition.NewCompetitionUseCase(competition.CompetitionDeps{CompetitionRepo: compRepo})

	r := chi.NewRouter()
	r.Use(CompetitionEnded(compUC))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
