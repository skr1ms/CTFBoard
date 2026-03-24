package helper

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type SettingsGetter = middleware.SettingsGetter

type OnErrorFunc func(w http.ResponseWriter, r *http.Request, err error, op, step string) bool

func ParseSearchQuery(w http.ResponseWriter, r *http.Request, q *string, maxLen int, onError OnErrorFunc, op, step string) (string, bool) {
	if q == nil || *q == "" {
		return "", true
	}
	if !httputil.ValidateSearchQ(*q) {
		onError(w, r, httperr.NewValidationErrorf("invalid search query"), op, step)
		return "", false
	}
	return httputil.SanitizeSearchQ(*q, maxLen), true
}

func RequireUser(w http.ResponseWriter, r *http.Request) (*domain.User, bool) {
	user, ok := middleware.GetUser(r.Context())
	if !ok || user == nil {
		httputil.HandleError(w, r, httperr.ErrNotAuthenticated())
		return nil, false
	}
	return user, true
}
