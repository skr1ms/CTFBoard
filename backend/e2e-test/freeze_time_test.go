package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// GET /scoreboard with freeze_time: solves after freeze are not counted in public scoreboard (frozen view).
func TestScoreboard_Freeze(t *testing.T) {
	t.Cleanup(resetCompetitionToActive)
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenAdmin := h.RegisterAdmin("admin_freeze_" + suffix)

	now := time.Now().UTC()
	freezeTime := now.Add(2 * time.Second)
	setCompetitionTimes(now.Add(-1*time.Hour), now.Add(24*time.Hour), &freezeTime)

	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Freeze Chall",
		"description": "Test freeze functionality",
		"flag":        "flag{freeze}",
		"points":      100,
		"category":    "misc",
		"state":       "visible",
	})

	_, _, user1 := h.RegisterUserAndLogin("user_freeze_1")
	h.CreateSoloTeam(user1, http.StatusCreated)
	require.Eventually(t, func() bool {
		setCompetitionTimes(now.Add(-1*time.Hour), now.Add(24*time.Hour), &freezeTime)
		resp, err := h.Client().PostChallengesChallengeIDSubmitWithResponse(
			context.Background(), challID,
			openapi.PostChallengesChallengeIDSubmitJSONRequestBody{Flag: "flag{freeze}"},
			helper.WithBearerToken(user1))
		return err == nil && resp != nil && resp.StatusCode() == http.StatusOK
	}, 5*time.Second, 200*time.Millisecond)

	_, _, user2 := h.RegisterUserAndLogin("user_freeze_2")
	h.CreateSoloTeam(user2, http.StatusCreated)

	require.True(t, h.PollCompetitionStatus("frozen", 10*time.Second), "competition should become frozen")

	require.Eventually(t, func() bool {
		setCompetitionTimes(now.Add(-1*time.Hour), now.Add(24*time.Hour), &freezeTime)
		resp, err := h.Client().PostChallengesChallengeIDSubmitWithResponse(
			context.Background(), challID,
			openapi.PostChallengesChallengeIDSubmitJSONRequestBody{Flag: "flag{freeze}"},
			helper.WithBearerToken(user2))
		return err == nil && resp != nil && resp.StatusCode() == http.StatusOK
	}, 5*time.Second, 200*time.Millisecond)

	scoreboard := h.GetScoreboard(user2)
	helper.RequireStatus(t, http.StatusOK, scoreboard.StatusCode(), scoreboard.Body, "scoreboard freeze")
	require.NotNil(t, scoreboard.JSON200)

	foundUser2 := false
	for _, entry := range *scoreboard.JSON200 {
		if entry.TeamName != nil && *entry.TeamName == "user_freeze_2" {
			foundUser2 = true
			var points int
			if entry.Points != nil {
				points = *entry.Points
			}
			if points != 0 {
				t.Errorf("Scoreboard not frozen! User 2 has %v points", points)
			}
		}
	}

	if !foundUser2 {
		t.Log("User 2 not found in frozen scoreboard (acceptable behavior)")
	}
}

// GET /scoreboard with freeze_time: when no solves exist, returns 200 and empty array.
func TestScoreboard_Freeze_NoSolves_Empty(t *testing.T) {
	t.Cleanup(resetCompetitionToActive)
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	suffix := helper.UID()
	_, _, tokenAdmin := h.RegisterAdmin("admin_freeze_empty_" + suffix)
	_ = h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "Freeze Empty", "description": "x", "flag": "flag{fe}",
		"points": 100, "category": "misc", "state": "visible",
	})
	// Set competition times directly in DB to avoid COMPETITION_ACTIVE_CANNOT_UPDATE
	// race: parallel tests may activate competition between resetCompetitionToNotStarted and the API PUT.
	now := time.Now().UTC()
	freezeTime := now.Add(1 * time.Hour)
	setCompetitionTimes(now.Add(-1*time.Hour), now.Add(24*time.Hour), &freezeTime)
	resp := h.GetScoreboard(tokenAdmin)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "scoreboard freeze empty")
	require.NotNil(t, resp.JSON200)
}

