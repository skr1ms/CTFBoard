package avatar

import (
	"context"
	"sync"
	"time"

	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/singleflight"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

const (
	contentTypeWebP                = "image/webp"
	resolveAvatarsBatchConcurrency = 10
	avatarSideEffectTimeout        = 5 * time.Second
)

type cachedAvatarURLs struct {
	FullURL  string `json:"full"`
	ThumbURL string `json:"thumb"`
}

type AvatarDeps struct {
	UserRepo     repo.UserRepository
	TeamRepo     repo.TeamRepository
	Storage      AvatarStorage
	Cache        cachekit.KeyValueStore
	TM           repo.TransactionManager
	AuditLogRepo repo.AuditLogRepository
	Config       domain.AvatarConfig
	Logger       logkit.Logger
}

type AvatarUseCase struct {
	deps      AvatarDeps
	processor *ImageProcessor
	wg        sync.WaitGroup
	presignSF singleflight.Group
}

var _ usecase.AvatarUseCase = (*AvatarUseCase)(nil)

func NewAvatarUseCase(deps AvatarDeps) *AvatarUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	return &AvatarUseCase{
		deps:      deps,
		processor: NewImageProcessor(deps.Config),
	}
}

func (uc *AvatarUseCase) Wait() { uc.wg.Wait() }

func (uc *AvatarUseCase) sideEffectCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), avatarSideEffectTimeout)
}

func (uc *AvatarUseCase) runAvatarMutation(ctx context.Context, fn func(context.Context) error) error {
	if uc.deps.TM == nil {
		return fn(ctx)
	}

	return uc.deps.TM.Run(ctx, fn)
}
