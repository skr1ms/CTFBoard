package helper

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
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
)

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
}

type RateLimitConfigCache struct {
	mu        sync.RWMutex
	sf        singleflight.Group
	cfg       *RateLimitConfig
	fetchedAt time.Time
	ttl       time.Duration
}

const rateLimitConfigCacheKey = "rate_limit_config"

func NewRateLimitConfigCache(ttl time.Duration) *RateLimitConfigCache {
	return &RateLimitConfigCache{ttl: ttl}
}

func (c *RateLimitConfigCache) Get(ctx context.Context, getter SettingsGetter) (*RateLimitConfig, error) {
	c.mu.RLock()
	if c.cfg != nil && time.Since(c.fetchedAt) < c.ttl {
		cfg := c.cfg
		c.mu.RUnlock()
		return cfg, nil
	}
	c.mu.RUnlock()

	v, err, _ := c.sf.Do(rateLimitConfigCacheKey, func() (any, error) {
		cfg, err := GetRateLimitConfig(context.WithoutCancel(ctx), getter)
		if err != nil {
			return nil, err
		}
		c.mu.Lock()
		c.cfg = cfg
		c.fetchedAt = time.Now()
		c.mu.Unlock()
		return cfg, nil
	})
	if err != nil {
		return nil, err
	}
	cfg, ok := v.(*RateLimitConfig)
	if !ok {
		return nil, fmt.Errorf("rate limit config cache: unexpected type %T", v)
	}
	return cfg, nil
}

func (c *RateLimitConfigCache) GetStale() *RateLimitConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cfg
}

func (c *RateLimitConfigCache) Invalidate() {
	c.sf.Forget(rateLimitConfigCacheKey)
	c.mu.Lock()
	c.cfg = nil
	c.fetchedAt = time.Time{}
	c.mu.Unlock()
}

func GetRateLimitConfig(ctx context.Context, getter SettingsGetter) (*RateLimitConfig, error) {
	settings, err := getter.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetRateLimitConfig - Get: %w", err)
	}

	submitUser := getOrDefault(settings.SubmitLimitPerUser, defaultSubmitPerUser)
	durMin := getOrDefault(settings.SubmitLimitDurationMin, defaultSubmitDurationMin)
	submitUserPerMin := submitUser / durMin
	if submitUserPerMin <= 0 {
		submitUserPerMin = 1
	}

	hintUnlockUserPerMin := submitUserPerMin * hintUnlockMultiplier
	if hintUnlockUserPerMin < defaultHintUnlockMinFloor {
		hintUnlockUserPerMin = defaultHintUnlockMinFloor
	}

	return &RateLimitConfig{
		LoginPerMinute:          getOrDefault(settings.RateLimitLoginPerMinute, defaultLoginPerMin),
		RegisterPerMinute:       getOrDefault(settings.RateLimitRegisterPerMinute, defaultRegisterPerMin),
		ForgotPasswordPerMinute: getOrDefault(settings.RateLimitForgotPasswordPerMinute, defaultForgotPasswordPerMin),
		ResetPasswordPerMinute:  getOrDefault(settings.RateLimitResetPasswordPerMinute, defaultResetPasswordPerMin),
		LogoutPerMinute:         getOrDefault(settings.RateLimitLogoutPerMinute, defaultLogoutPerMin),
		RefreshPerMinute:        getOrDefault(settings.RateLimitRefreshPerMinute, defaultRefreshPerMin),
		ScoreboardPerMinute:     getOrDefault(settings.RateLimitScoreboardPerMinute, defaultScoreboardPerMin),
		GeneralIPPerMinute:      getOrDefault(settings.RateLimitGeneralIPPerMinute, defaultGeneralIPPerMin),
		VerifyEmailPerMinute:    getOrDefault(settings.RateLimitVerifyEmailPerMinute, defaultVerifyEmailPerMin),
		OAuthCallbackPerMinute:  getOrDefault(settings.RateLimitOAuthCallbackPerMinute, defaultOAuthCallbackPerMin),
		OAuthRedirectPerMinute:  getOrDefault(settings.RateLimitOAuthRedirectPerMinute, defaultOAuthRedirectPerMin),
		SubmitUserPerMinute:     submitUserPerMin,
		SubmitIPPerMinute:       submitUserPerMin * submitIPMultiplier,
		HintUnlockUserPerMinute: hintUnlockUserPerMin,
		CommentPerMinute:        getOrDefault(settings.RateLimitCommentPerMinute, defaultCommentPerMinute),
	}, nil
}

func getOrDefault(value, defaultValue int) int {
	if value <= 0 {
		return defaultValue
	}
	return value
}
