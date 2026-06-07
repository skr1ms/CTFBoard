package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolveRepo_SoftBanByTeamIDAndUserID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "softban")
	ch := f.CreateChallenge(t, "softban_ch", 100)
	solve := f.CreateSolve(t, user.ID, team.ID, ch.ID)

	err := f.SolveRepo.SoftBanByTeamIDAndUserID(ctx, team.ID, user.ID)
	require.NoError(t, err)

	// solve should now be soft-banned (banned_user_id set to user_id)
	row := f.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM solves WHERE id = $1 AND banned_user_id IS NOT NULL", solve.ID)

	var count int
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, 1, count, "solve should have banned_user_id set after SoftBan")
}

func TestSolveRepo_RestoreByBannedUserID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "restore")
	ch := f.CreateChallenge(t, "restore_ch", 100)
	solve := f.CreateSolve(t, user.ID, team.ID, ch.ID)

	// Soft-ban the solve first
	require.NoError(t, f.SolveRepo.SoftBanByTeamIDAndUserID(ctx, team.ID, user.ID))

	// Now restore
	err := f.SolveRepo.RestoreByBannedUserID(ctx, user.ID)
	require.NoError(t, err)

	row := f.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM solves WHERE id = $1 AND banned_user_id IS NULL", solve.ID)

	var count int
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, 1, count, "solve should have banned_user_id cleared after restore")
}

func TestSolveRepo_BatchUpdateSolvePoints(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user1, team1 := f.CreateUserWithTeam(t, "bsp1")
	user2, team2 := f.CreateUserWithTeam(t, "bsp2")
	ch := f.CreateChallenge(t, "bsp_ch", 100)

	solve1 := f.CreateSolve(t, user1.ID, team1.ID, ch.ID)
	solve2 := f.CreateSolve(t, user2.ID, team2.ID, ch.ID)

	err := f.SolveRepo.BatchUpdateSolvePoints(ctx, []uuid.UUID{solve1.ID, solve2.ID}, []int{150, 175})
	require.NoError(t, err)

	row1 := f.Pool.QueryRow(ctx, "SELECT points_at_solve FROM solves WHERE id = $1", solve1.ID)

	var pts1 int
	require.NoError(t, row1.Scan(&pts1))
	assert.Equal(t, 150, pts1)

	row2 := f.Pool.QueryRow(ctx, "SELECT points_at_solve FROM solves WHERE id = $1", solve2.ID)

	var pts2 int
	require.NoError(t, row2.Scan(&pts2))
	assert.Equal(t, 175, pts2)
}
