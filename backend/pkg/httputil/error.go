package httputil

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ValidationErrorItem struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrorResponse struct {
	Code    string                `json:"code"`
	Message string                `json:"message"`
	Errors  []ValidationErrorItem `json:"errors,omitempty"`
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
	case http.StatusPaymentRequired:
		return "PAYMENT_REQUIRED"
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
		message := httpErr.Error()
		if httpErr.HTTPStatus() >= http.StatusInternalServerError {
			message = "Internal server error"
		}
		render.Status(r, httpErr.HTTPStatus())
		render.JSON(w, r, ErrorResponse{
			Code:    code,
			Message: message,
		})
		return
	}

	render.Status(r, http.StatusInternalServerError)
	render.JSON(w, r, ErrorResponse{Code: "INTERNAL_ERROR", Message: "Internal server error"})
}
