package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// POST /challenges/{challengeID}/hints/{hintID}/unlock: hint locked until user has points; unlock deducts cost; score reflects deduction.
func TestHint_Flow(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_hint")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Hint Chal",
		"description":   "Test hint functionality",
		"points":        100,
		"flag":          "flag{hint}",
		"category":      "misc",
		"initial_value": 100,
		"min_value":     100,
		"decay":         1,
	})

	hintContent := "Secret Hint Content"
	hintCost := 10
	hintID := h.CreateHint(tokenAdmin, challengeID, hintContent, hintCost)

	userName := "user_hint"
	_, _, tokenUser := h.RegisterUserAndLogin(userName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	hintObj := h.GetHintFromList(tokenUser, challengeID, hintID)
	require.NotNil(t, hintObj.Unlocked)
	require.False(t, *hintObj.Unlocked)
	require.Nil(t, hintObj.Content)
	require.NotNil(t, hintObj.Cost)
	require.Equal(t, hintCost, *hintObj.Cost)

	h.UnlockHint(tokenUser, challengeID, hintID, http.StatusPaymentRequired)

	h.SubmitFlag(tokenUser, challengeID, "flag{hint}", http.StatusOK)

	unlockResp := h.UnlockHint(tokenUser, challengeID, hintID, http.StatusOK)
	require.NotNil(t, unlockResp.JSON200)
	require.NotNil(t, unlockResp.JSON200.Unlocked)
	require.True(t, *unlockResp.JSON200.Unlocked)
	require.NotNil(t, unlockResp.JSON200.Content)
	require.Equal(t, hintContent, *unlockResp.JSON200.Content)

	hintObjUnlocked := h.GetHintFromList(tokenUser, challengeID, hintID)
	require.NotNil(t, hintObjUnlocked.Unlocked)
	require.True(t, *hintObjUnlocked.Unlocked)
	require.NotNil(t, hintObjUnlocked.Content)
	require.Equal(t, hintContent, *hintObjUnlocked.Content)

	h.AssertTeamScore(userName, 90)
}

// PUT /admin/hints/{ID}: admin updates hint content and cost; GET reflects new values.
func TestHint_Update_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_hint_update")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "Hint Update Chal", "description": "desc", "points": 50, "flag": "flag{up}", "category": "misc",
	})
	hintID := h.CreateHint(tokenAdmin, challengeID, "Original", 5)

	resp := h.UpdateHint(tokenAdmin, hintID, "Updated content", 8, http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.Content)
	require.Equal(t, "Updated content", *resp.JSON200.Content)
	require.NotNil(t, resp.JSON200.Cost)
	require.Equal(t, 8, *resp.JSON200.Cost)

	hintAfter := h.GetHintFromList(tokenAdmin, challengeID, hintID)
	require.NotNil(t, hintAfter.Cost)
	require.Equal(t, 8, *hintAfter.Cost)
}

// PUT /admin/hints/{ID}: non-existent hint returns 404.
func TestHint_Update_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_hint_up_err")

	h.UpdateHint(tokenAdmin, "00000000-0000-0000-0000-000000000000", "content", 10, http.StatusNotFound)
}

// DELETE /admin/hints/{ID}: admin deletes hint; GET /challenges/{ID}/hints no longer returns it.
func TestHint_Delete_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_hint_del")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "Hint Del Chal", "description": "desc", "points": 50, "flag": "flag{del}", "category": "misc",
	})
	hintID := h.CreateHint(tokenAdmin, challengeID, "To delete", 0)

	h.DeleteHint(tokenAdmin, hintID, http.StatusNoContent)

	resp, err := h.Client().GetChallengesChallengeIDHintsWithResponse(context.Background(), challengeID, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	require.NotNil(t, resp.JSON200)
	for _, c := range *resp.JSON200 {
		if c.ID != nil && *c.ID == hintID {
			t.Fatal("hint should be gone after delete")
		}
	}
}

// DELETE /admin/hints/{ID}: non-existent hint returns 204 (idempotent).
func TestHint_Delete_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_hint_del_err")

	h.DeleteHint(tokenAdmin, "00000000-0000-0000-0000-000000000000", http.StatusNoContent)
}

// POST /challenges/{ID}/hints/{hintID}/unlock: non-existent hint returns 404.
func TestHint_Unlock_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_hint_unlock_404")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Unlock Chal", "flag{u}", 50)
	_, _, tokenUser := h.RegisterUserAndLogin("unlock_user")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.UnlockHint(tokenUser, challengeID, "00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}
