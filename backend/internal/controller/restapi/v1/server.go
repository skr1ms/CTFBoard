package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// OnError logs err, gives the client a response via HandleError, and returns true.
// The method is declared here, not in helper/error, so as not to create an import loop v1 ↔ helper.
func (h *Server) OnError(w http.ResponseWriter, r *http.Request, err error, op, step string) bool {
	if err == nil {
		return false
	}
	h.infra.Logger.WithError(err).Error("restapi - v1 - " + op + " - " + step)
	helper.HandleError(w, r, err)
	return true
}

type Server struct {
	openapi.Unimplemented
	challenge helper.ChallengeDeps
	team      helper.TeamDeps
	user      helper.UserDeps
	comp      helper.CompetitionDeps
	admin     helper.AdminDeps
	infra     helper.InfraDeps
}

func NewServer(deps *helper.ServerDeps) *Server {
	if deps == nil {
		return nil
	}
	return &Server{
		challenge: deps.Challenge,
		team:      deps.Team,
		user:      deps.User,
		comp:      deps.Comp,
		admin:     deps.Admin,
		infra:     deps.Infra,
	}
}
