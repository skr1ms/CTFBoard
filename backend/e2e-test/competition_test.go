package e2e_test

import (
	"net/http"
	"testing"
	"time"
)

func TestCompetition_Status(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Verify Public Competition Status Endpoint
	h.GetCompetitionStatus().
		ContainsKey("status").
		ContainsKey("start_time").
		ContainsKey("end_time")
}

func TestCompetition_UpdateAndEnforce(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Register Admin and Create Challenge
	_, _, tokenAdmin := h.RegisterAdmin("admin_comp")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Comp Challenge",
		"description": "Test competition challenge",
		"flag":        "FLAG{comp}",
		"points":      100,
		"category":    "web",
		"is_hidden":   false,
	})

	// 2. Register Regular User
	_, _, tokenUser := h.RegisterUserAndLogin("comp_user")

	// 3. Admin Pauses Competition
	now := time.Now().UTC()
	h.UpdateCompetition(tokenAdmin, map[string]any{
		"name":       "Comp Name",
		"start_time": now.Add(-1 * time.Hour).Format(time.RFC3339),
		"end_time":   now.Add(24 * time.Hour).Format(time.RFC3339),
		"is_paused":  true,
	})

	h.GetCompetitionStatus().
		Value("status").String().IsEqual("paused")

	// 4. User Attempts to Submit Flag while Paused (Expect Forbidden)
	h.SubmitFlag(tokenUser, challengeID, "FLAG{comp}", http.StatusForbidden)

	// 5. Admin Resumes Competition
	h.UpdateCompetition(tokenAdmin, map[string]any{
		"name":       "Comp Name",
		"start_time": now.Add(-1 * time.Hour).Format(time.RFC3339),
		"end_time":   now.Add(24 * time.Hour).Format(time.RFC3339),
		"is_paused":  false,
	})

	// 6. User Successfully Submits Flag
	h.SubmitFlag(tokenUser, challengeID, "FLAG{comp}", http.StatusOK)
}
