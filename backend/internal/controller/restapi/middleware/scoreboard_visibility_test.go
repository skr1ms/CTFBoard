package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mocks"
)

func TestScoreboardVisibility_Public(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSettingsRepository(t)
	repo.On("Get", mock.Anything).Return(&entity.Settings{ScoreboardVisible: entity.ScoreboardVisiblePublic}, nil)

	handler := ScoreboardVisibility(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/scoreboard", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestScoreboardVisibility_Hidden(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSettingsRepository(t)
	repo.On("Get", mock.Anything).Return(&entity.Settings{ScoreboardVisible: entity.ScoreboardVisibleHidden}, nil)

	handler := ScoreboardVisibility(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/scoreboard", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestScoreboardVisibility_AdminsOnly_Forbidden(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSettingsRepository(t)
	repo.On("Get", mock.Anything).Return(&entity.Settings{ScoreboardVisible: entity.ScoreboardVisibleAdminsOnly}, nil)

	handler := ScoreboardVisibility(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/scoreboard", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestScoreboardVisibility_AdminsOnly_Allowed(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSettingsRepository(t)
	handler := ScoreboardVisibility(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/scoreboard", nil)
	ctx := context.WithValue(req.Context(), userContextKey, &entity.User{Role: entity.RoleAdmin})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
