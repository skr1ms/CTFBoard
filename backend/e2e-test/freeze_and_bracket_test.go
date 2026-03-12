package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

func TestFreezeTime_ScoreboardShowsFrozenSnapshot(t *testing.T) {
	t.Helper()
	t.Parallel()
	t.Cleanup(resetCompetitionToActive)
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, tokenAdmin := h.SetupCompetition("adm_frz_" + suffix)
	now := time.Now().UTC()
	setCompetitionTimes(now.Add(-2*time.Hour), now.Add(24*time.Hour), nil)

	challID := h.CreateBasicChallenge(tokenAdmin, "FrzChall "+suffix, "flag{frz_"+suffix+"}", 100)

	userA := "ua_" + suffix
	_, _, tokenA := h.RegisterUserAndLogin(userA)
	h.CreateSoloTeam(tokenA, http.StatusCreated)

	userB := "ub_" + suffix
	_, _, tokenB := h.RegisterUserAndLogin(userB)
	h.CreateSoloTeam(tokenB, http.StatusCreated)

	h.SubmitFlag(tokenA, challID, "flag{frz_"+suffix+"}", http.StatusOK)

	freezeNow := time.Now().UTC()
	setCompetitionTimes(now.Add(-2*time.Hour), now.Add(24*time.Hour), &freezeNow)
	invalidateScoreboardCache(context.Background())
	time.Sleep(6 * time.Second)

	h.SubmitFlag(tokenB, challID, "flag{frz_"+suffix+"}", http.StatusOK)

	resp := h.GetScoreboard(tokenA)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.NotNil(t, resp.JSON200)
	var foundA, foundB bool
	var pointsA, pointsB int
	for _, e := range *resp.JSON200 {
		if e.TeamName != nil {
			switch *e.TeamName {
			case userA:
				foundA = true
				if e.Points != nil {
					pointsA = *e.Points
				}
			case userB:
				foundB = true
				if e.Points != nil {
					pointsB = *e.Points
				}
			}
		}
	}
	require.True(t, foundA, "team A should be in frozen scoreboard")
	require.Equal(t, 100, pointsA, "frozen snapshot: team A should have 100 points")
	require.True(t, foundB, "team B may appear with 0 points")
	require.Equal(t, 0, pointsB, "frozen snapshot: team B solve was after freeze, should have 0 points")
}

func TestBracket_ScoreboardFilteredByBracket(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
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
	var onlyA bool
	for _, e := range *respBrA.JSON200 {
		if e.TeamName != nil && *e.TeamName == userA {
			onlyA = true
			require.NotNil(t, e.Points)
			require.Equal(t, 100, *e.Points)
		}
		if e.TeamName != nil && *e.TeamName == userB {
			t.Fatal("team B should not appear in bracket A scoreboard")
		}
	}
	require.True(t, onlyA, "team A should be in bracket A scoreboard")

	respBrB := h.GetScoreboardWithBracket(tokenA, bracketBID)
	require.Equal(t, http.StatusOK, respBrB.StatusCode())
	require.NotNil(t, respBrB.JSON200)
	for _, e := range *respBrB.JSON200 {
		if e.TeamName != nil && *e.TeamName == userB {
			require.NotNil(t, e.Points)
			require.Equal(t, 100, *e.Points)
			return
		}
	}
	t.Fatal("team B should be in bracket B scoreboard")
}
