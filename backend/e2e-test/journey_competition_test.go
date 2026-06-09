package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestE2E_CompetitionStateBlocksAndRestoresSubmit(t *testing.T) {
	s := newE2ESuite(t)

	admin := s.registerAdmin("state_admin")
	player := s.registerUser("state_player")
	s.createTeam(&player, "State team "+e2eUID("state_team"))
	challengeID := s.createChallenge(admin, "State challenge "+e2eUID("state"), "flag{state_ok}", 100)

	now := time.Now().UTC()
	setCompetitionTimes(now.Add(1*time.Hour), now.Add(2*time.Hour), nil)

	defer resetCompetitionToActive()

	list, err := s.client.GetChallengesWithResponse(context.Background(), nil, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "not-started challenge list", http.StatusOK, list.StatusCode(), list.Body)
	require.NotNil(t, list.JSON200)
	require.Empty(t, *list.JSON200)

	detail, err := s.client.GetChallengesChallengeIDWithResponse(context.Background(), challengeID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "not-started challenge detail", http.StatusForbidden, detail.StatusCode(), detail.Body)

	files, err := s.client.GetChallengesChallengeIDFilesWithResponse(context.Background(), challengeID, nil, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "not-started challenge files", http.StatusForbidden, files.StatusCode(), files.Body)

	hints, err := s.client.GetChallengesChallengeIDHintsWithResponse(context.Background(), challengeID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "not-started challenge hints", http.StatusForbidden, hints.StatusCode(), hints.Body)

	tags, err := s.client.GetChallengesChallengeIDTagsWithResponse(context.Background(), challengeID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "not-started challenge tags", http.StatusForbidden, tags.StatusCode(), tags.Body)

	requirements, err := s.client.GetChallengesChallengeIDRequirementsWithResponse(context.Background(), challengeID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "not-started challenge requirements", http.StatusForbidden, requirements.StatusCode(), requirements.Body)

	solution, err := s.client.GetChallengesChallengeIDSolutionWithResponse(context.Background(), challengeID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "not-started challenge solution", http.StatusForbidden, solution.StatusCode(), solution.Body)

	solutions, err := s.client.GetChallengesSolutionsWithResponse(context.Background(), e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "not-started challenge solutions", http.StatusForbidden, solutions.StatusCode(), solutions.Body)

	adminDetail, err := s.client.GetChallengesChallengeIDWithResponse(context.Background(), challengeID, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "admin challenge detail before start", http.StatusOK, adminDetail.StatusCode(), adminDetail.Body)

	s.submitFlag(player, challengeID, "flag{state_ok}", false, http.StatusForbidden)

	resetCompetitionToActive()
	s.submitFlag(player, challengeID, "flag{state_ok}", true, http.StatusOK)
}
