package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wahrwelt-kit/go-cachekit"

	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// setupProtectedRoutes creates the RequireTeamNotBanned middleware (shared across
// all route groups so the team cache is consulted only once per request) and then
// delegates to the sub-group setup functions: setupConditionalPublicRoutes
// (optionally-public routes gated by visibility config), setupBasicAuthRoutes,
// setupWebSocketRoute, and setupFileDownloadRoute.
func setupProtectedRoutes(
	router chi.Router,
	server *Server,
	deps *helper.ServerDeps,
	wrapper openapi.ServerInterfaceWrapper,
	verifyEmails bool,
	rateLimitCache *restapimiddleware.RateLimitConfigCache,
	sharedCache *cachekit.Cache,
	ipTracking func(http.Handler) http.Handler,
	notUserBanned func(http.Handler) http.Handler,
	scoreboardVis func(http.Handler) http.Handler,
) {
	notBanned := restapimiddleware.RequireTeamNotBanned(deps.Team.ReadUC, sharedCache)
	setupConditionalPublicRoutes(router, deps, wrapper, rateLimitCache, sharedCache, ipTracking, notBanned, notUserBanned)
	setupBannedAccessibleRoutes(router, deps, wrapper, sharedCache, ipTracking)
	setupBasicAuthRoutes(router, deps, wrapper, verifyEmails, rateLimitCache, sharedCache, ipTracking, notBanned, notUserBanned)
	setupWebSocketRoute(router, deps, wrapper, rateLimitCache, sharedCache, ipTracking, notBanned, notUserBanned, scoreboardVis)
	setupFileDownloadRoute(router, server, deps, wrapper, verifyEmails, rateLimitCache, sharedCache, notBanned, notUserBanned, ipTracking)
}

// setupBannedAccessibleRoutes registers routes accessible to banned users:
// GET /auth/me (so banned users can see their ban status) and the appeals endpoints.
// Uses Auth + InjectUser middleware WITHOUT RequireUserNotBanned.
func setupBannedAccessibleRoutes(
	router chi.Router,
	deps *helper.ServerDeps,
	wrapper openapi.ServerInterfaceWrapper,
	sharedCache *cachekit.Cache,
	ipTracking func(http.Handler) http.Handler,
) {
	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC, deps.Infra.Logger))
		r.Use(restapimiddleware.InjectUser(deps.User.UserUC, sharedCache, deps.Infra.Logger))
		r.Use(ipTracking)

		r.Get("/auth/me", wrapper.GetAuthMe)
		r.Post("/appeals", wrapper.PostAppeals)
		r.Get("/appeals/me", wrapper.GetAppealsMe)
	})
}

