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

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/websocket"
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
	UserRepo           repo.UserRepository
	TeamRepo           repo.TeamRepository
	TM                 repo.TransactionManager
	Cache              *cachekit.Cache
	ScoreboardCache    cache.ScoreboardCacheInvalidator
	ChallengeListCache cache.ChallengeListCacheInvalidator
	Broadcaster        websocket.SolveBroadcaster
	Logger             logkit.Logger
}

var _ usecase.SolveUseCase = (*SolveUseCase)(nil)

type localCacheRegistrar interface {
	RegisterLocalCache(fn func())
}

type localCacheLiveOnlyRegistrar interface {
	RegisterLocalCache(fn func())
	RegisterLocalCacheLiveOnly(fn func(keys []string))
}

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

	if uc.deps.ChallengeListCache != nil {
		uc.deps.ChallengeListCache.InvalidateForTeam(ctx, solve.TeamID)
	}

	if uc.deps.Broadcaster != nil && solvedChallenge != nil {
		comp, err := uc.getCompetition(ctx)
		if err != nil && uc.deps.Logger != nil {
			uc.deps.Logger.WithError(err).Warn("SolveUseCase - Create - getCompetition")
		}

		if comp == nil || !comp.IsFreezeActive() {
			uc.deps.Broadcaster.NotifySolve(solve.TeamID, solvedChallenge.Title, solvedChallenge.Points, isFirstBlood)
		}
	}

	return nil
}

func (uc *SolveUseCase) getCompetition(ctx context.Context) (*domain.Competition, error) {
	if uc.deps.CompetitionUC != nil {
		comp, err := uc.deps.CompetitionUC.Get(ctx)
		if err != nil && !errors.Is(err, httperr.ErrCompetitionNotFound) {
			return nil, fmt.Errorf("SolveUseCase - getCompetition - CompetitionUC.Get: %w", err)
		}

		return comp, nil
	}

	if uc.deps.CompetitionRepo != nil {
		comp, err := uc.deps.CompetitionRepo.Get(ctx)
		if err != nil && !errors.Is(err, httperr.ErrCompetitionNotFound) {
			return nil, fmt.Errorf("SolveUseCase - getCompetition - CompetitionRepo.Get: %w", err)
		}

		return comp, nil
	}

	return nil, nil
}

func (uc *SolveUseCase) solveCreateResolveTeamID(ctx context.Context, solve *domain.Solve) error {
	var comp *domain.Competition

	if uc.deps.CompetitionRepo != nil {
		c, err := uc.deps.CompetitionRepo.GetForUpdate(ctx)
		if err != nil && !errors.Is(err, httperr.ErrCompetitionNotFound) {
			return fmt.Errorf("SolveUseCase - Create - CompetitionRepo.GetForUpdate: %w", err)
		}

		comp = c
	} else {
		c, err := uc.getCompetition(ctx)
		if err != nil {
			return fmt.Errorf("SolveUseCase - Create - getCompetition: %w", err)
		}

		comp = c
	}

	if comp != nil && !comp.IsSubmissionAllowed() {
		return httperr.ErrSubmissionNotAllowed
	}

	if err := uc.deps.UserRepo.Lock(ctx, solve.UserID); err != nil {
		return fmt.Errorf("SolveUseCase - Create - UserRepo.Lock: %w", err)
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, solve.UserID)
	if err != nil {
		return fmt.Errorf("SolveUseCase - Create - UserRepo.GetByID: %w", err)
	}

	if user.IsBanned {
		return httperr.ErrUserBanned
	}

	if solve.TeamID == uuid.Nil {
		if user.TeamID == nil {
			return httperr.ErrNoTeamSelected
		}

		solve.TeamID = *user.TeamID
	}

	if user.TeamID == nil || *user.TeamID != solve.TeamID {
		return httperr.ErrNoTeamSelected
	}

	if uc.deps.TeamRepo != nil {
		if err := uc.deps.TeamRepo.Lock(ctx, solve.TeamID); err != nil {
			return fmt.Errorf("SolveUseCase - Create - TeamRepo.Lock: %w", err)
		}

		team, err := uc.deps.TeamRepo.GetByID(ctx, solve.TeamID)
		if err != nil {
			return fmt.Errorf("SolveUseCase - Create - TeamRepo.GetByID: %w", err)
		}

		if team.IsBanned {
			return httperr.ErrTeamBanned
		}

		if comp != nil {
			if comp.Mode == domain.ModeTeamsOnly && team.IsSolo {
				return httperr.ErrTeamModeRequired
			}

			if comp.Mode == domain.ModeSoloOnly && !team.IsSolo {
				return httperr.ErrSoloModeRequired
			}

			if comp.MinTeamSize > 0 && !team.IsSolo {
				count, err := uc.deps.TeamRepo.CountTeamMembers(ctx, solve.TeamID)
				if err != nil {
					return fmt.Errorf("SolveUseCase - Create - TeamRepo.CountTeamMembers: %w", err)
				}

				if count < comp.MinTeamSize {
					return httperr.ErrTeamBelowMinSize
				}
			}
		}
	}

	return nil
}

