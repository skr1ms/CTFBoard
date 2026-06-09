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

	requestTimeout     = 60 * time.Second
	longRequestTimeout = 10 * time.Minute
	rateLimitCacheTTL  = 30 * time.Second

	httpReadHeaderTimeout = 15 * time.Second
	httpWriteTimeout      = 10 * time.Minute
	httpIdleTimeout       = time.Minute

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
	mu             sync.RWMutex
	rateLimitCache *restapimiddleware.RateLimitConfigCache
}

type teamBracketIDGetter struct {
	r repo.TeamRepository
}
