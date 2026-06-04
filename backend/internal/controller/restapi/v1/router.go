package v1

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-cachekit"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
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
	rlKeyRatingUser        = "rating:user"
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
	rlKeyAvatarUploadUser  = "avatar:upload:user"
	defaultRLWindow        = time.Minute
	avatarUploadLimit      = 2
	avatarUploadWindow     = time.Minute
	teamOpRateLimit        = 5
	teamOpRateLimitWindow  = time.Minute
	adminExportZipLimit    = 3
	adminExportZipWindow   = time.Minute
	adminDestructiveLimit  = 3
	adminDestructiveWindow = time.Minute
	adminGeneralLimit      = 600
	adminGeneralWindow     = time.Minute
)

// dynamicRL is a factory that closes over the four shared rate-limit dependencies
// so each call site only needs to supply the per-endpoint key, window, field
// selector, and key function.
type dynamicRL func(key string, window time.Duration, field func(*restapimiddleware.RateLimitConfig) int64, keyFunc func(*http.Request) (string, error)) func(http.Handler) http.Handler

func newDynamicRL(
	redisClient *redis.Client,
	rateLimitCache *restapimiddleware.RateLimitConfigCache,
	getter restapimiddleware.SettingsGetter,
	logger logkit.Logger,
) dynamicRL {
	return func(key string, window time.Duration, field func(*restapimiddleware.RateLimitConfig) int64, keyFunc func(*http.Request) (string, error)) func(http.Handler) http.Handler {
		return restapimiddleware.DynamicRateLimit(redisClient, key, window, rateLimitCache, getter, field, keyFunc, logger)
	}
}

// ipKeyFunc returns a rate-limit key function that uses the client IP address from context.
func ipKeyFunc() func(*http.Request) (string, error) {
	return func(r *http.Request) (string, error) {
		return kitMiddleware.GetClientIPFromContext(r.Context()), nil
	}
}

// userIDKeyFunc returns a rate-limit key function that uses the authenticated user's ID.
func userIDKeyFunc(r *http.Request) (string, error) {
	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		return "", apperr.ErrNotAuthenticated
	}

	return user.ID.String(), nil
}

// protectedMiddlewareStack returns the shared middleware chain used by all
// authenticated route groups that require IP tracking:
// Auth -> InjectUser -> ipTracking -> notUserBanned.
func protectedMiddlewareStack(
	deps *helper.ServerDeps,
	sharedCache *cachekit.Cache,
	ipTracking func(http.Handler) http.Handler,
	notUserBanned func(http.Handler) http.Handler,
) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		restapimiddleware.Auth(deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC, deps.Infra.Logger),
		restapimiddleware.InjectUser(deps.User.UserUC, sharedCache, deps.Infra.Logger),
		ipTracking,
		notUserBanned,
	}
}

func scoreboardVisibilityMiddleware(deps *helper.ServerDeps) func(http.Handler) http.Handler {
	return restapimiddleware.VisibilityGuard(deps.Admin.CompetitionParamUC, "score_visibility")
}

