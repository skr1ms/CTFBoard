package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/wahrwelt-kit/go-cachekit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

const (
	DynamicLimitKeyPrefix       = "limiter:dynamic:"
	rateLimitMetricLabelLimiter = "limiter"
	rateLimitLogFieldKeyPrefix  = "key_prefix"
	rateLimitLogFieldLimiter    = "limiter"
	defaultLoginPerMin          = 10
	defaultRegisterPerMin       = 5
	defaultForgotPasswordPerMin = 3
	defaultResetPasswordPerMin  = 5
	defaultLogoutPerMin         = 10
	defaultRefreshPerMin        = 10
	defaultScoreboardPerMin     = 30
	defaultGeneralIPPerMin      = 100
	defaultVerifyEmailPerMin    = 10
	defaultOAuthCallbackPerMin  = 20
	defaultOAuthRedirectPerMin  = 20
	defaultHintUnlockMinFloor   = 30
	defaultSubmitPerUser        = 10
	defaultSubmitDurationMin    = 1
	submitIPMultiplier          = 3
	hintUnlockMultiplier        = 3
	defaultCommentPerMinute     = 30
	defaultRatingPerMinute      = 30
	DynamicLimitFallback        = 10
)

var defaultFallbackConfig = &RateLimitConfig{
	LoginPerMinute:          defaultLoginPerMin,
	RegisterPerMinute:       defaultRegisterPerMin,
	ForgotPasswordPerMinute: defaultForgotPasswordPerMin,
	ResetPasswordPerMinute:  defaultResetPasswordPerMin,
	LogoutPerMinute:         defaultLogoutPerMin,
	RefreshPerMinute:        defaultRefreshPerMin,
	ScoreboardPerMinute:     defaultScoreboardPerMin,
	GeneralIPPerMinute:      defaultGeneralIPPerMin,
	VerifyEmailPerMinute:    defaultVerifyEmailPerMin,
	OAuthCallbackPerMinute:  defaultOAuthCallbackPerMin,
	OAuthRedirectPerMinute:  defaultOAuthRedirectPerMin,
	SubmitUserPerMinute:     defaultSubmitPerUser,
	SubmitIPPerMinute:       defaultSubmitPerUser * submitIPMultiplier,
	HintUnlockUserPerMinute: defaultSubmitPerUser * hintUnlockMultiplier,
	CommentPerMinute:        defaultCommentPerMinute,
	RatingPerMinute:         defaultRatingPerMinute,
}

// SettingsGetter is the minimal interface required by rate-limit middleware to read dynamic settings.
type SettingsGetter interface {
	Get(ctx context.Context) (*domain.Settings, error)
}

// RateLimitConfig holds per-endpoint rate limits (requests per minute) derived from dynamic settings.
type RateLimitConfig struct {
	LoginPerMinute          int
	RegisterPerMinute       int
	ForgotPasswordPerMinute int
	ResetPasswordPerMinute  int
	LogoutPerMinute         int
	RefreshPerMinute        int
	ScoreboardPerMinute     int
	GeneralIPPerMinute      int
	VerifyEmailPerMinute    int
	OAuthCallbackPerMinute  int
	OAuthRedirectPerMinute  int
	SubmitUserPerMinute     int
	SubmitIPPerMinute       int
	HintUnlockUserPerMinute int
	CommentPerMinute        int
	RatingPerMinute         int
}

const rateLimitConfigCacheKey = "rate_limit_config"

// RateLimitConfigCache is a short-lived in-process cache for RateLimitConfig
// so settings are not fetched from Redis/DB on every request.
type RateLimitConfigCache struct {
	cv *cachekit.CachedValue[*RateLimitConfig]
}

// NewRateLimitConfigCache creates a RateLimitConfigCache with the given TTL.
func NewRateLimitConfigCache(ctx context.Context, ttl time.Duration) *RateLimitConfigCache {
	if ctx == nil {
		panic("middleware.NewRateLimitConfigCache: nil context")
	}

	return &RateLimitConfigCache{
		cv: cachekit.NewCachedValue[*RateLimitConfig](ctx, rateLimitConfigCacheKey, ttl),
	}
}

