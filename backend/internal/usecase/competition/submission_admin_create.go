package competition

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

// AdminCreate creates a submission record on behalf of an admin. It checks
// user and team ban status before persisting. When isCorrect is true and
// teamID is provided it additionally calls RecordSolveInTx with dynamic score
// decay inside a transaction so the solve and submission are written atomically
// the scoreboard cache is then invalidated for the affected team. Passing
// isCorrect true without a teamID is rejected with a validation error. If TM
// or SolveCreator are absent and a correct submission with a team is requested,
// the call returns an error rather than silently skipping the solve.
func (uc *SubmissionUseCase) AdminCreate(ctx context.Context, params usecase.AdminCreateSubmissionParams) (*domain.SubmissionWithDetails, error) {
	if params.IsCorrect && params.TeamID == nil {
		return nil, apperr.NewValidationErrorf("team_id is required when is_correct is true")
	}

	if uc.deps.UserRepo != nil {
		u, err := uc.deps.UserRepo.GetByID(ctx, params.UserID)
		if err != nil {
			uc.deps.Logger.WithError(err).WithFields(logkit.UserID(params.UserID.String())).Warn("SubmissionUseCase - AdminCreate: failed to check user ban status")
		} else if u.IsBanned {
			return nil, apperr.ErrUserBanned
		}
	}

	if params.TeamID != nil && uc.deps.TeamRepo != nil {
		t, err := uc.deps.TeamRepo.GetByID(ctx, *params.TeamID)
		if err != nil {
			uc.deps.Logger.WithError(err).WithFields(logkit.Fields{"team_id": params.TeamID.String()}).Warn("SubmissionUseCase - AdminCreate: failed to check team ban status")
		} else if t.IsBanned {
			return nil, apperr.ErrTeamBanned
		}
	}

	sub := &domain.Submission{
		ID:            uuid.New(),
		UserID:        params.UserID,
		TeamID:        params.TeamID,
		ChallengeID:   params.ChallengeID,
		SubmittedFlag: params.SubmittedFlag,
		IsCorrect:     params.IsCorrect,
		IP:            params.IP,
		CreatedAt:     time.Now(),
	}

	if params.IsCorrect && params.TeamID != nil && uc.deps.TM != nil && uc.deps.SolveCreator != nil {
		if err := uc.requireTxDeps("AdminCreate", true); err != nil {
			return nil, err
		}

		if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
			if err := uc.deps.SubmissionRepo.Create(ctx, sub); err != nil {
				return fmt.Errorf("SubmissionUseCase - AdminCreate - SubmissionRepo.Create: %w", err)
			}

			if err := uc.deps.SolveCreator.AdminCreateSolve(ctx, params.UserID, *params.TeamID, params.ChallengeID, true); err != nil {
				return fmt.Errorf("SubmissionUseCase - AdminCreate - SolveCreator.AdminCreateSolve: %w", err)
			}

			return nil
		}); err != nil {
			return nil, fmt.Errorf("SubmissionUseCase - AdminCreate - TM.Run: %w", err)
		}

		if uc.deps.CacheInvalidator != nil {
			uc.deps.CacheInvalidator.InvalidateScoreboardCacheForTeam(ctx, *params.TeamID)
		}
	} else {
		if params.IsCorrect && params.TeamID != nil {
			return nil, fmt.Errorf("SubmissionUseCase - AdminCreate: transaction and solve creator required for correct submission")
		}

		if err := uc.deps.SubmissionRepo.Create(ctx, sub); err != nil {
			return nil, fmt.Errorf("SubmissionUseCase - AdminCreate - SubmissionRepo.Create: %w", err)
		}
	}

	uc.invalidateStatisticsCache(ctx, "AdminCreate")

	result, err := uc.deps.SubmissionRepo.GetByID(ctx, sub.ID)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - AdminCreate - SubmissionRepo.GetByID: %w", err)
	}

	return result, nil
}
