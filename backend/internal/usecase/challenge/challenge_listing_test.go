package challenge

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	compMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mock"
)

func TestChallengeUseCase_GetAll_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	teamID := uuid.New()
	challenges := []*domain.ChallengeWithSolved{
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

	challenges := []*domain.ChallengeWithSolved{
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

func TestChallengeUseCaseApplyFrozenSolveCountsPreservesRequirementsMet(t *testing.T) {
	t.Parallel()

	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	now := time.Now()
	start := now.Add(-1 * time.Hour)
	freeze := now.Add(-1 * time.Minute)
	end := now.Add(1 * time.Hour)
	challengeID := uuid.New()
	requirementsMet := false

	comp := &domain.Competition{StartTime: &start, FreezeTime: &freeze, EndTime: &end}
	list := []*usecase.ChallengeWithTags{
		{
			ChallengeWithSolved: &domain.ChallengeWithSolved{
				Challenge: &domain.Challenge{ID: challengeID, Title: "locked", SolveCount: 42},
				Solved:    false,
			},
			Tags:            []*domain.Tag{{ID: uuid.New(), Name: "web"}},
			RequirementsMet: &requirementsMet,
		},
	}

	d.solveRepo.On("GetSolveCounts", mock.Anything, &freeze).Return(map[uuid.UUID]int{challengeID: 7}, nil).Once()

	got, err := uc.applyFrozenSolveCounts(context.Background(), comp, list)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.NotNil(t, got[0].RequirementsMet)
	assert.False(t, *got[0].RequirementsMet)
	assert.Equal(t, 7, got[0].Challenge.SolveCount)
	assert.Equal(t, list[0].Tags, got[0].Tags)
}

func TestChallengeUseCaseComputeRequirementsMetMapFailsClosedWhenGraphLoadFails(t *testing.T) {
	t.Parallel()

	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()
	compParams := compMock.NewMockCompetitionParamUseCase(t)
	uc.deps.CompParamUC = compParams

	teamID := uuid.New()
	challengeIDs := []uuid.UUID{uuid.New(), uuid.New()}

	compParams.EXPECT().
		GetBool(mock.Anything, "challenge_prerequisite_anonymize", false).
		Return(true).
		Once()
	d.challengeRepo.EXPECT().
		GetAllRequirementPairs(mock.Anything).
		Return(nil, assert.AnError).
		Once()

	got := uc.computeRequirementsMetMap(context.Background(), &teamID, challengeIDs)

	require.Len(t, got, len(challengeIDs))

	for _, id := range challengeIDs {
		assert.False(t, got[id])
	}
}

func TestChallengeUseCaseComputeRequirementsMetMapFailsClosedForGatedChallengesWhenSolvesLoadFails(t *testing.T) {
	t.Parallel()

	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()
	compParams := compMock.NewMockCompetitionParamUseCase(t)
	uc.deps.CompParamUC = compParams

	teamID := uuid.New()
	openID := uuid.New()
	gatedID := uuid.New()
	requiredID := uuid.New()
	challengeIDs := []uuid.UUID{openID, gatedID}

	compParams.EXPECT().
		GetBool(mock.Anything, "challenge_prerequisite_anonymize", false).
		Return(true).
		Once()
	d.challengeRepo.EXPECT().
		GetAllRequirementPairs(mock.Anything).
		Return([]*domain.ChallengeRequirementPair{
			{ChallengeID: gatedID, RequiredChallengeID: requiredID},
		}, nil).
		Once()
	d.solveRepo.EXPECT().
		GetSolvedChallengeIDsByTeam(mock.Anything, teamID, []uuid.UUID{requiredID}).
		Return(nil, assert.AnError).
		Once()

	got := uc.computeRequirementsMetMap(context.Background(), &teamID, challengeIDs)

	require.Len(t, got, 1)
	assert.False(t, got[gatedID])
	assert.NotContains(t, got, openID)
}