// NewRouter wires the entire HTTP route tree onto router. It creates the Server
// and the OpenAPI wrapper (with typed error handlers for missing/invalid params),
// builds shared infrastructure (cachekit, IP tracking, ban middleware, scoreboard
// visibility), then delegates to the four route-group setup functions:
// setupPublicRoutes, setupAuthOnlyRoutes, setupProtectedRoutes, and (transitively)
// setupScoreboardRoutes / setupFileDownloadRoute / setupWebSocketRoute.
func NewRouter(
	ctx context.Context,
	router chi.Router,
	deps *helper.ServerDeps,
	verifyEmails bool,
	rateLimitCache *restapimiddleware.RateLimitConfigCache,
) {
	server := NewServer(deps)
	wrapper := openapi.ServerInterfaceWrapper{
		Handler: server,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			var requiredHeader *openapi.RequiredHeaderError
			if errors.As(err, &requiredHeader) {
				server.OnError(w, r, httperr.New(err, http.StatusUnauthorized, "UNAUTHORIZED"), "OpenAPI", "RequiredHeader")

				return
			}

			var requiredParam *openapi.RequiredParamError
			if errors.As(err, &requiredParam) {
				server.OnError(w, r, httperr.New(err, http.StatusBadRequest, "BAD_REQUEST"), "OpenAPI", "RequiredParam")

				return
			}

			var invalidParam *openapi.InvalidParamFormatError
			if errors.As(err, &invalidParam) {
				server.OnError(w, r, httperr.New(err, http.StatusBadRequest, "BAD_REQUEST"), "OpenAPI", "InvalidParamFormat")

				return
			}

			server.OnError(w, r, httperr.New(errors.New("invalid request parameter"), http.StatusBadRequest, "BAD_REQUEST"), "OpenAPI", "BadRequest")
		},
	}
	sharedCache := cachekit.New(deps.Infra.RedisClient)
	ipTracking := restapimiddleware.IPTracking(ctx, deps.User.TrackingUC, deps.Infra.Logger)
	notUserBanned := restapimiddleware.RequireUserNotBanned()
	scoreboardVis := scoreboardVisibilityMiddleware(deps)
	setupPublicRoutes(router, wrapper, deps, deps.Infra.RedisClient, deps.Infra.Logger, rateLimitCache)
	router.Get("/auth/oauth/providers", server.GetAuthOauthProviders)
	setupAuthOnlyRoutes(router, deps, wrapper, sharedCache, rateLimitCache, notUserBanned)
	setupProtectedRoutes(router, server, deps, wrapper, verifyEmails, rateLimitCache, sharedCache, ipTracking, notUserBanned, scoreboardVis)
}

