package e2e_test

import (
	"net/http"
	"testing"
)

// POST /challenges/{challengeID}/hints/{hintID}/unlock: hint locked until user has points; unlock deducts cost; score reflects deduction.
func TestHint_Flow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_hint")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Hint Chal",
		"description": "Test hint functionality",
		"points":      100,
		"flag":        "flag{hint}",
		"category":    "misc",
	})

	hintContent := "Secret Hint Content"
	hintCost := 10
	hintID := h.CreateHint(tokenAdmin, challengeID, hintContent, hintCost)

	userName := "user_hint"
	_, _, tokenUser := h.RegisterUserAndLogin(userName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	hintObj := h.GetHintFromList(tokenUser, challengeID, hintID)
	hintObj.Value("unlocked").Boolean().IsFalse()
	hintObj.NotContainsKey("content")
	hintObj.Value("cost").Number().IsEqual(hintCost)

	h.UnlockHint(tokenUser, challengeID, hintID, http.StatusPaymentRequired)

	h.SubmitFlag(tokenUser, challengeID, "flag{hint}", http.StatusOK)

	unlockResp := h.UnlockHint(tokenUser, challengeID, hintID, http.StatusOK)
	unlockResp.Value("unlocked").Boolean().IsTrue()
	unlockResp.Value("content").String().IsEqual(hintContent)

	hintObjUnlocked := h.GetHintFromList(tokenUser, challengeID, hintID)
	hintObjUnlocked.Value("unlocked").Boolean().IsTrue()
	hintObjUnlocked.Value("content").String().IsEqual(hintContent)

	h.AssertTeamScore(userName, 90)
}
