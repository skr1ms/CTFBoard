package httputil

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

func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	var httpErr *entityError.HTTPError
	if errors.As(err, &httpErr) {
		render.Status(r, httpErr.HTTPStatus())
		render.JSON(w, r, ErrorResponse{
			Error: httpErr.Error(),
			Code:  httpErr.Code,
		})
		return
	}

	render.Status(r, http.StatusInternalServerError)
	render.JSON(w, r, ErrorResponse{Error: "Internal server error"})
}
