package team

import (
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase/team/mocks"
)

type AwardTestHelper struct {
	t       *testing.T
	repo    *mocks.MockAwardRepository
	txRepo  *mocks.MockTxRepository
	redis   redismock.ClientMock
	useCase *AwardUseCase
	teamID  uuid.UUID
	adminID uuid.UUID
}

func NewAwardTestHelper(t *testing.T) *AwardTestHelper {
	t.Helper()
	repo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	client, redis := redismock.NewClientMock()
	uc := NewAwardUseCase(repo, txRepo, client)
	return &AwardTestHelper{
		t:       t,
		repo:    repo,
		txRepo:  txRepo,
		redis:   redis,
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

func (h *AwardTestHelper) TxRepo() *mocks.MockTxRepository {
	h.t.Helper()
	return h.txRepo
}

func (h *AwardTestHelper) Redis() redismock.ClientMock {
	h.t.Helper()
	return h.redis
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
