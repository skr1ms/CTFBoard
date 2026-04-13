package v1

import (
	"net/http"
)

// (GET /ws).
func (h *Server) GetWs(w http.ResponseWriter, r *http.Request) {
	h.infra.WSController.HandleWS(w, r)
}
