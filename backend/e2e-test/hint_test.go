package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// POST /challenges/{challengeID}/hints/{hintID}/unlock: hint locked until user has points; unlock deducts cost; score reflects deduction.
func TestHint_Flow(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

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

	h.AssertTeamScore(tokenUser, userName, 90)
}

// PUT /admin/hints/{ID}: admin updates hint content and cost; GET reflects new values.
func TestHint_Update_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_hint_up_err")

	h.UpdateHint(tokenAdmin, "00000000-0000-0000-0000-000000000000", "content", 10, http.StatusNotFound)
}

// DELETE /admin/hints/{ID}: admin deletes hint; GET /challenges/{ID}/hints no longer returns it.
func TestHint_Delete_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_hint_del_err")

	h.DeleteHint(tokenAdmin, "00000000-0000-0000-0000-000000000000", http.StatusNoContent)
}

// POST /challenges/{ID}/hints/{hintID}/unlock: non-existent hint returns 404.
func TestHint_Unlock_NotFound(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_hint_unlock_404")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Unlock Chal", "flag{u}", 50)
	_, _, tokenUser := h.RegisterUserAndLogin("unlock_user")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.UnlockHint(tokenUser, challengeID, "00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}

// GET /challenges/{id}/hints: when challenge has unmet requirements (locked) returns 404.
func TestHint_LockedChallenge_HintsReturnNotFound(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("admin_hint_locked_" + suffix)
	prereqID := h.CreateBasicChallenge(tokenAdmin, "Prereq Hint", "flag{ph}", 50)
	mainID := h.CreateBasicChallenge(tokenAdmin, "Main Hint Locked", "flag{mh}", 100)
	h.SetChallengeRequirements(tokenAdmin, mainID, []string{prereqID})
	h.CreateHint(tokenAdmin, mainID, "Hint on locked challenge", 0)

	_, _, tokenUser := h.RegisterUserAndLogin("hint_locked_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.GetChallengesChallengeIDHintsExpectStatus(tokenUser, mainID, http.StatusNotFound)
}

// GET /admin/unlocks: admin lists all hint unlocks paginated.
func TestHint_AdminListUnlocks_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_unlocks_ok")
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Unlock List Chal",
		"description":   "desc",
		"points":        100,
		"flag":          "flag{unlock_list}",
		"category":      "misc",
		"initial_value": 100,
		"min_value":     100,
		"decay":         1,
	})
	hintID := h.CreateHint(tokenAdmin, challengeID, "Some hint", 0)

	_, _, tokenUser := h.RegisterUserAndLogin("unlocks_list_user")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{unlock_list}", http.StatusOK)
	h.UnlockHint(tokenUser, challengeID, hintID, http.StatusOK)

	page := 1
	perPage := 50
	resp, err := h.Client().GetAdminUnlocksWithResponse(context.Background(), &openapi.GetAdminUnlocksParams{
		Page:    &page,
		PerPage: &perPage,
	}, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "admin list unlocks")
	require.NotNil(t, resp.JSON200)
}

// GET /admin/unlocks: non-admin returns 403 Forbidden.
func TestHint_AdminListUnlocks_Forbidden(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.SetupCompetition("admin_unlocks_403")
	_, _, tokenUser := h.RegisterUserAndLogin("unlocks_forbid_user")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	page := 1
	perPage := 50
	resp, err := h.Client().GetAdminUnlocksWithResponse(context.Background(), &openapi.GetAdminUnlocksParams{
		Page:    &page,
		PerPage: &perPage,
	}, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, resp.StatusCode(), resp.Body, "admin list unlocks forbidden")
}
