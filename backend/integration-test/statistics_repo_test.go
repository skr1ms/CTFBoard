package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestStatisticsRepo_GetGeneralStats_Success(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)

	f.CreateUserWithTeam(t, uuid.New().String())
	f.CreateChallenge(t, uuid.New().String(), 100)

	stats, err := f.StatisticsRepo.GetGeneralStats(context.Background(), nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, stats.UserCount, 1)
	require.GreaterOrEqual(t, stats.TeamCount, 1)
	require.GreaterOrEqual(t, stats.ChallengeCount, 1)
}

func TestStatisticsRepo_GetChallengeStats_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.StatisticsRepo.GetChallengeStats(ctx, nil)
	require.Error(t, err)
}

func TestStatisticsRepo_GetChallengeStats_Success(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)

	user, team := f.CreateUserWithTeam(t, uuid.New().String())
	chall := f.CreateChallenge(t, uuid.New().String(), 100)
	f.CreateSolve(t, user.ID, team.ID, chall.ID)

	_, err := f.Pool.Exec(context.Background(), "UPDATE challenges SET solve_count = 1 WHERE ID = $1", chall.ID)
	require.NoError(t, err)

	stats, err := f.StatisticsRepo.GetChallengeStats(context.Background(), nil)
	require.NoError(t, err)

	found := false

	for _, s := range stats {
		if s.ID == chall.ID {
			require.Equal(t, 1, s.SolveCount)
			require.Equal(t, chall.Title, s.Title)

			found = true

			break
		}
	}

	require.True(t, found, "challenge statistic not found")
}

func TestStatisticsRepo_GetScoreboardHistory_Success(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)

	user1, team1 := f.CreateUserWithTeam(t, uuid.New().String())
	chall1 := f.CreateChallenge(t, uuid.New().String(), 100)

	ctx := context.Background()

	solveTime := time.Now().Add(-1 * time.Hour)
	_, err := f.Pool.Exec(ctx, "INSERT INTO solves (id, user_id, team_id, challenge_id, solved_at, points_at_solve) VALUES ($1, $2, $3, $4, $5, $6)", uuid.New(), user1.ID, team1.ID, chall1.ID, solveTime, 100)
	require.NoError(t, err)

	history, err := f.StatisticsRepo.GetScoreboardHistory(ctx, 1000, nil)
	require.NoError(t, err)

	found := false

	for _, h := range history {
		if h.TeamID == team1.ID {
			require.Equal(t, 100, h.Points)

			found = true
		}
	}

	require.True(t, found, "history for team1 not found")
}

func TestStatisticsRepo_GetScoreboardHistory_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.StatisticsRepo.GetScoreboardHistory(ctx, 10, nil)
	require.Error(t, err)
}

func TestStatisticsRepo_GetGeneralStats_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.StatisticsRepo.GetGeneralStats(ctx, nil)
	require.Error(t, err)
}

func TestStatisticsRepo_GetChallengeDetailStats_Success(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, uuid.New().String())
	chall := f.CreateChallenge(t, uuid.New().String(), 100)
	f.CreateSolve(t, user.ID, team.ID, chall.ID)

	_, err := f.Pool.Exec(ctx, "UPDATE challenges SET solve_count = 1 WHERE id = $1", chall.ID)
	require.NoError(t, err)

	stats, err := f.StatisticsRepo.GetChallengeDetailStats(ctx, chall.ID, nil)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, chall.ID, stats.ID)
	assert.Equal(t, chall.Title, stats.Title)
	assert.Equal(t, chall.Category, stats.Category)
	assert.Equal(t, chall.Points, stats.Points)
	assert.Equal(t, 1, stats.SolveCount)
	assert.GreaterOrEqual(t, stats.TotalTeams, 1)
	assert.NotNil(t, stats.FirstBlood)
	assert.Equal(t, team.ID, stats.FirstBlood.TeamID)
	assert.Equal(t, team.Name, stats.FirstBlood.TeamName)
	require.Len(t, stats.Solves, 1)
	assert.Equal(t, team.ID, stats.Solves[0].TeamID)
	assert.Equal(t, team.Name, stats.Solves[0].TeamName)
}