// setupBasicAuthRoutes registers all routes that always require authentication:
// profile read/update, per-user solves/fails/awards, notifications, API tokens,
// avatar upload/delete, team management, challenge write operations (submit,
// hint-unlock, comments, ratings), and admin routes.
func setupBasicAuthRoutes(
	router chi.Router,
	deps *helper.ServerDeps,
	wrapper openapi.ServerInterfaceWrapper,
	verifyEmails bool,
	rateLimitCache *restapimiddleware.RateLimitConfigCache,
	sharedCache *cachekit.Cache,
	ipTracking func(http.Handler) http.Handler,
	notBanned func(http.Handler) http.Handler,
	notUserBanned func(http.Handler) http.Handler,
) {
	rl := newDynamicRL(deps.Infra.RedisClient, rateLimitCache, deps.Admin.SettingsUC, deps.Infra.Logger)
	keyFunc := ipKeyFunc()
	generalIP := func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) }

	profileUpdateLimit := rl(rlKeyProfileUpdateIP, defaultRLWindow, generalIP, keyFunc)
	apiTokenLimit := rl(rlKeyAPITokenIP, defaultRLWindow, generalIP, keyFunc)
	notificationLimit := rl(rlKeyNotificationIP, defaultRLWindow, generalIP, keyFunc)
	shareCreateLimit := rl(rlKeyShareCreateUser, defaultRLWindow, generalIP, userIDKeyFunc)
	protectedReadLimit := rl(rlKeyProtectedReadIP, defaultRLWindow, generalIP, keyFunc)
	challengeReadLimit := rl(rlKeyChallengeReadIP, defaultRLWindow, generalIP, keyFunc)
	requireVerified := restapimiddleware.RequireVerifiedFromSettings(verifyEmails, deps.Admin.SettingsUC, deps.Infra.Logger)
	accountVisibility := restapimiddleware.VisibilityGuard(deps.Admin.CompetitionParamUC, "account_visibility")
	challengeVisibility := restapimiddleware.VisibilityGuard(deps.Admin.CompetitionParamUC, "challenge_visibility")
	challengeStarted := restapimiddleware.ChallengeVisibility(deps.Comp.CompetitionUC)

	router.Group(func(r chi.Router) {
		r.Use(protectedMiddlewareStack(deps, sharedCache, ipTracking, notUserBanned)...)

		r.With(restapimiddleware.RequireBearerAuth(), profileUpdateLimit).Patch("/auth/me", wrapper.PatchAuthMe)

		r.Group(func(me chi.Router) {
			me.Use(notBanned)
			me.Get("/users/me/solves", wrapper.GetUsersMeSolves)
			me.Get("/users/me/fails", wrapper.GetUsersMeFails)
			me.Get("/users/me/awards", wrapper.GetUsersMeAwards)
			me.Get("/users/me/submissions", wrapper.GetUsersMeSubmissions)
		})

		r.With(notificationLimit).Get("/user/notifications", wrapper.GetUserNotifications)
		r.With(notificationLimit).Get("/user/notifications/unread-count", wrapper.GetUserNotificationsUnreadCount)
		r.With(notificationLimit).Patch("/user/notifications/{ID}/read", wrapper.PatchUserNotificationsIDRead)

		r.Group(func(tokens chi.Router) {
			tokens.Use(restapimiddleware.RequireBearerAuth())
			tokens.Use(requireVerified)
			tokens.Get("/user/tokens", wrapper.GetUserTokens)
			tokens.With(apiTokenLimit).Post("/user/tokens", wrapper.PostUserTokens)
			tokens.With(apiTokenLimit).Delete("/user/tokens/{ID}", wrapper.DeleteUserTokensID)
		})

		avatarUploadLimitMw := restapimiddleware.RateLimit(deps.Infra.RedisClient, rlKeyAvatarUploadUser, avatarUploadLimit, avatarUploadWindow, userIDKeyFunc, deps.Infra.Logger)
		verified := r.With(requireVerified, notBanned)
		verified.With(avatarUploadLimitMw).Put("/users/me/avatar", wrapper.PutUsersMeAvatar)
		verified.Delete("/users/me/avatar", wrapper.DeleteUsersMeAvatar)
		verified.With(avatarUploadLimitMw).Put("/teams/me/avatar", wrapper.PutTeamsMeAvatar)
		verified.Delete("/teams/me/avatar", wrapper.DeleteTeamsMeAvatar)
		verified.With(restapimiddleware.RequireTeam(), shareCreateLimit).Post("/shares", wrapper.PostShares)

		r.Group(func(acc chi.Router) {
			acc.Use(notBanned)
			acc.Use(accountVisibility)
			acc.Use(protectedReadLimit)
			acc.Get("/users", wrapper.GetUsers)
			acc.Get("/users/{ID}", wrapper.GetUsersID)
			acc.Get("/teams", wrapper.GetTeams)
			acc.Get("/teams/{ID}", wrapper.GetTeamsID)
		})

		r.Group(func(ch chi.Router) {
			ch.Use(notBanned)
			ch.Use(challengeVisibility)
			ch.Use(challengeReadLimit)
			ch.Get("/challenges", wrapper.GetChallenges)

			ch.Group(func(direct chi.Router) {
				direct.Use(challengeStarted)
				direct.Get("/challenges/solutions", wrapper.GetChallengesSolutions)
				direct.Get("/challenges/{challengeID}", wrapper.GetChallengesChallengeID)
				direct.Get("/challenges/{challengeID}/files", wrapper.GetChallengesChallengeIDFiles)
				direct.Get("/challenges/{challengeID}/hints", wrapper.GetChallengesChallengeIDHints)
				direct.Get("/challenges/{challengeID}/tags", wrapper.GetChallengesChallengeIDTags)
				direct.Get("/challenges/{challengeID}/requirements", wrapper.GetChallengesChallengeIDRequirements)
				direct.Get("/challenges/{challengeID}/solution", wrapper.GetChallengesChallengeIDSolution)
			})
		})

		setupTeamRoutes(r, wrapper, requireVerified, deps.Infra.RedisClient, deps.Infra.Logger, notBanned)
		setupChallengeRoutes(r, wrapper, deps, rateLimitCache, requireVerified, sharedCache, notBanned)

		fileDownloadLimit := rl(rlKeyFileDownloadIP, defaultRLWindow, generalIP, ipKeyFunc())
		r.With(requireVerified, challengeVisibility, notBanned, fileDownloadLimit).Get("/files/by-id/{ID}/download", wrapper.GetFilesIDDownload)

		setupAdminRoutes(r, wrapper, deps.Infra.RedisClient, deps.Infra.Logger)
	})
}

