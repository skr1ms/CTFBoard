package v1

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
)

const (
	rlKeyLoginIP           = "auth:login:ip"
	rlKeyRegisterIP        = "auth:register:ip"
	rlKeyForgotIP          = "auth:forgot:ip"
	rlKeyResetIP           = "auth:reset:ip"
	rlKeyLogoutIP          = "auth:logout:ip"
	rlKeyRefreshIP         = "auth:refresh:ip"
	rlKeyVerifyEmailIP     = "auth:verify-email:ip"
	rlKeyOAuthCallbackIP   = "auth:oauth-callback:ip"
	rlKeyOAuthRedirectIP   = "auth:oauth-redirect:ip"
	rlKeyResendVerifyIP    = "auth:resend-verification:ip"
	rlKeyScoreboardIP      = "scoreboard:ip"
	rlKeyTeamOpUser        = "team:op:user"
	rlKeySubmitIP          = "submit:ip"
	rlKeySubmitUser        = "submit:user"
	rlKeyHintUnlockUser    = "hint:unlock:user"
	rlKeyCommentUser       = "comment:user"
	rlKeyProfileUpdateIP   = "auth:profile-update:ip"
	rlKeyAPITokenIP        = "user:api-token:ip"
	rlKeyNotificationIP    = "user:notification:ip"
	rlKeyAdminExportZip    = "admin:export:zip:user"
	rlKeyAdminDestructive  = "admin:destructive:user"
	rlKeyAdminGeneral      = "admin:general:user"
	rlKeyPublicReadIP      = "public:read:ip"
	rlKeyProtectedReadIP   = "protected:read:ip"
	rlKeyChallengeReadIP   = "challenge:read:ip"
	rlKeyFileDownloadIP    = "file:download:ip"
	rlKeyWebSocketIP       = "websocket:ip"
	teamOpRateLimit        = 5
	teamOpRateLimitWindow  = time.Minute
	adminExportZipLimit    = 3
	adminExportZipWindow   = 5 * time.Minute
	adminDestructiveLimit  = 3
	adminDestructiveWindow = 5 * time.Minute
	adminGeneralLimit      = 120
	adminGeneralWindow     = time.Minute
)

// ipKeyFunc returns a rate-limit key function that uses the client IP address.
func ipKeyFunc(trustedProxyCIDRs []string) func(*http.Request) (string, error) {
	return func(r *http.Request) (string, error) {
		return helper.GetClientIP(r, trustedProxyCIDRs), nil
	}
}

// userIDKeyFunc returns a rate-limit key function that uses the authenticated user's ID.
func userIDKeyFunc(r *http.Request) (string, error) {
	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		return "", errors.New("user not authenticated")
	}
	return user.ID.String(), nil
}

func scoreboardVisibilityMiddleware(deps *helper.ServerDeps) func(http.Handler) http.Handler {
	getter := deps.Admin.SettingsRepo
	if deps.Infra.ScoreboardVisibilityCache != nil {
		return deps.Infra.ScoreboardVisibilityCache.Middleware(getter)
	}
	return restapimiddleware.ScoreboardVisibility(getter)
}

func NewRouter(
	ctx context.Context,
	router chi.Router,
	deps *helper.ServerDeps,
	verifyEmails bool,
	rateLimitCache *helper.RateLimitConfigCache,
) {
	server := NewServer(deps)
	wrapper := openapi.ServerInterfaceWrapper{
		Handler: server,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			var requiredHeader *openapi.RequiredHeaderError
			if errors.As(err, &requiredHeader) {
				server.OnError(w, r, helper.New(err, http.StatusUnauthorized, "UNAUTHORIZED"), "OpenAPI", "RequiredHeader")
				return
			}
			var requiredParam *openapi.RequiredParamError
			if errors.As(err, &requiredParam) {
				server.OnError(w, r, helper.New(err, http.StatusBadRequest, "BAD_REQUEST"), "OpenAPI", "RequiredParam")
				return
			}
			var invalidParam *openapi.InvalidParamFormatError
			if errors.As(err, &invalidParam) {
				server.OnError(w, r, helper.New(err, http.StatusBadRequest, "BAD_REQUEST"), "OpenAPI", "InvalidParamFormat")
				return
			}
			server.OnError(w, r, helper.New(errors.New("invalid request parameter"), http.StatusBadRequest, "BAD_REQUEST"), "OpenAPI", "BadRequest")
		},
	}
	sharedCache := cache.New(deps.Infra.RedisClient)
	ipTracking := restapimiddleware.IPTracking(ctx, deps.User.TrackingUC, deps.Infra.TrustedProxyCIDRs, deps.Infra.Logger)
	notUserBanned := restapimiddleware.RequireUserNotBanned()
	scoreboardVis := scoreboardVisibilityMiddleware(deps)
	setupPublicRoutes(router, wrapper, deps, deps.Infra.RedisClient, deps.Infra.Logger, deps.Infra.TrustedProxyCIDRs, rateLimitCache)
	setupAuthOnlyRoutes(router, deps, wrapper, sharedCache, rateLimitCache, notUserBanned)
	setupProtectedRoutes(router, server, deps, wrapper, verifyEmails, rateLimitCache, sharedCache, ipTracking, notUserBanned, scoreboardVis)
}