func TestStatisticsRepo_GetChallengeDetailStats_NotFound(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	stats, err := f.StatisticsRepo.GetChallengeDetailStats(ctx, uuid.New(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrChallengeNotFound)
	assert.Nil(t, stats)
}

func TestStatisticsRepo_GetChallengeDetailStats_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	chall := f.CreateChallenge(t, uuid.New().String(), 100)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.StatisticsRepo.GetChallengeDetailStats(ctx, chall.ID, nil)
	require.Error(t, err)
}

func TestStatisticsRepo_GetTeamRegistrationTimeSeries_Success(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	f.CreateUserWithTeam(t, uuid.New().String())

	series, err := f.StatisticsRepo.GetTeamRegistrationTimeSeries(ctx)
	require.NoError(t, err)
	require.NotNil(t, series)
	require.GreaterOrEqual(t, len(series), 1)
}

func TestStatisticsRepo_GetTeamRegistrationTimeSeries_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.StatisticsRepo.GetTeamRegistrationTimeSeries(ctx)
	require.Error(t, err)
}

func TestStatisticsRepo_GetUserRegistrationTimeSeries_Success(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	f.CreateUser(t, uuid.New().String())

	series, err := f.StatisticsRepo.GetUserRegistrationTimeSeries(ctx)
	require.NoError(t, err)
	require.NotNil(t, series)
	require.GreaterOrEqual(t, len(series), 1)
}

func TestStatisticsRepo_GetUserRegistrationTimeSeries_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.StatisticsRepo.GetUserRegistrationTimeSeries(ctx)
	require.Error(t, err)
}

func TestStatisticsRepo_GetSolveMatrix_Success(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, uuid.New().String())
	chall := f.CreateChallenge(t, uuid.New().String(), 100)
	f.CreateSolve(t, user.ID, team.ID, chall.ID)

	matrix, err := f.StatisticsRepo.GetSolveMatrix(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, matrix)

	found := false

	for _, row := range matrix {
		if row.TeamID == team.ID && row.ChallengeID == chall.ID {
			assert.True(t, row.Solved)

			found = true

			break
		}
	}

	require.True(t, found, "solve matrix should contain row for team/challenge")
}

func TestStatisticsRepo_GetSolveMatrix_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.StatisticsRepo.GetSolveMatrix(ctx, nil)
	require.Error(t, err)
}

func TestStatisticsRepo_GetChallengeSolvePercentages(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "pct")
	chall := f.CreateChallenge(t, "pct_ch", 100)
	f.CreateSolve(t, user.ID, team.ID, chall.ID)

	result, err := f.StatisticsRepo.GetChallengeSolvePercentages(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	found := false

	for _, r := range result {
		if r.ID == chall.ID {
			assert.GreaterOrEqual(t, r.SolveCount, 1)

			found = true

			break
		}
	}

	assert.True(t, found, "challenge should appear in solve percentages")
}

func TestStatisticsRepo_GetChallengeSolvePercentages_CancelledContext(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.StatisticsRepo.GetChallengeSolvePercentages(ctx, nil)
	require.Error(t, err)
}

func TestStatisticsRepo_GetScoreDistribution(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "dist")
	chall := f.CreateChallenge(t, "dist_ch", 200)
	f.CreateSolve(t, user.ID, team.ID, chall.ID)

	buckets, err := f.StatisticsRepo.GetScoreDistribution(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, buckets)
	// at least one non-zero bucket for the team that solved
	total := 0

	for _, b := range buckets {
		total += b.Count
	}

	assert.GreaterOrEqual(t, total, 1)
}

func TestStatisticsRepo_GetScoreDistribution_CancelledContext(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.StatisticsRepo.GetScoreDistribution(ctx, nil)
	require.Error(t, err)
}

func TestStatisticsRepo_GetSubmissionTimeSeries(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "ts")
	chall := f.CreateChallenge(t, "ts_ch", 100)

	// Insert a submission row directly
	_, err := f.Pool.Exec(ctx,
		"INSERT INTO submissions (id, user_id, team_id, challenge_id, submitted_flag, is_correct, submission_type) VALUES ($1, $2, $3, $4, $5, TRUE, 'correct')",
		uuid.New(), user.ID, team.ID, chall.ID, "FLAG{test}",
	)
	require.NoError(t, err)

	series, err := f.StatisticsRepo.GetSubmissionTimeSeries(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, series)
	assert.GreaterOrEqual(t, series.TotalCorrect+series.TotalIncorrect, 1)
}

