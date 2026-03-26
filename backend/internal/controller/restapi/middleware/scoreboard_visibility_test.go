package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	compMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mock"
)

func TestScoreboardVisibility_Public(t *testing.T) {
	t.Parallel()
	repo := compMock.NewMockSettingsRepository(t)
	repo.On("Get", mock.Anything).Return(&domain.Settings{ScoreboardVisible: domain.ScoreboardVisiblePublic}, nil)

	handler := ScoreboardVisibility(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/scoreboard", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestScoreboardVisibility_Hidden(t *testing.T) {
	t.Parallel()
	repo := compMock.NewMockSettingsRepository(t)
	repo.On("Get", mock.Anything).Return(&domain.Settings{ScoreboardVisible: domain.ScoreboardVisibleHidden}, nil)

	handler := ScoreboardVisibility(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/scoreboard", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestScoreboardVisibility_AdminsOnly_Forbidden(t *testing.T) {
	t.Parallel()
	repo := compMock.NewMockSettingsRepository(t)
	repo.On("Get", mock.Anything).Return(&domain.Settings{ScoreboardVisible: domain.ScoreboardVisibleAdminsOnly}, nil)

	handler := ScoreboardVisibility(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/scoreboard", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestScoreboardVisibility_AdminsOnly_Allowed(t *testing.T) {
	t.Parallel()
	repo := compMock.NewMockSettingsRepository(t)
	handler := ScoreboardVisibility(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/scoreboard", http.NoBody)
	ctx := context.WithValue(req.Context(), userContextKey, &domain.User{Role: domain.RoleAdmin})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
