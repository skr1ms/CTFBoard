package helper

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

type SettingsGetter = middleware.SettingsGetter

type OnErrorFunc func(w http.ResponseWriter, r *http.Request, err error, op, step string) bool

// teamByIDGetter is the minimal interface needed for ban-check helpers.
type teamByIDGetter interface {
	GetByID(ctx context.Context, ID uuid.UUID) (*domain.Team, error)
}

func IsAdmin(user *domain.User) bool {
	return user.Role == domain.RoleAdmin
}

func ParseSearchQuery(w http.ResponseWriter, r *http.Request, q *string, maxLen int, onError OnErrorFunc, op, step string) (string, bool) {
	if q == nil || *q == "" {
		return "", true
	}

	if !httputil.ValidateSearchQ(*q) {
		onError(w, r, apperr.NewValidationErrorf("invalid search query"), op, step)

		return "", false
	}

	return httputil.SanitizeSearchQ(*q, maxLen), true
}

// ParseOptionalSearchQuery validates and sanitizes an optional search query parameter.
// Returns (nil, true) when q is nil or empty, (&sanitized, true) on success, (nil, false) on invalid input.
func ParseOptionalSearchQuery(w http.ResponseWriter, r *http.Request, q *string, maxLen int, onError OnErrorFunc, op, step string) (*string, bool) {
	if q == nil || *q == "" {
		return nil, true
	}

	s, ok := ParseSearchQuery(w, r, q, maxLen, onError, op, step)
	if !ok {
		return nil, false
	}

	return &s, true
}

func RequireUser(w http.ResponseWriter, r *http.Request) (*domain.User, bool) {
	user, ok := middleware.GetUser(r.Context())
	if !ok || user == nil {
		httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrNotAuthenticated))

		return nil, false
	}

	return user, true
}

// RequireTeamID checks that the user belongs to a team and returns the team UUID.
// Calls onError and returns (uuid.Nil, false) when TeamID is nil.
func RequireTeamID(w http.ResponseWriter, r *http.Request, user *domain.User, onError OnErrorFunc, op string) (uuid.UUID, bool) {
	if user.TeamID == nil {
		onError(w, r, apperr.ErrUserNotInTeam, op, "RequireTeam")

		return uuid.Nil, false
	}

	return *user.TeamID, true
}

// RequireUnbannedTeam loads the team by teamID and verifies it is not banned.
// Calls onError and returns (nil, false) on fetch error or ban.
func RequireUnbannedTeam(w http.ResponseWriter, r *http.Request, teamUC teamByIDGetter, teamID uuid.UUID, onError OnErrorFunc, op string) (*domain.Team, bool) {
	team, err := teamUC.GetByID(r.Context(), teamID)
	if onError(w, r, err, op, "TeamCheck") {
		return nil, false
	}

	if team.IsBanned {
		onError(w, r, apperr.ErrTeamBanned, op, "BanCheck")

		return nil, false
	}

	return team, true
}

// CheckOptionalTeamBan checks ban status when the user may or may not be in a team.
// Returns false (and calls onError) only if the team exists and is banned.
// Returns true when teamID is nil (no team - not banned).
func CheckOptionalTeamBan(w http.ResponseWriter, r *http.Request, teamUC teamByIDGetter, teamID *uuid.UUID, onError OnErrorFunc, op string) bool {
	if teamID == nil {
		return true
	}

	team, err := teamUC.GetByID(r.Context(), *teamID)
	if onError(w, r, err, op, "TeamCheck") {
		return false
	}

	if team.IsBanned {
		onError(w, r, apperr.ErrTeamBanned, op, "BanCheck")

		return false
	}

	return true
}
