package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestChallengeRepo_Create(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := &domain.Challenge{
		Title:        "Test Challenge",
		Description:  "Test Description",
		Category:     "Web",
		Points:       100,
		FlagHash:     "hash123",
		State:        domain.ChallengeStateVisible,
		InitialValue: 100,
		MinValue:     50,
		Decay:        10,
	}
	challenge.ID = uuid.New()
	err := f.TM.Run(ctx, func(ctx context.Context) error {
		return f.ChallengeRepo.Create(ctx, challenge)
	})
	require.NoError(t, err)
	assert.NotEmpty(t, challenge.ID)
}

func TestChallengeRepo_GetByID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateDynamicChallenge(t, "get_by_ID", 200, 100, 20)

	gotChallenge, err := f.ChallengeRepo.GetByID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, challenge.ID, gotChallenge.ID)
	assert.Equal(t, challenge.Title, gotChallenge.Title)
	assert.Equal(t, challenge.Points, gotChallenge.Points)
	assert.Equal(t, challenge.FlagHash, gotChallenge.FlagHash)
	assert.Equal(t, challenge.InitialValue, gotChallenge.InitialValue)
	assert.Equal(t, challenge.MinValue, gotChallenge.MinValue)
	assert.Equal(t, challenge.Decay, gotChallenge.Decay)
}

func TestChallengeRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	nonExistentID := uuid.New()
	_, err := f.ChallengeRepo.GetByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
}

func TestChallengeRepo_GetAll_NoTeam(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch1 := f.CreateChallenge(t, "public_1", 100)
	ch2 := f.CreateChallenge(t, "public_2", 200)

	hiddenChallenge := &domain.Challenge{
		Title:       "HIDden Challenge",
		Description: "Description",
		Category:    "Pwn",
		Points:      300,
		FlagHash:    "hash3",
		State:       domain.ChallengeStateHidden,
	}
	hiddenChallenge.ID = uuid.New()
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.ChallengeRepo.Create(txCtx, hiddenChallenge)
	})
	require.NoError(t, err)

	challenges, err := f.ChallengeRepo.GetAll(ctx, nil, nil)
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)

	for _, c := range challenges {
		ids[c.Challenge.ID] = true
	}

	assert.True(t, ids[ch1.ID], "ch1 should be in result")
	assert.True(t, ids[ch2.ID], "ch2 should be in result")

	for _, ch := range challenges {
		assert.Equal(t, domain.ChallengeStateVisible, ch.Challenge.State)
		assert.False(t, ch.Solved)
	}
}

func TestChallengeRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	challenges, err := f.ChallengeRepo.GetAll(ctx, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, challenges)
}

func TestChallengeRepo_GetAll_WithTeam(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "team_user")

	err := f.UserRepo.UpdateTeamID(ctx, user.ID, &team.ID)
	require.NoError(t, err)

	ch1 := f.CreateChallenge(t, "ch_1", 100)
	ch2 := f.CreateChallenge(t, "ch_2", 200)

	f.CreateSolve(t, user.ID, team.ID, ch1.ID)

	challenges, err := f.ChallengeRepo.GetAll(ctx, &team.ID, nil)
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)

	for _, c := range challenges {
		ids[c.Challenge.ID] = true
	}

	assert.True(t, ids[ch1.ID], "ch1 should be in result")
	assert.True(t, ids[ch2.ID], "ch2 should be in result")

	solved := false

	for _, ch := range challenges {
		if ch.Challenge.ID == ch1.ID {
			assert.True(t, ch.Solved)

			solved = true
		} else {
			assert.False(t, ch.Solved)
		}
	}

	assert.True(t, solved)
}

func TestChallengeRepo_Update(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateDynamicChallenge(t, "original", 100, 50, 10)

	challenge.Title = "Updated Title"
	challenge.Description = "Updated Description"
	challenge.Category = "Crypto"
	challenge.Points = 200
	challenge.FlagHash = "updated_hash"
	challenge.State = domain.ChallengeStateHidden
	challenge.InitialValue = 200
	challenge.MinValue = 80
	challenge.Decay = 15

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		err := f.ChallengeRepo.Update(txCtx, challenge)
		if err != nil {
			return err
		}

		return f.ChallengeRepo.SetTags(txCtx, challenge.ID, nil)
	})
	require.NoError(t, err)

	gotChallenge, err := f.ChallengeRepo.GetByID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", gotChallenge.Title)
	assert.Equal(t, "Updated Description", gotChallenge.Description)
	assert.Equal(t, "Crypto", gotChallenge.Category)
	assert.Equal(t, 200, gotChallenge.Points)
	assert.Equal(t, "updated_hash", gotChallenge.FlagHash)
	assert.Equal(t, domain.ChallengeStateHidden, gotChallenge.State)
	assert.Equal(t, 200, gotChallenge.InitialValue)
	assert.Equal(t, 80, gotChallenge.MinValue)
	assert.Equal(t, 15, gotChallenge.Decay)
}