// setupPublicRoutes registers all unauthenticated routes with per-endpoint dynamic
// rate limiters sourced from the settings use-case. Each limiter uses the client
// IP as its key. Routes are grouped into: auth endpoints (login, register, verify,
// forgot/reset password, refresh, logout, OAuth) and public read endpoints (competition
// status, tags, fields, brackets, pages, notifications, challenge types, healthcheck,
// robots.txt, ToS, privacy, and public configs), all rate-limited at GeneralIPPerMinute.
func setupPublicRoutes(router chi.Router, wrapper openapi.ServerInterfaceWrapper, deps *helper.ServerDeps, redisClient *redis.Client, logger logkit.Logger, rateLimitCache *restapimiddleware.RateLimitConfigCache) {
	rl := newDynamicRL(redisClient, rateLimitCache, deps.Admin.SettingsUC, logger)
	keyFunc := ipKeyFunc()

	loginLimit := rl(rlKeyLoginIP, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.LoginPerMinute) }, keyFunc)
	registerLimit := rl(rlKeyRegisterIP, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.RegisterPerMinute) }, keyFunc)
	forgotPasswordLimit := rl(rlKeyForgotIP, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.ForgotPasswordPerMinute) }, keyFunc)
	resetPasswordLimit := rl(rlKeyResetIP, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.ResetPasswordPerMinute) }, keyFunc)
	logoutLimit := rl(rlKeyLogoutIP, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.LogoutPerMinute) }, keyFunc)
	refreshLimit := rl(rlKeyRefreshIP, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.RefreshPerMinute) }, keyFunc)

	verifyEmailLimit := rl(rlKeyVerifyEmailIP, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.VerifyEmailPerMinute) }, keyFunc)
	oauthCallbackLimit := rl(rlKeyOAuthCallbackIP, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.OAuthCallbackPerMinute) }, keyFunc)
	oauthRedirectLimit := rl(rlKeyOAuthRedirectIP, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.OAuthRedirectPerMinute) }, keyFunc)
	publicReadLimit := rl(rlKeyPublicReadIP, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) }, keyFunc)

	router.Group(func(r chi.Router) {
		// Auth endpoints with rate limiting
		r.With(loginLimit).Post("/auth/login", wrapper.PostAuthLogin)
		r.With(registerLimit).Post("/auth/register", wrapper.PostAuthRegister)
		r.With(verifyEmailLimit).Post("/auth/verify-email", wrapper.PostAuthVerifyEmail)
		r.With(verifyEmailLimit).Post("/auth/resend-verification-by-email", wrapper.PostAuthResendVerificationByEmail)
		r.With(forgotPasswordLimit).Post("/auth/forgot-password", wrapper.PostAuthForgotPassword)
		r.With(resetPasswordLimit).Post("/auth/reset-password", wrapper.PostAuthResetPassword)
		r.With(refreshLimit).Post("/auth/refresh", wrapper.PostAuthRefresh)
		r.With(logoutLimit).Post("/auth/logout", wrapper.PostAuthLogout)

		// OAuth endpoints
		r.With(oauthRedirectLimit).Get("/auth/oauth/{provider}", wrapper.GetAuthOauthProvider)
		r.With(oauthCallbackLimit).Get("/auth/oauth/{provider}/callback", wrapper.GetAuthOauthProviderCallback)
		r.With(oauthCallbackLimit).Post("/auth/oauth/exchange", wrapper.PostAuthOauthExchange)

		// Public cacheable endpoints: rate-limited + ETag conditional GET support.
		r.Group(func(pub chi.Router) {
			pub.Use(publicReadLimit, restapimiddleware.ETag)
			pub.Get("/competition/status", wrapper.GetCompetitionStatus)
			pub.Get("/tags", wrapper.GetTags)
			pub.Get("/fields", wrapper.GetFields)
			pub.Get("/brackets", wrapper.GetBrackets)
			pub.Get("/pages", wrapper.GetPages)
			pub.Get("/pages/{slug}", wrapper.GetPagesSlug)
			pub.Get("/notifications", wrapper.GetNotifications)
			pub.Get("/challenges/types", wrapper.GetChallengesTypes)
		})

		// Non-cacheable public endpoints (no ETag, still rate-limited).
		r.With(publicReadLimit).Get("/healthcheck", wrapper.GetHealthcheck)
		r.With(publicReadLimit).Get("/robots.txt", wrapper.GetRobotsTxt)
		r.With(publicReadLimit).Get("/tos", wrapper.GetTos)
		r.With(publicReadLimit).Get("/privacy", wrapper.GetPrivacy)
		r.With(publicReadLimit).Get("/configs/public", wrapper.GetConfigsPublic)
	})
}

