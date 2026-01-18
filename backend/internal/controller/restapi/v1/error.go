package v1

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
)

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, entityError.ErrUserAlreadyExists):
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, ErrorResponse{Error: err.Error()})
		return

	case errors.Is(err, entityError.ErrUserNotFound):
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, ErrorResponse{Error: err.Error()})
		return

	case errors.Is(err, entityError.ErrChallengeNotFound):
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, ErrorResponse{Error: err.Error()})
		return

	case errors.Is(err, entityError.ErrUserMustBeInTeam):
		render.Status(r, http.StatusForbidden)
		render.JSON(w, r, ErrorResponse{Error: err.Error()})
		return

	case errors.Is(err, entityError.ErrAlreadySolved):
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, ErrorResponse{Error: err.Error()})
		return

	case errors.Is(err, entityError.ErrSolveNotFound):
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, ErrorResponse{Error: err.Error()})
		return

	case errors.Is(err, entityError.ErrTeamNotFound):
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, ErrorResponse{Error: err.Error()})
		return

	case errors.Is(err, entityError.ErrTeamAlreadyExists):
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, ErrorResponse{Error: err.Error()})
		return

	case errors.Is(err, entityError.ErrUserAlreadyInTeam):
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, ErrorResponse{Error: err.Error()})
		return

	case errors.Is(err, entityError.ErrInvalidCredentials):
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, ErrorResponse{Error: err.Error()})
		return

	default:
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse{Error: "Internal server error"})
	}
}
