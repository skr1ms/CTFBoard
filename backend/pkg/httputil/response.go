package httputil

import (
	"net/http"

	"github.com/go-chi/render"
)

func RenderJSON[T any](w http.ResponseWriter, r *http.Request, status int, data T) {
	render.Status(r, status)
	render.JSON(w, r, data)
}

func RenderNoContent(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func RenderCreated[T any](w http.ResponseWriter, r *http.Request, data T) {
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, data)
}

func RenderOK[T any](w http.ResponseWriter, r *http.Request, data T) {
	render.Status(r, http.StatusOK)
	render.JSON(w, r, data)
}

func RenderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	render.Status(r, status)
	render.JSON(w, r, map[string]string{"error": message})
}

func RenderErrorWithCode(w http.ResponseWriter, r *http.Request, status int, message, code string) {
	render.Status(r, status)
	render.JSON(w, r, map[string]any{
		"error": message,
		"code":  code,
	})
}
