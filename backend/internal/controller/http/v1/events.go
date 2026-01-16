package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/logger"
)

type eventsRoutes struct {
	solveUC *usecase.SolveUseCase
	logger  logger.Interface
}

func NewEventsRoutes(router chi.Router,
	solveUC *usecase.SolveUseCase,
	logger logger.Interface,
) {
	routes := eventsRoutes{
		solveUC: solveUC,
		logger:  logger,
	}

	router.Get("/events", routes.Events)
}

// @Summary      Scoreboard events stream
// @Description  Establishes SSE connection for receiving scoreboard updates every 5 seconds
// @Tags         Events
// @Produce      text/event-stream
// @Success      200  "Server-Sent Events stream"
// @Router       /events [get]
func (h *eventsRoutes) Events(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx := r.Context()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			entries, err := h.solveUC.GetScoreboard(ctx)
			if err != nil {
				h.logger.Error("http - v1 - Events - GetScoreboard", err)
				continue
			}

			data, err := json.Marshal(entries)
			if err != nil {
				h.logger.Error("http - v1 - Events - Marshal", err)
				continue
			}

			_, err = w.Write([]byte("data: " + string(data) + "\n\n"))
			if err != nil {
				h.logger.Error("http - v1 - Events - Write", err)
				return
			}

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}
