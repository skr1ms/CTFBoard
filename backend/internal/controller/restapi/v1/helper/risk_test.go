package helper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

type helperTeamGetter struct {
	team *domain.Team
	err  error
}

func (g helperTeamGetter) GetByID(context.Context, uuid.UUID) (*domain.Team, error) {
	return g.team, g.err
}

func TestTeamStatsVisibleToViewer(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	otherTeamID := uuid.New()

	tests := []struct {
		name   string
		team   *domain.Team
		viewer *domain.User
		want   bool
	}{
		{name: "nil team", team: nil, viewer: nil, want: false},
		{name: "public team visible anonymously", team: &domain.Team{ID: teamID}, viewer: nil, want: true},
		{name: "hidden team hidden anonymously", team: &domain.Team{ID: teamID, IsHidden: true}, viewer: nil, want: false},
		{
			name:   "hidden team visible to admin",
			team:   &domain.Team{ID: teamID, IsHidden: true},
			viewer: &domain.User{ID: uuid.New(), Role: domain.RoleAdmin},
			want:   true,
		},
		{
			name:   "hidden team visible to own member",
			team:   &domain.Team{ID: teamID, IsHidden: true},
			viewer: &domain.User{ID: uuid.New(), Role: domain.RoleUser, TeamID: &teamID},
			want:   true,
		},
		{
			name:   "hidden team hidden from other member",
			team:   &domain.Team{ID: teamID, IsHidden: true},
			viewer: &domain.User{ID: uuid.New(), Role: domain.RoleUser, TeamID: &otherTeamID},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, TeamStatsVisibleToViewer(tt.team, tt.viewer))
		})
	}
}

func TestParseSearchQuerySanitizesEscapableCharacters(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := newRequest(http.MethodGet, "/", http.NoBody)
	q := "team_%"

	got, ok := ParseSearchQuery(w, r, &q, 20, nilOnError, "op", "step")

	require.True(t, ok)
	assert.Equal(t, `team\_\%`, got)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestParseSearchQueryInvalidCallsOnError(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := newRequest(http.MethodGet, "/", http.NoBody)
	q := "bad\nquery"

	called := false
	got, ok := ParseSearchQuery(w, r, &q, 20, func(_ http.ResponseWriter, _ *http.Request, err error, op, step string) bool {
		called = true

		assert.Equal(t, "op", op)
		assert.Equal(t, "step", step)

		var validationErr *apperr.ValidationError
		assert.ErrorAs(t, err, &validationErr)

		return true
	}, "op", "step")

	assert.False(t, ok)
	assert.Empty(t, got)
	assert.True(t, called)
}

func TestParseOptionalSearchQueryEmptyAndValid(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := newRequest(http.MethodGet, "/", http.NoBody)

	got, ok := ParseOptionalSearchQuery(w, r, nil, 20, nilOnError, "op", "step")
	require.True(t, ok)
	assert.Nil(t, got)

	q := "abc_def"
	got, ok = ParseOptionalSearchQuery(w, r, &q, 20, nilOnError, "op", "step")
	require.True(t, ok)
	require.NotNil(t, got)
	assert.Equal(t, `abc\_def`, *got)
}

func TestRequireTeamID(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := newRequest(http.MethodGet, "/", http.NoBody)
	teamID := uuid.New()

	got, ok := RequireTeamID(w, r, &domain.User{TeamID: &teamID}, failOnError, "op")
	require.True(t, ok)
	assert.Equal(t, teamID, got)

	called := false
	got, ok = RequireTeamID(w, r, &domain.User{}, func(_ http.ResponseWriter, _ *http.Request, err error, op, step string) bool {
		called = true

		assert.ErrorIs(t, err, apperr.ErrUserNotInTeam)
		assert.Equal(t, "op", op)
		assert.Equal(t, "RequireTeam", step)

		return true
	}, "op")
	assert.False(t, ok)
	assert.Equal(t, uuid.Nil, got)
	assert.True(t, called)
}

func TestRequireUnbannedTeam(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := newRequest(http.MethodGet, "/", http.NoBody)
	teamID := uuid.New()

	team, ok := RequireUnbannedTeam(w, r, helperTeamGetter{team: &domain.Team{ID: teamID}}, teamID, failOnError, "op")
	require.True(t, ok)
	require.NotNil(t, team)
	assert.Equal(t, teamID, team.ID)

	called := false
	team, ok = RequireUnbannedTeam(w, r, helperTeamGetter{team: &domain.Team{ID: teamID, IsBanned: true}}, teamID, func(_ http.ResponseWriter, _ *http.Request, err error, op, step string) bool {
		if err == nil {
			return false
		}

		called = true

		assert.ErrorIs(t, err, apperr.ErrTeamBanned)
		assert.Equal(t, "op", op)
		assert.Equal(t, "BanCheck", step)

		return true
	}, "op")
	assert.False(t, ok)
	assert.Nil(t, team)
	assert.True(t, called)
}

func TestCheckOptionalTeamBan(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := newRequest(http.MethodGet, "/", http.NoBody)
	teamID := uuid.New()

	assert.True(t, CheckOptionalTeamBan(w, r, helperTeamGetter{}, nil, failOnError, "op"))
	assert.True(t, CheckOptionalTeamBan(w, r, helperTeamGetter{team: &domain.Team{ID: teamID}}, &teamID, failOnError, "op"))

	called := false
	ok := CheckOptionalTeamBan(w, r, helperTeamGetter{team: &domain.Team{ID: teamID, IsBanned: true}}, &teamID, func(_ http.ResponseWriter, _ *http.Request, err error, op, step string) bool {
		if err == nil {
			return false
		}

		called = true

		assert.ErrorIs(t, err, apperr.ErrTeamBanned)
		assert.Equal(t, "op", op)
		assert.Equal(t, "BanCheck", step)

		return true
	}, "op")
	assert.False(t, ok)
	assert.True(t, called)
}

func TestCookieHelpersAndFrontendCallbackURL(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()

	SetHTTPOnlyCookie(w, CookieOptions{
		Name:     "refresh",
		Value:    "token",
		Path:     "/api",
		MaxAge:   60,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	ClearHTTPOnlyCookie(w, "refresh", "/api", true, http.SameSiteStrictMode)

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 2)
	assert.True(t, cookies[0].HttpOnly)
	assert.True(t, cookies[0].Secure)
	assert.Equal(t, http.SameSiteStrictMode, cookies[0].SameSite)
	assert.Equal(t, -1, cookies[1].MaxAge)

	values := url.Values{}
	values.Set("state", "a b")
	assert.Equal(t, "https://ctf.example/auth/callback?state=a+b", FrontendCallbackURL("https://ctf.example/", values))
}

func nilOnError(http.ResponseWriter, *http.Request, error, string, string) bool {
	return false
}

func failOnError(_ http.ResponseWriter, _ *http.Request, err error, _, _ string) bool {
	if err != nil {
		panic("unexpected error")
	}

	return false
}