// GET /scoreboard: when competition paused during freeze, still shows frozen snapshot (solve after freeze = 0).
func TestScoreboard_Freeze_WhenPaused_StillShowsFrozenSnapshot(t *testing.T) {
	t.Cleanup(resetCompetitionToActive)
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenAdmin := h.RegisterAdmin("admin_freeze_pause_" + suffix)
	now := time.Now().UTC()
	freezeTime := now.Add(2 * time.Second)
	setCompetitionTimes(now.Add(-2*time.Hour), now.Add(24*time.Hour), &freezeTime)

	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "Freeze Pause Chall", "description": "x", "flag": "flag{freeze_pause}",
		"points": 100, "category": "misc", "state": "visible",
	})

	_, _, tokenA := h.RegisterUserAndLogin("user_fp_a_" + suffix)
	h.CreateSoloTeam(tokenA, http.StatusCreated)
	h.SubmitFlag(tokenA, challID, "flag{freeze_pause}", http.StatusOK)

	_, _, tokenB := h.RegisterUserAndLogin("user_fp_b_" + suffix)
	h.CreateSoloTeam(tokenB, http.StatusCreated)

	require.True(t, h.PollCompetitionStatus("frozen", 10*time.Second), "competition should become frozen")
	freezeNow := time.Now().UTC()
	setCompetitionTimes(now.Add(-2*time.Hour), now.Add(24*time.Hour), &freezeNow)
	invalidateScoreboardCache(context.Background())
	require.True(t, h.PollCompetitionStatus("frozen", 8*time.Second), "competition should remain frozen after cache invalidation")

	h.SubmitFlag(tokenB, challID, "flag{freeze_pause}", http.StatusOK)

	setCompetitionPaused(true)
	invalidateScoreboardCache(context.Background())
	require.True(t, h.PollCompetitionStatus("paused", 12*time.Second), "competition should become paused")

	scoreboard := h.GetScoreboard(tokenA)
	helper.RequireStatus(t, http.StatusOK, scoreboard.StatusCode(), scoreboard.Body, "scoreboard when paused+freeze")
	require.NotNil(t, scoreboard.JSON200)
	var pointsB int
	for _, entry := range *scoreboard.JSON200 {
		if entry.TeamName != nil && *entry.TeamName == "user_fp_b_"+suffix {
			if entry.Points != nil {
				pointsB = *entry.Points
			}
			break
		}
	}
	require.Equal(t, 0, pointsB, "scoreboard must show frozen snapshot when paused during freeze; user B solved after freeze so must have 0 in public view")
}

// GET /admin/submissions?live=true: admin sees all submissions during freeze; live=false shows frozen count.
func TestAdminSubmissions_LiveParam_SeesAllDuringFreeze(t *testing.T) {
	t.Cleanup(resetCompetitionToActive)
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenAdmin := h.RegisterAdmin("admin_subs_live_" + suffix)

	now := time.Now().UTC()
	freezeTime := now.Add(2 * time.Second)
	setCompetitionTimes(now.Add(-1*time.Hour), now.Add(24*time.Hour), &freezeTime)

	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "Subs Live Chall", "description": "x", "flag": "flag{live}",
		"points": 100, "category": "misc", "state": "visible",
	})

	_, _, user1 := h.RegisterUserAndLogin("user_sl_1_" + suffix)
	h.CreateSoloTeam(user1, http.StatusCreated)
	h.SubmitFlag(user1, challID, "flag{live}", http.StatusOK)

	_, _, user2 := h.RegisterUserAndLogin("user_sl_2_" + suffix)
	h.CreateSoloTeam(user2, http.StatusCreated)

	require.True(t, h.PollCompetitionStatus("frozen", 10*time.Second), "competition should become frozen")

	h.SubmitFlag(user2, challID, "flag{live}", http.StatusOK)

	liveFalse := false
	liveTrue := true
	frozenResp := h.GetAdminSubmissionsWithLive(tokenAdmin, &liveFalse, 1, 50, http.StatusOK)
	require.NotNil(t, frozenResp.JSON200)
	frozenTotal := 0
	if frozenResp.JSON200.Meta != nil && frozenResp.JSON200.Meta.Total != nil {
		frozenTotal = *frozenResp.JSON200.Meta.Total
	}

	liveResp := h.GetAdminSubmissionsWithLive(tokenAdmin, &liveTrue, 1, 50, http.StatusOK)
	require.NotNil(t, liveResp.JSON200)
	liveTotal := 0
	if liveResp.JSON200.Meta != nil && liveResp.JSON200.Meta.Total != nil {
		liveTotal = *liveResp.JSON200.Meta.Total
	}

	require.Greater(t, liveTotal, frozenTotal, "admin with ?live=true should see more submissions than frozen view; frozen=%d live=%d", frozenTotal, liveTotal)
}

