package competition

import (
	"context"
	"errors"
	"regexp"
	"sync"
	"time"

	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/singleflight"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

const (
	localTTL                  = 5 * time.Second
	redisTTL                  = 60 * time.Second
	negativeCacheTTL          = 30 * time.Second
	invalidateTimeout         = 2 * time.Second
	configsCacheKey           = "configs:all"
	configsInvChannel         = "configs:inv"
	loadAllKey                = "competition_params:loadAll"
	competitionParamKeyMaxLen = 100
)

var competitionParamKeyRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

var errCacheNotInitialized = errors.New("cache not initialized")

var publicConfigKeys = []string{
	"ctf_name", "ctf_description", "ctf_logo", "tos_url", "privacy_url",
	"theme_color_primary", "theme_color_secondary", "theme_header_html", "theme_footer_html", "theme_dark_mode",
	"social_github", "social_discord", "social_twitter", "social_website", "social_shares_enabled",
	"challenge_visibility", "score_visibility", "account_visibility", "registration_visibility",
	"setup_complete", "email_verification_required", "timezone",
}

var allowedCategories = map[string]struct{}{
	domain.ConfigCategoryGeneral:    {},
	domain.ConfigCategoryTheme:      {},
	domain.ConfigCategoryVisibility: {},
	domain.ConfigCategoryScoring:    {},
	domain.ConfigCategoryEmail:      {},
	domain.ConfigCategorySocial:     {},
	domain.ConfigCategoryLegal:      {},
	domain.ConfigCategoryAdvanced:   {},
}

var allowedVisibilityValues = map[string][]string{
	"challenge_visibility":    {"public", "private", "hidden", "admins"},
	"score_visibility":        {"public", "private", "hidden", "admins", "admins_only"},
	"account_visibility":      {"public", "private", "hidden", "admins"},
	"registration_visibility": {"public", "private"},
}

type CompetitionParamUseCase struct {
	deps          CompetitionParamDeps
	cache         map[string]*domain.CompetitionParam
	negativeCache map[string]time.Time
	mu            sync.RWMutex
	lastLoad      time.Time
	sf            singleflight.Group
}

type CompetitionParamDeps struct {
	Repo         repo.CompetitionParamRepository
	AuditLogRepo repo.AuditLogRepository
	TM           repo.TransactionManager
	Logger       logkit.Logger
	Cache        cachekit.KeyValueStore
	PubSub       cachekit.PubSubStore
	StopContext  context.Context
}

var _ usecase.CompetitionParamUseCase = (*CompetitionParamUseCase)(nil)

// NewCompetitionParamUseCase initializes a CompetitionParamUseCase with an in-process map
// cache and a negative-miss cache. If PubSub is configured, a background goroutine is
// started to subscribe to Redis invalidation events and call invalidateLocal on each message.
func NewCompetitionParamUseCase(deps CompetitionParamDeps) *CompetitionParamUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	uc := &CompetitionParamUseCase{
		deps:          deps,
		cache:         make(map[string]*domain.CompetitionParam),
		negativeCache: make(map[string]time.Time),
	}
	if uc.deps.PubSub != nil {
		if uc.deps.StopContext == nil {
			uc.deps.Logger.Warn("CompetitionParamUseCase: PubSub invalidation disabled: StopContext is nil")

			return uc
		}

		go cacheutil.SubscribeInvalidation(uc.deps.StopContext, uc.deps.PubSub, configsInvChannel, uc.invalidateLocal, uc.deps.Logger, "competition_params")
	}

	return uc
}