func setupPublicRoutes(router chi.Router, wrapper openapi.ServerInterfaceWrapper, deps *helper.ServerDeps, redisClient *redis.Client, logger logger.Logger, trustedProxyCIDRs []string, rateLimitCache *helper.RateLimitConfigCache) {
	getter := deps.Admin.SettingsUC
	keyFunc := ipKeyFunc(trustedProxyCIDRs)

	loginLimit := helper.RateLimitFromConfig(redisClient, rlKeyLoginIP, time.Minute, rateLimitCache, getter, func(c *helper.RateLimitConfig) int64 { return int64(c.LoginPerMinute) }, keyFunc, trustedProxyCIDRs, logger)
	registerLimit := helper.RateLimitFromConfig(redisClient, rlKeyRegisterIP, time.Minute, rateLimitCache, getter, func(c *helper.RateLimitConfig) int64 { return int64(c.RegisterPerMinute) }, keyFunc, trustedProxyCIDRs, logger)
	forgotPasswordLimit := helper.RateLimitFromConfig(redisClient, rlKeyForgotIP, time.Minute, rateLimitCache, getter, func(c *helper.RateLimitConfig) int64 { return int64(c.ForgotPasswordPerMinute) }, keyFunc, trustedProxyCIDRs, logger)
	resetPasswordLimit := helper.RateLimitFromConfig(redisClient, rlKeyResetIP, time.Minute, rateLimitCache, getter, func(c *helper.RateLimitConfig) int64 { return int64(c.ResetPasswordPerMinute) }, keyFunc, trustedProxyCIDRs, logger)
	logoutLimit := helper.RateLimitFromConfig(redisClient, rlKeyLogoutIP, time.Minute, rateLimitCache, getter, func(c *helper.RateLimitConfig) int64 { return int64(c.LogoutPerMinute) }, keyFunc, trustedProxyCIDRs, logger)
	refreshLimit := helper.RateLimitFromConfig(redisClient, rlKeyRefreshIP, time.Minute, rateLimitCache, getter, func(c *helper.RateLimitConfig) int64 { return int64(c.RefreshPerMinute) }, keyFunc, trustedProxyCIDRs, logger)

	verifyEmailLimit := helper.RateLimitFromConfig(redisClient, rlKeyVerifyEmailIP, time.Minute, rateLimitCache, getter, func(c *helper.RateLimitConfig) int64 { return int64(c.VerifyEmailPerMinute) }, keyFunc, trustedProxyCIDRs, logger)
	oauthCallbackLimit := helper.RateLimitFromConfig(redisClient, rlKeyOAuthCallbackIP, time.Minute, rateLimitCache, getter, func(c *helper.RateLimitConfig) int64 { return int64(c.OAuthCallbackPerMinute) }, keyFunc, trustedProxyCIDRs, logger)
	oauthRedirectLimit := helper.RateLimitFromConfig(redisClient, rlKeyOAuthRedirectIP, time.Minute, rateLimitCache, getter, func(c *helper.RateLimitConfig) int64 { return int64(c.OAuthRedirectPerMinute) }, keyFunc, trustedProxyCIDRs, logger)
	publicReadLimit := helper.RateLimitFromConfig(redisClient, rlKeyPublicReadIP, time.Minute, rateLimitCache, getter, func(c *helper.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) }, keyFunc, trustedProxyCIDRs, logger)

	router.Group(func(r chi.Router) {
		// Auth endpoints with rate limiting
		r.With(loginLimit).Post("/auth/login", wrapper.PostAuthLogin)
		r.With(registerLimit).Post("/auth/register", wrapper.PostAuthRegister)
		r.With(verifyEmailLimit).Post("/auth/verify-email", wrapper.PostAuthVerifyEmail)
		r.With(forgotPasswordLimit).Post("/auth/forgot-password", wrapper.PostAuthForgotPassword)
		r.With(resetPasswordLimit).Post("/auth/reset-password", wrapper.PostAuthResetPassword)
		r.With(refreshLimit).Post("/auth/refresh", wrapper.PostAuthRefresh)
		r.With(logoutLimit).Post("/auth/logout", wrapper.PostAuthLogout)

		// OAuth endpoints
		r.With(oauthRedirectLimit).Get("/auth/oauth/{provider}", wrapper.GetAuthOauthProvider)
		r.With(oauthCallbackLimit).Get("/auth/oauth/{provider}/callback", wrapper.GetAuthOauthProviderCallback)

		// Public endpoints (rate limited to mitigate DoS)
		r.With(publicReadLimit).Get("/competition/status", wrapper.GetCompetitionStatus)
		r.With(publicReadLimit).Get("/tags", wrapper.GetTags)
		r.With(publicReadLimit).Get("/fields", wrapper.GetFields)
		r.With(publicReadLimit).Get("/brackets", wrapper.GetBrackets)
		r.With(publicReadLimit).Get("/pages", wrapper.GetPages)
		r.With(publicReadLimit).Get("/pages/{slug}", wrapper.GetPagesSlug)
		r.With(publicReadLimit).Get("/notifications", wrapper.GetNotifications)
		r.With(publicReadLimit).Get("/challenges/types", wrapper.GetChallengesTypes)
		r.With(publicReadLimit).Get("/healthcheck", wrapper.GetHealthcheck)
		r.With(publicReadLimit).Get("/robots.txt", wrapper.GetRobotsTxt)
		r.With(publicReadLimit).Get("/tos", wrapper.GetTos)
		r.With(publicReadLimit).Get("/privacy", wrapper.GetPrivacy)
		r.With(publicReadLimit).Get("/configs/public", wrapper.GetConfigsPublic)
	})
}

