package competition

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// AdminCreate creates a submission record on behalf of an admin. It checks
// user and team ban status before persisting. When isCorrect is true and
// teamID is provided it additionally calls RecordSolveInTx with dynamic score
// decay inside a transaction so the solve and submission are written atomically
// the scoreboard cache is then invalidated for the affected team. Passing
// isCorrect true without a teamID is rejected with a validation error. If TM
// or SolveCreator are absent and a correct submission with a team is requested,
// the call returns an error rather than silently skipping the solve.
func (uc *SubmissionUseCase) AdminCreate(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID, challengeID uuid.UUID, submittedFlag string, isCorrect bool, ip string) (*domain.SubmissionWithDetails, error) {
	if isCorrect && teamID == nil {
		return nil, apperr.NewValidationErrorf("team_id is required when is_correct is true")
	}

	if uc.deps.UserRepo != nil {
		u, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			uc.deps.Logger.WithError(err).WithFields(logkit.UserID(userID.String())).Warn("SubmissionUseCase - AdminCreate: failed to check user ban status")
		} else if u.IsBanned {
			return nil, apperr.ErrUserBanned
		}
	}

	if teamID != nil && uc.deps.TeamRepo != nil {
		t, err := uc.deps.TeamRepo.GetByID(ctx, *teamID)
		if err != nil {
			uc.deps.Logger.WithError(err).WithFields(logkit.Fields{"team_id": teamID.String()}).Warn("SubmissionUseCase - AdminCreate: failed to check team ban status")
		} else if t.IsBanned {
			return nil, apperr.ErrTeamBanned
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
		if err := uc.requireTxDeps("AdminCreate", true); err != nil {
			return nil, err
		}

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
			uc.deps.CacheInvalidator.InvalidateScoreboardCacheForTeam(ctx, *teamID)
		}
	} else {
		if isCorrect && teamID != nil {
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
