package team

import (
	"testing"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	teamMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team/mock"
)

type teamTestDeps struct {
	teamRepo       *teamMock.MockTeamRepository
	userRepo       *teamMock.MockUserRepository
	solveRepo      *teamMock.MockSolveRepository
	submissionRepo *teamMock.MockSubmissionRepository
	awardRepo      *teamMock.MockAwardRepository
	compRepo       *teamMock.MockCompetitionRepository
	SettingsRepo   *teamMock.MockSettingsRepository
	challengeRepo  *teamMock.MockChallengeRepository
	tm             *teamMock.MockTransactionManager
}

func newTeamTestDeps(t *testing.T) *teamTestDeps {
	t.Helper()

	return &teamTestDeps{
		teamRepo:       teamMock.NewMockTeamRepository(t),
		userRepo:       teamMock.NewMockUserRepository(t),
		solveRepo:      teamMock.NewMockSolveRepository(t),
		submissionRepo: teamMock.NewMockSubmissionRepository(t),
		awardRepo:      teamMock.NewMockAwardRepository(t),
		compRepo:       teamMock.NewMockCompetitionRepository(t),
		SettingsRepo:   teamMock.NewMockSettingsRepository(t),
		challengeRepo:  teamMock.NewMockChallengeRepository(t),
		tm:             teamMock.NewMockTransactionManager(t),
	}
}

func (d *teamTestDeps) createUseCase() *TeamUseCase {
	return NewTeamUseCase(TeamDeps{
		TeamRepo: d.teamRepo, UserRepo: d.userRepo, SolveRepo: d.solveRepo,
		SubmissionRepo: d.submissionRepo, AwardRepo: d.awardRepo, CompRepo: d.compRepo,
		SettingsGetter: d.SettingsRepo, ChallengeRepo: d.challengeRepo, TM: d.tm,
		Guard: competition.NewGuard(d.compRepo), DefaultMaxTeamSize: 10,
	})
}

func newTestUser(id uuid.UUID, teamID *uuid.UUID, username, email string) *domain.User {
	return &domain.User{ID: id, Username: username, Email: email, TeamID: teamID}
}

func newTestTeam(id uuid.UUID, name string, captainID, inviteToken uuid.UUID, isSolo bool) *domain.Team {
	return &domain.Team{ID: id, Name: name, CaptainID: captainID, InviteToken: inviteToken, IsSolo: isSolo}
}
