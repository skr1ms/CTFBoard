package e2e_test

import (
	"net/http"
	"testing"
	"time"
)

func TestCompetition_Status(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	h.GetCompetitionStatus().
		ContainsKey("status").
		ContainsKey("start_time").
		ContainsKey("end_time")
}

func TestCompetition_UpdateAndEnforce(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, _, tokenAdmin := h.RegisterAdmin("admin_comp")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]interface{}{
		"title":       "Comp Challenge",
		"description": "Test competition challenge",
		"flag":        "FLAG{comp}",
		"points":      100,
		"category":    "web",
		"is_hidden":   false,
	})

	_, _, tokenUser := h.RegisterUserAndLogin("comp_user")

	now := time.Now().UTC()
	h.UpdateCompetition(tokenAdmin, map[string]interface{}{
		"name":       "Comp Name",
		"start_time": now.Add(-1 * time.Hour).Format(time.RFC3339),
		"end_time":   now.Add(24 * time.Hour).Format(time.RFC3339),
		"is_paused":  true,
	})

	h.GetCompetitionStatus().
		Value("status").String().IsEqual("paused")

	h.SubmitFlag(tokenUser, challengeID, "FLAG{comp}", http.StatusForbidden)

	h.UpdateCompetition(tokenAdmin, map[string]interface{}{
		"name":       "Comp Name",
		"start_time": now.Add(-1 * time.Hour).Format(time.RFC3339),
		"end_time":   now.Add(24 * time.Hour).Format(time.RFC3339),
		"is_paused":  false,
	})

	h.SubmitFlag(tokenUser, challengeID, "FLAG{comp}", http.StatusOK)
}