// Get returns a cached RateLimitConfig, fetching from settings via getter on cache miss.
func (c *RateLimitConfigCache) Get(ctx context.Context, getter SettingsGetter) (*RateLimitConfig, error) {
	return c.cv.Get(ctx, func(ctx context.Context) (*RateLimitConfig, error) {
		return GetRateLimitConfig(ctx, getter)
	})
}

// GetStale returns the last successfully fetched config without triggering a refresh, or nil if none.
func (c *RateLimitConfigCache) GetStale() *RateLimitConfig {
	cfg, ok := c.cv.GetStale()
	if !ok {
		return nil
	}

	return cfg
}

// Invalidate clears the cached config so the next Get fetches fresh values from settings.
func (c *RateLimitConfigCache) Invalidate() {
	c.cv.Invalidate()
}

func rateLimitDefault(value, defaultValue int) int {
	if v := max(value, 0); v != 0 {
		return v
	}

	return defaultValue
}

// GetRateLimitConfig builds a RateLimitConfig from dynamic settings, applying hardcoded defaults
// for any zero or missing values. Submit limits are derived from the per-user quota and duration.
func GetRateLimitConfig(ctx context.Context, getter SettingsGetter) (*RateLimitConfig, error) {
	settings, err := getter.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("RateLimitConfigCache - Get: %w", err)
	}

	submitUser := rateLimitDefault(settings.SubmitLimitPerUser, defaultSubmitPerUser)
	durMin := rateLimitDefault(settings.SubmitLimitDurationMin, defaultSubmitDurationMin)

	submitUserPerMin := submitUser / durMin
	if submitUserPerMin <= 0 {
		submitUserPerMin = 1
	}

	hintUnlockUserPerMin := max(submitUserPerMin*hintUnlockMultiplier, defaultHintUnlockMinFloor)

	return &RateLimitConfig{
		LoginPerMinute:          rateLimitDefault(settings.RateLimitLoginPerMinute, defaultLoginPerMin),
		RegisterPerMinute:       rateLimitDefault(settings.RateLimitRegisterPerMinute, defaultRegisterPerMin),
		ForgotPasswordPerMinute: rateLimitDefault(settings.RateLimitForgotPasswordPerMinute, defaultForgotPasswordPerMin),
		ResetPasswordPerMinute:  rateLimitDefault(settings.RateLimitResetPasswordPerMinute, defaultResetPasswordPerMin),
		LogoutPerMinute:         rateLimitDefault(settings.RateLimitLogoutPerMinute, defaultLogoutPerMin),
		RefreshPerMinute:        rateLimitDefault(settings.RateLimitRefreshPerMinute, defaultRefreshPerMin),
		ScoreboardPerMinute:     rateLimitDefault(settings.RateLimitScoreboardPerMinute, defaultScoreboardPerMin),
		GeneralIPPerMinute:      rateLimitDefault(settings.RateLimitGeneralIPPerMinute, defaultGeneralIPPerMin),
		VerifyEmailPerMinute:    rateLimitDefault(settings.RateLimitVerifyEmailPerMinute, defaultVerifyEmailPerMin),
		OAuthCallbackPerMinute:  rateLimitDefault(settings.RateLimitOAuthCallbackPerMinute, defaultOAuthCallbackPerMin),
		OAuthRedirectPerMinute:  rateLimitDefault(settings.RateLimitOAuthRedirectPerMinute, defaultOAuthRedirectPerMin),
		SubmitUserPerMinute:     submitUserPerMin,
		SubmitIPPerMinute:       submitUserPerMin * submitIPMultiplier,
		HintUnlockUserPerMinute: hintUnlockUserPerMin,
		CommentPerMinute:        rateLimitDefault(settings.RateLimitCommentPerMinute, defaultCommentPerMinute),
		RatingPerMinute:         defaultRatingPerMinute,
	}, nil
}
