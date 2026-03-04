package competition

import (
	"context"
	"fmt"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/google/uuid"
)

// AdminSolveCreator creates a solve record without flag validation.
type AdminSolveCreator interface {
	AdminCreateSolve(ctx context.Context, userID, teamID, challengeID uuid.UUID) error
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

type SubmissionUseCase struct {
	deps SubmissionDeps
}

var _ usecase.SubmissionUseCase = (*SubmissionUseCase)(nil)

type SubmissionDeps struct {
	SubmissionRepo   repo.SubmissionRepository
	TM               repo.TransactionManager
	SolveCreator     AdminSolveCreator
	SolveDeleter     AdminSolveDeleter
	CacheInvalidator CacheInvalidator
	Logger           logger.Logger
}

func NewSubmissionUseCase(deps SubmissionDeps) *SubmissionUseCase {
	if deps.Logger == nil {
		deps.Logger = logger.Noop()
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

func (uc *SubmissionUseCase) LogSubmission(ctx context.Context, sub *entity.Submission) error {
	if err := uc.deps.SubmissionRepo.Create(ctx, sub); err != nil {
		return fmt.Errorf("SubmissionUseCase - LogSubmission - SubmissionRepo.Create: %w", err)
	}
	return nil
}

func (uc *SubmissionUseCase) GetByChallenge(ctx context.Context, challengeID uuid.UUID, page, perPage int) (*usecase.Paginated[*entity.SubmissionWithDetails], error) {
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*entity.SubmissionWithDetails, error) {
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

func (uc *SubmissionUseCase) GetByUser(ctx context.Context, userID uuid.UUID, page, perPage int) (*usecase.Paginated[*entity.SubmissionWithDetails], error) {
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*entity.SubmissionWithDetails, error) {
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

func (uc *SubmissionUseCase) GetByTeam(ctx context.Context, teamID uuid.UUID, page, perPage int) (*usecase.Paginated[*entity.SubmissionWithDetails], error) {
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*entity.SubmissionWithDetails, error) {
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

func (uc *SubmissionUseCase) GetAll(ctx context.Context, page, perPage int) (*usecase.Paginated[*entity.SubmissionWithDetails], error) {
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*entity.SubmissionWithDetails, error) {
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

func (uc *SubmissionUseCase) GetStats(ctx context.Context, challengeID uuid.UUID) (*entity.SubmissionStats, error) {
	stats, err := uc.deps.SubmissionRepo.GetStats(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - GetStats - SubmissionRepo.GetStats: %w", err)
	}
	return stats, nil
}

func (uc *SubmissionUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*entity.SubmissionWithDetails, error) {
	sub, err := uc.deps.SubmissionRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - GetByID - SubmissionRepo.GetByID: %w", err)
	}
	return sub, nil
}

//nolint:gocognit,gocyclo
func (uc *SubmissionUseCase) Update(ctx context.Context, ID uuid.UUID, isCorrect bool) (*entity.SubmissionWithDetails, error) {
	if uc.deps.TM != nil && uc.deps.SolveCreator != nil && uc.deps.SolveDeleter != nil {
		var locked *entity.Submission
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
				if err := uc.deps.SolveCreator.AdminCreateSolve(ctx, locked.UserID, *locked.TeamID, locked.ChallengeID); err != nil {
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

	// Degraded path - only reached when tm/solve dependencies are missing.
	prev, err := uc.deps.SubmissionRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - Update - SubmissionRepo.GetByID: %w", err)
	}
	needsCreate := prev.TeamID != nil && !prev.IsCorrect && isCorrect
	needsDelete := prev.TeamID != nil && prev.IsCorrect && !isCorrect
	if (needsCreate || needsDelete) && (uc.deps.TM == nil || uc.deps.SolveCreator == nil) {
		uc.deps.Logger.WithFields(logger.Fields{
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
			if err = uc.deps.SolveCreator.AdminCreateSolve(ctx, prev.UserID, *prev.TeamID, prev.ChallengeID); err != nil {
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
	if err := uc.deps.SubmissionRepo.Delete(ctx, ID); err != nil {
		return fmt.Errorf("SubmissionUseCase - Delete - SubmissionRepo.Delete: %w", err)
	}
	return nil
}

//nolint:gocognit,gocyclo
func (uc *SubmissionUseCase) AdminCreate(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID, challengeID uuid.UUID, submittedFlag string, isCorrect bool, ip string) (*entity.SubmissionWithDetails, error) {
	sub := &entity.Submission{
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
			if err := uc.deps.SolveCreator.AdminCreateSolve(ctx, userID, *teamID, challengeID); err != nil {
				return fmt.Errorf("SubmissionUseCase - AdminCreate - SolveCreator.AdminCreateSolve: %w", err)
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("SubmissionUseCase - AdminCreate - TM.Run: %w", err)
		}
		if uc.deps.CacheInvalidator != nil {
			uc.deps.CacheInvalidator.InvalidateScoreboardCache(ctx)
			if teamID != nil {
				uc.deps.CacheInvalidator.InvalidateScoreboardCacheForTeam(ctx, *teamID)
			}
		}
	} else {
		if (isCorrect && teamID != nil) && (uc.deps.TM == nil || uc.deps.SolveCreator == nil) {
			uc.deps.Logger.WithFields(logger.Fields{
				"submission_id": sub.ID,
				"tm":            uc.deps.TM != nil,
				"solve_creator": uc.deps.SolveCreator != nil,
			}).Warn("SubmissionUseCase - AdminCreate: using non-transactional path; submission/solve state may be inconsistent on partial failure")
		}
		if err := uc.deps.SubmissionRepo.Create(ctx, sub); err != nil {
			return nil, fmt.Errorf("SubmissionUseCase - AdminCreate - SubmissionRepo.Create: %w", err)
		}
		if isCorrect && teamID != nil && uc.deps.SolveCreator != nil {
			if err := uc.deps.SolveCreator.AdminCreateSolve(ctx, userID, *teamID, challengeID); err != nil {
				return nil, fmt.Errorf("SubmissionUseCase - AdminCreate - SolveCreator.AdminCreateSolve: %w", err)
			}
		}
	}

	result, err := uc.deps.SubmissionRepo.GetByID(ctx, sub.ID)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - AdminCreate - SubmissionRepo.GetByID: %w", err)
	}
	return result, nil
}
