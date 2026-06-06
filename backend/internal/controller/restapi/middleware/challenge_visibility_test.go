package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	compMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mock"
)

func TestChallengeVisibility_Success(t *testing.T) {
	t.Parallel()
	compRepo := compMock.NewMockCompetitionRepository(t)
	uc := competition.NewCompetitionUseCase(competition.CompetitionDeps{CompetitionRepo: compRepo})

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	comp := &domain.Competition{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	compRepo.On("Get", mock.Anything).Return(comp, nil)

	handler := ChallengeVisibility(uc)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newRequest(http.MethodGet, "/challenges", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestChallengeVisibility_Forbidden(t *testing.T) {
	t.Parallel()
	compRepo := compMock.NewMockCompetitionRepository(t)
	uc := competition.NewCompetitionUseCase(competition.CompetitionDeps{CompetitionRepo: compRepo})

	startTime := time.Now().Add(1 * time.Hour)
	endTime := time.Now().Add(2 * time.Hour)
	comp := &domain.Competition{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	compRepo.On("Get", mock.Anything).Return(comp, nil)

	handler := ChallengeVisibility(uc)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newRequest(http.MethodGet, "/challenges", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestChallengeVisibility_AdminBypass(t *testing.T) {
	t.Parallel()
	compRepo := compMock.NewMockCompetitionRepository(t)
	uc := competition.NewCompetitionUseCase(competition.CompetitionDeps{CompetitionRepo: compRepo})

	handler := ChallengeVisibility(uc)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newRequest(http.MethodGet, "/challenges", http.NoBody)
	ctx := context.WithValue(req.Context(), userContextKey, &domain.User{Role: domain.RoleAdmin})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
