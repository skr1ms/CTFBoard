package team

import (
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase/team/mocks"
)

type TeamTestHelper struct {
	t    *testing.T
	deps *teamTestDeps
}

type teamTestDeps struct {
	teamRepo *mocks.MockTeamRepository
	userRepo *mocks.MockUserRepository
	compRepo *mocks.MockCompetitionRepository
	txRepo   *mocks.MockTxRepository
}

func NewTeamTestHelper(t *testing.T) *TeamTestHelper {
	t.Helper()
	return &TeamTestHelper{
		t: t,
		deps: &teamTestDeps{
			teamRepo: mocks.NewMockTeamRepository(t),
			userRepo: mocks.NewMockUserRepository(t),
			compRepo: mocks.NewMockCompetitionRepository(t),
			txRepo:   mocks.NewMockTxRepository(t),
		},
	}
}

func (h *TeamTestHelper) CreateUseCase() *TeamUseCase {
	h.t.Helper()
	return NewTeamUseCase(
		h.deps.teamRepo,
		h.deps.userRepo,
		h.deps.compRepo,
		h.deps.txRepo,
	)
}

func (h *TeamTestHelper) Deps() *teamTestDeps {
	h.t.Helper()
	return h.deps
}

func (h *TeamTestHelper) NewUser(id uuid.UUID, teamID *uuid.UUID, username, email string) *entity.User {
	h.t.Helper()
	return &entity.User{
		ID:       id,
		Username: username,
		Email:    email,
		TeamID:   teamID,
	}
}

func (h *TeamTestHelper) NewTeam(id uuid.UUID, name string, captainID, inviteToken uuid.UUID, isSolo bool) *entity.Team {
	h.t.Helper()
	return &entity.Team{
		ID:          id,
		Name:        name,
		CaptainID:   captainID,
		InviteToken: inviteToken,
		IsSolo:      isSolo,
	}
}

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
