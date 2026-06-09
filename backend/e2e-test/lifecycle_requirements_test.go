package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	openapiTypes "github.com/oapi-codegen/runtime/types"
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
	gatedID := s.createChallengeWithState(admin, "Freeze Gated "+suffix, "flag{"+suffix+"_gated}", 200, openapi.CreateChallengeRequestStateVisible)
	s.setChallengeRequirements(admin, gatedID, warmupID)

	gatedBefore, err := s.client.GetChallengesChallengeIDWithResponse(context.Background(), gatedID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "freeze gated before requirement", http.StatusNotFound, gatedBefore.StatusCode(), gatedBefore.Body)
	s.submitFlag(player, gatedID, "flag{"+suffix+"_gated}", false, http.StatusForbidden)

	hiddenPrereqID := s.createChallengeWithState(admin, "Hidden prereq "+suffix, "flag{"+suffix+"_hidden}", 50, openapi.CreateChallengeRequestStateHidden)
	hiddenGatedID := s.createChallengeWithState(admin, "Hidden gated "+suffix, "flag{"+suffix+"_hidden_gated}", 75, openapi.CreateChallengeRequestStateVisible)
	s.setChallengeRequirements(admin, hiddenGatedID, hiddenPrereqID)

	hiddenBlocked, err := s.client.GetChallengesChallengeIDWithResponse(context.Background(), hiddenGatedID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "hidden prerequisite blocks dependent challenge", http.StatusNotFound, hiddenBlocked.StatusCode(), hiddenBlocked.Body)

	s.submitFlag(player, warmupID, "flag{"+suffix+"_warmup}", true, http.StatusOK)

	userID := openapiTypes.UUID(uuid.MustParse(player.UserID))
	teamID := openapiTypes.UUID(uuid.MustParse(player.TeamID))
	hiddenPrereqUUID := openapiTypes.UUID(uuid.MustParse(hiddenPrereqID))
	adminSolve, err := s.client.PostAdminSubmissionsWithResponse(context.Background(), openapi.PostAdminSubmissionsJSONRequestBody{
		UserID:        userID,
		TeamID:        &teamID,
		ChallengeID:   hiddenPrereqUUID,
		SubmittedFlag: "flag{" + suffix + "_hidden}",
		IsCorrect:     true,
	}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "admin solve hidden prerequisite", http.StatusCreated, adminSolve.StatusCode(), adminSolve.Body)

	hiddenAllowed, err := s.client.GetChallengesChallengeIDWithResponse(context.Background(), hiddenGatedID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "hidden prerequisite solve unlocks dependent challenge", http.StatusOK, hiddenAllowed.StatusCode(), hiddenAllowed.Body)

	gatedAfter, err := s.client.GetChallengesChallengeIDWithResponse(context.Background(), gatedID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "freeze gated after requirement", http.StatusOK, gatedAfter.StatusCode(), gatedAfter.Body)

	hardLockedID := s.createChallengeWithState(admin, "Hard locked "+suffix, "flag{"+suffix+"_hard_locked}", 125, openapi.CreateChallengeRequestStateLocked)
	s.setChallengeRequirements(admin, hardLockedID, warmupID)
	hardLockedDetail, err := s.client.GetChallengesChallengeIDWithResponse(context.Background(), hardLockedID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "hard locked detail after requirement", http.StatusOK, hardLockedDetail.StatusCode(), hardLockedDetail.Body)
	s.submitFlag(player, hardLockedID, "flag{"+suffix+"_hard_locked}", false, http.StatusForbidden)

	freezeTime := time.Now().UTC()

	time.Sleep(20 * time.Millisecond)
	setCompetitionTimes(freezeTime.Add(-time.Hour), freezeTime.Add(24*time.Hour), &freezeTime)
	invalidateScoreboardCache(context.Background())

	s.submitFlag(player, gatedID, "flag{"+suffix+"_gated}", true, http.StatusOK)

	_, frozenPoints := s.requireScoreboardTeam(admin.Token, player.TeamName, 100, &openapi.GetScoreboardParams{})
	require.Equal(t, 100, frozenPoints)

	_, livePoints := s.requireScoreboardTeam(admin.Token, player.TeamName, 300, &openapi.GetScoreboardParams{Live: new(true)})
	require.Equal(t, 300, livePoints)
}
