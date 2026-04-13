package v1

import (
	"context"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-httpkit/httputil"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

const (
	buildDownloadURLsConcurrency = 10
	maxLogoutBodySize            = 4096
	maxSearchQueryLen            = 100
)

func (h *Server) pageParams(ctx context.Context, page, perPage *int) (int, int) {
	pageNum, perPageNum, err := helper.ResolvePageParams(ctx, h.admin.SettingsUC, page, perPage)
	if err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - pageParams - ResolvePageParams failed, using fallback")

		return httputil.ClampPage(page), httputil.ClampPerPage(perPage, usecase.DefaultPerPage, usecase.DefaultMaxPerPage)
	}

	return pageNum, perPageNum
}

func (h *Server) OnError(w http.ResponseWriter, r *http.Request, err error, op, step string) bool {
	return h.errHandler.Handle(w, r, errmap.MapAppError(err), "restapi - v1 - "+op+" - "+step)
}

// checkWriteupEnabled loads app settings and ensures writeups are enabled. If not, it
// reports the error and returns false. Use for solution/writeup endpoints.
func (h *Server) checkWriteupEnabled(w http.ResponseWriter, r *http.Request, handlerName, op string) bool {
	settings, err := h.admin.SettingsUC.Get(r.Context())
	if h.OnError(w, r, err, handlerName, "GetSettings") {
		return false
	}

	if !settings.WriteupEnabled {
		h.OnError(w, r, apperr.ErrWriteupsDisabled, handlerName, op)

		return false
	}

	return true
}

// forceLiveFromParams returns true only when the caller explicitly requests live
// data AND is an admin. Used on mixed-access routes (e.g. scoreboard, statistics)
// where both regular users and admins share the same endpoint.
func forceLiveFromParams(r *http.Request, live *bool) bool {
	if live == nil || !*live {
		return false
	}

	user, ok := middleware.GetUser(r.Context())

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

// buildDownloadURLs generates presigned download URLs for all files concurrently.
// An errgroup with SetLimit(buildDownloadURLsConcurrency=10) bounds parallelism to
// avoid overwhelming the storage backend. A mutex serialises writes into the result
// map. Any single URL error aborts the whole group and returns nil.
func (h *Server) buildDownloadURLs(ctx context.Context, files []*domain.File, teamID *uuid.UUID, isAdmin bool) (map[string]string, error) {
	if len(files) == 0 {
		return nil, nil
	}

	urls := make(map[string]string, len(files))

	var mu sync.Mutex

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(buildDownloadURLsConcurrency)

	for _, f := range files {
		g.Go(func() error {
			u, err := h.challenge.FileUC.GetDownloadURLWithAccess(gCtx, f.ID, teamID, isAdmin)
			if err != nil {
				return err
			}

			mu.Lock()
			urls[f.ID.String()] = u
			mu.Unlock()

			return nil
		})
	}

	err := g.Wait()
	if err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - buildDownloadURLs - GetDownloadURLWithAccess")

		return nil, err
	}

	return urls, nil
}