func setupAuthOnlyRoutes(router chi.Router, deps *helper.ServerDeps, wrapper openapi.ServerInterfaceWrapper, sharedCache *cache.Cache, rateLimitCache *helper.RateLimitConfigCache, notUserBanned func(http.Handler) http.Handler) {
	resendVerificationLimit := helper.RateLimitFromConfig(
		deps.Infra.RedisClient, rlKeyResendVerifyIP, time.Minute,
		rateLimitCache, deps.Admin.SettingsUC,
		func(c *helper.RateLimitConfig) int64 { return int64(c.VerifyEmailPerMinute) },
		ipKeyFunc(deps.Infra.TrustedProxyCIDRs), deps.Infra.TrustedProxyCIDRs, deps.Infra.Logger,
	)

	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC, deps.Infra.Logger))
		r.Use(restapimiddleware.InjectUser(deps.User.UserUC, sharedCache, deps.Infra.Logger))
		r.Use(notUserBanned)

		r.With(resendVerificationLimit).Post("/auth/resend-verification", wrapper.PostAuthResendVerification)
	})
}

func setupProtectedRoutes(
	router chi.Router,
	server *Server,
	deps *helper.ServerDeps,
	wrapper openapi.ServerInterfaceWrapper,
	verifyEmails bool,
	rateLimitCache *helper.RateLimitConfigCache,
	sharedCache *cache.Cache,
	ipTracking func(http.Handler) http.Handler,
	notUserBanned func(http.Handler) http.Handler,
	scoreboardVis func(http.Handler) http.Handler,
) {
	notBanned := restapimiddleware.RequireTeamNotBanned(deps.Team.TeamUC, sharedCache)
	setupBasicAuthRoutes(router, deps, wrapper, verifyEmails, rateLimitCache, sharedCache, ipTracking, notBanned, notUserBanned, scoreboardVis)
	setupScoreboardRoutes(router, deps, wrapper, rateLimitCache, sharedCache, ipTracking, notBanned, notUserBanned, scoreboardVis)
	setupFirstBloodRoute(router, deps, wrapper, verifyEmails, rateLimitCache, sharedCache, ipTracking, notBanned, notUserBanned, scoreboardVis)
	setupWebSocketRoute(router, deps, wrapper, rateLimitCache, sharedCache, ipTracking, notBanned, notUserBanned, scoreboardVis)
	setupFileDownloadRoute(router, server, deps, wrapper, verifyEmails, rateLimitCache, sharedCache, notBanned, notUserBanned, ipTracking)
}

func setupBasicAuthRoutes(
	router chi.Router,
	deps *helper.ServerDeps,
	wrapper openapi.ServerInterfaceWrapper,
	verifyEmails bool,
	rateLimitCache *helper.RateLimitConfigCache,
	sharedCache *cache.Cache,
	ipTracking func(http.Handler) http.Handler,
	notBanned func(http.Handler) http.Handler,
	notUserBanned func(http.Handler) http.Handler,
	scoreboardVis func(http.Handler) http.Handler,
) {
	getter := deps.Admin.SettingsUC
	keyFunc := ipKeyFunc(deps.Infra.TrustedProxyCIDRs)

	profileUpdateLimit := helper.RateLimitFromConfig(
		deps.Infra.RedisClient, rlKeyProfileUpdateIP, time.Minute,
		rateLimitCache, getter,
		func(c *helper.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) },
		keyFunc, deps.Infra.TrustedProxyCIDRs, deps.Infra.Logger,
	)
	apiTokenLimit := helper.RateLimitFromConfig(
		deps.Infra.RedisClient, rlKeyAPITokenIP, time.Minute,
		rateLimitCache, getter,
		func(c *helper.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) },
		keyFunc, deps.Infra.TrustedProxyCIDRs, deps.Infra.Logger,
	)
	notificationLimit := helper.RateLimitFromConfig(
		deps.Infra.RedisClient, rlKeyNotificationIP, time.Minute,
		rateLimitCache, getter,
		func(c *helper.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) },
		keyFunc, deps.Infra.TrustedProxyCIDRs, deps.Infra.Logger,
	)
	protectedReadLimit := helper.RateLimitFromConfig(
		deps.Infra.RedisClient, rlKeyProtectedReadIP, time.Minute,
		rateLimitCache, getter,
		func(c *helper.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) },
		keyFunc, deps.Infra.TrustedProxyCIDRs, deps.Infra.Logger,
	)

	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC, deps.Infra.Logger))
		r.Use(restapimiddleware.InjectUser(deps.User.UserUC, sharedCache, deps.Infra.Logger))
		r.Use(ipTracking)
		r.Use(notUserBanned)

		r.Get("/auth/me", wrapper.GetAuthMe)
		r.With(profileUpdateLimit).Patch("/auth/me", wrapper.PatchAuthMe)
		r.With(protectedReadLimit).Get("/users", wrapper.GetUsers)
		r.Group(func(r2 chi.Router) {
			r2.Use(protectedReadLimit)
			r2.Use(scoreboardVis)
			r2.Get("/users/{ID}", wrapper.GetUsersID)
			r2.Get("/teams/{ID}", wrapper.GetTeamsID)
		})
		r.Group(func(me chi.Router) {
			me.Use(notBanned)
			me.Get("/users/me/solves", wrapper.GetUsersMeSolves)
			me.Get("/users/me/fails", wrapper.GetUsersMeFails)
			me.Get("/users/me/awards", wrapper.GetUsersMeAwards)
			me.Get("/users/me/submissions", wrapper.GetUsersMeSubmissions)
		})

		r.With(notificationLimit).Get("/user/notifications", wrapper.GetUserNotifications)
		r.With(notificationLimit).Patch("/user/notifications/{ID}/read", wrapper.PatchUserNotificationsIDRead)

		r.Group(func(tokens chi.Router) {
			tokens.Use(restapimiddleware.RequireVerified(verifyEmails))
			tokens.Get("/user/tokens", wrapper.GetUserTokens)
			tokens.With(apiTokenLimit).Post("/user/tokens", wrapper.PostUserTokens)
			tokens.With(apiTokenLimit).Delete("/user/tokens/{ID}", wrapper.DeleteUserTokensID)
		})

		setupTeamRoutes(r, wrapper, verifyEmails, deps.Infra.RedisClient, deps.Infra.TrustedProxyCIDRs, deps.Infra.Logger, notBanned, scoreboardVis)
		setupChallengeRoutes(r, wrapper, deps, rateLimitCache, verifyEmails, sharedCache, notBanned)

		fileDownloadLimit := helper.RateLimitFromConfig(
			deps.Infra.RedisClient, rlKeyFileDownloadIP, time.Minute,
			rateLimitCache, getter,
			func(c *helper.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) },
			ipKeyFunc(deps.Infra.TrustedProxyCIDRs), deps.Infra.TrustedProxyCIDRs, deps.Infra.Logger,
		)
		r.With(restapimiddleware.RequireVerified(verifyEmails), notBanned, fileDownloadLimit).Get("/files/by-id/{ID}/download", wrapper.GetFilesIDDownload)

		setupAdminRoutes(r, wrapper, deps.Infra.RedisClient, deps.Infra.TrustedProxyCIDRs, deps.Infra.Logger)
	})
}

