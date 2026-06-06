package wire

import (
	"context"

	"github.com/redis/go-redis/v9"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/internal/loginlockout"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

func (f healthCheckerFunc) Check(ctx context.Context) error {
	return f(ctx)
}

func ProvideRuntimeSettingsInvalidator() *runtimeSettingsInvalidator {
	return &runtimeSettingsInvalidator{}
}

func (i *runtimeSettingsInvalidator) SetRateLimitCache(c *restapimiddleware.RateLimitConfigCache) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.rateLimitCache = c
}

func (i *runtimeSettingsInvalidator) SetScoreboardVisibilityCache(c *restapimiddleware.ScoreboardVisibilityCache) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.scoreboardVisibilityCache = c
}

func (i *runtimeSettingsInvalidator) InvalidateRuntimeSettingsCaches() {
	i.mu.RLock()
	rateLimitCache := i.rateLimitCache
	scoreboardVisibilityCache := i.scoreboardVisibilityCache
	i.mu.RUnlock()

	if rateLimitCache != nil {
		rateLimitCache.Invalidate()
	}

	if scoreboardVisibilityCache != nil {
		scoreboardVisibilityCache.Invalidate()
	}
}

func ProvideCompetitionGuard(compUC *competition.CompetitionUseCase) *competition.Guard {
	return competition.NewGuard(compUC)
}

func ProvideValidator() (validator.Validator, error) {
	return validator.New()
}

func ProvideFailedLoginTracker(redisClient *redis.Client) *loginlockout.Tracker {
	return loginlockout.NewTracker(redisClient, loginLockoutMaxAttempts, loginLockoutTTL)
}

func ProvideCrypto(cfg *config.Config) (crypto.Service, error) {
	if cfg.FlagEncryptionKey == "" {
		return nil, nil
	}

	return crypto.NewCryptoService(cfg.FlagEncryptionKey)
}
