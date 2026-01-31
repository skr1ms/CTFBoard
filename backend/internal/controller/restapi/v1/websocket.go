package v1

import (
	"net/http"
)

// WebSocket connection
// (GET /ws)
func (h *Server) GetWs(w http.ResponseWriter, r *http.Request) {
	h.wsController.HandleWS(w, r)
}
