package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestE2E_SoloOnlyRegistration_AutoCreatesSoloTeam(t *testing.T) {
	s := newE2ESuite(t)

	defer resetCompetitionToActive()

	suffix := e2eUID("solo_auto")
	admin := s.registerAdmin("solo_auto_admin")
	start := time.Now().UTC().Add(time.Hour)
	setCompetitionTimes(start, start.Add(24*time.Hour), nil)

	mode := openapi.SoloOnly
	s.configureCompetitionMode(admin, "Solo Auto "+suffix, &mode)
	setCompetitionTimes(time.Now().UTC().Add(-time.Hour), time.Now().UTC().Add(24*time.Hour), nil)

	player := s.registerUser("solo_auto_player")
	myTeam, err := s.client.GetTeamsMyWithResponse(context.Background(), e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "get auto-created solo team", http.StatusOK, myTeam.StatusCode(), myTeam.Body)
	require.NotNil(t, myTeam.JSON200)
	require.NotNil(t, myTeam.JSON200.ID)
	require.NotNil(t, myTeam.JSON200.Name)
	require.NotNil(t, myTeam.JSON200.IsSolo)
	require.True(t, *myTeam.JSON200.IsSolo)

	player.TeamID = *myTeam.JSON200.ID
	player.TeamName = *myTeam.JSON200.Name

	challengeID := s.createChallenge(admin, "Solo Auto Challenge "+suffix, "flag{"+suffix+"}", 100)
	s.submitFlag(player, challengeID, "flag{"+suffix+"}", true, http.StatusOK)
	s.requireScoreboardTeam(admin.Token, player.TeamName, 100, &openapi.GetScoreboardParams{Live: new(true)})
}
