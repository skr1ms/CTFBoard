package e2e

import (
	"testing"

	"github.com/google/uuid"
)

func TestHint_Flow(t *testing.T) {
	e := setupE2E(t)

	adminName := "admin_" + uuid.New().String()[:8]
	_, _, adminToken := registerAdmin(e, adminName)

	chal := map[string]interface{}{
		"title":       "Hint Chal",
		"description": "Desc",
		"category":    "Web",
		"points":      100,
		"flag":        "flag{hint}",
		"is_hidden":   false,
	}
	challengeID := createChallenge(e, adminToken, chal)

	hintReq := map[string]interface{}{
		"content":     "Secret Hint Content",
		"cost":        10,
		"order_index": 1,
	}
	hintResp := e.POST("/api/v1/admin/challenges/{id}/hints", challengeID).
		WithHeader("Authorization", adminToken).
		WithJSON(hintReq).
		Expect().
		Status(201).
		JSON().Object()

	hintID := hintResp.Value("id").String().Raw()
	hintResp.Value("content").String().IsEqual("Secret Hint Content")

	userName := "user_" + uuid.New().String()[:8]
	uEmail, uPass := registerUser(e, userName)
	userToken := login(e, uEmail, uPass)

	teamName := userName

	hintsArr := e.GET("/api/v1/challenges/{id}/hints", challengeID).
		WithHeader("Authorization", userToken).
		Expect().
		Status(200).
		JSON().Array()

	hintsArr.Length().IsEqual(1)
	firstHint := hintsArr.Value(0).Object()
	firstHint.Value("id").String().IsEqual(hintID)
	firstHint.NotContainsKey("content")
	firstHint.Value("unlocked").Boolean().IsFalse()
	firstHint.Value("cost").Number().IsEqual(10)

	e.POST("/api/v1/challenges/{cid}/hints/{hid}/unlock", challengeID, hintID).
		WithHeader("Authorization", userToken).
		Expect().
		Status(402)

	submitFlag(e, userToken, challengeID, "flag{hint}")
	unlockResp := e.POST("/api/v1/challenges/{cid}/hints/{hid}/unlock", challengeID, hintID).
		WithHeader("Authorization", userToken).
		Expect().
		Status(200).
		JSON().Object()

	unlockResp.Value("unlocked").Boolean().IsTrue()
	unlockResp.Value("content").String().IsEqual("Secret Hint Content")

	hintsArrUnlocked := e.GET("/api/v1/challenges/{id}/hints", challengeID).
		WithHeader("Authorization", userToken).
		Expect().
		Status(200).
		JSON().Array()

	firstHintUnlocked := hintsArrUnlocked.Value(0).Object()
	firstHintUnlocked.Value("unlocked").Boolean().IsTrue()
	firstHintUnlocked.Value("content").String().IsEqual("Secret Hint Content")

	scoreboard := e.GET("/api/v1/scoreboard").
		Expect().
		Status(200).
		JSON().Array()

	found := false
	for _, val := range scoreboard.Iter() {
		obj := val.Object()
		if obj.Value("team_name").String().Raw() == teamName {
			obj.Value("points").Number().IsEqual(90)
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("team %s not found in scoreboard", teamName)
	}
}
