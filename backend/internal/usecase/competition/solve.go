package competition

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/websocket"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

const (
	localScoreboardTTL      = 2 * time.Second
	localScoreCacheCapacity = 32
	scoreboardRedisTTL      = 15 * time.Second
)

type SolveUseCase struct {
	deps            SolveDeps
	localScoreCache *cache.TTLCache[string, []*entity.ScoreboardEntry]
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
	Cache              *cache.Cache
	ScoreboardCache    cache.ScoreboardCacheInvalidator
	ChallengeListCache cache.ChallengeListCacheInvalidator
	Broadcaster        websocket.SolveBroadcaster
	Logger             logger.Logger
}

var _ usecase.SolveUseCase = (*SolveUseCase)(nil)

// localCacheRegistrar is a structural interface satisfied by cache.ScoreboardCacheService.
// Using a local interface avoids importing the concrete type and keeps the dependency
// flowing in the right direction.
type localCacheRegistrar interface {
	RegisterLocalCache(fn func())
}

func NewSolveUseCase(deps SolveDeps) *SolveUseCase {
	if deps.Logger == nil {
		deps.Logger = logger.Noop()
	}
	uc := &SolveUseCase{
		deps:            deps,
		localScoreCache: cache.NewTTLCache[string, []*entity.ScoreboardEntry](localScoreboardTTL, localScoreCacheCapacity),
	}
	// If the scoreboard cache service supports local-cache registration, hook in
	// so that external invalidations (awards, bans, etc.) also clear localScoreCache.
	if r, ok := deps.ScoreboardCache.(localCacheRegistrar); ok {
		r.RegisterLocalCache(uc.ClearLocalScoreCache)
	}
	return uc
}

func (uc *SolveUseCase) Create(ctx context.Context, solve *entity.Solve) error {
	var isFirstBlood bool
	var solvedChallenge *entity.Challenge
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
		uc.deps.Broadcaster.NotifySolve(solve.TeamID, solvedChallenge.Title, solvedChallenge.Points, isFirstBlood)
	}
	return nil
}

