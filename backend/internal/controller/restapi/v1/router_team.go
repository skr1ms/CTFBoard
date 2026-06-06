package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-logkit"

	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func setupTeamRoutes(r chi.Router, wrapper openapi.ServerInterfaceWrapper, requireVerified func(http.Handler) http.Handler, redisClient *redis.Client, log logkit.Logger, notBanned func(http.Handler) http.Handler) {
	r.Get("/teams/my", wrapper.GetTeamsMy)

	r.Group(func(me chi.Router) {
		me.Use(restapimiddleware.RequireTeam())
		me.Use(notBanned)
		me.Get("/teams/me/solves", wrapper.GetTeamsMeSolves)
		me.Get("/teams/me/fails", wrapper.GetTeamsMeFails)
		me.Get("/teams/me/awards", wrapper.GetTeamsMeAwards)
		me.Get("/teams/me/invite", wrapper.GetTeamsMeInvite)
		me.Post("/teams/me/invite", wrapper.PostTeamsMeInvite)
	})

	verified := r.With(requireVerified, notBanned)
	verified.Patch("/teams/me", wrapper.PatchTeamsMe)
	verified.Post("/teams/leave", wrapper.PostTeamsLeave)
	verified.Delete("/teams/me", wrapper.DeleteTeamsMe)
	verified.Delete("/teams/members/{ID}", wrapper.DeleteTeamsMembersID)
	verified.Post("/teams/transfer-captain", wrapper.PostTeamsTransferCaptain)

	teamOpLimit := restapimiddleware.RateLimit(redisClient, rlKeyTeamOpUser, teamOpRateLimit, teamOpRateLimitWindow, userIDKeyFunc, log)
	verified.With(teamOpLimit).Post("/teams", wrapper.PostTeams)
	verified.With(teamOpLimit).Post("/teams/join", wrapper.PostTeamsJoin)
	verified.With(teamOpLimit).Post("/teams/solo", wrapper.PostTeamsSolo)
}
