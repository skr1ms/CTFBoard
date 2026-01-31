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
	httputil.RenderError(w, r, http.StatusBadRequest, ErrInvalidID.Error)
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	httputil.HandleError(w, r, err)
}
