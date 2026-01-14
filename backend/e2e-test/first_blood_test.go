package e2e

import (
	"testing"
	"time"
)

func TestFirstBlood_Display(t *testing.T) {
	e := setupE2E(t)

	_, _, tokenAdmin := registerAdmin(e, "adminfb")

	challengeId := createChallenge(e, tokenAdmin, map[string]interface{}{
		"title":       "First Blood Test",
		"description": "Test challenge",
		"points":      100,
		"flag":        "FLAG{firstblood}",
		"category":    "web",
		"difficulty":  "easy",
		"is_hidden":   false,
	})

	emailUser1, passUser1 := registerUser(e, "fbuser1")
	tokenUser1 := login(e, emailUser1, passUser1)

	emailUser2, passUser2 := registerUser(e, "fbuser2")
	tokenUser2 := login(e, emailUser2, passUser2)

	submitFlag(e, tokenUser1, challengeId, "FLAG{firstblood}")
	time.Sleep(100 * time.Millisecond)

	submitFlag(e, tokenUser2, challengeId, "FLAG{firstblood}")

	resp := e.GET("/api/v1/challenges/{id}/first-blood", challengeId).
		Expect().
		Status(200).
		JSON().
		Object()

	resp.Value("username").String().IsEqual("fbuser1")
	resp.Value("team_name").String().IsEqual("fbuser1")
	resp.ContainsKey("user_id")
	resp.ContainsKey("team_id")
	resp.ContainsKey("solved_at")
}

func TestFirstBlood_NotFound(t *testing.T) {
	e := setupE2E(t)

	_, _, tokenAdmin := registerAdmin(e, "adminfb2")

	challengeId := createChallenge(e, tokenAdmin, map[string]interface{}{
		"title":       "No Solves Test",
		"description": "Test challenge",
		"points":      100,
		"flag":        "FLAG{nosolves}",
		"category":    "web",
		"difficulty":  "easy",
		"is_hidden":   false,
	})

	e.GET("/api/v1/challenges/{id}/first-blood", challengeId).
		Expect().
		Status(404).
		JSON().
		Object().
		Value("error").String().IsEqual("no solves yet")
}
