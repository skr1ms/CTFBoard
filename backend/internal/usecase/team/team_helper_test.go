package team

import (
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team/mocks"
	"github.com/google/uuid"
)

type TeamTestHelper struct {
	t    *testing.T
	deps *teamTestDeps
}

type teamTestDeps struct {
	teamRepo       *mocks.MockTeamRepository
	userRepo       *mocks.MockUserRepository
	solveRepo      *mocks.MockSolveRepository
	submissionRepo *mocks.MockSubmissionRepository
	awardRepo      *mocks.MockAwardRepository
	compRepo       *mocks.MockCompetitionRepository
	SettingsRepo   *mocks.MockSettingsRepository
	challengeRepo  *mocks.MockChallengeRepository
	tm             *mocks.MockTransactionManager
}

func NewTeamTestHelper(t *testing.T) *TeamTestHelper {
	t.Helper()
	return &TeamTestHelper{
		t: t,
		deps: &teamTestDeps{
			teamRepo:       mocks.NewMockTeamRepository(t),
			userRepo:       mocks.NewMockUserRepository(t),
			solveRepo:      mocks.NewMockSolveRepository(t),
			submissionRepo: mocks.NewMockSubmissionRepository(t),
			awardRepo:      mocks.NewMockAwardRepository(t),
			compRepo:       mocks.NewMockCompetitionRepository(t),
			SettingsRepo:   mocks.NewMockSettingsRepository(t),
			challengeRepo:  mocks.NewMockChallengeRepository(t),
			tm:             mocks.NewMockTransactionManager(t),
		},
	}
}

func (h *TeamTestHelper) CreateUseCase() *TeamUseCase {
	h.t.Helper()
	return NewTeamUseCase(TeamDeps{
		TeamRepo:           h.deps.teamRepo,
		UserRepo:           h.deps.userRepo,
		SolveRepo:          h.deps.solveRepo,
		SubmissionRepo:     h.deps.submissionRepo,
		AwardRepo:          h.deps.awardRepo,
		CompRepo:           h.deps.compRepo,
		SettingsGetter:     h.deps.SettingsRepo,
		ChallengeRepo:      h.deps.challengeRepo,
		TM:                 h.deps.tm,
		Guard:              competition.NewGuard(h.deps.compRepo),
		DefaultMaxTeamSize: 10,
	})
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
