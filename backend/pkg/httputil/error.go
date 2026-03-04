package httputil

import (
	"errors"
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/go-chi/render"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func codeFromStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusGone:
		return "GONE"
	case http.StatusTooManyRequests:
		return "RATE_LIMIT_EXCEEDED"
	default:
		return "INTERNAL_ERROR"
	}
}

func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	var httpErr *httperr.HTTPError
	if errors.As(err, &httpErr) {
		code := httpErr.Code
		if code == "" {
			code = codeFromStatus(httpErr.HTTPStatus())
		}
		render.Status(r, httpErr.HTTPStatus())
		render.JSON(w, r, ErrorResponse{
			Code:    code,
			Message: httpErr.Error(),
		})
		return
	}

	render.Status(r, http.StatusInternalServerError)
	render.JSON(w, r, ErrorResponse{Code: "INTERNAL_ERROR", Message: "Internal server error"})
}