func setupScoreboardRoutes(router chi.Router, deps *helper.ServerDeps, wrapper openapi.ServerInterfaceWrapper, rateLimitCache *helper.RateLimitConfigCache, sharedCache *cache.Cache, ipTracking, notBanned, notUserBanned, scoreboardVis func(http.Handler) http.Handler) {
	getter := deps.Admin.SettingsUC
	scoreboardLimit := helper.RateLimitFromConfig(
		deps.Infra.RedisClient, rlKeyScoreboardIP, time.Minute,
		rateLimitCache, getter,
		func(c *helper.RateLimitConfig) int64 { return int64(c.ScoreboardPerMinute) },
		ipKeyFunc(deps.Infra.TrustedProxyCIDRs), deps.Infra.TrustedProxyCIDRs, deps.Infra.Logger,
	)

	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC, deps.Infra.Logger))
		r.Use(restapimiddleware.InjectUser(deps.User.UserUC, sharedCache, deps.Infra.Logger))
		r.Use(ipTracking)
		r.Use(notUserBanned)
		r.Use(notBanned)
		r.Use(scoreboardVis)
		r.Use(scoreboardLimit)

		r.Get("/scoreboard", wrapper.GetScoreboard)
		r.Get("/scoreboard/graph", wrapper.GetScoreboardGraph)
		r.Get("/users/{ID}/solves", wrapper.GetUsersIDSolves)
		r.Get("/users/{ID}/awards", wrapper.GetUsersIDAwards)
		r.Get("/teams/solves/{teamID}", wrapper.GetTeamsIDSolves)
		r.Get("/teams/awards/{teamID}", wrapper.GetTeamsIDAwards)
		r.Get("/users/{ID}/fails", wrapper.GetUsersIDFails)
		r.Get("/teams/fails/{teamID}", wrapper.GetTeamsIDFails)
		r.Get("/statistics/general", wrapper.GetStatisticsGeneral)
		r.Get("/statistics/challenges", wrapper.GetStatisticsChallenges)
		r.Get("/statistics/challenges/{ID}", wrapper.GetStatisticsChallengesID)
		r.Get("/statistics/challenges/solves/percentages", wrapper.GetStatisticsChallengesSolvesPercentages)
		r.Get("/statistics/scores/distribution", wrapper.GetStatisticsScoresDistribution)
		r.Get("/statistics/submissions", wrapper.GetStatisticsSubmissions)
		r.Get("/statistics/submissions/{type}", wrapper.GetStatisticsSubmissionsType)
		r.Get("/statistics/teams", wrapper.GetStatisticsTeams)
		r.Get("/statistics/users", wrapper.GetStatisticsUsers)
		r.Get("/statistics/scoreboard", wrapper.GetStatisticsScoreboard)
	})
}

