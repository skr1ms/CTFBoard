package competition

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
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
	// submission updates) to avoid leaving bracket scoreboards stale.
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

func NewSubmissionUseCase(deps SubmissionDeps) *SubmissionUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}
	// Guard against partial configuration: the transactional path requires all
	// three deps together. A mismatch means the caller made a wiring mistake.
	// Panic at startup rather than silently running the degraded path in production.
	txDeps := []bool{deps.TM != nil, deps.SolveCreator != nil, deps.SolveDeleter != nil}
	hasAny := txDeps[0] || txDeps[1] || txDeps[2]
	hasAll := txDeps[0] && txDeps[1] && txDeps[2]
	if hasAny && !hasAll {
		panic("SubmissionUseCase: TM, SolveCreator, and SolveDeleter must all be provided together or not at all")
	}
	return &SubmissionUseCase{deps: deps}
}

func (uc *SubmissionUseCase) LogSubmission(ctx context.Context, sub *domain.Submission) error {
	if err := uc.deps.SubmissionRepo.Create(ctx, sub); err != nil {
		return fmt.Errorf("SubmissionUseCase - LogSubmission - SubmissionRepo.Create: %w", err)
	}
	return nil
}

func (uc *SubmissionUseCase) GetByChallenge(ctx context.Context, challengeID uuid.UUID, page, perPage int, forceLive bool) (*usecase.Paginated[*domain.SubmissionWithDetails], error) {
	comp, freezeTime := uc.getCompAndFreezeTime(ctx)
	if !forceLive && comp != nil && comp.IsFreezeActive() && comp.FreezeTime != nil {
		result, err := usecase.FetchPage(ctx, page, perPage,
			func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
				return uc.deps.SubmissionRepo.GetByChallengeFrozen(ctx, challengeID, *freezeTime, limit, offset)
			},
			func(ctx context.Context) (int64, error) {
				return uc.deps.SubmissionRepo.CountByChallengeFrozen(ctx, challengeID, *freezeTime)
			},
		)
		if err != nil {
			return nil, fmt.Errorf("SubmissionUseCase - GetByChallenge: %w", err)
		}
		return result, nil
	}
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
			return uc.deps.SubmissionRepo.GetByChallenge(ctx, challengeID, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.SubmissionRepo.CountByChallenge(ctx, challengeID)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - GetByChallenge: %w", err)
	}
	return result, nil
}

func (uc *SubmissionUseCase) GetByUser(ctx context.Context, userID uuid.UUID, page, perPage int, forceLive bool) (*usecase.Paginated[*domain.SubmissionWithDetails], error) {
	comp, freezeTime := uc.getCompAndFreezeTime(ctx)
	if !forceLive && comp != nil && comp.IsFreezeActive() && comp.FreezeTime != nil {
		result, err := usecase.FetchPage(ctx, page, perPage,
			func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
				return uc.deps.SubmissionRepo.GetByUserFrozen(ctx, userID, *freezeTime, limit, offset)
			},
			func(ctx context.Context) (int64, error) {
				return uc.deps.SubmissionRepo.CountByUserFrozen(ctx, userID, *freezeTime)
			},
		)
		if err != nil {
			return nil, fmt.Errorf("SubmissionUseCase - GetByUser: %w", err)
		}
		return result, nil
	}
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
			return uc.deps.SubmissionRepo.GetByUser(ctx, userID, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.SubmissionRepo.CountByUser(ctx, userID)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - GetByUser: %w", err)
	}
	return result, nil
}

func (uc *SubmissionUseCase) GetByTeam(ctx context.Context, teamID uuid.UUID, page, perPage int, forceLive bool) (*usecase.Paginated[*domain.SubmissionWithDetails], error) {
	comp, freezeTime := uc.getCompAndFreezeTime(ctx)
	if !forceLive && comp != nil && comp.IsFreezeActive() && comp.FreezeTime != nil {
		result, err := usecase.FetchPage(ctx, page, perPage,
			func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
				return uc.deps.SubmissionRepo.GetByTeamFrozen(ctx, teamID, *freezeTime, limit, offset)
			},
			func(ctx context.Context) (int64, error) {
				return uc.deps.SubmissionRepo.CountByTeamFrozen(ctx, teamID, *freezeTime)
			},
		)
		if err != nil {
			return nil, fmt.Errorf("SubmissionUseCase - GetByTeam: %w", err)
		}
		return result, nil
	}
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
			return uc.deps.SubmissionRepo.GetByTeam(ctx, teamID, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.SubmissionRepo.CountByTeam(ctx, teamID)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - GetByTeam: %w", err)
	}
	return result, nil
}

