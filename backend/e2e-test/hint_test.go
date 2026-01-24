package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestHint_Flow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]
	_, _, tokenAdmin := h.RegisterAdmin("admin_" + suffix)

	h.StartCompetition(tokenAdmin)

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

	userName := "user_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(userName)

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
