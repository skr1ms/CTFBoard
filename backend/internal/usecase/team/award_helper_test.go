package team

import (
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team/mocks"
	"github.com/google/uuid"
)

type AwardTestHelper struct {
	t       *testing.T
	repo    *mocks.MockAwardRepository
	tm      *mocks.MockTransactionManager
	useCase *AwardUseCase
	teamID  uuid.UUID
	adminID uuid.UUID
}

func NewAwardTestHelper(t *testing.T) *AwardTestHelper {
	t.Helper()
	repo := mocks.NewMockAwardRepository(t)
	tm := mocks.NewMockTransactionManager(t)
	uc := NewAwardUseCase(AwardDeps{AwardRepo: repo, TM: tm})
	return &AwardTestHelper{
		t:       t,
		repo:    repo,
		tm:      tm,
		useCase: uc,
		teamID:  uuid.New(),
		adminID: uuid.New(),
	}
}

func (h *AwardTestHelper) CreateUseCase() *AwardUseCase {
	h.t.Helper()
	return h.useCase
}

func (h *AwardTestHelper) Repo() *mocks.MockAwardRepository {
	h.t.Helper()
	return h.repo
}

func (h *AwardTestHelper) TM() *mocks.MockTransactionManager {
	h.t.Helper()
	return h.tm
}

func (h *AwardTestHelper) TeamID() uuid.UUID {
	return h.teamID
}

func (h *AwardTestHelper) AdminID() uuid.UUID {
	return h.adminID
}

func (h *AwardTestHelper) NewAward(teamID uuid.UUID, value int, createdAt time.Time) *entity.Award {
	h.t.Helper()
	return &entity.Award{
		ID:        uuid.New(),
		TeamID:    teamID,
		Value:     value,
		CreatedAt: createdAt,
	}
}
