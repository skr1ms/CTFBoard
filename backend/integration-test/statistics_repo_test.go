package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStatisticsRepo_GetGeneralStats_Success(t *testing.T) {
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)

	f.CreateUserWithTeam(t, uuid.New().String())
	f.CreateChallenge(t, uuid.New().String(), 100)

	stats, err := f.StatisticsRepo.GetGeneralStats(context.Background())
	require.NoError(t, err)
	require.GreaterOrEqual(t, stats.UserCount, 1)
	require.GreaterOrEqual(t, stats.TeamCount, 1)
	require.GreaterOrEqual(t, stats.ChallengeCount, 1)
}

func TestStatisticsRepo_GetChallengeStats_Success(t *testing.T) {
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)

	user, team := f.CreateUserWithTeam(t, uuid.New().String())
	chall := f.CreateChallenge(t, uuid.New().String(), 100)
	f.CreateSolve(t, user.Id, team.Id, chall.Id)

	_, err := f.Pool.Exec(context.Background(), "UPDATE challenges SET solve_count = 1 WHERE id = $1", chall.Id)
	require.NoError(t, err)

	stats, err := f.StatisticsRepo.GetChallengeStats(context.Background())
	require.NoError(t, err)

	found := false
	for _, s := range stats {
		if s.Id == chall.Id {
			require.Equal(t, 1, s.SolveCount)
			require.Equal(t, chall.Title, s.Title)
			found = true
			break
		}
	}
	require.True(t, found, "challenge statistic not found")
}

func TestStatisticsRepo_GetScoreboardHistory_Success(t *testing.T) {
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)

	user1, team1 := f.CreateUserWithTeam(t, uuid.New().String())
	chall1 := f.CreateChallenge(t, uuid.New().String(), 100)

	ctx := context.Background()

	solveTime := time.Now().Add(-1 * time.Hour)
	_, err := f.Pool.Exec(ctx, "INSERT INTO solves (id, user_id, team_id, challenge_id, solved_at) VALUES ($1, $2, $3, $4, $5)", uuid.New(), user1.Id, team1.Id, chall1.Id, solveTime)
	require.NoError(t, err)

	history, err := f.StatisticsRepo.GetScoreboardHistory(ctx, 10)
	require.NoError(t, err)

	found := false
	for _, h := range history {
		if h.TeamId == team1.Id {
			require.Equal(t, 100, h.Points)
			found = true
		}
	}
	require.True(t, found, "history for team1 not found")
}

func TestStatisticsRepo_GetGeneralStats_Error_CancelledContext(t *testing.T) {
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.StatisticsRepo.GetGeneralStats(ctx)
	require.Error(t, err)
}
