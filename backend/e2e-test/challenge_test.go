package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestChallenge_Lifecycle(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition
	_, tokenAdmin := h.SetupCompetition("admin_lifecycle")

	// 2. Create Challenge
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Test Challenge",
		"description": "Test Description",
		"points":      100,
		"flag":        "FLAG{test}",
		"category":    "web",
		"difficulty":  "easy",
		"is_hidden":   false,
	})

	// 3. Register User
	suffix := uuid.New().String()[:8]
	userName := "chall_usr_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(userName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	// 4. Verify Challenge Initial State
	challenge := h.FindChallengeInList(tokenUser, challengeID)
	challenge.Value("title").String().IsEqual("Test Challenge")
	challenge.Value("solved").Boolean().IsFalse()
	challenge.Value("solve_count").Number().IsEqual(0)

	// 5. Submit Correct Flag
	h.SubmitFlag(tokenUser, challengeID, "FLAG{test}", http.StatusOK)

	// 6. Verify Challenge Solved State
	challengeAfterSolve := h.FindChallengeInList(tokenUser, challengeID)
	challengeAfterSolve.Value("solved").Boolean().IsTrue()
	challengeAfterSolve.Value("solve_count").Number().IsEqual(1)

	// 7. Submit Duplicate Flag (Expect Conflict)
	h.SubmitFlag(tokenUser, challengeID, "FLAG{test}", http.StatusConflict)
}

func TestChallenge_DynamicScoring(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition
	_, tokenAdmin := h.SetupCompetition("adm_dyn")

	// 2. Create Dynamic Challenge
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Dynamic Challenge",
		"description":   "Test dynamic scoring",
		"points":        500,
		"initial_value": 500,
		"min_value":     100,
		"decay":         1,
		"flag":          "FLAG{dynamic}",
		"category":      "web",
		"difficulty":    "easy",
		"is_hidden":     false,
	})

	// 3. First Solver
	suffix := uuid.New().String()[:8]
	_, _, tokenUser1 := h.RegisterUserAndLogin("solver1_" + suffix)
	h.CreateSoloTeam(tokenUser1, http.StatusCreated)
	h.SubmitFlag(tokenUser1, challengeID, "FLAG{dynamic}", http.StatusOK)

	// 4. Check Score Decay (1 solve)
	// CTFd Logic: First blood gets Max Points (500)
	challengeState1 := h.FindChallengeInList(tokenUser1, challengeID)
	challengeState1.Value("points").Number().IsEqual(500)
	challengeState1.Value("solve_count").Number().IsEqual(1)

	// 5. Second Solver
	_, _, tokenUser2 := h.RegisterUserAndLogin("solver2_" + suffix)
	h.CreateSoloTeam(tokenUser2, http.StatusCreated)
	h.SubmitFlag(tokenUser2, challengeID, "FLAG{dynamic}", http.StatusOK)

	// 6. Check Score Decay (2 solves)
	// count = 2-1 = 1. Decay = 1.
	// 1 >= Min Value (100).
	challengeState2 := h.FindChallengeInList(tokenUser2, challengeID)
	challengeState2.Value("points").Number().IsEqual(100)
	challengeState2.Value("solve_count").Number().IsEqual(2)
}

func TestChallenge_CreateHIDden(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition
	_, tokenAdmin := h.SetupCompetition("admin_hidden")

	// 2. Create HIDden Challenge
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "HIDden Challenge",
		"description": "Test hidden challenge",
		"points":      200,
		"flag":        "FLAG{hidden}",
		"category":    "crypto",
		"is_hidden":   true,
	})

	// 3. Register User
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("user2_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	// 4. Verify Challenge is Not Visible
	h.AssertChallengeMissing(tokenUser, challengeID)
}

func TestChallenge_Update(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition
	_, tokenAdmin := h.SetupCompetition("admin_update")

	// 2. Create Challenge
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Original Title",
		"description": "Original description",
		"points":      100,
		"flag":        "FLAG{original}",
		"category":    "web",
		"is_hidden":   false,
	})

	// 3. Update Challenge
	h.UpdateChallenge(tokenAdmin, challengeID, map[string]any{
		"title":       "Updated Title",
		"description": "Updated Description",
		"points":      150,
		"flag":        "FLAG{updated}",
		"category":    "pwn",
		"difficulty":  "hard",
		"is_hidden":   false,
	})

	// 4. Verify Updates
	challenge := h.FindChallengeInList(tokenAdmin, challengeID)
	challenge.Value("title").String().IsEqual("Updated Title")
	challenge.Value("description").String().IsEqual("Updated Description")
	challenge.Value("points").Number().IsEqual(150)
}

func TestChallenge_SubmitInvalidFlag(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition
	_, tokenAdmin := h.SetupCompetition("admin_invalid")

	// 2. Create Challenge
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Test Challenge",
		"description": "Test invalid flag",
		"flag":        "FLAG{correct}",
		"points":      100,
		"category":    "misc",
	})

	// 3. Register User
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("user3_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	// 4. Submit Wrong Flag (Expect 400)
	h.SubmitFlag(tokenUser, challengeID, "FLAG{wrong}", http.StatusBadRequest)
}

func TestChallenge_Delete(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition
	_, tokenAdmin := h.SetupCompetition("admin_delete")

	// 2. Create Challenge
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "To Delete",
		"description": "Test delete challenge",
		"flag":        "FLAG{delete}",
		"points":      50,
		"category":    "misc",
	})

	// 3. Delete Challenge
	h.DeleteChallenge(tokenAdmin, challengeID)

	// 4. Verify Challenge Gone
	h.AssertChallengeMissing(tokenAdmin, challengeID)
}

func TestChallenge_AccessWithoutAuth(t *testing.T) {
	e := setupE2E(t)

	// 1. Attempt access without token
	e.GET("/api/v1/challenges").
		Expect().
		Status(http.StatusUnauthorized)
}
