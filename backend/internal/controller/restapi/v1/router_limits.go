package v1

import (
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"

	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
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
		return helper.ClientIP(r), nil
	}
}

// userIDKeyFunc returns a rate-limit key function that uses the authenticated user's ID.
func userIDKeyFunc(r *http.Request) (string, error) {
	user, ok := helper.CurrentUser(r)
	if !ok {
		return "", helper.ErrNotAuthenticated
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
