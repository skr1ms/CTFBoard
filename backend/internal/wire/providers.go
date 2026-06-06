package wire

import (
	"context"
	"sync"
	"time"

	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

const (
	rlKeyForgot   = "forgot"
	rlKeyResend   = "resend"
	rlKeyResetTok = "reset-token"
	rlKeyGeneral  = "general:ip"

	requestTimeout    = 60 * time.Second
	rateLimitCacheTTL = 30 * time.Second

	httpReadTimeout  = 15 * time.Second
	httpWriteTimeout = 100 * time.Second
	httpIdleTimeout  = time.Minute

	loginLockoutMaxAttempts      = 5
	loginLockoutTTL              = time.Minute
	forgotPasswordRateLimit      = 10
	resendVerificationRateLimit  = 10
	resetPasswordTokenRateLimit  = 5
	resetPasswordTokenRateWindow = time.Minute
	perKeyRateLimitWindow        = 24 * time.Hour
	corsPreflightMaxAgeSeconds   = 300
)

type healthCheckerFunc func(context.Context) error

type runtimeSettingsInvalidator struct {
	mu                        sync.RWMutex
	rateLimitCache            *restapimiddleware.RateLimitConfigCache
	scoreboardVisibilityCache *restapimiddleware.ScoreboardVisibilityCache
}

type teamBracketIDGetter struct {
	r repo.TeamRepository
}
