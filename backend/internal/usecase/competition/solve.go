package competition

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jellydator/ttlcache/v3"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/singleflight"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/computil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

const (
	localScoreboardTTL      = 2 * time.Second
	localScoreCacheCapacity = 32
	scoreboardRedisTTL      = 15 * time.Second
)

type SolveUseCase struct {
	deps            SolveDeps
	localScoreCache *ttlcache.Cache[string, []*domain.ScoreboardEntry]
	scoreboardSF    singleflight.Group
}
type SolveDeps struct {
	SolveRepo          repo.SolveRepository
	ChallengeRepo      repo.ChallengeRepository
	CompetitionRepo    repo.CompetitionRepository
	CompetitionUC      usecase.CompetitionUseCase
	CompParamUC        usecase.CompetitionParamUseCase
	UserRepo           repo.UserRepository
	TeamRepo           repo.TeamRepository
	TM                 repo.TransactionManager
	Cache              *cachekit.Cache
	ScoreboardCache    cacheutil.ScoreboardCacheInvalidator
	StatsCache         StatisticsCacheInvalidator
	ChallengeListCache cacheutil.ChallengeListCacheInvalidator
	Broadcaster        solveBroadcaster
	Logger             logkit.Logger
}

var _ usecase.SolveUseCase = (*SolveUseCase)(nil)

type solveBroadcaster interface {
	NotifySolve(teamID uuid.UUID, challengeTitle string, points int, isFirstBlood bool)
}

type localCacheRegistrar interface {
	RegisterLocalCache(fn func())
}

type localCacheLiveOnlyRegistrar interface {
	RegisterLocalCache(fn func())
	RegisterLocalCacheLiveOnly(fn func(keys []string))
}

// NewSolveUseCase initializes a SolveUseCase: starts a background ttlcache goroutine for
// in-process scoreboard caching and registers cache-invalidation callbacks on ScoreboardCache
// (full-clear or live-only, depending on which interface the implementation satisfies).
func NewSolveUseCase(deps SolveDeps) *SolveUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	localCache := ttlcache.New(
		ttlcache.WithTTL[string, []*domain.ScoreboardEntry](localScoreboardTTL),
		ttlcache.WithCapacity[string, []*domain.ScoreboardEntry](localScoreCacheCapacity),
	)
	go localCache.Start()

	uc := &SolveUseCase{
		deps:            deps,
		localScoreCache: localCache,
	}
	if r, ok := deps.ScoreboardCache.(localCacheLiveOnlyRegistrar); ok {
		r.RegisterLocalCache(uc.ClearLocalScoreCache)
		r.RegisterLocalCacheLiveOnly(uc.ClearLocalScoreCacheLiveOnly)
	} else if r, ok := deps.ScoreboardCache.(localCacheRegistrar); ok {
		r.RegisterLocalCache(uc.ClearLocalScoreCache)
	}

	return uc
}

func (uc *SolveUseCase) StopLocalScoreboardCache() {
	if uc == nil || uc.localScoreCache == nil {
		return
	}

	uc.localScoreCache.Stop()
}

// Create records a new solve inside a transaction. It validates competition timing,
// ban status, and team-size constraints via solveCreateResolveTeamID, then calls
// solveCreateUpsert which inserts the solve and triggers dynamic score decay.
// After the transaction it invalidates the scoreboard cache scoped to the team and,
// when the competition is not frozen, broadcasts the solve event to connected clients.
func (uc *SolveUseCase) Create(ctx context.Context, solve *domain.Solve) error {
	var (
		isFirstBlood    bool
		solvedChallenge *domain.Challenge
	)

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.solveCreateResolveTeamID(ctx, solve); err != nil {
			return fmt.Errorf("SolveUseCase - Create - solveCreateResolveTeamID: %w", err)
		}

		challenge, fb, err := uc.solveCreateUpsert(ctx, solve)
		if err != nil {
			return fmt.Errorf("SolveUseCase - Create - solveCreateUpsert: %w", err)
		}

		solvedChallenge = challenge
		isFirstBlood = fb

		return nil
	})
	if err != nil {
		return fmt.Errorf("SolveUseCase - Create - TM.Run: %w", err)
	}

	uc.invalidateScoreboardCache(ctx, solve.TeamID)
	uc.invalidateStatisticsCache(ctx, "Create")

	if uc.deps.ChallengeListCache != nil {
		uc.deps.ChallengeListCache.InvalidateForTeam(ctx, solve.TeamID)
	}

	if uc.deps.Broadcaster != nil && solvedChallenge != nil {
		comp := computil.Cached(ctx, uc.deps.CompetitionUC, uc.deps.CompetitionRepo)

		if comp == nil || !comp.IsFreezeActive() {
			uc.deps.Broadcaster.NotifySolve(solve.TeamID, solvedChallenge.Title, solvedChallenge.Points, isFirstBlood)
		}
	}

	return nil
}

