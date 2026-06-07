package challenge

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

func TestChallengeUseCase_GetAll_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	teamID := uuid.New()
	challenges := []*repo.ChallengeWithSolved{
		newTestChallengeWithSolved(&domain.Challenge{
			ID: uuid.New(), Title: "Test Challenge", Description: "Test Description", Category: "Web", Points: 100,
		}, true),
	}

	d.compRepo.On("Get", mock.Anything).Return(nil, nil)
	d.challengeRepo.On("GetAll", mock.Anything, &teamID, mock.Anything).Return(challenges, nil)
	d.tagRepo.On("GetByChallengeIDs", mock.Anything, mock.Anything).Return(map[uuid.UUID][]*domain.Tag{}, nil)

	result, err := uc.GetAll(context.Background(), &teamID, nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, challenges[0].Challenge.Title, result[0].Challenge.Title)
}

func TestChallengeUseCase_GetAll_NoTeamID(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challenges := []*repo.ChallengeWithSolved{
		newTestChallengeWithSolved(&domain.Challenge{
			ID:          uuid.New(),
			Title:       "Test Challenge",
			Description: "Test Description",
			Category:    "Web",
			Points:      100,
		}, false),
	}

	d.compRepo.On("Get", mock.Anything).Return(nil, nil)
	d.challengeRepo.On("GetAll", mock.Anything, (*uuid.UUID)(nil), mock.Anything).Return(challenges, nil)
	d.tagRepo.On("GetByChallengeIDs", mock.Anything, mock.Anything).Return(map[uuid.UUID][]*domain.Tag{}, nil)

	result, err := uc.GetAll(context.Background(), nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
}

func TestChallengeUseCase_GetAll_Error(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	teamID := uuid.New()
	expectedError := assert.AnError

	d.compRepo.On("Get", mock.Anything).Return(nil, nil)
	d.challengeRepo.On("GetAll", mock.Anything, &teamID, mock.Anything).Return(nil, expectedError)

	result, err := uc.GetAll(context.Background(), &teamID, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
}