func (uc *SubmissionUseCase) GetAll(ctx context.Context, page, perPage int, forceLive bool) (*usecase.Paginated[*domain.SubmissionWithDetails], error) {
	comp, freezeTime := uc.getCompAndFreezeTime(ctx)
	if !forceLive && comp != nil && comp.IsFreezeActive() && comp.FreezeTime != nil {
		result, err := usecase.FetchPage(ctx, page, perPage,
			func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
				return uc.deps.SubmissionRepo.GetAllFrozen(ctx, *freezeTime, limit, offset)
			},
			func(ctx context.Context) (int64, error) {
				return uc.deps.SubmissionRepo.CountAllFrozen(ctx, *freezeTime)
			},
		)
		if err != nil {
			return nil, fmt.Errorf("SubmissionUseCase - GetAll: %w", err)
		}
		return result, nil
	}
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
			return uc.deps.SubmissionRepo.GetAll(ctx, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.SubmissionRepo.CountAll(ctx)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - GetAll: %w", err)
	}
	return result, nil
}

func (uc *SubmissionUseCase) GetStats(ctx context.Context, challengeID uuid.UUID, forceLive bool) (*domain.SubmissionStats, error) {
	comp, freezeTime := uc.getCompAndFreezeTime(ctx)
	if !forceLive && comp != nil && comp.IsFreezeActive() && comp.FreezeTime != nil {
		stats, err := uc.deps.SubmissionRepo.GetStatsFrozen(ctx, challengeID, *freezeTime)
		if err != nil {
			return nil, fmt.Errorf("SubmissionUseCase - GetStats - SubmissionRepo.GetStatsFrozen: %w", err)
		}
		return stats, nil
	}
	stats, err := uc.deps.SubmissionRepo.GetStats(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - GetStats - SubmissionRepo.GetStats: %w", err)
	}
	return stats, nil
}

func (uc *SubmissionUseCase) getCompAndFreezeTime(ctx context.Context) (*domain.Competition, *time.Time) {
	if uc.deps.CompGetter == nil {
		return nil, nil
	}
	comp, err := uc.deps.CompGetter.Get(ctx)
	if err != nil || comp == nil || comp.FreezeTime == nil {
		return comp, nil
	}
	return comp, comp.FreezeTime
}

func (uc *SubmissionUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*domain.SubmissionWithDetails, error) {
	sub, err := uc.deps.SubmissionRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - GetByID - SubmissionRepo.GetByID: %w", err)
	}
	return sub, nil
}

