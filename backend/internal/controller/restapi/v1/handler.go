package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
)

type HandlerResult struct {
	Data   any
	Status int
}

func OK(data any) HandlerResult {
	return HandlerResult{Data: data, Status: http.StatusOK}
}

func Created(data any) HandlerResult {
	return HandlerResult{Data: data, Status: http.StatusCreated}
}

func NoContent() HandlerResult {
	return HandlerResult{Status: http.StatusNoContent}
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request) (HandlerResult, error)

func (h *Server) Handle(name string, fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := fn(w, r)
		if err != nil {
			h.infra.Logger.WithError(err).Error("restapi - v1 - " + name)
			helper.HandleError(w, r, err)
			return
		}
		switch result.Status {
		case http.StatusNoContent:
			helper.RenderNoContent(w, r)
		case http.StatusCreated:
			helper.RenderCreated(w, r, result.Data)
		default:
			helper.RenderOK(w, r, result.Data)
		}
	}
}