// solveCreateResolveTeamID resolves and validates the team ID for a new solve
// inside the caller's transaction. It acquires an advisory row lock on both
// the user and team rows to serialize concurrent submissions. The function
// checks that neither the user nor the team is banned, verifies that the user
// belongs to the requested team, and enforces competition-mode rules: solo
// mode rejects non-solo teams, teams-only mode rejects solo teams, and when a
// minimum team size is configured it counts current members before allowing
// the solve. Returns a domain error if submission is not currently permitted
// (competition not started, paused, or ended).
func (uc *SolveUseCase) solveCreateResolveTeamID(ctx context.Context, solve *domain.Solve) error {
	var comp *domain.Competition

	if uc.deps.CompetitionRepo != nil {
		c, err := uc.deps.CompetitionRepo.GetForUpdate(ctx)
		if err != nil && !errors.Is(err, apperr.ErrCompetitionNotFound) {
			return fmt.Errorf("SolveUseCase - Create - CompetitionRepo.GetForUpdate: %w", err)
		}

		comp = c
	} else {
		comp = computil.Cached(ctx, uc.deps.CompetitionUC, uc.deps.CompetitionRepo)
	}

	if comp != nil && !comp.IsSubmissionAllowed() {
		return apperr.ErrSubmissionNotAllowed
	}

	if err := uc.deps.UserRepo.Lock(ctx, solve.UserID); err != nil {
		return fmt.Errorf("SolveUseCase - Create - UserRepo.Lock: %w", err)
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, solve.UserID)
	if err != nil {
		return fmt.Errorf("SolveUseCase - Create - UserRepo.GetByID: %w", err)
	}

	if solve.TeamID == uuid.Nil {
		if user.TeamID == nil {
			return apperr.ErrNoTeamSelected
		}

		solve.TeamID = *user.TeamID
	}

	if user.TeamID == nil || *user.TeamID != solve.TeamID {
		return apperr.ErrNoTeamSelected
	}

	if uc.deps.TeamRepo != nil {
		if err := uc.deps.TeamRepo.Lock(ctx, solve.TeamID); err != nil {
			return fmt.Errorf("SolveUseCase - Create - TeamRepo.Lock: %w", err)
		}

		team, err := uc.deps.TeamRepo.GetByID(ctx, solve.TeamID)
		if err != nil {
			return fmt.Errorf("SolveUseCase - Create - TeamRepo.GetByID: %w", err)
		}

		if err := guard.ValidateSubmissionEligibility(ctx, user, team, comp, uc.deps.TeamRepo); err != nil {
			return err
		}
	} else {
		if user.IsBanned {
			return apperr.ErrUserBanned
		}

		if user.WasInBannedTeam && user.Role != domain.RoleAdmin {
			return apperr.ErrUserWasInBannedTeam
		}
	}

	return nil
}

// solveCreateUpsert loads the challenge, verifies it is not hidden, and calls
// RecordSolveInTx which upserts the solve row and recalculates dynamic scoring.
// Returns the challenge and whether this is the first solve (first blood).
func (uc *SolveUseCase) solveCreateUpsert(ctx context.Context, solve *domain.Solve) (*domain.Challenge, bool, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, solve.ChallengeID)
	if err != nil {
		return nil, false, fmt.Errorf("SolveUseCase - Create - ChallengeRepo.GetByID: %w", err)
	}

	if err := guard.EnsureChallengeVisible(challenge); err != nil {
		return nil, false, err
	}

	solveCount, err := RecordSolveInTx(ctx, solve, challenge, uc.deps.ChallengeRepo, uc.deps.SolveRepo, scoring.GetDecayFn(ctx, uc.deps.CompParamUC))
	if err != nil {
		return nil, false, fmt.Errorf("SolveUseCase - Create - RecordSolveInTx: %w", err)
	}

	return challenge, solveCount == 1, nil
}