func setupFirstBloodRoute(router chi.Router, deps *helper.ServerDeps, wrapper openapi.ServerInterfaceWrapper, _ bool, rateLimitCache *helper.RateLimitConfigCache, sharedCache *cache.Cache, ipTracking, notBanned, notUserBanned, scoreboardVis func(http.Handler) http.Handler) {
	challengeReadLimit := helper.RateLimitFromConfig(
		deps.Infra.RedisClient, rlKeyChallengeReadIP, time.Minute,
		rateLimitCache, deps.Admin.SettingsUC,
		func(c *helper.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) },
		ipKeyFunc(deps.Infra.TrustedProxyCIDRs), deps.Infra.TrustedProxyCIDRs, deps.Infra.Logger,
	)
	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC, deps.Infra.Logger))
		r.Use(restapimiddleware.InjectUser(deps.User.UserUC, sharedCache, deps.Infra.Logger))
		r.Use(ipTracking)
		r.Use(notUserBanned)
		r.Use(notBanned)
		r.Use(scoreboardVis)
		r.Use(restapimiddleware.ChallengeVisibility(deps.Comp.CompetitionUC))
		r.Use(challengeReadLimit)

		r.Get("/challenges/{challengeID}/first-blood", wrapper.GetChallengesChallengeIDFirstBlood)
	})
}

func setupWebSocketRoute(router chi.Router, deps *helper.ServerDeps, wrapper openapi.ServerInterfaceWrapper, rateLimitCache *helper.RateLimitConfigCache, sharedCache *cache.Cache, ipTracking, notBanned, notUserBanned, scoreboardVis func(http.Handler) http.Handler) {
	getter := deps.Admin.SettingsUC
	wsLimit := helper.RateLimitFromConfig(
		deps.Infra.RedisClient, rlKeyWebSocketIP, time.Minute,
		rateLimitCache, getter,
		func(c *helper.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) },
		ipKeyFunc(deps.Infra.TrustedProxyCIDRs), deps.Infra.TrustedProxyCIDRs, deps.Infra.Logger,
	)
	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC, deps.Infra.Logger))
		r.Use(restapimiddleware.InjectUser(deps.User.UserUC, sharedCache, deps.Infra.Logger))
		r.Use(ipTracking)
		r.Use(notUserBanned)
		r.Use(notBanned)
		r.Use(scoreboardVis)
		r.Use(wsLimit)

		r.Get("/ws", wrapper.GetWs)
	})
}

func setupFileDownloadRoute(router chi.Router, server *Server, deps *helper.ServerDeps, _ openapi.ServerInterfaceWrapper, verifyEmails bool, rateLimitCache *helper.RateLimitConfigCache, sharedCache *cache.Cache, notBanned, notUserBanned, ipTracking func(http.Handler) http.Handler) {
	fileDownloadLimit := helper.RateLimitFromConfig(
		deps.Infra.RedisClient, rlKeyFileDownloadIP, time.Minute,
		rateLimitCache, deps.Admin.SettingsUC,
		func(c *helper.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) },
		ipKeyFunc(deps.Infra.TrustedProxyCIDRs), deps.Infra.TrustedProxyCIDRs, deps.Infra.Logger,
	)
	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC, deps.Infra.Logger))
		r.Use(restapimiddleware.InjectUser(deps.User.UserUC, sharedCache, deps.Infra.Logger))
		r.Use(ipTracking)
		r.Use(notUserBanned)
		r.Use(restapimiddleware.RequireVerified(verifyEmails))
		r.Use(restapimiddleware.ChallengeVisibility(deps.Comp.CompetitionUC))
		r.Use(notBanned)
		r.Use(fileDownloadLimit)
		r.Get("/files/download/*", func(w http.ResponseWriter, req *http.Request) {
			path := chi.URLParam(req, "*")
			server.GetFilesDownloadPath(w, req, path, openapi.GetFilesDownloadPathParams{Token: req.URL.Query().Get("token")})
		})
	})
}

func setupTeamRoutes(r chi.Router, wrapper openapi.ServerInterfaceWrapper, verifyEmails bool, redisClient *redis.Client, trustedProxyCIDRs []string, log logger.Logger, notBanned, scoreboardVisibility func(http.Handler) http.Handler) {
	r.Group(func(sv chi.Router) {
		sv.Use(scoreboardVisibility)
		sv.Get("/teams", wrapper.GetTeams)
		sv.Get("/teams/my", wrapper.GetTeamsMy)
	})

	r.Group(func(me chi.Router) {
		me.Use(restapimiddleware.RequireTeamOrNotFound())
		me.Use(notBanned)
		me.Get("/teams/me/solves", wrapper.GetTeamsMeSolves)
		me.Get("/teams/me/fails", wrapper.GetTeamsMeFails)
		me.Get("/teams/me/awards", wrapper.GetTeamsMeAwards)
		me.Get("/teams/me/invite", wrapper.GetTeamsMeInvite)
		me.Post("/teams/me/invite", wrapper.PostTeamsMeInvite)
	})

	verified := r.With(restapimiddleware.RequireVerified(verifyEmails), notBanned)
	verified.Patch("/teams/me", wrapper.PatchTeamsMe)
	verified.Post("/teams/leave", wrapper.PostTeamsLeave)
	verified.Delete("/teams/me", wrapper.DeleteTeamsMe)
	verified.Delete("/teams/members/{ID}", wrapper.DeleteTeamsMembersID)
	verified.Post("/teams/transfer-captain", wrapper.PostTeamsTransferCaptain)

	teamOpLimit := restapimiddleware.RateLimit(redisClient, rlKeyTeamOpUser, teamOpRateLimit, teamOpRateLimitWindow, userIDKeyFunc, trustedProxyCIDRs, log)
	verified.With(teamOpLimit).Post("/teams", wrapper.PostTeams)
	verified.With(teamOpLimit).Post("/teams/join", wrapper.PostTeamsJoin)
	verified.With(teamOpLimit).Post("/teams/solo", wrapper.PostTeamsSolo)
}

