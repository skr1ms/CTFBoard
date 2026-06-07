package challenge

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
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
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)
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
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)
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
