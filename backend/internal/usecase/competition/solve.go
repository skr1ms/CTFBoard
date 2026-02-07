package competition

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/cache"
	"github.com/skr1ms/CTFBoard/pkg/usecaseutil"
	"github.com/skr1ms/CTFBoard/pkg/websocket"
)

type SolveDeps struct {
	SolveRepo       repo.SolveRepository
	ChallengeRepo   repo.ChallengeRepository
	CompetitionRepo repo.CompetitionRepository
	UserRepo        repo.UserRepository
	TeamRepo        repo.TeamRepository
	TxRepo          repo.TxRepository
	Cache           *cache.Cache
	ScoreboardCache cache.ScoreboardCacheInvalidator
	Broadcaster     websocket.SolveBroadcaster
}

type SolveUseCase struct {
	deps SolveDeps
}

func NewSolveUseCase(deps SolveDeps) *SolveUseCase {
	return &SolveUseCase{deps: deps}
}

func (uc *SolveUseCase) Create(ctx context.Context, solve *entity.Solve) error {
	var isFirstBlood bool
	var solvedChallenge *entity.Challenge
	err := uc.deps.TxRepo.RunTransaction(ctx, func(ctx context.Context, tx repo.Transaction) error {
		if err := uc.solveCreateResolveTeamID(ctx, tx, solve); err != nil {
			return err
		}
		challenge, fb, err := uc.solveCreateUpsertInTx(ctx, tx, solve)
		if err != nil {
			return err
		}
		solvedChallenge = challenge
		isFirstBlood = fb
		return nil
	})
	if err != nil {
		if errors.Is(err, entityError.ErrAlreadySolved) || errors.Is(err, entityError.ErrNoTeamSelected) {
			return err
		}
		return usecaseutil.Wrap(err, "SolveUseCase - Create - Transaction")
	}
	uc.invalidateScoreboardCache(ctx, solve.TeamID)
	if uc.deps.Broadcaster != nil && solvedChallenge != nil {
		uc.deps.Broadcaster.NotifySolve(solve.TeamID, solvedChallenge.Title, solvedChallenge.Points, isFirstBlood)
	}
	return nil
}

func (uc *SolveUseCase) solveCreateResolveTeamID(ctx context.Context, tx repo.Transaction, solve *entity.Solve) error {
	if solve.TeamID != uuid.Nil {
		return nil
	}
	if err := uc.deps.TxRepo.LockUserTx(ctx, tx, solve.UserID); err != nil {
		return usecaseutil.Wrap(err, "SolveUseCase - Create - LockUserTx")
	}
	user, err := uc.deps.UserRepo.GetByID(ctx, solve.UserID)
	if err != nil {
		return usecaseutil.Wrap(err, "SolveUseCase - Create - GetByID")
	}
	if user.TeamID == nil {
		return entityError.ErrNoTeamSelected
	}
	solve.TeamID = *user.TeamID
	return nil
}

func (uc *SolveUseCase) solveCreateUpsertInTx(ctx context.Context, tx repo.Transaction, solve *entity.Solve) (*entity.Challenge, bool, error) {
	challenge, err := uc.deps.TxRepo.GetChallengeByIDTx(ctx, tx, solve.ChallengeID)
	if err != nil {
		return nil, false, usecaseutil.Wrap(err, "SolveUseCase - Create - GetChallengeByIDTx")
	}
	existing, err := uc.deps.TxRepo.GetSolveByTeamAndChallengeTx(ctx, tx, solve.TeamID, solve.ChallengeID)
	if err == nil && existing != nil {
		return nil, false, entityError.ErrAlreadySolved
	}
	isFirstBlood := challenge.SolveCount == 0
	if err := uc.deps.TxRepo.CreateSolveTx(ctx, tx, solve); err != nil {
		return nil, false, usecaseutil.Wrap(err, "SolveUseCase - Create - CreateSolveTx")
	}
	newCount, err := uc.deps.TxRepo.IncrementChallengeSolveCountTx(ctx, tx, solve.ChallengeID)
	if err != nil {
		return nil, false, usecaseutil.Wrap(err, "SolveUseCase - Create - IncrementChallengeSolveCountTx")
	}
	if challenge.Decay > 0 {
		newPoints := CalculateDynamicScore(challenge.InitialValue, challenge.MinValue, challenge.Decay, newCount)
		if newPoints != challenge.Points {
			if err := uc.deps.TxRepo.UpdateChallengePointsTx(ctx, tx, challenge.ID, newPoints); err != nil {
				return nil, false, usecaseutil.Wrap(err, "SolveUseCase - Create - UpdateChallengePointsTx")
			}
			challenge.Points = newPoints
		}
	}
	return challenge, isFirstBlood, nil
}

func (uc *SolveUseCase) GetScoreboard(ctx context.Context, bracketID *uuid.UUID) ([]*repo.ScoreboardEntry, error) {
	comp, err := uc.deps.CompetitionRepo.Get(ctx)
	if err != nil && !errors.Is(err, entityError.ErrCompetitionNotFound) {
		return nil, usecaseutil.Wrap(err, "SolveUseCase - GetScoreboard - GetCompetition")
	}

	cacheKey, frozen := uc.getScoreboardCacheKey(comp, bracketID)

	return cache.GetOrLoad(uc.deps.Cache, ctx, cacheKey, 15*time.Second, func() ([]*repo.ScoreboardEntry, error) {
		var entries []*repo.ScoreboardEntry
		if frozen {
			if comp != nil && comp.FreezeTime != nil {
				entries, err = uc.deps.SolveRepo.GetScoreboardByBracketFrozen(ctx, *comp.FreezeTime, bracketID)
			} else {
				entries, err = uc.deps.SolveRepo.GetScoreboardByBracketFrozen(ctx, time.Time{}, bracketID)
			}
		} else {
			entries, err = uc.deps.SolveRepo.GetScoreboardByBracket(ctx, bracketID)
		}
		if err != nil {
			return nil, usecaseutil.Wrap(err, "SolveUseCase - GetScoreboard")
		}
		return entries, nil
	})
}

func (uc *SolveUseCase) getScoreboardCacheKey(comp *entity.Competition, bracketID *uuid.UUID) (string, bool) {
	frozen := comp != nil && comp.FreezeTime != nil && time.Now().After(*comp.FreezeTime)
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

func (uc *SolveUseCase) GetFirstBlood(ctx context.Context, challengeID uuid.UUID) (*repo.FirstBloodEntry, error) {
	entry, err := uc.deps.SolveRepo.GetFirstBlood(ctx, challengeID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "SolveUseCase - GetFirstBlood")
	}
	return entry, nil
}