// setupConditionalPublicRoutes registers routes that may be visible to guests.
// OptionalAuth authenticates requests when credentials are present but lets
// guests through. VisibilityGuard then enforces score_visibility:
//
//	score_visibility    -> /scoreboard, /statistics, per-user/team/challenge solves/fails/awards
//
// Account and challenge read routes are intentionally auth-only and are wired in
// setupBasicAuthRoutes, where account_visibility/challenge_visibility still gate
// authenticated non-admin users.
func setupConditionalPublicRoutes(
	router chi.Router,
	deps *helper.ServerDeps,
	wrapper openapi.ServerInterfaceWrapper,
	rateLimitCache *restapimiddleware.RateLimitConfigCache,
	sharedCache *cachekit.Cache,
	ipTracking func(http.Handler) http.Handler,
	notBanned func(http.Handler) http.Handler,
	notUserBanned func(http.Handler) http.Handler,
) {
	rl := newDynamicRL(deps.Infra.RedisClient, rateLimitCache, deps.Admin.SettingsUC, deps.Infra.Logger)
	keyFunc := ipKeyFunc()

	scoreboardLimit := rl(rlKeyScoreboardIP, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.ScoreboardPerMinute) }, keyFunc)

	compParamUC := deps.Admin.CompetitionParamUC

	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.OptionalAuth(deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC, deps.Infra.Logger))
		r.Use(restapimiddleware.OptionalInjectUser(deps.User.UserUC, sharedCache, deps.Infra.Logger))
		r.Use(ipTracking)
		r.Use(notUserBanned)

		// Scoreboard & statistics gated by score_visibility
		r.Group(func(sb chi.Router) {
			sb.Use(notBanned)
			sb.Use(restapimiddleware.VisibilityGuard(compParamUC, "score_visibility"))
			sb.Use(scoreboardLimit)
			sb.Get("/scoreboard", wrapper.GetScoreboard)
			sb.Get("/scoreboard/graph", wrapper.GetScoreboardGraph)
			sb.Get("/users/{ID}/solves", wrapper.GetUsersIDSolves)
			sb.Get("/users/{ID}/fails", wrapper.GetUsersIDFails)
			sb.Get("/users/{ID}/awards", wrapper.GetUsersIDAwards)
			sb.Get("/teams/solves/{teamID}", wrapper.GetTeamsIDSolves)
			sb.Get("/teams/fails/{teamID}", wrapper.GetTeamsIDFails)
			sb.Get("/teams/awards/{teamID}", wrapper.GetTeamsIDAwards)
			sb.Get("/challenges/{challengeID}/solves", wrapper.GetChallengesChallengeIDSolves)
			sb.Get("/challenges/{challengeID}/first-blood", wrapper.GetChallengesChallengeIDFirstBlood)
			sb.Get("/statistics/general", wrapper.GetStatisticsGeneral)
			sb.Get("/statistics/challenges", wrapper.GetStatisticsChallenges)
			sb.Get("/statistics/challenges/{ID}", wrapper.GetStatisticsChallengesID)
			sb.Get("/statistics/challenges/solves/percentages", wrapper.GetStatisticsChallengesSolvesPercentages)
			sb.Get("/statistics/scores/distribution", wrapper.GetStatisticsScoresDistribution)
			sb.Get("/statistics/submissions", wrapper.GetStatisticsSubmissions)
			sb.Get("/statistics/submissions/{type}", wrapper.GetStatisticsSubmissionsType)
			sb.Get("/statistics/teams", wrapper.GetStatisticsTeams)
			sb.Get("/statistics/users", wrapper.GetStatisticsUsers)
			sb.Get("/statistics/scoreboard", wrapper.GetStatisticsScoreboard)
		})
	})
}
