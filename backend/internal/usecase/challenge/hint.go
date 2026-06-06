package challenge

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/computil"
)

type HintUseCase struct {
	deps HintDeps
}

type HintDeps struct {
	HintRepo        repo.HintRepository
	AwardRepo       repo.AwardRepository
	TM              repo.TransactionManager
	SolveRepo       repo.SolveRepository
	CompRepo        repo.CompetitionRepository
	CompUC          computil.CompetitionGetter
	TeamRepo        repo.TeamRepository
	UserRepo        repo.UserRepository
	ChallengeRepo   repo.ChallengeRepository
	ScoreboardCache cacheutil.ScoreboardCacheInvalidator
	Logger          logkit.Logger
}

var _ usecase.HintUseCase = (*HintUseCase)(nil)

func NewHintUseCase(deps HintDeps) *HintUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	return &HintUseCase{deps: deps}
}

func (uc *HintUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Hint, error) {
	hint, err := uc.deps.HintRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - GetByID - HintRepo.GetByID: %w", err)
	}

	return hint, nil
}
