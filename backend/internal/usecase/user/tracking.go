package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type TrackingDeps struct {
	TrackingRepo repo.TrackingRepository
}

type TrackingUseCase struct {
	deps TrackingDeps
}

var _ usecase.TrackingUseCase = (*TrackingUseCase)(nil)

func NewTrackingUseCase(deps TrackingDeps) *TrackingUseCase {
	return &TrackingUseCase{deps: deps}
}

func (uc *TrackingUseCase) Track(ctx context.Context, userID uuid.UUID, ip, userAgent string) error {
	entry := &domain.TrackingEntry{
		UserID:    userID,
		IP:        ip,
		UserAgent: userAgent,
	}

	err := uc.deps.TrackingRepo.Create(ctx, entry)
	if err != nil {
		return fmt.Errorf("TrackingUseCase - Track - TrackingRepo.Create: %w", err)
	}

	return nil
}

func (uc *TrackingUseCase) GetByUser(ctx context.Context, userID uuid.UUID, page, perPage int) (*usecase.Paginated[*domain.TrackingEntry], error) {
	offset := (page - 1) * perPage

	entries, err := uc.deps.TrackingRepo.GetByUser(ctx, userID, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("TrackingUseCase - GetByUser - TrackingRepo.GetByUser: %w", err)
	}

	total, err := uc.deps.TrackingRepo.CountByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("TrackingUseCase - GetByUser - TrackingRepo.CountByUser: %w", err)
	}

	return usecase.NewPaginated(entries, int64(total), page, perPage), nil
}

func (uc *TrackingUseCase) TrackChallengeOpen(ctx context.Context, userID, challengeID uuid.UUID, ip string) error {
	entry := &domain.ChallengeOpen{
		UserID:      userID,
		ChallengeID: challengeID,
		IP:          ip,
	}

	err := uc.deps.TrackingRepo.CreateChallengeOpen(ctx, entry)
	if err != nil {
		return fmt.Errorf("TrackingUseCase - TrackChallengeOpen - TrackingRepo.CreateChallengeOpen: %w", err)
	}

	return nil
}