func setupChallengeRoutes(
	r chi.Router,
	wrapper openapi.ServerInterfaceWrapper,
	deps *helper.ServerDeps,
	rateLimitCache *helper.RateLimitConfigCache,
	verifyEmails bool,
	_ *cache.Cache,
	notBanned func(http.Handler) http.Handler,
) {
	competitionUC := deps.Comp.CompetitionUC
	log := deps.Infra.Logger
	getter := deps.Admin.SettingsUC
	challengeReadLimit := helper.RateLimitFromConfig(
		deps.Infra.RedisClient, rlKeyChallengeReadIP, time.Minute,
		rateLimitCache, getter,
		func(c *helper.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) },
		ipKeyFunc(deps.Infra.TrustedProxyCIDRs), deps.Infra.TrustedProxyCIDRs, log,
	)

	r.Group(func(challenges chi.Router) {
		challenges.Use(challengeReadLimit)
		challenges.Use(restapimiddleware.ChallengeVisibility(competitionUC))
		challenges.Use(notBanned)
		challenges.Get("/challenges", wrapper.GetChallenges)
		challenges.Get("/challenges/solutions", wrapper.GetChallengesSolutions)
		challenges.Get("/challenges/{challengeID}", wrapper.GetChallengesChallengeID)
		challenges.Get("/challenges/{challengeID}/solves", wrapper.GetChallengesChallengeIDSolves)
		challenges.Get("/challenges/{challengeID}/files", wrapper.GetChallengesChallengeIDFiles)
		challenges.Get("/challenges/{challengeID}/hints", wrapper.GetChallengesChallengeIDHints)
		challenges.Get("/challenges/{challengeID}/tags", wrapper.GetChallengesChallengeIDTags)
		challenges.Get("/challenges/{challengeID}/requirements", wrapper.GetChallengesChallengeIDRequirements)
		challenges.Get("/challenges/{challengeID}/solution", wrapper.GetChallengesChallengeIDSolution)
	})

	commentLimit := helper.RateLimitFromConfig(
		deps.Infra.RedisClient, rlKeyCommentUser, time.Minute,
		rateLimitCache, getter,
		func(c *helper.RateLimitConfig) int64 { return int64(c.CommentPerMinute) },
		userIDKeyFunc, deps.Infra.TrustedProxyCIDRs, log,
	)
	r.Group(func(comments chi.Router) {
		comments.Use(restapimiddleware.CompetitionEnded(competitionUC))
		comments.Use(restapimiddleware.RequireVerified(verifyEmails))
		comments.Use(notBanned)
		comments.Use(commentLimit)
		comments.Get("/challenges/{challengeID}/comments", wrapper.GetChallengesChallengeIDComments)
		comments.Post("/challenges/{challengeID}/comments", wrapper.PostChallengesChallengeIDComments)
		comments.Delete("/comments/{ID}", wrapper.DeleteCommentsID)
	})

	submitIPLimit := helper.RateLimitFromConfig(
		deps.Infra.RedisClient, rlKeySubmitIP, time.Minute,
		rateLimitCache, getter,
		func(c *helper.RateLimitConfig) int64 { return int64(c.SubmitIPPerMinute) },
		ipKeyFunc(deps.Infra.TrustedProxyCIDRs), deps.Infra.TrustedProxyCIDRs, log,
	)

	submitUserLimit := helper.RateLimitFromConfig(
		deps.Infra.RedisClient, rlKeySubmitUser, time.Minute,
		rateLimitCache, getter,
		func(c *helper.RateLimitConfig) int64 { return int64(c.SubmitUserPerMinute) },
		userIDKeyFunc, deps.Infra.TrustedProxyCIDRs, log,
	)

	hintUnlockUserLimit := helper.RateLimitFromConfig(
		deps.Infra.RedisClient, rlKeyHintUnlockUser, time.Minute,
		rateLimitCache, getter,
		func(c *helper.RateLimitConfig) int64 { return int64(c.HintUnlockUserPerMinute) },
		userIDKeyFunc, deps.Infra.TrustedProxyCIDRs, log,
	)

	r.Group(func(sub chi.Router) {
		sub.Use(restapimiddleware.CompetitionActive(competitionUC))
		sub.Use(restapimiddleware.RequireVerified(verifyEmails))
		sub.Use(restapimiddleware.RequireTeam())
		sub.Use(notBanned)
		sub.Use(submitIPLimit)
		sub.Use(submitUserLimit)
		sub.Post("/challenges/{challengeID}/submit", wrapper.PostChallengesChallengeIDSubmit)
	})

	// Unlock Hints
	r.Group(func(sub chi.Router) {
		sub.Use(restapimiddleware.CompetitionActive(competitionUC))
		sub.Use(restapimiddleware.RequireVerified(verifyEmails))
		sub.Use(restapimiddleware.RequireTeam())
		sub.Use(notBanned)
		sub.Use(hintUnlockUserLimit)
		sub.Post("/challenges/{challengeID}/hints/{hintID}/unlock", wrapper.PostChallengesChallengeIDHintsHintIDUnlock)
	})
}