func TestChallengeRepo_Delete(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "to_delete", 100)

	err := f.ChallengeRepo.Delete(ctx, challenge.ID)
	require.NoError(t, err)

	_, err = f.ChallengeRepo.GetByID(ctx, challenge.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
}

func TestChallengeRepo_GetByIDTx(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateDynamicChallenge(t, "tx_get", 200, 100, 20)
	_, err := f.Pool.Exec(ctx, "UPDATE challenges SET solve_count = 5 WHERE ID = $1", challenge.ID)
	require.NoError(t, err)

	challenge.SolveCount = 5

	err = f.TM.Run(ctx, func(txCtx context.Context) error {
		gotChallenge, err := f.ChallengeRepo.GetByIDForUpdate(txCtx, challenge.ID)
		if err != nil {
			return err
		}

		_ = gotChallenge

		return nil
	})
	require.NoError(t, err)
	gotChallenge, err := f.ChallengeRepo.GetByID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, challenge.ID, gotChallenge.ID)
	assert.Equal(t, challenge.Title, gotChallenge.Title)
	assert.Equal(t, challenge.Points, gotChallenge.Points)
	assert.Equal(t, challenge.SolveCount, gotChallenge.SolveCount)
}

func TestChallengeRepo_GetByIDTx_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	nonExistentID := uuid.New()
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		_, err := f.ChallengeRepo.GetByIDForUpdate(txCtx, nonExistentID)

		return err
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
}

func TestChallengeRepo_IncrementSolveCountTx(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateDynamicChallenge(t, "inc_solve", 100, 50, 10)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		_, err := f.ChallengeRepo.IncrementSolveCount(txCtx, challenge.ID)

		return err
	})
	require.NoError(t, err)

	gotChallenge, err := f.ChallengeRepo.GetByID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, gotChallenge.SolveCount)
}

func TestChallengeRepo_UpdatePointsTx(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateDynamicChallenge(t, "update_pts", 500, 100, 10)

	newPoints := 350
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.ChallengeRepo.UpdatePoints(txCtx, challenge.ID, newPoints)
	})
	require.NoError(t, err)

	gotChallenge, err := f.ChallengeRepo.GetByID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, newPoints, gotChallenge.Points)
}

func TestChallengeRepo_AtomicDynamicScoring(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	initialValue := 500
	minValue := 100
	decay := 10

	challenge := f.CreateDynamicChallenge(t, "atomic_scoring", initialValue, minValue, decay)

	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		gotChallenge, err := f.ChallengeRepo.GetByIDForUpdate(txCtx, challenge.ID)
		if err != nil {
			return err
		}

		solveCount := gotChallenge.SolveCount + 1

		newPoints := max(int(float64(gotChallenge.MinValue)+(float64(gotChallenge.InitialValue-gotChallenge.MinValue)/(1+float64(solveCount-1)/float64(gotChallenge.Decay)))), gotChallenge.MinValue)

		_, err = f.ChallengeRepo.IncrementSolveCount(txCtx, challenge.ID)
		if err != nil {
			return err
		}

		return f.ChallengeRepo.UpdatePoints(txCtx, challenge.ID, newPoints)
	})
	require.NoError(t, err)

	finalChallenge, err := f.ChallengeRepo.GetByID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, finalChallenge.SolveCount)
	// For first solve the dynamic formula yields InitialValue
	assert.Equal(t, initialValue, finalChallenge.Points)
}

func TestChallengeRepo_GetRequirements_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	mainCh := f.CreateChallenge(t, "main_req", 100)
	prereqCh := f.CreateChallenge(t, "prereq_req", 50)

	err := f.ChallengeRepo.SetRequirements(ctx, mainCh.ID, []uuid.UUID{prereqCh.ID})
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

func TestChallengeRepo_GetRequirements_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)

	challenge := f.CreateChallenge(t, "cancel_reqs", 100)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.ChallengeRepo.GetRequirements(ctx, challenge.ID)
	assert.Error(t, err)
}

func TestChallengeRepo_SetRequirements_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	mainCh := f.CreateChallenge(t, "main_set", 200)
	prereq1 := f.CreateChallenge(t, "prereq1_set", 50)
	prereq2 := f.CreateChallenge(t, "prereq2_set", 75)

	err := f.ChallengeRepo.SetRequirements(ctx, mainCh.ID, []uuid.UUID{prereq1.ID, prereq2.ID})
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

	err := f.ChallengeRepo.SetRequirements(ctx, mainCh.ID, []uuid.UUID{nonExistentID})

	assert.Error(t, err)
}

