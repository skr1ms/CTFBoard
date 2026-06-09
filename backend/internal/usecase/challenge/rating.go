package challenge

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

type RatingUseCase struct {
	deps RatingDeps
}

type RatingDeps struct {
	ChallengeRepo repo.ChallengeRepository
	SolveRepo     repo.SolveRepository
	RatingRepo    repo.RatingRepository
	UserRepo      repo.UserRepository
	TeamRepo      repo.TeamRepository
	TM            repo.TransactionManager
}

var _ usecase.RatingUseCase = (*RatingUseCase)(nil)

func NewRatingUseCase(deps RatingDeps) *RatingUseCase {
	return &RatingUseCase{deps: deps}
}

func (uc *RatingUseCase) PutRating(ctx context.Context, challengeID, userID, teamID uuid.UUID, value int, review string) (*domain.Rating, error) {
	if uc.deps.ChallengeRepo == nil {
		return nil, fmt.Errorf("RatingUseCase - PutRating: ChallengeRepo not configured")
	}

	if uc.deps.SolveRepo == nil {
		return nil, fmt.Errorf("RatingUseCase - PutRating: SolveRepo not configured")
	}

	if uc.deps.RatingRepo == nil {
		return nil, fmt.Errorf("RatingUseCase - PutRating: RatingRepo not configured")
	}

	if uc.deps.UserRepo == nil {
		return nil, fmt.Errorf("RatingUseCase - PutRating: UserRepo not configured")
	}

	if uc.deps.TeamRepo == nil {
		return nil, fmt.Errorf("RatingUseCase - PutRating: TeamRepo not configured")
	}

	ch, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		if errors.Is(err, apperr.ErrChallengeNotFound) {
			return nil, apperr.ErrChallengeNotFound
		}

		return nil, fmt.Errorf("RatingUseCase - PutRating - ChallengeRepo.GetByID: %w", err)
	}

	if err := guard.EnsureChallengeVisible(ch); err != nil {
		return nil, err
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("RatingUseCase - PutRating - UserRepo.GetByID: %w", err)
	}

	if user.IsBanned {
		return nil, apperr.ErrUserBanned
	}

	if user.WasInBannedTeam && user.Role != domain.RoleAdmin {
		return nil, apperr.ErrUserWasInBannedTeam
	}

	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("RatingUseCase - PutRating - TeamRepo.GetByID: %w", err)
	}

	if team.IsBanned {
		return nil, apperr.ErrTeamBanned
	}

	rating := &domain.Rating{
		ChallengeID: challengeID,
		UserID:      userID,
		TeamID:      teamID,
		Value:       value,
		Review:      review,
	}

	if uc.deps.TM == nil {
		return nil, fmt.Errorf("RatingUseCase - PutRating: TransactionManager not configured")
	}

	err = uc.deps.TM.Run(ctx, func(txCtx context.Context) error {
		if err := uc.deps.UserRepo.Lock(txCtx, userID); err != nil {
			return fmt.Errorf("RatingUseCase - PutRating - UserRepo.Lock: %w", err)
		}

		freshUser, err := uc.deps.UserRepo.GetByID(txCtx, userID)
		if err != nil {
			return fmt.Errorf("RatingUseCase - PutRating - UserRepo.GetByID (fresh): %w", err)
		}

		if freshUser.IsBanned {
			return apperr.ErrUserBanned
		}

		if freshUser.TeamID == nil || *freshUser.TeamID != teamID {
			return apperr.ErrUserNotInTeam
		}

		if err := uc.deps.TeamRepo.Lock(txCtx, teamID); err != nil {
			return fmt.Errorf("RatingUseCase - PutRating - TeamRepo.Lock: %w", err)
		}

		freshTeam, err := uc.deps.TeamRepo.GetByID(txCtx, teamID)
		if err != nil {
			return fmt.Errorf("RatingUseCase - PutRating - TeamRepo.GetByID (fresh): %w", err)
		}

		if err := guard.ValidateSubmissionEligibility(txCtx, freshUser, freshTeam, nil, uc.deps.TeamRepo); err != nil {
			return err
		}

		if _, err := uc.deps.SolveRepo.GetByTeamAndChallengeForUpdate(txCtx, teamID, challengeID); err != nil {
			if errors.Is(err, apperr.ErrSolveNotFound) {
				return apperr.ErrSolveRequiredForRating
			}

			return fmt.Errorf("RatingUseCase - PutRating - SolveRepo.GetByTeamAndChallengeForUpdate: %w", err)
		}

		if err := uc.deps.RatingRepo.Upsert(txCtx, rating); err != nil {
			return fmt.Errorf("RatingUseCase - PutRating - RatingRepo.Upsert: %w", err)
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, apperr.ErrSolveRequiredForRating) {
			return nil, apperr.ErrSolveRequiredForRating
		}

		return nil, fmt.Errorf("RatingUseCase - PutRating - TM.Run: %w", err)
	}

	return rating, nil
}

func (uc *RatingUseCase) GetRatingsByChallengeID(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*domain.Rating, error) {
	if uc.deps.ChallengeRepo == nil {
		return nil, fmt.Errorf("RatingUseCase - GetRatingsByChallengeID: ChallengeRepo not configured")
	}

	if uc.deps.RatingRepo == nil {
		return nil, fmt.Errorf("RatingUseCase - GetRatingsByChallengeID: RatingRepo not configured")
	}

	ch, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		if errors.Is(err, apperr.ErrChallengeNotFound) {
			return nil, apperr.ErrChallengeNotFound
		}

		return nil, fmt.Errorf("RatingUseCase - GetRatingsByChallengeID - ChallengeRepo.GetByID: %w", err)
	}

	if err := guard.EnsureChallengeVisible(ch); err != nil {
		return nil, err
	}

	if err := ensureRequirementsSatisfiedForRead(ctx, challengeID, teamID, uc.deps.ChallengeRepo, uc.deps.SolveRepo, "RatingUseCase - GetRatingsByChallengeID"); err != nil {
		return nil, err
	}

	list, err := uc.deps.RatingRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("RatingUseCase - GetRatingsByChallengeID - RatingRepo.GetByChallengeID: %w", err)
	}

	return list, nil
}
