package user

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

const appealCooldown = 7 * 24 * time.Hour

// BanAppealUseCase handles creation and review of ban appeals.
type BanAppealUseCase struct {
	appealRepo repo.BanAppealRepository
	userRepo   repo.UserRepository
	tm         repo.TransactionManager
}

var _ usecase.BanAppealUseCase = (*BanAppealUseCase)(nil)

func NewBanAppealUseCase(
	appealRepo repo.BanAppealRepository,
	userRepo repo.UserRepository,
	tm repo.TransactionManager,
) *BanAppealUseCase {
	return &BanAppealUseCase{
		appealRepo: appealRepo,
		userRepo:   userRepo,
		tm:         tm,
	}
}

// CreateAppeal creates a new ban appeal for the given user. Returns
// ErrAppealRateLimited when a previous appeal exists within the cooldown window.
func (uc *BanAppealUseCase) CreateAppeal(ctx context.Context, userID uuid.UUID, message string) (*domain.BanAppeal, error) {
	latest, err := uc.appealRepo.GetLatestByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if latest != nil && time.Since(latest.CreatedAt) < appealCooldown {
		return nil, apperr.ErrAppealRateLimited
	}

	appeal := &domain.BanAppeal{
		UserID:   userID,
		Message:  message,
		Decision: domain.AppealDecisionPending,
	}

	if err := uc.appealRepo.Create(ctx, appeal); err != nil {
		return nil, err
	}

	return appeal, nil
}

// CanAppeal reports whether the user is eligible to submit a new appeal.
// Returns (true, false, nil) when the user has no pending appeal and is outside
// the cooldown window. Returns (false, true, nil) when a pending appeal exists.
// Returns (false, false, nil) when within the cooldown window.
func (uc *BanAppealUseCase) CanAppeal(ctx context.Context, userID uuid.UUID) (bool, bool, error) {
	latest, err := uc.appealRepo.GetLatestByUserID(ctx, userID)
	if err != nil {
		return false, false, err
	}

	if latest == nil {
		return true, false, nil
	}

	if latest.Decision == domain.AppealDecisionPending {
		return false, true, nil
	}

	if time.Since(latest.CreatedAt) < appealCooldown {
		return false, false, nil
	}

	return true, false, nil
}

// GetAppealsByUser returns all appeals for the given user.
func (uc *BanAppealUseCase) GetAppealsByUser(ctx context.Context, userID uuid.UUID) ([]*domain.BanAppeal, error) {
	return uc.appealRepo.GetByUserID(ctx, userID)
}

// ListAppeals returns a paginated list of appeals, optionally filtered by decision.
func (uc *BanAppealUseCase) ListAppeals(ctx context.Context, decision *domain.AppealDecision, page, perPage int) (*usecase.Paginated[*domain.BanAppeal], error) {
	if page < 1 {
		page = 1
	}

	if perPage < 1 || perPage > usecase.DefaultMaxPerPage {
		perPage = usecase.DefaultPerPage
	}

	offset := (page - 1) * perPage

	appeals, total, err := uc.appealRepo.List(ctx, decision, perPage, offset)
	if err != nil {
		return nil, err
	}

	return usecase.NewPaginated(appeals, total, page, perPage), nil
}

// ReviewAppeal reviews an existing appeal. When decision is resolved, the user is automatically unbanned.
func (uc *BanAppealUseCase) ReviewAppeal(ctx context.Context, appealID uuid.UUID, decision domain.AppealDecision, adminResponse *string, actorID uuid.UUID) (*domain.BanAppeal, error) {
	appeal, err := uc.appealRepo.GetByID(ctx, appealID)
	if err != nil {
		return nil, err
	}

	if appeal.Decision != domain.AppealDecisionPending {
		return nil, apperr.ErrAccessDenied
	}

	appeal.Decision = decision
	appeal.AdminResponse = adminResponse

	if decision == domain.AppealDecisionResolved {
		err = uc.tm.Run(ctx, func(ctx context.Context) error {
			if err := uc.appealRepo.Update(ctx, appeal); err != nil {
				return err
			}

			return uc.userRepo.Unban(ctx, appeal.UserID)
		})
	} else {
		err = uc.appealRepo.Update(ctx, appeal)
	}

	if err != nil {
		return nil, err
	}

	return appeal, nil
}
