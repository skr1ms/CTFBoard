package e2e_test

import (
	"net/http"
	"testing"
)

func TestHint_Flow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition (Create Admin & Start)
	_, tokenAdmin := h.SetupCompetition("admin_hint")

	// 2. Create Challenge
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Hint Chal",
		"description": "Test hint functionality",
		"points":      100,
		"flag":        "flag{hint}",
		"category":    "misc",
	})

	// 3. Create Hint
	hintContent := "Secret Hint Content"
	hintCost := 10
	hintID := h.CreateHint(tokenAdmin, challengeID, hintContent, hintCost)

	// 4. Register User
	userName := "user_hint"
	_, _, tokenUser := h.RegisterUserAndLogin(userName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	// 5. Verify Hint is Locked
	hintObj := h.GetHintFromList(tokenUser, challengeID, hintID)
	hintObj.Value("unlocked").Boolean().IsFalse()
	hintObj.NotContainsKey("content")
	hintObj.Value("cost").Number().IsEqual(hintCost)

	// 6. Attempt Unlock without points (Expect 402)
	h.UnlockHint(tokenUser, challengeID, hintID, http.StatusPaymentRequired)

	// 7. Solve to gain points
	h.SubmitFlag(tokenUser, challengeID, "flag{hint}", http.StatusOK)

	// 8. Unlock Hint (Expect Success)
	unlockResp := h.UnlockHint(tokenUser, challengeID, hintID, http.StatusOK)
	unlockResp.Value("unlocked").Boolean().IsTrue()
	unlockResp.Value("content").String().IsEqual(hintContent)

	// 9. Verify Hint is Unlocked in List
	hintObjUnlocked := h.GetHintFromList(tokenUser, challengeID, hintID)
	hintObjUnlocked.Value("unlocked").Boolean().IsTrue()
	hintObjUnlocked.Value("content").String().IsEqual(hintContent)

	// 10. Verify Score Deduction
	h.AssertTeamScore(userName, 90) // 100 - 10 = 90
}
