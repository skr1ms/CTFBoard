package challenge

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

func TestChallengeUseCase_GetRequirements_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	cat := "web"
	req1 := &repo.ChallengeRequirement{
		ChallengeID:    uuid.New(),
		ChallengeTitle: "Prereq One",
		Category:       &cat,
	}
	req2 := &repo.ChallengeRequirement{
		ChallengeID:    uuid.New(),
		ChallengeTitle: "Prereq Two",
		Category:       nil,
	}
	requirements := []*repo.ChallengeRequirement{req1, req2}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID}, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return(requirements, nil)

	got, err := uc.GetRequirements(context.Background(), challengeID)

	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, req1.ChallengeID, got[0].ChallengeID)
	assert.Equal(t, req1.ChallengeTitle, got[0].ChallengeTitle)
	assert.Equal(t, req2.ChallengeID, got[1].ChallengeID)
	assert.Equal(t, req2.ChallengeTitle, got[1].ChallengeTitle)
}

func TestChallengeUseCase_GetRequirements_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, apperr.ErrChallengeNotFound)

	got, err := uc.GetRequirements(context.Background(), challengeID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
	assert.Nil(t, got)
}

func TestChallengeUseCase_SetRequirements_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	reqIDs := []uuid.UUID{uuid.New(), uuid.New()}

	reqChallenges := make(map[uuid.UUID]*domain.Challenge)

	for _, reqID := range reqIDs {
		reqChallenges[reqID] = &domain.Challenge{ID: reqID}
	}

	d.challengeRepo.On("GetByIDs", mock.Anything, reqIDs).Return(reqChallenges, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		assert.True(t, ok)
		ctx, ok := args.Get(0).(context.Context)
		assert.True(t, ok)
		assert.NoError(t, fn(ctx))
	}).Return(nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID}, nil)
	d.challengeRepo.On("GetAllRequirementPairs", mock.Anything).Return([]*domain.ChallengeRequirementPair{}, nil)
	d.challengeRepo.On("SetRequirements", mock.Anything, challengeID, reqIDs).Return(nil)

	err := uc.SetRequirements(context.Background(), challengeID, reqIDs)

	assert.NoError(t, err)
}

func TestChallengeUseCase_SetRequirements_ChallengeNotFound(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	reqIDs := []uuid.UUID{uuid.New()}
	d.challengeRepo.On("GetByIDs", mock.Anything, reqIDs).Return(map[uuid.UUID]*domain.Challenge{}, nil)

	err := uc.SetRequirements(context.Background(), challengeID, reqIDs)

	assert.Error(t, err)

	var ve *apperr.ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestChallengeUseCase_SetRequirements_Cycle(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	reqIDs := []uuid.UUID{challengeID}

	reqChallenges := map[uuid.UUID]*domain.Challenge{challengeID: {ID: challengeID}}
	d.challengeRepo.On("GetByIDs", mock.Anything, reqIDs).Return(reqChallenges, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		assert.True(t, ok)
		ctx, ok := args.Get(0).(context.Context)
		assert.True(t, ok)
		assert.Error(t, fn(ctx))
	}).Return(apperr.NewValidationErrorf("requirements contain a cycle"))
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(&domain.Challenge{ID: challengeID}, nil)
	d.challengeRepo.On("GetAllRequirementPairs", mock.Anything).Return([]*domain.ChallengeRequirementPair{}, nil)

	err := uc.SetRequirements(context.Background(), challengeID, reqIDs)

	assert.Error(t, err)

	var ve2 *apperr.ValidationError
	assert.True(t, errors.As(err, &ve2))
	assert.Contains(t, err.Error(), "cycle")
}

func TestChallengeUseCase_GetDetail_RequirementsNotMet_ReturnsNotFound(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	prereqID := uuid.New()
	challenge := newTestChallenge(challengeID, "Locked", "Web", 100, "")
	requirements := []*repo.ChallengeRequirement{
		{ChallengeID: prereqID, ChallengeTitle: "Prereq", Category: nil},
	}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return(requirements, nil)
	d.solveRepo.On("GetSolvedChallengeIDsByTeam", mock.Anything, teamID, mock.Anything).Return([]uuid.UUID{}, nil)

	detail, err := uc.GetDetail(context.Background(), challengeID, &teamID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
	assert.Nil(t, detail)
}

func TestAnonymizedChallengeDetail_HidesMetadata(t *testing.T) {
	t.Parallel()

	nextID := uuid.New()
	challenge := newTestChallenge(uuid.New(), "Locked", "Web", 100, "")
	challenge.Attribution = "Author"
	challenge.ConnectionInfo = "nc host 31337"
	challenge.NextChallengeID = &nextID

	detail := anonymizedChallengeDetail(challenge)

	assert.Equal(t, "???", detail.Challenge.Title)
	assert.Equal(t, "???", detail.Challenge.Description)
	assert.Empty(t, detail.Challenge.Attribution)
	assert.Empty(t, detail.Challenge.ConnectionInfo)
	assert.Nil(t, detail.Challenge.NextChallengeID)
	assert.Equal(t, domain.ChallengeStateLocked, detail.Challenge.State)
}

func TestChallengeUseCase_SubmitFlag_RequirementsNotMet(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	prereqID := uuid.New()
	flag := "flag{test}"
	challenge := newTestChallenge(challengeID, "Main Challenge", "Web", 100, challengeTestSha256Hash(flag))
	team := newTestTeam(teamID)
	requirements := []*repo.ChallengeRequirement{
		{ChallengeID: prereqID, ChallengeTitle: "Prereq", Category: nil},
	}

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return(requirements, nil)
	d.solveRepo.On("GetSolvedChallengeIDsByTeam", mock.Anything, teamID, mock.Anything).Return([]uuid.UUID{}, nil)

	valid, err := uc.SubmitFlag(context.Background(), submitFlagParams(challengeID, flag, userID, &teamID))

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrRequirementsNotMet)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_RequirementsMet_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	prereqID := uuid.New()
	flag := "flag{test}"
	challenge := newTestChallenge(challengeID, "Main Challenge", "Web", 100, challengeTestSha256Hash(flag))
	team := newTestTeam(teamID)
	requirements := []*repo.ChallengeRequirement{
		{ChallengeID: prereqID, ChallengeTitle: "Prereq", Category: nil},
	}

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return(requirements, nil)
	d.solveRepo.On("GetSolvedChallengeIDsByTeam", mock.Anything, teamID, mock.Anything).Return([]uuid.UUID{prereqID}, nil)

	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}

		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}

		_ = fn(ctx)
	})
	d.challengeRepo.On("GetByIDForUpdate", mock.Anything, challengeID).Return(challenge, nil)

	valid, err := uc.SubmitFlag(context.Background(), submitFlagParams(challengeID, flag, userID, &teamID))

	assert.NoError(t, err)
	assert.True(t, valid)
}