func TestChallengeRepo_GetByIDs_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch1 := f.CreateChallenge(t, "getbyids_1", 100)
	ch2 := f.CreateChallenge(t, "getbyids_2", 200)
	ch3 := f.CreateChallenge(t, "getbyids_3", 300)

	ids := []uuid.UUID{ch1.ID, ch2.ID, ch3.ID}
	got, err := f.ChallengeRepo.GetByIDs(ctx, ids)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, ch1.ID, got[ch1.ID].ID)
	assert.Equal(t, 100, got[ch1.ID].Points)
	assert.Equal(t, ch2.ID, got[ch2.ID].ID)
	assert.Equal(t, 200, got[ch2.ID].Points)
	assert.Equal(t, ch3.ID, got[ch3.ID].ID)
	assert.Equal(t, 300, got[ch3.ID].Points)
}

func TestChallengeRepo_GetByIDs_EmptyIDs(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	got, err := f.ChallengeRepo.GetByIDs(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, got)

	got, err = f.ChallengeRepo.GetByIDs(ctx, []uuid.UUID{})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestChallengeRepo_GetByIDs_PartialMatch(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch1 := f.CreateChallenge(t, "partial_1", 50)
	nonExistentID := uuid.New()

	ids := []uuid.UUID{ch1.ID, nonExistentID}
	got, err := f.ChallengeRepo.GetByIDs(ctx, ids)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, ch1.ID, got[ch1.ID].ID)
	assert.Equal(t, 50, got[ch1.ID].Points)
}

func TestChallengeRepo_BatchDecrementSolveCount(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch1 := f.CreateChallenge(t, "dec1", 100)
	ch2 := f.CreateChallenge(t, "dec2", 100)

	// Seed solve counts to 2 each
	_, err := f.Pool.Exec(ctx, "UPDATE challenges SET solve_count = 2 WHERE id = ANY($1)", []uuid.UUID{ch1.ID, ch2.ID})
	require.NoError(t, err)

	err = f.ChallengeRepo.BatchDecrementSolveCount(ctx, []uuid.UUID{ch1.ID, ch2.ID})
	require.NoError(t, err)

	got1, err := f.ChallengeRepo.GetByID(ctx, ch1.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got1.SolveCount)

	got2, err := f.ChallengeRepo.GetByID(ctx, ch2.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got2.SolveCount)
}

func TestChallengeRepo_BatchIncrementSolveCount(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch1 := f.CreateChallenge(t, "inc1", 100)
	ch2 := f.CreateChallenge(t, "inc2", 100)

	err := f.ChallengeRepo.BatchIncrementSolveCount(ctx, []uuid.UUID{ch1.ID, ch2.ID})
	require.NoError(t, err)

	got1, err := f.ChallengeRepo.GetByID(ctx, ch1.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got1.SolveCount)

	got2, err := f.ChallengeRepo.GetByID(ctx, ch2.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got2.SolveCount)
}

func TestChallengeRepo_BatchUpdatePoints(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	ch1 := f.CreateChallenge(t, "bup1", 100)
	ch2 := f.CreateChallenge(t, "bup2", 200)

	err := f.ChallengeRepo.BatchUpdatePoints(ctx, []uuid.UUID{ch1.ID, ch2.ID}, []int{150, 250})
	require.NoError(t, err)

	got1, err := f.ChallengeRepo.GetByID(ctx, ch1.ID)
	require.NoError(t, err)
	assert.Equal(t, 150, got1.Points)

	got2, err := f.ChallengeRepo.GetByID(ctx, ch2.ID)
	require.NoError(t, err)
	assert.Equal(t, 250, got2.Points)
}

func TestChallengeRepo_RecalculateSolveCounts(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "recalc")
	ch := f.CreateChallenge(t, "recalc_ch", 100)
	f.CreateSolve(t, user.ID, team.ID, ch.ID)

	// Corrupt the solve_count
	_, err := f.Pool.Exec(ctx, "UPDATE challenges SET solve_count = 99 WHERE id = $1", ch.ID)
	require.NoError(t, err)

	err = f.ChallengeRepo.RecalculateSolveCounts(ctx, []uuid.UUID{ch.ID})
	require.NoError(t, err)

	got, err := f.ChallengeRepo.GetByID(ctx, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got.SolveCount, "RecalculateSolveCounts should fix corrupted solve_count")
}
