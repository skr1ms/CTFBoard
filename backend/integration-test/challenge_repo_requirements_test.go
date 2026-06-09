package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	challengeuc "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge"
)

func TestChallengeRepo_GetRequirements_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	mainCh := f.CreateChallenge(t, "main_req", 100)
	prereqCh := f.CreateChallenge(t, "prereq_req", 50)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.ChallengeRepo.SetRequirements(txCtx, mainCh.ID, []uuid.UUID{prereqCh.ID})
	})
	require.NoError(t, err)

	got, err := f.ChallengeRepo.GetRequirements(ctx, mainCh.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, prereqCh.ID, got[0].ChallengeID)
	assert.Equal(t, prereqCh.Title, got[0].ChallengeTitle)
	assert.NotNil(t, got[0].Category)
	assert.Equal(t, "Web", *got[0].Category)
}

func TestChallengeRepo_GetRequirements_Empty(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "no_reqs", 100)

	got, err := f.ChallengeRepo.GetRequirements(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestChallengeRepo_GetRequirementsForEnforcement_IncludesHiddenPrerequisites(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	mainCh := f.CreateChallenge(t, "main_hidden_req", 100)
	hiddenPrereq := f.CreateChallenge(t, "hidden_prereq", 50)
	hiddenPrereq.State = "hidden"

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.ChallengeRepo.Update(txCtx, hiddenPrereq)
	})
	require.NoError(t, err)

	err = f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.ChallengeRepo.SetRequirements(txCtx, mainCh.ID, []uuid.UUID{hiddenPrereq.ID})
	})
	require.NoError(t, err)

	publicReqs, err := f.ChallengeRepo.GetRequirements(ctx, mainCh.ID)
	require.NoError(t, err)
	assert.Empty(t, publicReqs)

	enforcedReqs, err := f.ChallengeRepo.GetRequirementsForEnforcement(ctx, mainCh.ID)
	require.NoError(t, err)
	require.Len(t, enforcedReqs, 1)
	assert.Equal(t, hiddenPrereq.ID, enforcedReqs[0].ChallengeID)
}

func TestChallengeRepo_SetRequirements_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	mainCh := f.CreateChallenge(t, "main_set", 200)
	prereq1 := f.CreateChallenge(t, "prereq1_set", 50)
	prereq2 := f.CreateChallenge(t, "prereq2_set", 75)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.ChallengeRepo.SetRequirements(txCtx, mainCh.ID, []uuid.UUID{prereq1.ID, prereq2.ID})
	})
	require.NoError(t, err)

	got, err := f.ChallengeRepo.GetRequirements(ctx, mainCh.ID)
	require.NoError(t, err)
	require.Len(t, got, 2)
	ids := map[uuid.UUID]bool{got[0].ChallengeID: true, got[1].ChallengeID: true}
	assert.True(t, ids[prereq1.ID])
	assert.True(t, ids[prereq2.ID])
}

func TestChallengeRepo_SetRequirements_InvalidRequiredID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	mainCh := f.CreateChallenge(t, "main_invalid", 100)
	nonExistentID := uuid.New()

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.ChallengeRepo.SetRequirements(txCtx, mainCh.ID, []uuid.UUID{nonExistentID})
	})

	assert.Error(t, err)
}

func TestChallengeUseCase_SetRequirements_ConcurrentCycleRejected(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challengeA := f.CreateChallenge(t, "cycle_a", 100)
	challengeB := f.CreateChallenge(t, "cycle_b", 100)
	uc := challengeuc.NewChallengeUseCase(challengeuc.ChallengeDeps{
		ChallengeRepo: f.ChallengeRepo,
		TM:            f.TM,
	})

	start := make(chan struct{})
	errs := make(chan error, 2)

	go func() {
		<-start

		errs <- uc.SetRequirements(ctx, challengeA.ID, []uuid.UUID{challengeB.ID})
	}()

	go func() {
		<-start

		errs <- uc.SetRequirements(ctx, challengeB.ID, []uuid.UUID{challengeA.ID})
	}()

	close(start)

	err1 := <-errs
	err2 := <-errs
	require.True(t, (err1 == nil) != (err2 == nil), "exactly one concurrent requirements update should commit: err1=%v err2=%v", err1, err2)

	var validationErr *apperr.ValidationError

	if err1 != nil {
		require.True(t, errors.As(err1, &validationErr))
		assert.Contains(t, err1.Error(), "cycle")
	} else {
		require.True(t, errors.As(err2, &validationErr))
		assert.Contains(t, err2.Error(), "cycle")
	}

	pairs, err := f.ChallengeRepo.GetAllRequirementPairs(ctx)
	require.NoError(t, err)

	edgeAB := false
	edgeBA := false

	for _, pair := range pairs {
		if pair.ChallengeID == challengeA.ID && pair.RequiredChallengeID == challengeB.ID {
			edgeAB = true
		}

		if pair.ChallengeID == challengeB.ID && pair.RequiredChallengeID == challengeA.ID {
			edgeBA = true
		}
	}

	require.False(t, edgeAB && edgeBA, "requirements graph must remain acyclic")
}