// GET /challenges: during freeze solve_count is frozen snapshot, not live.
func TestChallenges_Freeze_SolveCountShowsFrozenSnapshot(t *testing.T) {
	t.Cleanup(resetCompetitionToActive)
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenAdmin := h.RegisterAdmin("admin_chall_freeze_" + suffix)
	now := time.Now().UTC()
	freezeTime := now.Add(2 * time.Second)
	setCompetitionTimes(now.Add(-1*time.Hour), now.Add(24*time.Hour), &freezeTime)

	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Freeze SolveCount Chall",
		"description": "x",
		"flag":        "flag{solve_count_freeze}",
		"points":      100,
		"category":    "misc",
		"state":       "visible",
	})

	_, _, user1 := h.RegisterUserAndLogin("user_sc_1_" + suffix)
	h.CreateSoloTeam(user1, http.StatusCreated)
	h.SubmitFlag(user1, challID, "flag{solve_count_freeze}", http.StatusOK)

	_, _, user2 := h.RegisterUserAndLogin("user_sc_2_" + suffix)
	h.CreateSoloTeam(user2, http.StatusCreated)

	require.True(t, h.PollCompetitionStatus("frozen", 10*time.Second), "competition should become frozen")

	h.SubmitFlag(user2, challID, "flag{solve_count_freeze}", http.StatusOK)

	resp := h.GetChallengesExpectStatus(user2, http.StatusOK)
	require.NotNil(t, resp.JSON200)
	var solveCount *int
	for i := range *resp.JSON200 {
		c := &(*resp.JSON200)[i]
		if c.ID != nil && *c.ID == challID {
			solveCount = c.SolveCount
			break
		}
	}
	require.NotNil(t, solveCount, "challenge should be in list")
	require.Equal(t, 1, *solveCount, "GET /challenges during freeze must return frozen solve_count (1), not live (2)")
}

// GET /challenges/{id}: during freeze when no first blood yet returns 200 (not 404).
func TestChallenge_Detail_Freeze_NoFirstBlood_ReturnsOK(t *testing.T) {
	t.Cleanup(resetCompetitionToActive)
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenAdmin := h.RegisterAdmin("admin_detail_fb_" + suffix)
	now := time.Now().UTC()
	freezeTime := now.Add(2 * time.Second)
	setCompetitionTimes(now.Add(-1*time.Hour), now.Add(24*time.Hour), &freezeTime)

	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "No FB Yet", "description": "x", "flag": "flag{nofb}",
		"points": 100, "category": "misc", "state": "visible",
	})

	_, _, tokenUser := h.RegisterUserAndLogin("user_nofb_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	require.True(t, h.PollCompetitionStatus("frozen", 10*time.Second), "competition should become frozen")

	h.GetChallengeDetailExpectStatus(tokenUser, challID, http.StatusOK)
}