func (uc *SolveUseCase) getCompetition(ctx context.Context) (*entity.Competition, error) {
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

//nolint:gocognit,gocyclo // team resolution + ban + competition-time + mode + min-size checks
func (uc *SolveUseCase) solveCreateResolveTeamID(ctx context.Context, solve *entity.Solve) error {
	if solve.TeamID == uuid.Nil {
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
		if user.TeamID == nil {
			return httperr.ErrNoTeamSelected
		}
		solve.TeamID = *user.TeamID
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
		comp, err := uc.getCompetition(ctx)
		if err != nil {
			return fmt.Errorf("SolveUseCase - Create - getCompetition: %w", err)
		}
		if comp != nil {
			if !comp.IsSubmissionAllowed() {
				return httperr.ErrSubmissionNotAllowed
			}
			if comp.Mode == entity.ModeTeamsOnly && team.IsSolo {
				return httperr.ErrTeamModeRequired
			}
			if comp.Mode == entity.ModeSoloOnly && !team.IsSolo {
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

func (uc *SolveUseCase) solveCreateUpsert(ctx context.Context, solve *entity.Solve) (*entity.Challenge, bool, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, solve.ChallengeID)
	if err != nil {
		return nil, false, fmt.Errorf("SolveUseCase - Create - ChallengeRepo.GetByID: %w", err)
	}
	if challenge.IsHidden {
		return nil, false, httperr.ErrChallengeNotFound
	}
	existing, err := uc.deps.SolveRepo.GetByTeamAndChallengeForUpdate(ctx, solve.TeamID, solve.ChallengeID)
	if err == nil && existing != nil {
		return nil, false, httperr.ErrAlreadySolved
	}
	if err != nil && !errors.Is(err, httperr.ErrSolveNotFound) {
		return nil, false, fmt.Errorf("SolveUseCase - Create - SolveRepo.GetByTeamAndChallengeForUpdate: %w", err)
	}
	newCount, err := uc.deps.ChallengeRepo.IncrementSolveCount(ctx, solve.ChallengeID)
	if err != nil {
		return nil, false, fmt.Errorf("SolveUseCase - Create - ChallengeRepo.IncrementSolveCount: %w", err)
	}
	isFirstBlood := newCount == 1
	pointsAtSolve, err := scoring.ApplySolveScore(ctx,
		challenge.InitialValue, challenge.MinValue, challenge.Decay, challenge.Points, newCount,
		func(ctx context.Context, pts int) error {
			if err := uc.deps.ChallengeRepo.UpdatePoints(ctx, challenge.ID, pts); err != nil {
				return fmt.Errorf("SolveUseCase - Create - ChallengeRepo.UpdatePoints: %w", err)
			}
			challenge.Points = pts
			return nil
		},
	)
	if err != nil {
		return nil, false, fmt.Errorf("SolveUseCase - Create - ApplySolveScore: %w", err)
	}
	solve.PointsAtSolve = pointsAtSolve
	if err := uc.deps.SolveRepo.Create(ctx, solve); err != nil {
		return nil, false, fmt.Errorf("SolveUseCase - Create - SolveRepo.Create: %w", err)
	}
	return challenge, isFirstBlood, nil
}

func (uc *SolveUseCase) GetScoreboard(ctx context.Context, bracketID *uuid.UUID) ([]*entity.ScoreboardEntry, error) {
	comp, err := uc.getCompetition(ctx)
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetScoreboard - getCompetition: %w", err)
	}

	cacheKey, frozen := uc.getScoreboardCacheKey(comp, bracketID)

	if entries, ok := uc.localScoreCache.Get(cacheKey); ok {
		return entries, nil
	}

	v, err, _ := uc.scoreboardSF.Do(cacheKey, func() (any, error) {
		if entries, ok := uc.localScoreCache.Get(cacheKey); ok {
			return entries, nil
		}
		entries, err := cache.GetOrLoad(uc.deps.Cache, ctx, cacheKey, scoreboardRedisTTL, func() ([]*entity.ScoreboardEntry, error) {
			if frozen {
				return uc.deps.SolveRepo.GetScoreboardByBracketFrozen(ctx, *comp.FreezeTime, bracketID)
			}
			return uc.deps.SolveRepo.GetScoreboardByBracket(ctx, bracketID)
		})
		if err != nil {
			return nil, fmt.Errorf("SolveUseCase - GetScoreboard - cache.GetOrLoad: %w", err)
		}
		uc.localScoreCache.Set(cacheKey, entries)
		return entries, nil
	})
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetScoreboard - scoreboardSF.Do: %w", err)
	}
	entries, ok := v.([]*entity.ScoreboardEntry)
	if !ok {
		return nil, fmt.Errorf("SolveUseCase - GetScoreboard: unexpected type from singleflight")
	}
	return entries, nil
}

func (uc *SolveUseCase) getScoreboardCacheKey(comp *entity.Competition, bracketID *uuid.UUID) (string, bool) {
	frozen := comp != nil && comp.GetStatus() == entity.CompetitionStatusFrozen
	if bracketID == nil || *bracketID == uuid.Nil {
		if frozen {
			return cache.KeyScoreboardFrozen, true
		}
		return cache.KeyScoreboard, false
	}
	idStr := bracketID.String()
	if frozen {
		return cache.KeyScoreboardBracketFrozen(idStr), true
	}
	return cache.KeyScoreboardBracket(idStr), false
}

func (uc *SolveUseCase) invalidateScoreboardCache(ctx context.Context, teamID uuid.UUID) {
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateForTeam(ctx, teamID)
	}
}

// ClearLocalScoreCache evicts all entries from the in-process scoreboard TTL
// cache. It is registered as a hook on ScoreboardCacheService so that external
// invalidations (e.g. award creation, team ban) also clear this layer.
func (uc *SolveUseCase) ClearLocalScoreCache() {
	uc.localScoreCache.DeleteAll()
}

func (uc *SolveUseCase) GetFirstBlood(ctx context.Context, challengeID uuid.UUID) (*entity.FirstBloodEntry, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetFirstBlood - ChallengeRepo.GetByID: %w", err)
	}
	if challenge.IsHidden {
		return nil, httperr.ErrChallengeNotFound
	}
	entry, err := uc.deps.SolveRepo.GetFirstBlood(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetFirstBlood - SolveRepo.GetFirstBlood: %w", err)
	}
	return entry, nil
}
