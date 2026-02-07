package team

import (
	"testing"

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
		nil,
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