func setupAuthOnlyRoutes(router chi.Router, deps *helper.ServerDeps, wrapper openapi.ServerInterfaceWrapper, sharedCache *cachekit.Cache, rateLimitCache *restapimiddleware.RateLimitConfigCache, notUserBanned func(http.Handler) http.Handler) {
	rl := newDynamicRL(deps.Infra.RedisClient, rateLimitCache, deps.Admin.SettingsUC, deps.Infra.Logger)
	resendVerificationLimit := rl(rlKeyResendVerifyIP, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.VerifyEmailPerMinute) }, ipKeyFunc())

	router.Group(func(r chi.Router) {
		r.Use(restapimiddleware.Auth(deps.Infra.JWTService, deps.User.APITokenUC, deps.User.UserUC, deps.Infra.Logger))
		r.Use(restapimiddleware.InjectUser(deps.User.UserUC, sharedCache, deps.Infra.Logger))
		r.Use(notUserBanned)

		r.With(resendVerificationLimit).Post("/auth/resend-verification", wrapper.PostAuthResendVerification)
	})
}

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
	notBanned := restapimiddleware.RequireTeamNotBanned(deps.Team.TeamUC, sharedCache)
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

	router.Group(func(r chi.Router) {
		r.Use(protectedMiddlewareStack(deps, sharedCache, ipTracking, notUserBanned)...)

		r.With(profileUpdateLimit).Patch("/auth/me", wrapper.PatchAuthMe)

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

		avatarUploadLimitMw := restapimiddleware.RateLimit(deps.Infra.RedisClient, rlKeyAvatarUploadUser, avatarUploadLimit, avatarUploadWindow, userIDKeyFunc, deps.Infra.Logger)
		verified := r.With(restapimiddleware.RequireVerified(verifyEmails), notBanned)
		verified.With(avatarUploadLimitMw).Put("/users/me/avatar", wrapper.PutUsersMeAvatar)
		verified.Delete("/users/me/avatar", wrapper.DeleteUsersMeAvatar)
		verified.With(avatarUploadLimitMw).Put("/teams/me/avatar", wrapper.PutTeamsMeAvatar)
		verified.Delete("/teams/me/avatar", wrapper.DeleteTeamsMeAvatar)

		setupTeamRoutes(r, wrapper, verifyEmails, deps.Infra.RedisClient, deps.Infra.Logger, notBanned)
		setupChallengeRoutes(r, wrapper, deps, rateLimitCache, verifyEmails, sharedCache, notBanned)

		fileDownloadLimit := rl(rlKeyFileDownloadIP, defaultRLWindow, generalIP, ipKeyFunc())
		r.With(restapimiddleware.RequireVerified(verifyEmails), notBanned, fileDownloadLimit).Get("/files/by-id/{ID}/download", wrapper.GetFilesIDDownload)

		setupAdminRoutes(r, wrapper, deps.Infra.RedisClient, deps.Infra.Logger)
	})
}

