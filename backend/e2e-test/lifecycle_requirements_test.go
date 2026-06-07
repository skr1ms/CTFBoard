package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestE2E_ChallengeRequirementsAndFreeze(t *testing.T) {
	s := newE2ESuite(t)

	defer resetCompetitionToActive()

	admin := s.registerAdmin("freeze_admin")
	player := s.registerUser("freeze_player")
	s.createSoloTeam(&player)

	suffix := e2eUID("freeze")
	warmupID := s.createChallengeWithState(admin, "Freeze Warmup "+suffix, "flag{"+suffix+"_warmup}", 100, openapi.CreateChallengeRequestStateVisible)
	lockedID := s.createChallengeWithState(admin, "Freeze Locked "+suffix, "flag{"+suffix+"_locked}", 200, openapi.CreateChallengeRequestStateVisible)
	s.setChallengeRequirements(admin, lockedID, warmupID)

	lockedBefore, err := s.client.GetChallengesChallengeIDWithResponse(context.Background(), lockedID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "freeze locked before requirement", http.StatusNotFound, lockedBefore.StatusCode(), lockedBefore.Body)
	s.submitFlag(player, lockedID, "flag{"+suffix+"_locked}", false, http.StatusForbidden)

	s.submitFlag(player, warmupID, "flag{"+suffix+"_warmup}", true, http.StatusOK)

	lockedAfter, err := s.client.GetChallengesChallengeIDWithResponse(context.Background(), lockedID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "freeze locked after requirement", http.StatusOK, lockedAfter.StatusCode(), lockedAfter.Body)

	freezeTime := time.Now().UTC()

	time.Sleep(20 * time.Millisecond)
	setCompetitionTimes(freezeTime.Add(-time.Hour), freezeTime.Add(24*time.Hour), &freezeTime)
	invalidateScoreboardCache(context.Background())

	s.submitFlag(player, lockedID, "flag{"+suffix+"_locked}", true, http.StatusOK)

	_, frozenPoints := s.requireScoreboardTeam(admin.Token, player.TeamName, 100, &openapi.GetScoreboardParams{})
	require.Equal(t, 100, frozenPoints)

	_, livePoints := s.requireScoreboardTeam(admin.Token, player.TeamName, 300, &openapi.GetScoreboardParams{Live: new(true)})
	require.Equal(t, 300, livePoints)
}
