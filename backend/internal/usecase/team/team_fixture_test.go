package team

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	teamMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team/mock"
)

type teamTestDeps struct {
	teamRepo        *teamMock.MockTeamRepository
	userRepo        *teamMock.MockUserRepository
	solveRepo       *teamMock.MockSolveRepository
	submissionRepo  *teamMock.MockSubmissionRepository
	awardRepo       *teamMock.MockAwardRepository
	ratingRepo      *teamMock.MockRatingRepository
	compRepo        *teamMock.MockCompetitionRepository
	SettingsRepo    *teamMock.MockSettingsRepository
	challengeRepo   *teamMock.MockChallengeRepository
	tm              *teamMock.MockTransactionManager
	scoreboardCache cacheutil.ScoreboardCacheInvalidator
	jwtRevoker      *testJWTRevoker
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
		jwtRevoker:     &testJWTRevoker{},
	}
}

func (d *teamTestDeps) enableRatingRepo(t *testing.T) {
	t.Helper()

	d.ratingRepo = teamMock.NewMockRatingRepository(t)
}

func (d *teamTestDeps) createUseCase() *TeamUseCase {
	deps := TeamDeps{
		TeamRepo: d.teamRepo, UserRepo: d.userRepo, SolveRepo: d.solveRepo,
		SubmissionRepo: d.submissionRepo, AwardRepo: d.awardRepo, CompRepo: d.compRepo,
		SettingsGetter: d.SettingsRepo, ChallengeRepo: d.challengeRepo, TM: d.tm,
		Guard: competition.NewGuard(d.compRepo), ScoreboardCache: d.scoreboardCache, JWTRevoker: d.jwtRevoker, DefaultMaxTeamSize: 10,
	}

	if d.ratingRepo != nil {
		deps.RatingRepo = d.ratingRepo
	}

	return NewTeamUseCase(deps)
}

type testJWTRevoker struct {
	revoked []uuid.UUID
	err     error
}

func (r *testJWTRevoker) RevokeAllForUser(_ context.Context, userID uuid.UUID) error {
	r.revoked = append(r.revoked, userID)

	return r.err
}

func newTestUser(id uuid.UUID, teamID *uuid.UUID, username, email string) *domain.User {
	return &domain.User{ID: id, Username: username, Email: email, TeamID: teamID}
}

func newTestTeam(id uuid.UUID, name string, captainID, inviteToken uuid.UUID, isSolo bool) *domain.Team {
	return &domain.Team{ID: id, Name: name, CaptainID: captainID, InviteToken: inviteToken, IsSolo: isSolo}
}
