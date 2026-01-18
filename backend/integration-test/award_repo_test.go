package integration_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CreateTx Tests

func TestAwardRepo_CreateTx(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "award")

	tx, err := f.DB.BeginTx(ctx, nil)
	require.NoError(t, err)

	award := f.CreateAwardTx(t, tx, team.Id, 100, "Test award")
	assert.NotEmpty(t, award.Id)

	err = tx.Commit()
	require.NoError(t, err)

	total, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 100, total)
}

func TestAwardRepo_CreateTx_Rollback(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "rollback_award")

	tx, err := f.DB.BeginTx(ctx, nil)
	require.NoError(t, err)

	f.CreateAwardTx(t, tx, team.Id, 200, "Rollback award")

	err = tx.Rollback()
	require.NoError(t, err)

	total, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

// GetTeamTotalAwards Tests

func TestAwardRepo_GetTeamTotalAwards(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "total_award")

	awards := []int{100, 50, -25, 75}
	expectedTotal := 0
	for i, value := range awards {
		tx, err := f.DB.BeginTx(ctx, nil)
		require.NoError(t, err)

		f.CreateAwardTx(t, tx, team.Id, value, "Award "+string(rune('A'+i)))

		err = tx.Commit()
		require.NoError(t, err)

		expectedTotal += value
	}

	total, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, expectedTotal, total)
}

func TestAwardRepo_GetTeamTotalAwards_NoAwards(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "no_award")

	total, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestAwardRepo_NegativeAward(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	_, team := f.CreateUserWithTeam(t, "neg_award")

	tx1, err := f.DB.BeginTx(ctx, nil)
	require.NoError(t, err)
	f.CreateAwardTx(t, tx1, team.Id, 100, "Bonus")
	err = tx1.Commit()
	require.NoError(t, err)

	tx2, err := f.DB.BeginTx(ctx, nil)
	require.NoError(t, err)
	f.CreateAwardTx(t, tx2, team.Id, -30, "Penalty for hint")
	err = tx2.Commit()
	require.NoError(t, err)

	total, err := f.AwardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 70, total)
}