func TestStatisticsRepo_GetSubmissionTimeSeries_CancelledContext(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.StatisticsRepo.GetSubmissionTimeSeries(ctx, nil)
	require.Error(t, err)
}

func TestStatisticsRepo_GetSubmissionTimeSeriesByType(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "tstype")
	chall := f.CreateChallenge(t, "tstype_ch", 100)

	_, err := f.Pool.Exec(ctx,
		"INSERT INTO submissions (id, user_id, team_id, challenge_id, submitted_flag, is_correct, submission_type) VALUES ($1, $2, $3, $4, $5, FALSE, 'incorrect')",
		uuid.New(), user.ID, team.ID, chall.ID, "WRONG{flag}",
	)
	require.NoError(t, err)

	series, err := f.StatisticsRepo.GetSubmissionTimeSeriesByType(ctx, false, nil)
	require.NoError(t, err)
	require.NotNil(t, series)
	// should have at least one data point for the wrong submission we just inserted
	total := 0

	for _, pt := range series {
		total += pt.Count
	}

	assert.GreaterOrEqual(t, total, 1)
}

func TestStatisticsRepo_GetAdminStatisticsFunnel_ScenariosAndFilters(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()
	now := time.Now().UTC()

	openedUser, openedTeam := f.CreateUserWithTeam(t, "funnel_opened")
	attemptedUser, attemptedTeam := f.CreateUserWithTeam(t, "funnel_attempted")
	solvedUser, solvedTeam := f.CreateUserWithTeam(t, "funnel_solved")
	neverUser, neverTeam := f.CreateUserWithTeam(t, "funnel_never")
	bannedUser, bannedTeam := f.CreateUserWithTeam(t, "funnel_banned_team")
	hiddenUser, hiddenTeam := f.CreateUserWithTeam(t, "funnel_hidden_team")
	userBanned, userBannedTeam := f.CreateUserWithTeam(t, "funnel_banned_user")

	openedChallenge := f.CreateChallenge(t, "funnel_opened_ch", 100)
	attemptedChallenge := f.CreateChallenge(t, "funnel_attempted_ch", 100)
	solvedChallenge := f.CreateChallenge(t, "funnel_solved_ch", 100)
	neverChallenge := f.CreateChallenge(t, "funnel_never_ch", 100)
	filteredChallenge := f.CreateChallenge(t, "funnel_filtered_ch", 100)
	hiddenChallenge := f.CreateChallenge(t, "funnel_hidden_ch", 100)

	_, err := f.Pool.Exec(ctx, "UPDATE challenges SET state = $1 WHERE id = $2", domain.ChallengeStateHidden, hiddenChallenge.ID)
	require.NoError(t, err)
	_, err = f.Pool.Exec(ctx, "UPDATE teams SET is_banned = TRUE WHERE id = $1", bannedTeam.ID)
	require.NoError(t, err)
	_, err = f.Pool.Exec(ctx, "UPDATE teams SET is_hidden = TRUE WHERE id = $1", hiddenTeam.ID)
	require.NoError(t, err)
	_, err = f.Pool.Exec(ctx, "UPDATE users SET is_banned = TRUE WHERE id = $1", userBanned.ID)
	require.NoError(t, err)

	insertChallengeOpenAt(t, f, openedUser.ID, &openedTeam.ID, openedChallenge.ID, now.Add(-6*time.Minute))
	insertSubmissionAt(t, f, attemptedUser.ID, &attemptedTeam.ID, attemptedChallenge.ID, false, now.Add(-5*time.Minute), nil, nil)
	insertChallengeOpenAt(t, f, solvedUser.ID, &solvedTeam.ID, solvedChallenge.ID, now.Add(-4*time.Minute))
	insertSubmissionAt(t, f, solvedUser.ID, &solvedTeam.ID, solvedChallenge.ID, true, now.Add(-3*time.Minute), nil, nil)
	insertSolveAt(t, f, solvedUser.ID, solvedTeam.ID, solvedChallenge.ID, now.Add(-2*time.Minute), 100, nil, nil)
	insertChallengeOpenAt(t, f, neverUser.ID, &neverTeam.ID, hiddenChallenge.ID, now.Add(-2*time.Minute))

	insertChallengeOpenAt(t, f, bannedUser.ID, &bannedTeam.ID, filteredChallenge.ID, now.Add(-5*time.Minute))
	insertSubmissionAt(t, f, bannedUser.ID, &bannedTeam.ID, filteredChallenge.ID, false, now.Add(-4*time.Minute), nil, &bannedTeam.ID)
	insertSolveAt(t, f, bannedUser.ID, bannedTeam.ID, filteredChallenge.ID, now.Add(-3*time.Minute), 100, nil, &bannedTeam.ID)
	insertChallengeOpenAt(t, f, hiddenUser.ID, &hiddenTeam.ID, filteredChallenge.ID, now.Add(-5*time.Minute))
	insertSubmissionAt(t, f, hiddenUser.ID, &hiddenTeam.ID, filteredChallenge.ID, false, now.Add(-4*time.Minute), nil, &hiddenTeam.ID)
	insertSolveAt(t, f, hiddenUser.ID, hiddenTeam.ID, filteredChallenge.ID, now.Add(-3*time.Minute), 100, nil, &hiddenTeam.ID)
	insertChallengeOpenAt(t, f, userBanned.ID, &userBannedTeam.ID, filteredChallenge.ID, now.Add(-5*time.Minute))
	insertSubmissionAt(t, f, userBanned.ID, &userBannedTeam.ID, filteredChallenge.ID, false, now.Add(-4*time.Minute), &userBanned.ID, nil)
	insertSolveAt(t, f, userBanned.ID, userBannedTeam.ID, filteredChallenge.ID, now.Add(-3*time.Minute), 100, &userBanned.ID, nil)

	funnel, err := f.StatisticsRepo.GetAdminStatisticsFunnel(ctx, 1000, nil)
	require.NoError(t, err)
	require.NotNil(t, funnel)

	openedRow := requireFunnelChallengeRow(t, funnel, openedChallenge.ID)
	assert.Equal(t, 1, openedRow.OpenedCount)
	assert.Equal(t, 0, openedRow.AttemptedCount)
	assert.Equal(t, 0, openedRow.SolvedCount)

	attemptedRow := requireFunnelChallengeRow(t, funnel, attemptedChallenge.ID)
	assert.Equal(t, 0, attemptedRow.OpenedCount)
	assert.Equal(t, 1, attemptedRow.AttemptedCount)
	assert.Equal(t, 0, attemptedRow.SolvedCount)

	solvedRow := requireFunnelChallengeRow(t, funnel, solvedChallenge.ID)
	assert.Equal(t, 1, solvedRow.OpenedCount)
	assert.Equal(t, 1, solvedRow.AttemptedCount)
	assert.Equal(t, 1, solvedRow.SolvedCount)

	neverRow := requireFunnelChallengeRow(t, funnel, neverChallenge.ID)
	assert.Equal(t, 0, neverRow.OpenedCount)
	assert.Equal(t, 0, neverRow.AttemptedCount)
	assert.Equal(t, 0, neverRow.SolvedCount)

	filteredRow := requireFunnelChallengeRow(t, funnel, filteredChallenge.ID)
	assert.Equal(t, 0, filteredRow.OpenedCount)
	assert.Equal(t, 0, filteredRow.AttemptedCount)
	assert.Equal(t, 0, filteredRow.SolvedCount)
	assert.Nil(t, findFunnelChallengeRow(funnel, hiddenChallenge.ID))

	openedTeamCell := requireFunnelTeamCell(t, funnel, openedTeam.ID, openedChallenge.ID)
	assert.True(t, openedTeamCell.Opened)
	assert.False(t, openedTeamCell.Attempted)
	assert.False(t, openedTeamCell.Solved)
	require.NotNil(t, openedTeamCell.FirstOpenedAt)
	assert.Nil(t, openedTeamCell.FirstAttemptedAt)
	assert.Nil(t, openedTeamCell.SolvedAt)

	attemptedTeamCell := requireFunnelTeamCell(t, funnel, attemptedTeam.ID, attemptedChallenge.ID)
	assert.False(t, attemptedTeamCell.Opened)
	assert.True(t, attemptedTeamCell.Attempted)
	assert.False(t, attemptedTeamCell.Solved)

	solvedTeamCell := requireFunnelTeamCell(t, funnel, solvedTeam.ID, solvedChallenge.ID)
	assert.True(t, solvedTeamCell.Opened)
	assert.True(t, solvedTeamCell.Attempted)
	assert.True(t, solvedTeamCell.Solved)

	neverTeamCell := requireFunnelTeamCell(t, funnel, neverTeam.ID, neverChallenge.ID)
	assert.False(t, neverTeamCell.Opened)
	assert.False(t, neverTeamCell.Attempted)
	assert.False(t, neverTeamCell.Solved)

	solvedUserCell := requireFunnelUserCell(t, funnel, solvedUser.ID, solvedChallenge.ID)
	assert.True(t, solvedUserCell.Opened)
	assert.True(t, solvedUserCell.Attempted)
	assert.True(t, solvedUserCell.Solved)

	assert.Nil(t, findFunnelTeamRow(funnel, bannedTeam.ID))
	assert.Nil(t, findFunnelTeamRow(funnel, hiddenTeam.ID))
	assert.Nil(t, findFunnelUserRow(funnel, userBanned.ID))
}

