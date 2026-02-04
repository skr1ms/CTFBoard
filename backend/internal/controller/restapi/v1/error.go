package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

var ErrInvalidID = &ErrorResponse{Error: "invalid ID format", Code: "INVALID_ID"}

func (e *ErrorResponse) Render(_ http.ResponseWriter, r *http.Request) error {
	return nil
}

func RenderInvalidID(w http.ResponseWriter, r *http.Request) {
	RenderErrorWithCode(w, r, http.StatusBadRequest, ErrInvalidID.Error, ErrInvalidID.Code)
}

func RenderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	httputil.RenderError(w, r, status, message)
}

func RenderErrorWithCode(w http.ResponseWriter, r *http.Request, status int, message, code string) {
	httputil.RenderErrorWithCode(w, r, status, message, code)
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	httputil.HandleError(w, r, err)
}

func (h *Server) OnError(w http.ResponseWriter, r *http.Request, err error, op, step string) bool {
	if err == nil {
		return false
	}
	h.logger.WithError(err).Error("restapi - v1 - " + op + " - " + step)
	handleError(w, r, err)
	return true
}