func setupAdminRoutes(r chi.Router, wrapper openapi.ServerInterfaceWrapper, redisClient *redis.Client, trustedProxyCIDRs []string, log logger.Logger) {
	adminGeneralLimitMw := restapimiddleware.RateLimit(redisClient, rlKeyAdminGeneral, adminGeneralLimit, adminGeneralWindow, userIDKeyFunc, trustedProxyCIDRs, log)
	r.Group(func(adm chi.Router) {
		adm.Use(restapimiddleware.Admin)
		adm.Use(adminGeneralLimitMw)

		setupAdminConfigRoutes(adm, wrapper)
		setupAdminChallengeRoutes(adm, wrapper)
		setupAdminAwardRoutes(adm, wrapper)
		setupAdminUserRoutes(adm, wrapper)
		setupAdminTeamRoutes(adm, wrapper)
		setupAdminBracketRoutes(adm, wrapper)
		setupAdminTagRoutes(adm, wrapper)
		setupAdminFieldRoutes(adm, wrapper)
		setupAdminPageRoutes(adm, wrapper)
		setupAdminNotificationRoutes(adm, wrapper)
		setupAdminSubmissionRoutes(adm, wrapper)
		setupAdminUtilityRoutes(adm, wrapper, redisClient, trustedProxyCIDRs, log)
	})
}

func setupAdminConfigRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/competition", wrapper.GetAdminCompetition)
	adm.Put("/admin/competition", wrapper.PutAdminCompetition)
	adm.Get("/admin/settings", wrapper.GetAdminSettings)
	adm.Put("/admin/settings", wrapper.PutAdminSettings)
	adm.Get("/admin/configs", wrapper.GetAdminConfigs)
	adm.Get("/admin/configs/categories", wrapper.GetAdminConfigsCategories)
	adm.Get("/admin/configs/category/{category}", wrapper.GetAdminConfigsCategory)
	adm.Put("/admin/configs/batch", wrapper.PutAdminConfigsBatch)
	adm.Get("/admin/configs/{key}", wrapper.GetAdminConfigsKey)
	adm.Put("/admin/configs/{key}", wrapper.PutAdminConfigsKey)
	adm.Delete("/admin/configs/{key}", wrapper.DeleteAdminConfigsKey)
}

func setupAdminChallengeRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Post("/admin/challenges", wrapper.PostAdminChallenges)
	adm.Put("/admin/challenges/{ID}", wrapper.PutAdminChallengesID)
	adm.Delete("/admin/challenges/{ID}", wrapper.DeleteAdminChallengesID)
	adm.Post("/admin/challenges/{challengeID}/files", wrapper.PostAdminChallengesChallengeIDFiles)
	adm.Post("/admin/challenges/{challengeID}/hints", wrapper.PostAdminChallengesChallengeIDHints)
	adm.Get("/admin/challenges/{challengeID}/flags", wrapper.GetAdminChallengesChallengeIDFlags)
	adm.Put("/admin/challenges/{challengeID}/requirements", wrapper.PutAdminChallengesChallengeIDRequirements)
	adm.Post("/admin/challenges/{challengeID}/solution", wrapper.PostAdminChallengesChallengeIDSolution)
	adm.Delete("/admin/challenges/{challengeID}/solution", wrapper.DeleteAdminChallengesChallengeIDSolution)
}

func setupAdminAwardRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/awards", wrapper.GetAdminAwards)
	adm.Post("/admin/awards", wrapper.PostAdminAwards)
	adm.Get("/admin/awards/{ID}", wrapper.GetAdminAwardsID)
	adm.Delete("/admin/awards/{ID}", wrapper.DeleteAdminAwardsID)
	adm.Get("/admin/awards/team/{teamID}", wrapper.GetAdminAwardsTeamTeamID)
}

func setupAdminUserRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/users", wrapper.GetAdminUsers)
	adm.Post("/admin/users", wrapper.PostAdminUsers)
	adm.Patch("/admin/users/{ID}", wrapper.PatchAdminUsersID)
	adm.Delete("/admin/users/{ID}", wrapper.DeleteAdminUsersID)
	adm.Get("/admin/users/{ID}/tracking", wrapper.GetAdminUsersIDTracking)
	adm.Get("/admin/users/{ID}/missing-challenges", wrapper.GetAdminUsersIDMissingChallenges)
	adm.Post("/admin/users/{ID}/ban", wrapper.PostAdminUsersIDBan)
	adm.Delete("/admin/users/{ID}/ban", wrapper.DeleteAdminUsersIDBan)
}

func setupAdminTeamRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/teams", wrapper.GetAdminTeams)
	adm.Patch("/admin/teams/{ID}", wrapper.PatchAdminTeamsID)
	adm.Delete("/admin/teams/{ID}", wrapper.DeleteAdminTeamsID)
	adm.Get("/admin/teams/{ID}/members", wrapper.GetAdminTeamsIDMembers)
	adm.Post("/admin/teams/{ID}/members", wrapper.PostAdminTeamsIDMembers)
	adm.Delete("/admin/teams/{ID}/members/{userID}", wrapper.DeleteAdminTeamsIDMembersUserID)
	adm.Get("/admin/teams/{ID}/missing-challenges", wrapper.GetAdminTeamsIDMissingChallenges)
	adm.Post("/admin/teams/{ID}/ban", wrapper.PostAdminTeamsIDBan)
	adm.Delete("/admin/teams/{ID}/ban", wrapper.DeleteAdminTeamsIDBan)
	adm.Patch("/admin/teams/{ID}/hidden", wrapper.PatchAdminTeamsIDHidden)
	adm.Patch("/admin/teams/{ID}/bracket", wrapper.PatchAdminTeamsIDBracket)
}

func setupAdminBracketRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Post("/admin/brackets", wrapper.PostAdminBrackets)
	adm.Get("/admin/brackets/{ID}", wrapper.GetAdminBracketsID)
	adm.Put("/admin/brackets/{ID}", wrapper.PutAdminBracketsID)
	adm.Delete("/admin/brackets/{ID}", wrapper.DeleteAdminBracketsID)
}

func setupAdminTagRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Post("/admin/tags", wrapper.PostAdminTags)
	adm.Put("/admin/tags/{ID}", wrapper.PutAdminTagsID)
	adm.Delete("/admin/tags/{ID}", wrapper.DeleteAdminTagsID)
}

func setupAdminFieldRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Post("/admin/fields", wrapper.PostAdminFields)
	adm.Put("/admin/fields/{ID}", wrapper.PutAdminFieldsID)
	adm.Delete("/admin/fields/{ID}", wrapper.DeleteAdminFieldsID)
}

func setupAdminPageRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/pages", wrapper.GetAdminPages)
	adm.Post("/admin/pages", wrapper.PostAdminPages)
	adm.Get("/admin/pages/{ID}", wrapper.GetAdminPagesID)
	adm.Put("/admin/pages/{ID}", wrapper.PutAdminPagesID)
	adm.Delete("/admin/pages/{ID}", wrapper.DeleteAdminPagesID)
}

func setupAdminNotificationRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Post("/admin/notifications", wrapper.PostAdminNotifications)
	adm.Post("/admin/notifications/user/{userID}", wrapper.PostAdminNotificationsUserUserID)
	adm.Put("/admin/notifications/{ID}", wrapper.PutAdminNotificationsID)
	adm.Delete("/admin/notifications/{ID}", wrapper.DeleteAdminNotificationsID)
}

func setupAdminSubmissionRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/submissions", wrapper.GetAdminSubmissions)
	adm.Post("/admin/submissions", wrapper.PostAdminSubmissions)
	adm.Get("/admin/submissions/{ID}", wrapper.GetAdminSubmissionsID)
	adm.Patch("/admin/submissions/{ID}", wrapper.PatchAdminSubmissionsID)
	adm.Delete("/admin/submissions/{ID}", wrapper.DeleteAdminSubmissionsID)
	adm.Get("/admin/submissions/challenge/{challengeID}", wrapper.GetAdminSubmissionsChallengeChallengeID)
	adm.Get("/admin/submissions/challenge/{challengeID}/stats", wrapper.GetAdminSubmissionsChallengeChallengeIDStats)
	adm.Get("/admin/submissions/user/{userID}", wrapper.GetAdminSubmissionsUserUserID)
	adm.Get("/admin/submissions/team/{teamID}", wrapper.GetAdminSubmissionsTeamTeamID)
}

func setupAdminUtilityRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper, redisClient *redis.Client, trustedProxyCIDRs []string, log logger.Logger) {
	adm.Put("/admin/hints/{ID}", wrapper.PutAdminHintsID)
	adm.Delete("/admin/hints/{ID}", wrapper.DeleteAdminHintsID)
	adm.Delete("/admin/files/{ID}", wrapper.DeleteAdminFilesID)
	adm.Get("/admin/unlocks", wrapper.GetAdminUnlocks)
	adm.Get("/admin/statistics/solve-matrix", wrapper.GetAdminStatisticsSolveMatrix)
	destructiveLimit := restapimiddleware.RateLimit(redisClient, rlKeyAdminDestructive, adminDestructiveLimit, adminDestructiveWindow, userIDKeyFunc, trustedProxyCIDRs, log)
	adm.With(destructiveLimit).Post("/admin/reset", wrapper.PostAdminReset)
	adm.With(destructiveLimit).Post("/admin/import", wrapper.PostAdminImport)
	adm.With(destructiveLimit).Post("/admin/import/csv", wrapper.PostAdminImportCsv)
	adm.Get("/admin/export", wrapper.GetAdminExport)
	exportZipLimit := restapimiddleware.RateLimit(redisClient, rlKeyAdminExportZip, adminExportZipLimit, adminExportZipWindow, userIDKeyFunc, trustedProxyCIDRs, log)
	adm.With(exportZipLimit).Get("/admin/export/zip", wrapper.GetAdminExportZip)
	adm.Get("/admin/export/csv", wrapper.GetAdminExportCsv)
	adm.Get("/debug", wrapper.GetDebug)
}
