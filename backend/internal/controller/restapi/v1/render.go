package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

func RenderOK[T any](w http.ResponseWriter, r *http.Request, data T) {
	httputil.RenderOK(w, r, data)
}

func RenderCreated[T any](w http.ResponseWriter, r *http.Request, data T) {
	httputil.RenderCreated(w, r, data)
}

func RenderNoContent(w http.ResponseWriter, r *http.Request) {
	httputil.RenderNoContent(w, r)
}

func RenderJSON[T any](w http.ResponseWriter, r *http.Request, status int, data T) {
	httputil.RenderJSON(w, r, status, data)
}