// GetScoreboard returns scoreboard entries using singleflight and a two-layer
// cache (in-process ttlcache -> Redis). The cache key encodes whether the
// scoreboard is frozen and, when frozen, the freeze timestamp, so that frozen
// and live views are cached independently and a freeze-time shift (e.g. after
// an unpause) produces a fresh key and bypasses any stale entry. The actual
// database query runs inside a read-only transaction to ensure a consistent
// snapshot. bracketID scopes the result to a specific bracket; nil returns the
// global scoreboard. forceLive bypasses the freeze check and always returns
// the live view.
func (uc *SolveUseCase) GetScoreboard(ctx context.Context, bracketID *uuid.UUID, forceLive bool) ([]*domain.ScoreboardEntry, error) {
	comp := computil.Cached(ctx, uc.deps.CompetitionUC, uc.deps.CompetitionRepo)
	cacheKey, frozen := uc.getScoreboardCacheKey(comp, bracketID, forceLive)

	if item := uc.localScoreCache.Get(cacheKey); item != nil {
		return item.Value(), nil
	}

	v, err, _ := uc.scoreboardSF.Do(cacheKey, func() (any, error) {
		if item := uc.localScoreCache.Get(cacheKey); item != nil {
			return item.Value(), nil
		}

		loadCtx, cancel := cacheutil.LoaderContext(ctx)
		defer cancel()

		entries, err := cachekit.GetOrLoad(uc.deps.Cache, loadCtx, cacheKey, scoreboardRedisTTL, func(loadCtx context.Context) ([]*domain.ScoreboardEntry, error) {
			var result []*domain.ScoreboardEntry

			var ft *time.Time

			if frozen {
				ft = comp.FreezeTime
			}

			errRO := uc.deps.TM.ReadOnly(loadCtx, func(roCtx context.Context) error {
				var errSB error

				result, errSB = uc.deps.SolveRepo.GetScoreboardByBracket(roCtx, bracketID, ft)

				return errSB
			})
			if errRO != nil {
				return nil, errRO
			}

			return result, nil
		})
		if err != nil {
			return nil, fmt.Errorf("SolveUseCase - GetScoreboard - cache.GetOrLoad: %w", err)
		}

		uc.localScoreCache.Set(cacheKey, entries, ttlcache.DefaultTTL)

		return entries, nil
	})
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetScoreboard - scoreboardSF.Do: %w", err)
	}

	entries, ok := v.([]*domain.ScoreboardEntry)
	if !ok {
		return nil, fmt.Errorf("SolveUseCase - GetScoreboard: unexpected type from singleflight")
	}

	return entries, nil
}

// getScoreboardCacheKey returns the cache key and whether the scoreboard is frozen
// When the competition ends naturally (end_time passes without admin Update), the next
// request uses the live key because IsFreezeActive() is false; frozen cache entries
// may still be present until TTL expiry (localScoreboardTTL + scoreboardRedisTTL)
// Frozen keys include freeze_time so that cache is invalidated when freeze_time shifts (e.g. on unpause).
func (uc *SolveUseCase) getScoreboardCacheKey(comp *domain.Competition, bracketID *uuid.UUID, forceLive bool) (string, bool) {
	frozen := !forceLive && comp != nil && comp.IsFreezeActive()

	if bracketID == nil || *bracketID == uuid.Nil {
		if frozen {
			return cacheutil.KeyScoreboardFrozenAt(comp.FreezeTime.Unix()), true
		}

		return cacheutil.KeyScoreboard, false
	}

	idStr := bracketID.String()

	if frozen {
		return cacheutil.KeyScoreboardBracketFrozenAt(idStr, comp.FreezeTime.Unix()), true
	}

	return cacheutil.KeyScoreboardBracket(idStr), false
}

func (uc *SolveUseCase) invalidateScoreboardCache(ctx context.Context, teamID uuid.UUID) {
	comp := computil.Cached(ctx, uc.deps.CompetitionUC, uc.deps.CompetitionRepo)
	frozen := comp != nil && comp.IsFreezeActive()
	cacheutil.InvalidateWithFreezeAwareness(ctx, uc.deps.ScoreboardCache, teamID, frozen)
}

func (uc *SolveUseCase) ClearLocalScoreCache() {
	uc.localScoreCache.DeleteAll()
}

func (uc *SolveUseCase) ClearLocalScoreCacheLiveOnly(keys []string) {
	for _, key := range keys {
		uc.localScoreCache.Delete(key)
	}
}

func (uc *SolveUseCase) GetFirstBlood(ctx context.Context, challengeID uuid.UUID, forceLive bool) (*domain.FirstBloodEntry, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetFirstBlood - ChallengeRepo.GetByID: %w", err)
	}

	if err := guard.EnsureChallengeVisible(challenge); err != nil {
		return nil, err
	}

	comp := computil.Cached(ctx, uc.deps.CompetitionUC, uc.deps.CompetitionRepo)

	var ft *time.Time

	if !forceLive && comp != nil && comp.IsFreezeActive() {
		ft = comp.FreezeTime
	}

	entry, err := uc.deps.SolveRepo.GetFirstBlood(ctx, challengeID, ft)
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetFirstBlood - SolveRepo.GetFirstBlood: %w", err)
	}

	return entry, nil
}
