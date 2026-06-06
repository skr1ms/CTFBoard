package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

// ListUsers returns a paginated user list inside a read-only transaction.
// When field=="ip" the search targets tracked IP addresses; otherwise it
// performs a standard username/email search.
func (uc *UserUseCase) ListUsers(ctx context.Context, search *string, field string, page, perPage int) (*usecase.Paginated[*domain.User], error) {
	offset := (page - 1) * perPage

	var (
		users []*domain.User
		total int64
	)

	err := uc.deps.TM.ReadOnly(ctx, func(roCtx context.Context) error {
		if field == "ip" && search != nil && *search != "" {
			var err error

			users, err = uc.deps.UserRepo.SearchByIP(roCtx, *search, perPage, offset)
			if err != nil {
				return fmt.Errorf("UserUseCase - ListUsers - UserRepo.SearchByIP: %w", err)
			}

			total, err = uc.deps.UserRepo.CountSearchByIP(roCtx, *search)
			if err != nil {
				return fmt.Errorf("UserUseCase - ListUsers - UserRepo.CountSearchByIP: %w", err)
			}

			return nil
		}

		var err error

		users, err = uc.deps.UserRepo.Search(roCtx, search, perPage, offset)
		if err != nil {
			return fmt.Errorf("UserUseCase - ListUsers - UserRepo.Search: %w", err)
		}

		total, err = uc.deps.UserRepo.CountSearch(roCtx, search)
		if err != nil {
			return fmt.Errorf("UserUseCase - ListUsers - UserRepo.CountSearch: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - ListUsers - TM.ReadOnly: %w", err)
	}

	return usecase.NewPaginated(users, total, page, perPage), nil
}

func (uc *UserUseCase) GetUserSolves(ctx context.Context, userID uuid.UUID) ([]*domain.SolveWithDetails, error) {
	if _, err := uc.deps.UserRepo.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserSolves - UserRepo.GetByID: %w", err)
	}

	solves, err := uc.deps.SolveRepo.GetByUserIDWithDetails(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserSolves - SolveRepo.GetByUserIDWithDetails: %w", err)
	}

	return scoring.FilterSolveDetailsByFreezeFromRepo(ctx, uc.deps.CompRepo, solves)
}

func (uc *UserUseCase) GetUserFails(ctx context.Context, userID uuid.UUID, page, perPage int) (*usecase.Paginated[*domain.SubmissionWithDetails], error) {
	if _, err := uc.deps.UserRepo.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserFails - UserRepo.GetByID: %w", err)
	}

	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
			return uc.deps.SubmissionRepo.GetFailsByUser(ctx, userID, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.SubmissionRepo.CountFailsByUser(ctx, userID)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserFails: %w", err)
	}

	return result, nil
}

func (uc *UserUseCase) GetUserAwards(ctx context.Context, userID uuid.UUID) ([]*domain.Award, error) {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserAwards - UserRepo.GetByID: %w", err)
	}

	if user.TeamID == nil {
		return []*domain.Award{}, nil
	}

	if uc.deps.AwardRepo == nil {
		return []*domain.Award{}, nil
	}

	awards, err := uc.deps.AwardRepo.GetByTeamID(ctx, *user.TeamID)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetUserAwards - AwardRepo.GetByTeamID: %w", err)
	}

	return awards, nil
}

func (uc *UserUseCase) GetMySubmissions(ctx context.Context, userID uuid.UUID, page, perPage int) (*usecase.Paginated[*domain.SubmissionWithDetails], error) {
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
			return uc.deps.SubmissionRepo.GetByUser(ctx, userID, nil, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.SubmissionRepo.CountByUser(ctx, userID, nil)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - GetMySubmissions: %w", err)
	}

	return result, nil
}
