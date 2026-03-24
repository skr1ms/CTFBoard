package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// GET /scoreboard with freeze_time: solves after freeze are not counted; frozen snapshot.
func TestFreezeTime_ScoreboardShowsFrozenSnapshot(t *testing.T) {
	t.Cleanup(resetCompetitionToActive)
	resetCompetitionToActive()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenAdmin := h.RegisterAdmin("adm_frz_" + suffix)
	now := time.Now().UTC()
	setCompetitionTimes(now.Add(-2*time.Hour), now.Add(24*time.Hour), nil)
	require.Eventually(t, func() bool {
		resp := h.GetCompetitionStatus()
		return resp.JSON200 != nil && resp.JSON200.Status != nil && *resp.JSON200.Status == "active"
	}, 6*time.Second, 50*time.Millisecond)

	challID := h.CreateBasicChallenge(tokenAdmin, "FrzChall "+suffix, "flag{frz_"+suffix+"}", 100)

	userA := "ua_" + suffix
	_, _, tokenA := h.RegisterUserAndLogin(userA)
	h.CreateSoloTeam(tokenA, http.StatusCreated)

	userB := "ub_" + suffix
	_, _, tokenB := h.RegisterUserAndLogin(userB)
	h.CreateSoloTeam(tokenB, http.StatusCreated)

	h.SubmitFlag(tokenA, challID, "flag{frz_"+suffix+"}", http.StatusOK)

	freezeIn2s := time.Now().UTC().Add(2 * time.Second)
	setCompetitionTimes(now.Add(-2*time.Hour), now.Add(24*time.Hour), &freezeIn2s)
	invalidateScoreboardCache(context.Background())
	require.True(t, h.PollCompetitionStatus("frozen", 15*time.Second), "competition should become frozen")

	h.SubmitFlag(tokenB, challID, "flag{frz_"+suffix+"}", http.StatusOK)

	resp := h.GetScoreboard(tokenA)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.NotNil(t, resp.JSON200)
	entryA, okA := lo.Find(*resp.JSON200, func(e openapi.ScoreboardEntryResponse) bool { return e.TeamName != nil && *e.TeamName == userA })
	require.True(t, okA, "team A should be in frozen scoreboard")
	require.NotNil(t, entryA.Points)
	require.Equal(t, 100, *entryA.Points, "frozen snapshot: team A should have 100 points")
	entryB, okB := lo.Find(*resp.JSON200, func(e openapi.ScoreboardEntryResponse) bool { return e.TeamName != nil && *e.TeamName == userB })
	require.True(t, okB, "team B may appear with 0 points")
	require.NotNil(t, entryB.Points)
	require.Equal(t, 0, *entryB.Points, "frozen snapshot: team B solve was after freeze, should have 0 points")
}

// GET /scoreboard?bracket_id=: returns only teams in that bracket.
func TestBracket_ScoreboardFilteredByBracket(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("adm_br_" + suffix)

	brA := h.CreateBracket(tokenAdmin, "BracketA", "A", false, http.StatusCreated)
	require.NotNil(t, brA.JSON201)
	require.NotNil(t, brA.JSON201.ID)
	bracketAID := *brA.JSON201.ID

	brB := h.CreateBracket(tokenAdmin, "BracketB", "B", false, http.StatusCreated)
	require.NotNil(t, brB.JSON201)
	require.NotNil(t, brB.JSON201.ID)
	bracketBID := *brB.JSON201.ID

	challID := h.CreateBasicChallenge(tokenAdmin, "BracketChall "+suffix, "flag{br_"+suffix+"}", 100)

	userA := "uba_" + suffix
	_, _, tokenA := h.RegisterUserAndLogin(userA)
	h.CreateSoloTeam(tokenA, http.StatusCreated)
	myA := h.GetMyTeam(tokenA, http.StatusOK)
	require.NotNil(t, myA.JSON200)
	require.NotNil(t, myA.JSON200.ID)
	h.SetTeamBracket(tokenAdmin, *myA.JSON200.ID, bracketAID, http.StatusOK)

	userB := "ubb_" + suffix
	_, _, tokenB := h.RegisterUserAndLogin(userB)
	h.CreateSoloTeam(tokenB, http.StatusCreated)
	myB := h.GetMyTeam(tokenB, http.StatusOK)
	require.NotNil(t, myB.JSON200)
	require.NotNil(t, myB.JSON200.ID)
	h.SetTeamBracket(tokenAdmin, *myB.JSON200.ID, bracketBID, http.StatusOK)

	h.SubmitFlag(tokenA, challID, "flag{br_"+suffix+"}", http.StatusOK)
	h.SubmitFlag(tokenB, challID, "flag{br_"+suffix+"}", http.StatusOK)

	invalidateScoreboardCache(context.Background())

	respAll := h.GetScoreboard(tokenA)
	require.Equal(t, http.StatusOK, respAll.StatusCode())
	require.NotNil(t, respAll.JSON200)
	require.GreaterOrEqual(t, len(*respAll.JSON200), 2)

	respBrA := h.GetScoreboardWithBracket(tokenA, bracketAID)
	require.Equal(t, http.StatusOK, respBrA.StatusCode())
	require.NotNil(t, respBrA.JSON200)
	entryA, okA := lo.Find(*respBrA.JSON200, func(e openapi.ScoreboardEntryResponse) bool { return e.TeamName != nil && *e.TeamName == userA })
	require.True(t, okA, "team A should be in bracket A scoreboard")
	require.NotNil(t, entryA.Points)
	require.Equal(t, 100, *entryA.Points)
	_, hasB := lo.Find(*respBrA.JSON200, func(e openapi.ScoreboardEntryResponse) bool { return e.TeamName != nil && *e.TeamName == userB })
	require.False(t, hasB, "team B should not appear in bracket A scoreboard")

	respBrB := h.GetScoreboardWithBracket(tokenA, bracketBID)
	require.Equal(t, http.StatusOK, respBrB.StatusCode())
	require.NotNil(t, respBrB.JSON200)
	entryB, okB := lo.Find(*respBrB.JSON200, func(e openapi.ScoreboardEntryResponse) bool { return e.TeamName != nil && *e.TeamName == userB })
	require.True(t, okB, "team B should be in bracket B scoreboard")
	require.NotNil(t, entryB.Points)
	require.Equal(t, 100, *entryB.Points)
}