// setupConditionalPublicRoutes registers routes whose accessibility depends on
// visibility config keys. An optional auth stack (OptionalAuth + OptionalInjectUser)
// authenticates requests when credentials are present but lets guests through.
// VisibilityGuard then enforces the per-section config key:
//
//	score_visibility    -> /scoreboard, /statistics, per-user/team/challenge solves/fails/awards
//	account_visibility  -> /users, /teams (public read)
//	challenge_visibility -> /challenges (read-only metadata and files)
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
	generalIP := func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) }

	scoreboardLimit := rl(rlKeyScoreboardIP, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.ScoreboardPerMinute) }, keyFunc)
	protectedReadLimit := rl(rlKeyProtectedReadIP, defaultRLWindow, generalIP, keyFunc)
	challengeReadLimit := rl(rlKeyChallengeReadIP, defaultRLWindow, generalIP, keyFunc)

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

		// User and team profiles gated by account_visibility
		r.Group(func(acc chi.Router) {
			acc.Use(notBanned)
			acc.Use(restapimiddleware.VisibilityGuard(compParamUC, "account_visibility"))
			acc.Use(protectedReadLimit)
			acc.Get("/users", wrapper.GetUsers)
			acc.Get("/users/{ID}", wrapper.GetUsersID)
			acc.Get("/teams", wrapper.GetTeams)
			acc.Get("/teams/{ID}", wrapper.GetTeamsID)
		})

		// Challenge read-only endpoints gated by challenge_visibility
		r.Group(func(ch chi.Router) {
			ch.Use(notBanned)
			ch.Use(restapimiddleware.VisibilityGuard(compParamUC, "challenge_visibility"))
			ch.Use(challengeReadLimit)
			ch.Get("/challenges", wrapper.GetChallenges)
			ch.Get("/challenges/solutions", wrapper.GetChallengesSolutions)
			ch.Get("/challenges/{challengeID}", wrapper.GetChallengesChallengeID)
			ch.Get("/challenges/{challengeID}/files", wrapper.GetChallengesChallengeIDFiles)
			ch.Get("/challenges/{challengeID}/hints", wrapper.GetChallengesChallengeIDHints)
			ch.Get("/challenges/{challengeID}/tags", wrapper.GetChallengesChallengeIDTags)
			ch.Get("/challenges/{challengeID}/requirements", wrapper.GetChallengesChallengeIDRequirements)
			ch.Get("/challenges/{challengeID}/solution", wrapper.GetChallengesChallengeIDSolution)
		})
	})
}

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

	router.Group(func(r chi.Router) {
		r.Use(protectedMiddlewareStack(deps, sharedCache, ipTracking, notUserBanned)...)
		r.Use(restapimiddleware.RequireVerified(verifyEmails))
		r.Use(restapimiddleware.ChallengeVisibility(deps.Comp.CompetitionUC))
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

func setupTeamRoutes(r chi.Router, wrapper openapi.ServerInterfaceWrapper, verifyEmails bool, redisClient *redis.Client, log logkit.Logger, notBanned func(http.Handler) http.Handler) {
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

	verified := r.With(restapimiddleware.RequireVerified(verifyEmails), notBanned)
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

// setupChallengeRoutes wires auth-required challenge routes:
// (1) comment/rating endpoints gated by CompetitionEnded / RequireVerified;
// (2) submit and hint-unlock behind CompetitionActive + RequireTeam +
// SubmitRateLimitWithAudit (dual IP+user rate limit with async audit log on excess).
// Read-only challenge endpoints live in setupConditionalPublicRoutes.
func setupChallengeRoutes(
	r chi.Router,
	wrapper openapi.ServerInterfaceWrapper,
	deps *helper.ServerDeps,
	rateLimitCache *restapimiddleware.RateLimitConfigCache,
	verifyEmails bool,
	_ *cachekit.Cache,
	notBanned func(http.Handler) http.Handler,
) {
	competitionUC := deps.Comp.CompetitionUC
	log := deps.Infra.Logger
	getter := deps.Admin.SettingsUC
	rl := newDynamicRL(deps.Infra.RedisClient, rateLimitCache, getter, log)

	commentLimit := rl(rlKeyCommentUser, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.CommentPerMinute) }, userIDKeyFunc)
	ratingLimit := rl(rlKeyRatingUser, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.RatingPerMinute) }, userIDKeyFunc)

	r.Group(func(comments chi.Router) {
		comments.Use(restapimiddleware.CompetitionEnded(competitionUC))
		comments.Use(restapimiddleware.RequireVerified(verifyEmails))
		comments.Use(notBanned)
		comments.Use(commentLimit)
		comments.Get("/challenges/{challengeID}/comments", wrapper.GetChallengesChallengeIDComments)
		comments.Post("/challenges/{challengeID}/comments", wrapper.PostChallengesChallengeIDComments)
		comments.Delete("/comments/{ID}", wrapper.DeleteCommentsID)
	})

	r.Group(func(ratings chi.Router) {
		ratings.Use(restapimiddleware.ChallengeVisibility(competitionUC))
		ratings.Use(restapimiddleware.RequireVerified(verifyEmails))
		ratings.Use(notBanned)
		ratings.Use(ratingLimit)
		ratings.Get("/challenges/{challengeID}/ratings", wrapper.GetChallengesChallengeIDRatings)
		ratings.Put("/challenges/{challengeID}/rating", wrapper.PutChallengesChallengeIDRating)
	})

	submitLimitWithAudit := restapimiddleware.SubmitRateLimitWithAudit(
		deps.Infra.RedisClient, rlKeySubmitIP, rlKeySubmitUser, defaultRLWindow,
		rateLimitCache, getter,
		ipKeyFunc(), userIDKeyFunc,
		log,
		deps.Comp.SubmissionUC,
		deps.Infra.RatelimitAuditWG,
	)

	hintUnlockUserLimit := rl(rlKeyHintUnlockUser, defaultRLWindow, func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.HintUnlockUserPerMinute) }, userIDKeyFunc)

	r.Group(func(sub chi.Router) {
		sub.Use(restapimiddleware.CompetitionActive(competitionUC))
		sub.Use(restapimiddleware.RequireVerified(verifyEmails))
		sub.Use(restapimiddleware.RequireTeam())
		sub.Use(notBanned)
		sub.Use(submitLimitWithAudit)
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

// setupAdminRoutes applies the Admin role-check middleware and a shared per-user
// general rate limiter (adminGeneralLimit / adminGeneralWindow) to the admin
// subtree, then delegates to 11 sub-group helpers covering: configs, challenges,
// awards, users, teams, brackets, tags, fields, pages, notifications, and utility
// (export, import, reset, debug) - the last group adds its own tighter destructive
// and export-zip rate limits.
func setupAdminRoutes(r chi.Router, wrapper openapi.ServerInterfaceWrapper, redisClient *redis.Client, log logkit.Logger) {
	adminGeneralLimitMw := restapimiddleware.RateLimit(redisClient, rlKeyAdminGeneral, adminGeneralLimit, adminGeneralWindow, userIDKeyFunc, log)

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
		setupAdminAppealRoutes(adm, wrapper)
		setupAdminStorageRoutes(adm, wrapper)
		setupAdminUtilityRoutes(adm, wrapper, redisClient, log)
	})
}

func setupAdminAppealRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/appeals", wrapper.GetAdminAppeals)
	adm.Patch("/admin/appeals/{ID}", wrapper.PatchAdminAppealsID)
}

func setupAdminStorageRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper) {
	adm.Get("/admin/storage", wrapper.GetAdminStorage)
	adm.Delete("/admin/storage/{path}", wrapper.DeleteAdminStoragePath)
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
	adm.Post("/admin/challenges/recalc-points", wrapper.PostAdminChallengesRecalcPoints)
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
	adm.Put("/admin/users/{ID}/avatar", wrapper.PutAdminUsersIDAvatar)
	adm.Delete("/admin/users/{ID}/avatar", wrapper.DeleteAdminUsersIDAvatar)
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
	adm.Put("/admin/teams/{ID}/avatar", wrapper.PutAdminTeamsIDAvatar)
	adm.Delete("/admin/teams/{ID}/avatar", wrapper.DeleteAdminTeamsIDAvatar)
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

func setupAdminUtilityRoutes(adm chi.Router, wrapper openapi.ServerInterfaceWrapper, redisClient *redis.Client, log logkit.Logger) {
	adm.Put("/admin/hints/{ID}", wrapper.PutAdminHintsID)
	adm.Delete("/admin/hints/{ID}", wrapper.DeleteAdminHintsID)
	adm.Delete("/admin/files/{ID}", wrapper.DeleteAdminFilesID)
	adm.Get("/admin/unlocks", wrapper.GetAdminUnlocks)
	adm.Get("/admin/statistics/solve-matrix", wrapper.GetAdminStatisticsSolveMatrix)

	destructiveLimit := restapimiddleware.RateLimit(redisClient, rlKeyAdminDestructive, adminDestructiveLimit, adminDestructiveWindow, userIDKeyFunc, log)
	adm.With(destructiveLimit).Post("/admin/reset", wrapper.PostAdminReset)
	adm.With(destructiveLimit).Post("/admin/import", wrapper.PostAdminImport)
	adm.With(destructiveLimit).Post("/admin/import/csv", wrapper.PostAdminImportCsv)
	adm.Get("/admin/export", wrapper.GetAdminExport)

	exportZipLimit := restapimiddleware.RateLimit(redisClient, rlKeyAdminExportZip, adminExportZipLimit, adminExportZipWindow, userIDKeyFunc, log)
	adm.With(exportZipLimit).Get("/admin/export/zip", wrapper.GetAdminExportZip)
	adm.Get("/admin/export/csv", wrapper.GetAdminExportCsv)
	adm.Get("/debug", wrapper.GetDebug)
}
