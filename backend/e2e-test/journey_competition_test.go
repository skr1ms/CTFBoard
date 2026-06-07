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
	s.createSoloTeam(&player)
	challengeID := s.createChallenge(admin, "State challenge "+e2eUID("state"), "flag{state_ok}", 100)

	now := time.Now().UTC()
	setCompetitionTimes(now.Add(1*time.Hour), now.Add(2*time.Hour), nil)

	defer resetCompetitionToActive()

	list, err := s.client.GetChallengesWithResponse(context.Background(), nil, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "not-started challenge list", http.StatusOK, list.StatusCode(), list.Body)
	require.NotNil(t, list.JSON200)
	require.Empty(t, *list.JSON200)

	s.submitFlag(player, challengeID, "flag{state_ok}", false, http.StatusForbidden)

	resetCompetitionToActive()
	s.submitFlag(player, challengeID, "flag{state_ok}", true, http.StatusOK)
}
