package e2e

import (
	"net/http"
	"testing"
	"time"
)

func TestCompetition_Status(t *testing.T) {
	e := setupE2E(t)

	e.GET("/api/v1/competition/status").
		Expect().
		Status(http.StatusOK).
		JSON().
		Object().
		ContainsKey("status").
		ContainsKey("start_time").
		ContainsKey("end_time")
}

func TestCompetition_UpdateAndEnforce(t *testing.T) {
	e := setupE2E(t)

	_, _, tokenAdmin := registerAdmin(e, "admin_comp")

	challengeId := createChallenge(e, tokenAdmin, map[string]interface{}{
		"title":       "Comp Challenge",
		"description": "Desc",
		"points":      100,
		"flag":        "FLAG{comp}",
		"category":    "web",
		"difficulty":  "easy",
		"is_hidden":   false,
	})

	emailUser, passUser := registerUser(e, "comp_user")
	tokenUser := login(e, emailUser, passUser)

	e.PUT("/api/v1/admin/competition").
		WithHeader("Authorization", tokenAdmin).
		WithJSON(map[string]interface{}{
			"name":       "Comp Name",
			"start_time": time.Now().Add(-1 * time.Hour),
			"is_paused":  true,
		}).
		Expect().
		Status(http.StatusOK)

	e.GET("/api/v1/competition/status").
		Expect().
		Status(http.StatusOK).
		JSON().
		Object().
		Value("status").String().IsEqual("paused")

	e.POST("/api/v1/challenges/{id}/submit", challengeId).
		WithHeader("Authorization", tokenUser).
		WithJSON(map[string]string{
			"flag": "FLAG{comp}",
		}).
		Expect().
		Status(http.StatusForbidden)

	e.PUT("/api/v1/admin/competition").
		WithHeader("Authorization", tokenAdmin).
		WithJSON(map[string]interface{}{
			"name":       "Comp Name",
			"start_time": time.Now().Add(-1 * time.Hour),
			"is_paused":  false,
		}).
		Expect().
		Status(http.StatusOK)

	e.POST("/api/v1/challenges/{id}/submit", challengeId).
		WithHeader("Authorization", tokenUser).
		WithJSON(map[string]string{
			"flag": "FLAG{comp}",
		}).
		Expect().
		Status(http.StatusOK)
}