func (uc *SolveUseCase) solveCreateUpsert(ctx context.Context, solve *domain.Solve) (*domain.Challenge, bool, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, solve.ChallengeID)
	if err != nil {
		return nil, false, fmt.Errorf("SolveUseCase - Create - ChallengeRepo.GetByID: %w", err)
	}

	if challenge.State == domain.ChallengeStateHidden {
		return nil, false, httperr.ErrChallengeNotFound
	}

	solveCount, err := RecordSolveInTx(ctx, solve, challenge, uc.deps.ChallengeRepo, uc.deps.SolveRepo)
	if err != nil {
		return nil, false, fmt.Errorf("SolveUseCase - Create - RecordSolveInTx: %w", err)
	}

	return challenge, solveCount == 1, nil
}

func (uc *SolveUseCase) GetScoreboard(ctx context.Context, bracketID *uuid.UUID, forceLive bool) ([]*domain.ScoreboardEntry, error) {
	comp, err := uc.getCompetition(ctx)
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetScoreboard - getCompetition: %w", err)
	}

	cacheKey, frozen := uc.getScoreboardCacheKey(comp, bracketID, forceLive)

	if item := uc.localScoreCache.Get(cacheKey); item != nil {
		return item.Value(), nil
	}

	v, err, _ := uc.scoreboardSF.Do(cacheKey, func() (any, error) {
		sfCtx := context.WithoutCancel(ctx)

		if item := uc.localScoreCache.Get(cacheKey); item != nil {
			return item.Value(), nil
		}

		entries, err := cachekit.GetOrLoad(uc.deps.Cache, sfCtx, cacheKey, scoreboardRedisTTL, func(context.Context) ([]*domain.ScoreboardEntry, error) {
			var result []*domain.ScoreboardEntry

			errRO := uc.deps.TM.ReadOnly(sfCtx, func(roCtx context.Context) error {
				var err2 error

				if frozen {
					result, err2 = uc.deps.SolveRepo.GetScoreboardByBracketFrozen(roCtx, *comp.FreezeTime, bracketID)
				} else {
					result, err2 = uc.deps.SolveRepo.GetScoreboardByBracket(roCtx, bracketID)
				}

				return err2
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

// getScoreboardCacheKey returns the cache key and whether the scoreboard is frozen.
// When the competition ends naturally (end_time passes without admin Update), the next
// request uses the live key because IsFreezeActive() is false; frozen cache entries
// may still be present until TTL expiry (localScoreboardTTL + scoreboardRedisTTL).
// Frozen keys include freeze_time so that cache is invalidated when freeze_time shifts (e.g. on unpause).
func (uc *SolveUseCase) getScoreboardCacheKey(comp *domain.Competition, bracketID *uuid.UUID, forceLive bool) (string, bool) {
	frozen := !forceLive && comp != nil && comp.IsFreezeActive()

	if bracketID == nil || *bracketID == uuid.Nil {
		if frozen {
			return cache.KeyScoreboardFrozenAt(comp.FreezeTime.Unix()), true
		}

		return cache.KeyScoreboard, false
	}

	idStr := bracketID.String()

	if frozen {
		return cache.KeyScoreboardBracketFrozenAt(idStr, comp.FreezeTime.Unix()), true
	}

	return cache.KeyScoreboardBracket(idStr), false
}

func (uc *SolveUseCase) invalidateScoreboardCache(ctx context.Context, teamID uuid.UUID) {
	if uc.deps.ScoreboardCache == nil {
		return
	}

	comp, err := uc.getCompetition(ctx)
	if err == nil && comp != nil && comp.IsFreezeActive() {
		uc.deps.ScoreboardCache.InvalidateLiveOnly(ctx, teamID)

		return
	}

	uc.deps.ScoreboardCache.InvalidateForTeam(ctx, teamID)
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

	if challenge.State == domain.ChallengeStateHidden {
		return nil, httperr.ErrChallengeNotFound
	}

	comp, err := uc.getCompetition(ctx)
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetFirstBlood - getCompetition: %w", err)
	}

	if !forceLive && comp != nil && comp.IsFreezeActive() {
		entry, err := uc.deps.SolveRepo.GetFirstBloodFrozen(ctx, challengeID, *comp.FreezeTime)
		if err != nil {
			return nil, fmt.Errorf("SolveUseCase - GetFirstBlood - SolveRepo.GetFirstBloodFrozen: %w", err)
		}

		return entry, nil
	}

	entry, err := uc.deps.SolveRepo.GetFirstBlood(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetFirstBlood - SolveRepo.GetFirstBlood: %w", err)
	}

	return entry, nil
}