func (uc *SubmissionUseCase) Update(ctx context.Context, ID uuid.UUID, isCorrect bool) (*domain.SubmissionWithDetails, error) {
	if uc.deps.TM != nil && uc.deps.SolveCreator != nil && uc.deps.SolveDeleter != nil {
		var locked *domain.Submission
		if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
			// Re-read with FOR UPDATE inside the transaction to avoid TOCTOU on concurrent admin edits.
			var err error
			locked, err = uc.deps.SubmissionRepo.GetByIDForUpdate(ctx, ID)
			if err != nil {
				return fmt.Errorf("SubmissionUseCase - Update - SubmissionRepo.GetByIDForUpdate: %w", err)
			}
			if err := uc.deps.SubmissionRepo.Update(ctx, ID, isCorrect); err != nil {
				return fmt.Errorf("SubmissionUseCase - Update - SubmissionRepo.Update: %w", err)
			}
			switch {
			case locked.TeamID != nil && !locked.IsCorrect && isCorrect:
				if err := uc.deps.SolveCreator.AdminCreateSolve(ctx, locked.UserID, *locked.TeamID, locked.ChallengeID, true); err != nil {
					return fmt.Errorf("SubmissionUseCase - Update - SolveCreator.AdminCreateSolve: %w", err)
				}
			case locked.TeamID != nil && locked.IsCorrect && !isCorrect:
				if err := uc.deps.SolveDeleter.AdminDeleteSolve(ctx, *locked.TeamID, locked.ChallengeID); err != nil {
					return fmt.Errorf("SubmissionUseCase - Update - SolveDeleter.AdminDeleteSolve: %w", err)
				}
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("SubmissionUseCase - Update - TM.Run: %w", err)
		}
		if uc.deps.CacheInvalidator != nil {
			if locked != nil && locked.TeamID != nil {
				uc.deps.CacheInvalidator.InvalidateScoreboardCacheForTeam(ctx, *locked.TeamID)
			} else {
				uc.deps.CacheInvalidator.InvalidateScoreboardCache(ctx)
			}
		}
		sub, err := uc.deps.SubmissionRepo.GetByID(ctx, ID)
		if err != nil {
			return nil, fmt.Errorf("SubmissionUseCase - Update - SubmissionRepo.GetByID: %w", err)
		}
		return sub, nil
	}

	// Degraded path: TM/SolveCreator/SolveDeleter are nil (NewSubmissionUseCase panics if only some are set).
	// Non-transactional: concurrent admin updates can desync submission and solve state; log warning below.
	prev, err := uc.deps.SubmissionRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - Update - SubmissionRepo.GetByID: %w", err)
	}
	needsCreate := prev.TeamID != nil && !prev.IsCorrect && isCorrect
	needsDelete := prev.TeamID != nil && prev.IsCorrect && !isCorrect
	if (needsCreate || needsDelete) && (uc.deps.TM == nil || uc.deps.SolveCreator == nil) {
		uc.deps.Logger.WithFields(logkit.Fields{
			"submission_id": ID,
			"tm":            uc.deps.TM != nil,
			"solve_creator": uc.deps.SolveCreator != nil,
			"solve_deleter": uc.deps.SolveDeleter != nil,
		}).Warn("SubmissionUseCase - Update: using non-transactional path; submission/solve state may be inconsistent on partial failure")
	}
	if err = uc.deps.SubmissionRepo.Update(ctx, ID, isCorrect); err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - Update - SubmissionRepo.Update: %w", err)
	}
	if prev.TeamID != nil {
		switch {
		case needsCreate && uc.deps.SolveCreator != nil:
			if err = uc.deps.SolveCreator.AdminCreateSolve(ctx, prev.UserID, *prev.TeamID, prev.ChallengeID, true); err != nil {
				return nil, fmt.Errorf("SubmissionUseCase - Update - SolveCreator.AdminCreateSolve: %w", err)
			}
		case needsDelete && uc.deps.SolveDeleter != nil:
			if err = uc.deps.SolveDeleter.AdminDeleteSolve(ctx, *prev.TeamID, prev.ChallengeID); err != nil {
				return nil, fmt.Errorf("SubmissionUseCase - Update - SolveDeleter.AdminDeleteSolve: %w", err)
			}
		}
	}
	sub, err := uc.deps.SubmissionRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - Update - SubmissionRepo.GetByID: %w", err)
	}
	return sub, nil
}