func TestStatisticsRepo_GetAdminStatisticsFunnel_RespectsFreezeTime(t *testing.T) {
	t.Parallel()
	pool := SetupTestPool(t)
	f := NewTestFixture(pool.Pool)
	ctx := context.Background()

	user, team := f.CreateUserWithTeam(t, "funnel_freeze")
	challenge := f.CreateChallenge(t, "funnel_freeze_ch", 100)
	now := time.Now().UTC()
	beforeFreeze := now.Add(-2 * time.Hour)
	freezeTime := now.Add(-1 * time.Hour)
	afterFreeze := now.Add(-30 * time.Minute)

	insertChallengeOpenAt(t, f, user.ID, &team.ID, challenge.ID, beforeFreeze)
	insertSubmissionAt(t, f, user.ID, &team.ID, challenge.ID, true, afterFreeze, nil, nil)
	insertSolveAt(t, f, user.ID, team.ID, challenge.ID, afterFreeze.Add(time.Minute), 100, nil, nil)

	frozen, err := f.StatisticsRepo.GetAdminStatisticsFunnel(ctx, 1000, &freezeTime)
	require.NoError(t, err)
	frozenChallenge := requireFunnelChallengeRow(t, frozen, challenge.ID)
	assert.Equal(t, 1, frozenChallenge.OpenedCount)
	assert.Equal(t, 0, frozenChallenge.AttemptedCount)
	assert.Equal(t, 0, frozenChallenge.SolvedCount)

	frozenTeamCell := requireFunnelTeamCell(t, frozen, team.ID, challenge.ID)
	assert.True(t, frozenTeamCell.Opened)
	assert.False(t, frozenTeamCell.Attempted)
	assert.False(t, frozenTeamCell.Solved)

	live, err := f.StatisticsRepo.GetAdminStatisticsFunnel(ctx, 1000, nil)
	require.NoError(t, err)
	liveChallenge := requireFunnelChallengeRow(t, live, challenge.ID)
	assert.Equal(t, 1, liveChallenge.OpenedCount)
	assert.Equal(t, 1, liveChallenge.AttemptedCount)
	assert.Equal(t, 1, liveChallenge.SolvedCount)

	liveTeamCell := requireFunnelTeamCell(t, live, team.ID, challenge.ID)
	assert.True(t, liveTeamCell.Opened)
	assert.True(t, liveTeamCell.Attempted)
	assert.True(t, liveTeamCell.Solved)
}

