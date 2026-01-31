package httputil

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
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

func ParseuuidParam(w http.ResponseWriter, r *http.Request, paramName string) (uuid.UUID, bool) {
	paramValue := chi.URLParam(r, paramName)
	if paramValue == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "missing parameter: " + paramName})
		return uuid.Nil, false
	}

	parsed, err := uuid.Parse(paramValue)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": "invalid uuid format for parameter: " + paramName,
			"code":  "INVALID_UUID",
		})
		return uuid.Nil, false
	}

	return parsed, true
}

func ParseAuthUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID := GetUserID(r.Context())
	if userID == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{
			"error": "not authenticated",
			"code":  "UNAUTHORIZED",
		})
		return uuid.Nil, false
	}

	useruuid, err := uuid.Parse(userID)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": "invalid user ID format",
			"code":  "INVALID_USER_ID",
		})
		return uuid.Nil, false
	}

	return useruuid, true
}
