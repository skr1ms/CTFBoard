package competition

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type AdminSolveCreator interface {
	AdminCreateSolve(ctx context.Context, userID, teamID, challengeID uuid.UUID, skipCompetitionCheck bool) error
}

type AdminSolveDeleter interface {
	AdminDeleteSolve(ctx context.Context, teamID, challengeID uuid.UUID) error
}

type CacheInvalidator interface {
	InvalidateScoreboardCache(ctx context.Context)
	// InvalidateScoreboardCacheForTeam invalidates both the global and bracket-specific
	// scoreboard cache keys for the given team. Use when a teamID is known (e.g. admin
	// submission updates) to avoid leaving bracket scoreboards stale
	InvalidateScoreboardCacheForTeam(ctx context.Context, teamID uuid.UUID)
}

type submissionCompGetter interface {
	Get(ctx context.Context) (*domain.Competition, error)
}

type SubmissionUseCase struct {
	deps SubmissionDeps
}

var _ usecase.SubmissionUseCase = (*SubmissionUseCase)(nil)

type SubmissionDeps struct {
	SubmissionRepo   repo.SubmissionRepository
	CompGetter       submissionCompGetter
	TM               repo.TransactionManager
	SolveCreator     AdminSolveCreator
	SolveDeleter     AdminSolveDeleter
	CacheInvalidator CacheInvalidator
	Logger           logkit.Logger
	UserRepo         repo.UserRepository
	TeamRepo         repo.TeamRepository
}

// NewSubmissionUseCase constructs a SubmissionUseCase. It panics at startup
// when TM, SolveCreator, and SolveDeleter are only partially configured, since
// solve-sensitive admin writes require those dependencies together.
func NewSubmissionUseCase(deps SubmissionDeps) *SubmissionUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}
	// Guard against partial configuration: the transactional path requires all
	// three deps together. A mismatch means the caller made a wiring mistake.
	txDeps := []bool{deps.TM != nil, deps.SolveCreator != nil, deps.SolveDeleter != nil}
	hasAny := txDeps[0] || txDeps[1] || txDeps[2]

	hasAll := txDeps[0] && txDeps[1] && txDeps[2]
	if hasAny && !hasAll {
		panic("SubmissionUseCase: TM, SolveCreator, and SolveDeleter must all be provided together or not at all")
	}

	return &SubmissionUseCase{deps: deps}
}

func (uc *SubmissionUseCase) requireTxDeps(op string, needsCreator bool) error {
	if uc.deps.TM == nil {
		return fmt.Errorf("SubmissionUseCase - %s: transaction manager required", op)
	}

	if needsCreator && uc.deps.SolveCreator == nil {
		return fmt.Errorf("SubmissionUseCase - %s: solve creator required", op)
	}

	if uc.deps.SolveDeleter == nil {
		return fmt.Errorf("SubmissionUseCase - %s: solve deleter required", op)
	}

	return nil
}
