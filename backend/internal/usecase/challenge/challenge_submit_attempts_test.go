package challenge

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestChallengeUseCase_SubmitFlag_MaxAttemptsReached(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithSubmissionRepo()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	challenge := newTestChallenge(challengeID, "Test Challenge", "Web", 100, challengeTestSha256Hash("flag{correct}"))
	challenge.MaxAttempts = 2
	team := newTestTeam(teamID)

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirementsForEnforcement", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.submissionRepo.EXPECT().CountSubmissionsByTeamAndChallenge(mock.Anything, teamID, challengeID).Return(int64(2), nil)

	valid, err := uc.SubmitFlag(context.Background(), submitFlagParams(challengeID, "flag{wrong}", userID, &teamID))

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrMaxAttemptsReached)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_MaxAttemptsOneUnderLimit(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithSubmissionRepo()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	challenge := newTestChallenge(challengeID, "Test Challenge", "Web", 100, challengeTestSha256Hash("flag{correct}"))
	challenge.MaxAttempts = 2
	team := newTestTeam(teamID)

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirementsForEnforcement", mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil)
	d.submissionRepo.EXPECT().CountSubmissionsByTeamAndChallenge(mock.Anything, teamID, challengeID).Return(int64(1), nil).Twice()
	d.submissionRepo.EXPECT().AcquireAdvisoryLockForSubmit(mock.Anything, teamID, challengeID).Return(nil).Once()
	d.submissionRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(s *domain.Submission) bool {
		return s.ChallengeID == challengeID && s.TeamID != nil && *s.TeamID == teamID && !s.IsCorrect
	})).Return(nil).Once()
	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}

		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}

		assert.NoError(t, fn(ctx))
	})

	valid, err := uc.SubmitFlag(context.Background(), submitFlagParams(challengeID, "flag{wrong}", userID, &teamID))

	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_IncorrectRechecksMembershipBeforeWrite(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc := NewChallengeUseCase(ChallengeDeps{
		ChallengeRepo:  d.challengeRepo,
		SolveRepo:      d.solveRepo,
		SubmissionRepo: d.submissionRepo,
		TM:             d.tm,
		CompRepo:       d.compRepo,
		TeamRepo:       d.teamRepo,
		UserRepo:       d.userRepo,
		SolveRecord:    stubSolveRecord,
	})

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	challenge := newTestChallenge(challengeID, "Test Challenge", "Web", 100, challengeTestSha256Hash("flag{correct}"))
	team := newTestTeam(teamID)
	initialUser := &domain.User{ID: userID, TeamID: &teamID}
	removedUser := &domain.User{ID: userID}

	d.compRepo.EXPECT().Get(mock.Anything).Return(newActiveCompetition(), nil).Twice()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(initialUser, nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(challenge, nil).Once()
	d.challengeRepo.EXPECT().GetRequirementsForEnforcement(mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(removedUser, nil).Once()

	valid, err := uc.SubmitFlag(context.Background(), submitFlagParams(challengeID, "flag{wrong}", userID, &teamID))

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamMemberNotFound)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_IncorrectRechecksCompetitionBeforeWrite(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc := NewChallengeUseCase(ChallengeDeps{
		ChallengeRepo:  d.challengeRepo,
		SolveRepo:      d.solveRepo,
		SubmissionRepo: d.submissionRepo,
		TM:             d.tm,
		CompRepo:       d.compRepo,
		TeamRepo:       d.teamRepo,
		UserRepo:       d.userRepo,
		SolveRecord:    stubSolveRecord,
	})

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	challenge := newTestChallenge(challengeID, "Test Challenge", "Web", 100, challengeTestSha256Hash("flag{correct}"))
	team := newTestTeam(teamID)
	initialUser := &domain.User{ID: userID, TeamID: &teamID}
	paused := newActiveCompetition()
	paused.IsPaused = true

	d.compRepo.EXPECT().Get(mock.Anything).Return(newActiveCompetition(), nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(initialUser, nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(challenge, nil).Once()
	d.challengeRepo.EXPECT().GetRequirementsForEnforcement(mock.Anything, challengeID).Return([]*domain.ChallengeRequirement{}, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.compRepo.EXPECT().Get(mock.Anything).Return(paused, nil).Once()

	valid, err := uc.SubmitFlag(context.Background(), submitFlagParams(challengeID, "flag{wrong}", userID, &teamID))

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrSubmissionNotAllowed)
	assert.False(t, valid)
	d.submissionRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}
