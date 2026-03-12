package v1

import (
	"context"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

const buildDownloadURLsConcurrency = 10

func (h *Server) pageParams(ctx context.Context, page, perPage *int) (int, int) {
	pageNum, perPageNum, err := helper.ResolvePageParams(ctx, h.admin.SettingsUC, page, perPage)
	if err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - pageParams - ResolvePageParams failed, using fallback")
		return helper.ClampPage(page), helper.ClampPerPage(perPage, usecase.DefaultPerPage, usecase.DefaultMaxPerPage)
	}
	return pageNum, perPageNum
}

func (h *Server) OnError(w http.ResponseWriter, r *http.Request, err error, op, step string) bool {
	if err == nil {
		return false
	}
	msg := "restapi - v1 - " + op + " - " + step
	if helper.IsExpectedClientError(err) {
		h.infra.Logger.WithError(err).Info(msg)
	} else {
		h.infra.Logger.WithError(err).Error(msg)
	}
	helper.HandleError(w, r, err)
	return true
}

// checkWriteupEnabled loads app settings and ensures writeups are enabled. If not, it
// reports the error and returns false. Use for solution/writeup endpoints.
func (h *Server) checkWriteupEnabled(w http.ResponseWriter, r *http.Request, handlerName, op string) bool {
	settings, err := h.admin.SettingsUC.Get(r.Context())
	if h.OnError(w, r, err, handlerName, "GetSettings") {
		return false
	}
	if !settings.WriteupEnabled {
		h.OnError(w, r, helper.ErrWriteupsDisabled, handlerName, op)
		return false
	}
	return true
}

var _ openapi.ServerInterface = (*Server)(nil)

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

func (h *Server) buildDownloadURLs(ctx context.Context, files []*entity.File, teamID *uuid.UUID, isAdmin bool) (map[string]string, error) {
	if len(files) == 0 {
		return nil, nil
	}
	urls := make(map[string]string, len(files))
	var mu sync.Mutex
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(buildDownloadURLsConcurrency)
	for _, f := range files {
		f := f
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
	if err := g.Wait(); err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - buildDownloadURLs - GetDownloadURLWithAccess")
		return nil, err
	}
	return urls, nil
}
