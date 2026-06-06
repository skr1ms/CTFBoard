package helper

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-httpkit/httputil"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

type SettingsGetter = middleware.SettingsGetter

type OnErrorFunc func(w http.ResponseWriter, r *http.Request, err error, op, step string) bool

const challengeOpenTrackTimeout = 5 * time.Second

var (
	ErrAccessDenied          = apperr.ErrAccessDenied
	ErrCompetitionNotStarted = apperr.ErrCompetitionNotStarted
	ErrDebugNotEnabled       = apperr.ErrDebugNotEnabled
	ErrNotAuthenticated      = apperr.ErrNotAuthenticated
	ErrSetupAlreadyComplete  = apperr.ErrSetupAlreadyComplete
	ErrTeamNotFound          = apperr.ErrTeamNotFound
	ErrTokenRequired         = apperr.ErrTokenRequired
	ErrTooManyRequests       = apperr.ErrTooManyRequests
	ErrWriteupsDisabled      = apperr.ErrWriteupsDisabled
)

func ValidationErrorf(format string, args ...any) error {
	return apperr.NewValidationErrorf(format, args...)
}

// teamByIDGetter is the minimal interface needed for ban-check helpers.
type teamByIDGetter interface {
	GetByID(ctx context.Context, ID uuid.UUID) (*domain.Team, error)
}

type challengeOpenTracker interface {
	TrackChallengeOpen(ctx context.Context, userID, challengeID uuid.UUID, ip string) error
}

func ClientIP(r *http.Request) string {
	return kitMiddleware.GetClientIPFromContext(r.Context())
}

func CurrentUser(r *http.Request) (*domain.User, bool) {
	user, ok := middleware.GetUser(r.Context())
	if !ok || user == nil {
		return nil, false
	}

	return user, true
}

// TrackChallengeOpenAsync records the side-effect outside the handler path while
// keeping the goroutine detached from the response cancellation and bounded by
// its own timeout.
func TrackChallengeOpenAsync(reqCtx context.Context, logger logkit.Logger, tracker challengeOpenTracker, userID, challengeID uuid.UUID, ip string) {
	if tracker == nil {
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("restapi - v1 - TrackChallengeOpenAsync - panic", logkit.Fields{"recover": fmt.Sprint(r)})
			}
		}()

		ctx, cancel := context.WithTimeout(context.WithoutCancel(reqCtx), challengeOpenTrackTimeout)
		defer cancel()

		_ = tracker.TrackChallengeOpen(ctx, userID, challengeID, ip)
	}()
}

func IsAdmin(user *domain.User) bool {
	return user.Role == domain.RoleAdmin
}

func IsCompetitionNotStarted(comp *domain.Competition) bool {
	return comp != nil && comp.GetStatus() == domain.CompetitionStatusNotStarted
}

func IsCompetitionStatusNotStarted(status domain.CompetitionStatus) bool {
	return status == domain.CompetitionStatusNotStarted
}

func UserMatchesOrAdmin(user *domain.User, targetID uuid.UUID) bool {
	return user != nil && (user.ID == targetID || IsAdmin(user))
}

func TeamStatsVisibleToViewer(team *domain.Team, viewer *domain.User) bool {
	if team == nil {
		return false
	}

	if !team.IsHidden {
		return true
	}

	if viewer == nil {
		return false
	}

	if IsAdmin(viewer) {
		return true
	}

	return viewer.TeamID != nil && *viewer.TeamID == team.ID
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

func HandleAppError(w http.ResponseWriter, r *http.Request, handler *httputil.ErrorHandler, err error, op, step string) bool {
	if handler == nil {
		handler = &httputil.ErrorHandler{}
	}

	return handler.Handle(w, r, errmap.MapAppError(err), "restapi - v1 - "+op+" - "+step)
}

func RequireUser(w http.ResponseWriter, r *http.Request) (*domain.User, bool) {
	user, ok := CurrentUser(r)
	if !ok {
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
