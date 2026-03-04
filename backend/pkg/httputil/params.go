package httputil

import (
	"context"
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

func ParseUUID(w http.ResponseWriter, r *http.Request, id string) (uuid.UUID, bool) {
	if id == "" {
		HandleError(w, r, httperr.ErrInvalidID)
		return uuid.Nil, false
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		HandleError(w, r, httperr.ErrInvalidID)
		return uuid.Nil, false
	}
	return parsed, true
}

func ParseUUIDField(w http.ResponseWriter, r *http.Request, value, field string) (uuid.UUID, bool) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		HandleError(w, r, httperr.NewValidationErrorf("invalid %s", field))
		return uuid.Nil, false
	}
	return parsed, true
}

func ParseAuthUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID := GetUserID(r.Context())
	if userID == "" {
		HandleError(w, r, httperr.ErrNotAuthenticated)
		return uuid.Nil, false
	}
	return ParseUUID(w, r, userID)
}
