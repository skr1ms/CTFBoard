package challenge

import (
	"context"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
)

func TestChallengeUseCase_SubmitFlag_AlreadySolved(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)

	_, redis := redismock.NewClientMock()
	uc := NewChallengeUseCase(ChallengeDeps{
		ChallengeRepo: d.challengeRepo, SolveRepo: d.solveRepo, SubmissionRepo: nil, TM: d.tm,
		CompRepo: d.compRepo, TeamRepo: d.teamRepo, AuditLogRepo: d.auditLogRepo,
		TagRepo: d.tagRepo, Crypto: nil,
		SolveRecord: func(_ context.Context, _ *domain.Solve, _ *domain.Challenge, _ repo.ChallengeRepository, _ repo.SolveRepository, _ ...scoring.DecayFunction) (int, error) {
			return 0, apperr.ErrAlreadySolved
		},
	})
	_ = redis

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := challengeTestSha256Hash(flag)
	challenge := &domain.Challenge{
		ID:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	team := newTestTeam(teamID)

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetByIDForUpdate", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirementsForEnforcement", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})

	valid, err := uc.SubmitFlag(context.Background(), submitFlagParams(challengeID, flag, userID, &teamID))

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestChallengeUseCase_SubmitFlag_BeginTxError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := challengeTestSha256Hash(flag)
	challenge := &domain.Challenge{
		ID:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	team := newTestTeam(teamID)
	expectedError := assert.AnError

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirementsForEnforcement", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Return(expectedError)

	valid, err := uc.SubmitFlag(context.Background(), submitFlagParams(challengeID, flag, userID, &teamID))

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_CreateTxError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	expectedError := assert.AnError

	uc := NewChallengeUseCase(ChallengeDeps{
		ChallengeRepo: d.challengeRepo, SolveRepo: d.solveRepo, SubmissionRepo: nil, TM: d.tm,
		CompRepo: d.compRepo, TeamRepo: d.teamRepo, AuditLogRepo: d.auditLogRepo,
		TagRepo: d.tagRepo, Crypto: nil,
		SolveRecord: func(_ context.Context, _ *domain.Solve, _ *domain.Challenge, _ repo.ChallengeRepository, _ repo.SolveRepository, _ ...scoring.DecayFunction) (int, error) {
			return 0, expectedError
		},
	})

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := challengeTestSha256Hash(flag)
	challenge := &domain.Challenge{
		ID:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	team := newTestTeam(teamID)

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetByIDForUpdate", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirementsForEnforcement", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})

	valid, err := uc.SubmitFlag(context.Background(), submitFlagParams(challengeID, flag, userID, &teamID))

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_GetByTeamAndChallengeTxUnexpectedError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	expectedError := assert.AnError

	uc := NewChallengeUseCase(ChallengeDeps{
		ChallengeRepo: d.challengeRepo, SolveRepo: d.solveRepo, SubmissionRepo: nil, TM: d.tm,
		CompRepo: d.compRepo, TeamRepo: d.teamRepo, AuditLogRepo: d.auditLogRepo,
		TagRepo: d.tagRepo, Crypto: nil,
		SolveRecord: func(_ context.Context, _ *domain.Solve, _ *domain.Challenge, _ repo.ChallengeRepository, _ repo.SolveRepository, _ ...scoring.DecayFunction) (int, error) {
			return 0, expectedError
		},
	})

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := challengeTestSha256Hash(flag)
	challenge := &domain.Challenge{
		ID:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	team := newTestTeam(teamID)

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)
	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetByIDForUpdate", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirementsForEnforcement", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})

	valid, err := uc.SubmitFlag(context.Background(), submitFlagParams(challengeID, flag, userID, &teamID))

	assert.Error(t, err)
	assert.False(t, valid)
}
