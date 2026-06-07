package user

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	userMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mock"
)

type appealTestDeps struct {
	repo     *userMock.MockBanAppealRepository
	userRepo *userMock.MockUserRepository
	tm       *userMock.MockTransactionManager
}

func newAppealTestDeps(t *testing.T) *appealTestDeps {
	t.Helper()

	return &appealTestDeps{
		repo:     userMock.NewMockBanAppealRepository(t),
		userRepo: userMock.NewMockUserRepository(t),
		tm:       userMock.NewMockTransactionManager(t),
	}
}

func (d *appealTestDeps) createUseCase() *BanAppealUseCase {
	return NewBanAppealUseCase(d.repo, d.userRepo, d.tm)
}

func (d *appealTestDeps) setupTxRun() {
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).Maybe()
}

func newTestAppeal(userID uuid.UUID, createdAt time.Time, decision domain.AppealDecision) *domain.BanAppeal {
	return &domain.BanAppeal{
		ID:        uuid.New(),
		UserID:    userID,
		Message:   "please unban me",
		Decision:  decision,
		CreatedAt: createdAt,
	}
}
