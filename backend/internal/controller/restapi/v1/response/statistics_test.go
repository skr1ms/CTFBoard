package response

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestFromAdminStatisticsFunnelNil(t *testing.T) {
	t.Parallel()

	got := FromAdminStatisticsFunnel(nil)

	assert.Nil(t, got.Challenges)
	assert.Nil(t, got.Teams)
	assert.Nil(t, got.TeamCells)
	assert.Nil(t, got.Users)
	assert.Nil(t, got.UserCells)
}

func TestFromAdminStatisticsFunnelMapsAllSections(t *testing.T) {
	t.Parallel()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	openedAt := time.Unix(100, 0).UTC()
	attemptedAt := time.Unix(200, 0).UTC()
	solvedAt := time.Unix(300, 0).UTC()

	got := FromAdminStatisticsFunnel(&domain.AdminStatisticsFunnel{
		Challenges: []*domain.FunnelChallengeRow{
			{
				ChallengeID:       challengeID,
				ChallengeTitle:    "web warmup",
				ChallengeCategory: "Web",
				OpenedCount:       3,
				AttemptedCount:    2,
				SolvedCount:       1,
			},
		},
		Teams: []*domain.FunnelTeamRow{
			{
				TeamID:         teamID,
				TeamName:       "blue",
				OpenedCount:    3,
				AttemptedCount: 2,
				SolvedCount:    1,
			},
		},
		TeamCells: []*domain.FunnelTeamCell{
			{
				TeamID:           teamID,
				ChallengeID:      challengeID,
				Opened:           true,
				Attempted:        true,
				Solved:           true,
				FirstOpenedAt:    &openedAt,
				FirstAttemptedAt: &attemptedAt,
				SolvedAt:         &solvedAt,
			},
		},
		Users: []*domain.FunnelUserRow{
			{
				UserID:         userID,
				Username:       "alice",
				OpenedCount:    1,
				AttemptedCount: 1,
				SolvedCount:    1,
			},
		},
		UserCells: []*domain.FunnelUserCell{
			{
				UserID:           userID,
				ChallengeID:      challengeID,
				Opened:           true,
				Attempted:        true,
				Solved:           true,
				FirstOpenedAt:    &openedAt,
				FirstAttemptedAt: &attemptedAt,
				SolvedAt:         &solvedAt,
			},
		},
	})

	require.NotNil(t, got.Challenges)
	require.Len(t, *got.Challenges, 1)
	assert.Equal(t, challengeID, *(*got.Challenges)[0].ChallengeID)
	assert.Equal(t, "web warmup", *(*got.Challenges)[0].ChallengeTitle)
	assert.Equal(t, "Web", *(*got.Challenges)[0].ChallengeCategory)
	assert.Equal(t, 3, *(*got.Challenges)[0].OpenedCount)
	assert.Equal(t, 2, *(*got.Challenges)[0].AttemptedCount)
	assert.Equal(t, 1, *(*got.Challenges)[0].SolvedCount)

	require.NotNil(t, got.Teams)
	require.Len(t, *got.Teams, 1)
	assert.Equal(t, teamID, *(*got.Teams)[0].TeamID)
	assert.Equal(t, "blue", *(*got.Teams)[0].TeamName)

	require.NotNil(t, got.TeamCells)
	require.Len(t, *got.TeamCells, 1)
	assert.Equal(t, teamID, *(*got.TeamCells)[0].TeamID)
	assert.Equal(t, challengeID, *(*got.TeamCells)[0].ChallengeID)
	assert.True(t, *(*got.TeamCells)[0].Opened)
	assert.True(t, *(*got.TeamCells)[0].Attempted)
	assert.True(t, *(*got.TeamCells)[0].Solved)
	assert.Equal(t, openedAt, *(*got.TeamCells)[0].FirstOpenedAt)
	assert.Equal(t, attemptedAt, *(*got.TeamCells)[0].FirstAttemptedAt)
	assert.Equal(t, solvedAt, *(*got.TeamCells)[0].SolvedAt)

	require.NotNil(t, got.Users)
	require.Len(t, *got.Users, 1)
	assert.Equal(t, userID, *(*got.Users)[0].UserID)
	assert.Equal(t, "alice", *(*got.Users)[0].Username)

	require.NotNil(t, got.UserCells)
	require.Len(t, *got.UserCells, 1)
	assert.Equal(t, userID, *(*got.UserCells)[0].UserID)
	assert.Equal(t, challengeID, *(*got.UserCells)[0].ChallengeID)
	assert.True(t, *(*got.UserCells)[0].Opened)
	assert.True(t, *(*got.UserCells)[0].Attempted)
	assert.True(t, *(*got.UserCells)[0].Solved)
	assert.Equal(t, openedAt, *(*got.UserCells)[0].FirstOpenedAt)
	assert.Equal(t, attemptedAt, *(*got.UserCells)[0].FirstAttemptedAt)
	assert.Equal(t, solvedAt, *(*got.UserCells)[0].SolvedAt)
}
