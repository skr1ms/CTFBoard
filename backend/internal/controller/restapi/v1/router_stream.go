package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wahrwelt-kit/go-cachekit"

	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func setupWebSocketRoute(router chi.Router, deps *helper.ServerDeps, wrapper openapi.ServerInterfaceWrapper, rateLimitCache *restapimiddleware.RateLimitConfigCache, sharedCache *cachekit.Cache, ipTracking, notBanned, notUserBanned, scoreboardVis func(http.Handler) http.Handler) {
	rl := newDynamicRL(deps.Infra.RedisClient, rateLimitCache, deps.Admin.SettingsUC, deps.Infra.Logger)
	wsLimit := rl(rlKeyWebSocketIP, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) }, ipKeyFunc())

	router.Group(func(r chi.Router) {
		r.Use(protectedMiddlewareStack(deps, sharedCache, ipTracking, notUserBanned)...)
		r.Use(notBanned)
		r.Use(scoreboardVis)
		r.Use(wsLimit)

		r.Get("/ws", wrapper.GetWs)

		if deps.Infra.SSEHandler != nil {
			r.Get("/sse", deps.Infra.SSEHandler.ServeHTTP)
		}
	})
}

// setupFileDownloadRoute registers the chi wildcard GET /files/download/* route
// behind a dedicated middleware stack (Auth, InjectUser, IP tracking, notUserBanned,
// RequireVerified, ChallengeVisibility, notBanned, file-download rate limit).
// It bridges the chi.URLParam wildcard to the typed OpenAPI handler rather than
// registering the handler directly, preserving OpenAPI param extraction.
// Also registers the public /avatars/* wildcard (public read rate limit only).
func setupFileDownloadRoute(router chi.Router, server *Server, deps *helper.ServerDeps, _ openapi.ServerInterfaceWrapper, verifyEmails bool, rateLimitCache *restapimiddleware.RateLimitConfigCache, sharedCache *cachekit.Cache, notBanned, notUserBanned, ipTracking func(http.Handler) http.Handler) {
	rl := newDynamicRL(deps.Infra.RedisClient, rateLimitCache, deps.Admin.SettingsUC, deps.Infra.Logger)
	generalIP := func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) }
	fileDownloadLimit := rl(rlKeyFileDownloadIP, defaultRLWindow, generalIP, ipKeyFunc())
	requireVerified := restapimiddleware.RequireVerifiedFromSettings(verifyEmails, deps.Admin.SettingsUC, deps.Infra.Logger)
	challengeVisibility := restapimiddleware.VisibilityGuard(deps.Admin.CompetitionParamUC, "challenge_visibility")

	router.Group(func(r chi.Router) {
		r.Use(protectedMiddlewareStack(deps, sharedCache, ipTracking, notUserBanned)...)
		r.Use(requireVerified)
		r.Use(restapimiddleware.ChallengeVisibility(deps.Comp.CompetitionUC))
		r.Use(challengeVisibility)
		r.Use(notBanned)
		r.Use(fileDownloadLimit)
		r.Get("/files/download/*", func(w http.ResponseWriter, req *http.Request) {
			path := chi.URLParam(req, "*")
			server.GetFilesDownloadPath(w, req, path, openapi.GetFilesDownloadPathParams{Token: req.URL.Query().Get("token")})
		})
	})

	avatarLimit := rl(rlKeyPublicReadIP, defaultRLWindow, generalIP, ipKeyFunc())
	router.With(avatarLimit).Get("/avatars/*", func(w http.ResponseWriter, req *http.Request) {
		path := chi.URLParam(req, "*")
		server.GetAvatarByPath(w, req, path)
	})
}
