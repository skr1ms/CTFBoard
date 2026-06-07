package v1

import (
	"context"
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	maxSearchQueryLen = 100
)

func (h *Server) pageParams(ctx context.Context, page, perPage *int) (int, int) {
	pageNum, perPageNum, err := helper.ResolvePageParams(ctx, h.admin.SettingsUC, page, perPage)
	if err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - pageParams - ResolvePageParams failed, using fallback")

		return helper.DefaultPageParams(page, perPage)
	}

	return pageNum, perPageNum
}

func (h *Server) OnError(w http.ResponseWriter, r *http.Request, err error, op, step string) bool {
	return helper.HandleAppError(w, r, h.errHandler, err, op, step)
}

// forceLiveFromParams returns true only when the caller explicitly requests live
// data AND is an admin. Used on mixed-access routes (e.g. scoreboard, statistics)
// where both regular users and admins share the same endpoint.
func forceLiveFromParams(r *http.Request, live *bool) bool {
	if live == nil || !*live {
		return false
	}

	user, ok := helper.CurrentUser(r)

	return ok && helper.IsAdmin(user)
}

// adminForceLive returns true when the caller requests live data. Role is not
// re-checked here because these endpoints are already protected by admin
// middleware.
func adminForceLive(live *bool) bool {
	return live != nil && *live
}

var _ openapi.ServerInterface = (*Server)(nil)

type Server struct {
	openapi.Unimplemented

	challenge  helper.ChallengeDeps
	team       helper.TeamDeps
	user       helper.UserDeps
	comp       helper.CompetitionDeps
	admin      helper.AdminDeps
	infra      helper.InfraDeps
	errHandler *httputil.ErrorHandler
}

func NewServer(deps *helper.ServerDeps) *Server {
	if deps == nil {
		return nil
	}

	return &Server{
		challenge:  deps.Challenge,
		team:       deps.Team,
		user:       deps.User,
		comp:       deps.Comp,
		admin:      deps.Admin,
		infra:      deps.Infra,
		errHandler: &httputil.ErrorHandler{Logger: deps.Infra.Logger},
	}
}