func (uc *SubmissionUseCase) Delete(ctx context.Context, ID uuid.UUID) error {
	var teamIDToInvalidate *uuid.UUID
	if uc.deps.TM != nil && uc.deps.SolveDeleter != nil {
		if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
			locked, err := uc.deps.SubmissionRepo.GetByIDForUpdate(ctx, ID)
			if err != nil {
				if errors.Is(err, httperr.ErrSubmissionNotFound) {
					return nil
				}
				return fmt.Errorf("SubmissionUseCase - Delete - SubmissionRepo.GetByIDForUpdate: %w", err)
			}
			if locked.IsCorrect && locked.TeamID != nil {
				if err := uc.deps.SolveDeleter.AdminDeleteSolve(ctx, *locked.TeamID, locked.ChallengeID); err != nil {
					return fmt.Errorf("SubmissionUseCase - Delete - SolveDeleter.AdminDeleteSolve: %w", err)
				}
				teamIDToInvalidate = locked.TeamID
			}
			if err := uc.deps.SubmissionRepo.Delete(ctx, ID); err != nil {
				return fmt.Errorf("SubmissionUseCase - Delete - SubmissionRepo.Delete: %w", err)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("SubmissionUseCase - Delete - TM.Run: %w", err)
		}
	} else {
		sub, err := uc.deps.SubmissionRepo.GetByID(ctx, ID)
		if err != nil {
			if errors.Is(err, httperr.ErrSubmissionNotFound) {
				return nil
			}
			return fmt.Errorf("SubmissionUseCase - Delete - SubmissionRepo.GetByID: %w", err)
		}
		if err := uc.deps.SubmissionRepo.Delete(ctx, ID); err != nil {
			return fmt.Errorf("SubmissionUseCase - Delete - SubmissionRepo.Delete: %w", err)
		}
		if sub.IsCorrect && sub.TeamID != nil {
			teamIDToInvalidate = sub.TeamID
		}
	}
	if uc.deps.CacheInvalidator != nil && teamIDToInvalidate != nil {
		uc.deps.CacheInvalidator.InvalidateScoreboardCacheForTeam(ctx, *teamIDToInvalidate)
	}
	return nil
}

func (uc *SubmissionUseCase) AdminCreate(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID, challengeID uuid.UUID, submittedFlag string, isCorrect bool, ip string) (*domain.SubmissionWithDetails, error) {
	if isCorrect && teamID == nil {
		return nil, httperr.NewValidationErrorf("team_id is required when is_correct is true")
	}
	if uc.deps.UserRepo != nil {
		u, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			uc.deps.Logger.WithError(err).WithFields(logkit.UserID(userID.String())).Warn("SubmissionUseCase - AdminCreate: failed to check user ban status")
		} else if u.IsBanned {
			return nil, httperr.ErrUserBanned
		}
	}
	if teamID != nil && uc.deps.TeamRepo != nil {
		t, err := uc.deps.TeamRepo.GetByID(ctx, *teamID)
		if err != nil {
			uc.deps.Logger.WithError(err).WithFields(logkit.Fields{"team_id": teamID.String()}).Warn("SubmissionUseCase - AdminCreate: failed to check team ban status")
		} else if t.IsBanned {
			return nil, httperr.ErrTeamBanned
		}
	}
	sub := &domain.Submission{
		ID:            uuid.New(),
		UserID:        userID,
		TeamID:        teamID,
		ChallengeID:   challengeID,
		SubmittedFlag: submittedFlag,
		IsCorrect:     isCorrect,
		IP:            ip,
		CreatedAt:     time.Now(),
	}

	if isCorrect && teamID != nil && uc.deps.TM != nil && uc.deps.SolveCreator != nil {
		if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
			if err := uc.deps.SubmissionRepo.Create(ctx, sub); err != nil {
				return fmt.Errorf("SubmissionUseCase - AdminCreate - SubmissionRepo.Create: %w", err)
			}
			if err := uc.deps.SolveCreator.AdminCreateSolve(ctx, userID, *teamID, challengeID, true); err != nil {
				return fmt.Errorf("SubmissionUseCase - AdminCreate - SolveCreator.AdminCreateSolve: %w", err)
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("SubmissionUseCase - AdminCreate - TM.Run: %w", err)
		}
		if uc.deps.CacheInvalidator != nil {
			if teamID != nil {
				uc.deps.CacheInvalidator.InvalidateScoreboardCacheForTeam(ctx, *teamID)
			} else {
				uc.deps.CacheInvalidator.InvalidateScoreboardCache(ctx)
			}
		}
	} else {
		if isCorrect && teamID != nil && (uc.deps.TM == nil || uc.deps.SolveCreator == nil) {
			return nil, fmt.Errorf("SubmissionUseCase - AdminCreate: transaction and solve creator required for correct submission")
		}
		if err := uc.deps.SubmissionRepo.Create(ctx, sub); err != nil {
			return nil, fmt.Errorf("SubmissionUseCase - AdminCreate - SubmissionRepo.Create: %w", err)
		}
	}

	result, err := uc.deps.SubmissionRepo.GetByID(ctx, sub.ID)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - AdminCreate - SubmissionRepo.GetByID: %w", err)
	}
	return result, nil
}
