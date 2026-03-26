package challenge

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
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
		return nil, fmt.Errorf("rating: ChallengeRepo not configured")
	}

	if uc.deps.SolveRepo == nil {
		return nil, fmt.Errorf("rating: SolveRepo not configured")
	}

	if uc.deps.RatingRepo == nil {
		return nil, fmt.Errorf("rating: RatingRepo not configured")
	}

	ch, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		if errors.Is(err, httperr.ErrChallengeNotFound) {
			return nil, httperr.ErrChallengeNotFound
		}

		return nil, fmt.Errorf("RatingUseCase - PutRating - ChallengeRepo.GetByID: %w", err)
	}

	if ch.State == domain.ChallengeStateHidden {
		return nil, httperr.ErrChallengeNotFound
	}

	if uc.deps.UserRepo != nil {
		user, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("RatingUseCase - PutRating - UserRepo.GetByID: %w", err)
		}

		if user.IsBanned {
			return nil, httperr.ErrUserBanned
		}
	}

	if uc.deps.TeamRepo != nil {
		team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return nil, fmt.Errorf("RatingUseCase - PutRating - TeamRepo.GetByID: %w", err)
		}

		if team.IsBanned {
			return nil, httperr.ErrTeamBanned
		}
	}

	rating := &domain.Rating{
		ChallengeID: challengeID,
		UserID:      userID,
		TeamID:      teamID,
		Value:       value,
		Review:      review,
	}

	if uc.deps.TM == nil {
		return nil, fmt.Errorf("rating: TransactionManager not configured")
	}

	err = uc.deps.TM.Run(ctx, func(txCtx context.Context) error {
		if _, err := uc.deps.SolveRepo.GetByTeamAndChallengeForUpdate(txCtx, teamID, challengeID); err != nil {
			if errors.Is(err, httperr.ErrSolveNotFound) {
				return httperr.ErrSolveRequiredForRating
			}

			return fmt.Errorf("SolveRepo.GetByTeamAndChallengeForUpdate: %w", err)
		}

		err := uc.deps.RatingRepo.Upsert(txCtx, rating)
		if err != nil {
			return fmt.Errorf("RatingRepo.Upsert: %w", err)
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, httperr.ErrSolveRequiredForRating) {
			return nil, httperr.ErrSolveRequiredForRating
		}

		return nil, fmt.Errorf("RatingUseCase - PutRating - TM.Run: %w", err)
	}

	return rating, nil
}

func (uc *RatingUseCase) GetRatingsByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Rating, error) {
	if uc.deps.ChallengeRepo == nil {
		return nil, fmt.Errorf("rating: ChallengeRepo not configured")
	}

	if uc.deps.RatingRepo == nil {
		return nil, fmt.Errorf("rating: RatingRepo not configured")
	}

	ch, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		if errors.Is(err, httperr.ErrChallengeNotFound) {
			return nil, httperr.ErrChallengeNotFound
		}

		return nil, fmt.Errorf("RatingUseCase - GetRatingsByChallengeID - ChallengeRepo.GetByID: %w", err)
	}

	if ch.State == domain.ChallengeStateHidden {
		return nil, httperr.ErrChallengeNotFound
	}

	list, err := uc.deps.RatingRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("RatingUseCase - GetRatingsByChallengeID - RatingRepo.GetByChallengeID: %w", err)
	}

	return list, nil
}
