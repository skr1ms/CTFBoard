package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"

	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

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
		r.With(publicReadLimit, restapimiddleware.ETag).Get("/auth/oauth/providers", wrapper.GetAuthOauthProviders)
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
			pub.Get("/notifications/count", wrapper.GetNotificationsCount)
			pub.Get("/notifications", wrapper.GetNotifications)
			pub.Get("/challenges/types", wrapper.GetChallengesTypes)
		})

		// Non-cacheable public endpoints (no ETag, still rate-limited).
		r.With(publicReadLimit).Get("/healthcheck", wrapper.GetHealthcheck)
		r.With(publicReadLimit).Get("/robots.txt", wrapper.GetRobotsTxt)
		r.With(publicReadLimit).Get("/tos", wrapper.GetTos)
		r.With(publicReadLimit).Get("/privacy", wrapper.GetPrivacy)
		r.With(publicReadLimit).Get("/configs/public", wrapper.GetConfigsPublic)
		r.With(publicReadLimit).Get("/shares/solve", wrapper.GetSharesSolve)
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
