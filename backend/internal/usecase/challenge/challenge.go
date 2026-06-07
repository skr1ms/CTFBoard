package challenge

import (
	"context"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/semaphore"
	"golang.org/x/sync/singleflight"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

// SolveRecordFn is the function type for recording a solve inside a transaction.
// Using a function value instead of a direct package import decouples the challenge
// usecase from the competition usecase package.
type SolveRecordFn func(ctx context.Context, solve *domain.Solve, challenge *domain.Challenge, challengeRepo repo.ChallengeRepository, solveRepo repo.SolveRepository, decayFn ...scoring.DecayFunction) (int, error)

const (
	// challengeListCachePrefix is kept only for backward-compatible invalidation of old keys.
	challengeListCachePrefix = "challenges:list:"

	// Two-layer cache: shared base (challenges without per-team solve status) + lightweight per-team solved-ID set.
	challengeBaseCachePrefix   = "challenges:base:"
	challengeBaseTTL           = 30 * time.Second
	challengeSolvedCachePrefix = "challenges:solved:"
	challengeSolvedTTL         = 10 * time.Second

	// Frozen solve counts are immutable for the duration of the freeze; cache with a long TTL.
	frozenSolveCountsCachePrefix = "challenges:frozen_counts:"
	frozenSolveCountsTTL         = 10 * time.Minute
)

type ChallengeUseCase struct {
	deps              ChallengeDeps
	regexCache        *cachekit.LRFUCache[string, *regexp.Regexp]
	regexSem          *semaphore.Weighted
	regexSf           singleflight.Group
	listBaseSF        singleflight.Group // deduplicates concurrent base-cache loader calls on cache miss
	challengeDetailSf singleflight.Group // for GetDetail (returns *usecase.ChallengeDetail)
	challengeFetchSf  singleflight.Group // for submitGetChallenge (returns *domain.Challenge)
	requirementsSf    singleflight.Group
}

type ChallengeDeps struct {
	ChallengeRepo   repo.ChallengeRepository
	TagRepo         repo.TagRepository
	SolveRepo       repo.SolveRepository
	SubmissionRepo  repo.SubmissionRepository
	FileRepo        repo.FileRepository
	Storage         FileStorage
	HintUC          usecase.HintUseCase
	TM              repo.TransactionManager
	CompRepo        repo.CompetitionRepository
	CompUC          usecase.CompetitionUseCase
	CompParamUC     usecase.CompetitionParamUseCase
	TeamRepo        repo.TeamRepository
	UserRepo        repo.UserRepository
	ScoreboardCache cacheutil.ScoreboardCacheInvalidator
	ListCache       *cachekit.Cache
	Broadcaster     solveBroadcaster
	AuditLogRepo    repo.AuditLogRepository
	Crypto          crypto.Service
	Logger          logkit.Logger
	SolveRecord     SolveRecordFn
	RegexSem        *semaphore.Weighted
}

var (
	_ usecase.ChallengeReadUseCase   = (*ChallengeUseCase)(nil)
	_ usecase.ChallengeSubmitUseCase = (*ChallengeUseCase)(nil)
	_ usecase.ChallengeAdminUseCase  = (*ChallengeUseCase)(nil)
	_ usecase.ChallengeUseCase       = (*ChallengeUseCase)(nil)
)

type solveBroadcaster interface {
	NotifySolve(teamID uuid.UUID, challengeTitle string, points int, isFirstBlood bool)
}

func NewChallengeUseCase(deps ChallengeDeps) *ChallengeUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	if deps.RegexSem == nil {
		deps.RegexSem = semaphore.NewWeighted(maxConcurrentRegex)
	}

	return &ChallengeUseCase{
		deps:       deps,
		regexCache: cachekit.NewLRFUCache[string, *regexp.Regexp](cachekit.DefaultLRFUCacheSize),
		regexSem:   deps.RegexSem,
	}
}
