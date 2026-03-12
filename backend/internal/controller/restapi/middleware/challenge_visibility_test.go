package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mocks"
)

func TestChallengeVisibility_Success(t *testing.T) {
	t.Parallel()
	compRepo := mocks.NewMockCompetitionRepository(t)
	uc := competition.NewCompetitionUseCase(competition.CompetitionDeps{CompetitionRepo: compRepo})

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	comp := &entity.Competition{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	compRepo.On("Get", mock.Anything).Return(comp, nil)

	handler := ChallengeVisibility(uc)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/challenges", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestChallengeVisibility_Forbidden(t *testing.T) {
	t.Parallel()
	compRepo := mocks.NewMockCompetitionRepository(t)
	uc := competition.NewCompetitionUseCase(competition.CompetitionDeps{CompetitionRepo: compRepo})

	startTime := time.Now().Add(1 * time.Hour)
	endTime := time.Now().Add(2 * time.Hour)
	comp := &entity.Competition{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	compRepo.On("Get", mock.Anything).Return(comp, nil)

	handler := ChallengeVisibility(uc)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/challenges", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestChallengeVisibility_AdminBypass(t *testing.T) {
	t.Parallel()
	compRepo := mocks.NewMockCompetitionRepository(t)
	uc := competition.NewCompetitionUseCase(competition.CompetitionDeps{CompetitionRepo: compRepo})

	handler := ChallengeVisibility(uc)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/challenges", nil)
	ctx := context.WithValue(req.Context(), userContextKey, &entity.User{Role: entity.RoleAdmin})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
