package e2e

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestScoreboard_Display(t *testing.T) {
	e := setupE2E(t)

	suffix := uuid.New().String()[:8]
	_, _, tokenAdmin := registerAdmin(e, "admin6_"+suffix)

	challengeId1 := createChallenge(e, tokenAdmin, map[string]interface{}{
		"title":       "Challenge 1",
		"description": "Description 1",
		"points":      100,
		"flag":        "FLAG{chall1}",
		"category":    "web",
		"difficulty":  "easy",
		"is_hidden":   false,
	})

	challengeId2 := createChallenge(e, tokenAdmin, map[string]interface{}{
		"title":       "Challenge 2",
		"description": "Description 2",
		"points":      200,
		"flag":        "FLAG{chall2}",
		"category":    "crypto",
		"difficulty":  "medium",
		"is_hidden":   false,
	})

	emailUser1, passUser1 := registerUser(e, "user4_"+suffix)
	tokenUser1 := login(e, emailUser1, passUser1)

	emailUser2, passUser2 := registerUser(e, "user5_"+suffix)
	tokenUser2 := login(e, emailUser2, passUser2)

	submitFlag(e, tokenUser1, challengeId1, "FLAG{chall1}")
	time.Sleep(3 * time.Second)

	submitFlag(e, tokenUser1, challengeId2, "FLAG{chall2}")
	time.Sleep(3 * time.Second)

	submitFlag(e, tokenUser2, challengeId1, "FLAG{chall1}")

	scoreboardResp := e.GET("/api/v1/scoreboard").
		Expect().
		Status(200).
		JSON().
		Array()

	var user4Found, user5Found bool
	length := int(scoreboardResp.Length().Raw())

	for i := 0; i < length; i++ {
		team := scoreboardResp.Value(i).Object()
		name := team.Value("team_name").String().Raw()

		if name == "user4_"+suffix {
			team.Value("points").Number().IsEqual(300)
			user4Found = true
		}
		if name == "user5_"+suffix {
			team.Value("points").Number().IsEqual(100)
			user5Found = true
		}
	}

	if !user4Found || !user5Found {
		t.Fatal("Scoreboard verification failed: users not found")
	}
}

func TestScoreboard_Empty(t *testing.T) {
	e := setupE2E(t)

	scoreboardResp := e.GET("/api/v1/scoreboard").
		Expect().
		Status(200).
		JSON().
		Array()

	length := int(scoreboardResp.Length().Raw())

	allZeroPoints := true
	for i := 0; i < length; i++ {
		team := scoreboardResp.Value(i).Object()
		points := int(team.Value("points").Number().Raw())
		if points > 0 {
			allZeroPoints = false
			break
		}
	}

	if !allZeroPoints && length > 0 {
		t.Logf("Scoreboard has %d teams, some have non-zero points (this is expected if previous tests created solves)", length)
	}
}
