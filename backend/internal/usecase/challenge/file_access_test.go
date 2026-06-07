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

func TestFileUseCase_GetByChallengeIDWithAccess_WriteupSolvedOnlyAllowed(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc := d.createFileUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	files := []*domain.File{{ID: uuid.New(), Type: domain.FileTypeWriteup, ChallengeID: &challengeID}}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(newTestChallenge(challengeID, "Web", "web", 100, "hash"), nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(&domain.ChallengeSolution{
		ChallengeID: challengeID,
		State:       domain.SolutionStateSolvedOnly,
	}, nil)
	d.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(&domain.Solve{TeamID: teamID, ChallengeID: challengeID}, nil)
	d.fileRepo.On("GetByChallengeID", mock.Anything, challengeID, domain.FileTypeWriteup).Return(files, nil)

	result, err := uc.GetByChallengeIDWithAccess(context.Background(), challengeID, domain.FileTypeWriteup, &teamID, false)

	assert.NoError(t, err)
	assert.Equal(t, files, result)
}

func TestFileUseCase_GetByChallengeIDWithAccess_WriteupHiddenDenied(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc := d.createFileUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(newTestChallenge(challengeID, "Web", "web", 100, "hash"), nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(&domain.ChallengeSolution{
		ChallengeID: challengeID,
		State:       domain.SolutionStateHidden,
	}, nil)

	result, err := uc.GetByChallengeIDWithAccess(context.Background(), challengeID, domain.FileTypeWriteup, &teamID, false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrWriteupAccessDenied)
}

func TestFileUseCase_GetByChallengeIDWithAccess_WriteupsDisabledDenied(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc := d.createFileUseCase()
	uc.deps.SettingsRepo = &solutionTestSettingsRepo{settings: &domain.Settings{WriteupEnabled: false}}

	challengeID := uuid.New()

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(newTestChallenge(challengeID, "Web", "web", 100, "hash"), nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(&domain.ChallengeSolution{
		ChallengeID: challengeID,
		State:       domain.SolutionStateAfterEvent,
	}, nil)

	result, err := uc.GetByChallengeIDWithAccess(context.Background(), challengeID, domain.FileTypeWriteup, nil, false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrWriteupsDisabled)
}