func insertChallengeOpenAt(t *testing.T, f *TestFixture, userID uuid.UUID, teamID *uuid.UUID, challengeID uuid.UUID, openedAt time.Time) {
	t.Helper()

	_, err := f.Pool.Exec(context.Background(),
		"INSERT INTO challenge_opens (id, user_id, team_id, challenge_id, ip, opened_at) VALUES ($1, $2, $3, $4, $5, $6)",
		uuid.New(), userID, nullableUUID(teamID), challengeID, "127.0.0.1", openedAt,
	)
	require.NoError(t, err)
}

func insertSubmissionAt(t *testing.T, f *TestFixture, userID uuid.UUID, teamID *uuid.UUID, challengeID uuid.UUID, isCorrect bool, createdAt time.Time, bannedUserID, bannedTeamID *uuid.UUID) {
	t.Helper()

	submissionType := domain.SubmissionTypeFromCorrect(isCorrect)
	_, err := f.Pool.Exec(context.Background(),
		`INSERT INTO submissions
			(id, user_id, team_id, challenge_id, submitted_flag, is_correct, submission_type, ip, created_at, banned_user_id, banned_team_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		uuid.New(), userID, nullableUUID(teamID), challengeID, "FLAG{funnel}", isCorrect, submissionType, "127.0.0.1", createdAt, nullableUUID(bannedUserID), nullableUUID(bannedTeamID),
	)
	require.NoError(t, err)
}

func insertSolveAt(t *testing.T, f *TestFixture, userID, teamID, challengeID uuid.UUID, solvedAt time.Time, points int, bannedUserID, bannedTeamID *uuid.UUID) {
	t.Helper()

	_, err := f.Pool.Exec(context.Background(),
		`INSERT INTO solves
			(id, user_id, team_id, challenge_id, solved_at, points_at_solve, banned_user_id, banned_team_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		uuid.New(), userID, teamID, challengeID, solvedAt, points, nullableUUID(bannedUserID), nullableUUID(bannedTeamID),
	)
	require.NoError(t, err)
}

func nullableUUID(id *uuid.UUID) any {
	if id == nil {
		return nil
	}

	return *id
}

func requireFunnelChallengeRow(t *testing.T, funnel *domain.AdminStatisticsFunnel, challengeID uuid.UUID) *domain.FunnelChallengeRow {
	t.Helper()

	row := findFunnelChallengeRow(funnel, challengeID)
	require.NotNil(t, row, "missing funnel challenge row for %s", challengeID)

	return row
}

func findFunnelChallengeRow(funnel *domain.AdminStatisticsFunnel, challengeID uuid.UUID) *domain.FunnelChallengeRow {
	for _, row := range funnel.Challenges {
		if row.ChallengeID == challengeID {
			return row
		}
	}

	return nil
}

func findFunnelTeamRow(funnel *domain.AdminStatisticsFunnel, teamID uuid.UUID) *domain.FunnelTeamRow {
	for _, row := range funnel.Teams {
		if row.TeamID == teamID {
			return row
		}
	}

	return nil
}

func findFunnelUserRow(funnel *domain.AdminStatisticsFunnel, userID uuid.UUID) *domain.FunnelUserRow {
	for _, row := range funnel.Users {
		if row.UserID == userID {
			return row
		}
	}

	return nil
}

func requireFunnelTeamCell(t *testing.T, funnel *domain.AdminStatisticsFunnel, teamID, challengeID uuid.UUID) *domain.FunnelTeamCell {
	t.Helper()

	for _, cell := range funnel.TeamCells {
		if cell.TeamID == teamID && cell.ChallengeID == challengeID {
			return cell
		}
	}

	require.Failf(t, "missing funnel team cell", "team=%s challenge=%s", teamID, challengeID)

	return nil
}

func requireFunnelUserCell(t *testing.T, funnel *domain.AdminStatisticsFunnel, userID, challengeID uuid.UUID) *domain.FunnelUserCell {
	t.Helper()

	for _, cell := range funnel.UserCells {
		if cell.UserID == userID && cell.ChallengeID == challengeID {
			return cell
		}
	}

	require.Failf(t, "missing funnel user cell", "user=%s challenge=%s", userID, challengeID)

	return nil
}
