package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestChallenge_Lifecycle(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]

	_, _, tokenAdmin := h.RegisterAdmin("admin_" + suffix)

	h.StartCompetition(tokenAdmin)

	challengeID := h.CreateChallenge(tokenAdmin, map[string]interface{}{
		"title":       "Test Challenge",
		"description": "Test Description",
		"points":      100,
		"flag":        "FLAG{test}",
		"category":    "web",
		"difficulty":  "easy",
		"is_hidden":   false,
	})

	emailUser, passUser := h.RegisterUser("chall_usr_" + suffix)
	loginResp := h.Login(emailUser, passUser, http.StatusOK)
	tokenUser := "Bearer " + loginResp.Value("access_token").String().Raw()

	challenge := h.FindChallengeInList(tokenUser, challengeID)

	challenge.Value("title").String().IsEqual("Test Challenge")
	challenge.Value("solved").Boolean().IsFalse()
	challenge.Value("solve_count").Number().IsEqual(0)

	h.SubmitFlag(tokenUser, challengeID, "FLAG{test}", http.StatusOK)

	challengeAfterSolve := h.FindChallengeInList(tokenUser, challengeID)
	challengeAfterSolve.Value("solved").Boolean().IsTrue()
	challengeAfterSolve.Value("solve_count").Number().IsEqual(1)

	h.SubmitFlag(tokenUser, challengeID, "FLAG{test}", http.StatusConflict)
}

func TestChallenge_DynamicScoring(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]
	_, _, tokenAdmin := h.RegisterAdmin("adm_dyn_" + suffix)

	h.StartCompetition(tokenAdmin)

	challengeID := h.CreateChallenge(tokenAdmin, map[string]interface{}{
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

	_, _, tokenUser1 := h.RegisterUserAndLogin("solver1_" + suffix)
	h.SubmitFlag(tokenUser1, challengeID, "FLAG{dynamic}", http.StatusOK)

	challengeState1 := h.FindChallengeInList(tokenUser1, challengeID)
	challengeState1.Value("points").Number().IsEqual(300)
	challengeState1.Value("solve_count").Number().IsEqual(1)

	_, _, tokenUser2 := h.RegisterUserAndLogin("solver2_" + suffix)
	h.SubmitFlag(tokenUser2, challengeID, "FLAG{dynamic}", http.StatusOK)

	challengeState2 := h.FindChallengeInList(tokenUser2, challengeID)
	challengeState2.Value("points").Number().IsEqual(200)
	challengeState2.Value("solve_count").Number().IsEqual(2)
}

func TestChallenge_CreateHidden(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]
	_, _, tokenAdmin := h.RegisterAdmin("admin2_" + suffix)

	h.StartCompetition(tokenAdmin)

	challengeID := h.CreateChallenge(tokenAdmin, map[string]interface{}{
		"title":       "Hidden Challenge",
		"description": "Test hidden challenge",
		"points":      200,
		"flag":        "FLAG{hidden}",
		"category":    "crypto",
		"is_hidden":   true,
	})

	_, _, tokenUser := h.RegisterUserAndLogin("user2_" + suffix)

	h.AssertChallengeMissing(tokenUser, challengeID)
}

func TestChallenge_Update(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]
	_, _, tokenAdmin := h.RegisterAdmin("admin3_" + suffix)

	h.StartCompetition(tokenAdmin)

	challengeID := h.CreateChallenge(tokenAdmin, map[string]interface{}{
		"title":       "Original Title",
		"description": "Original description",
		"points":      100,
		"flag":        "FLAG{original}",
		"category":    "web",
		"is_hidden":   false,
	})

	e.PUT("/api/v1/admin/challenges/{id}", challengeID).
		WithHeader("Authorization", tokenAdmin).
		WithJSON(map[string]interface{}{
			"title":       "Updated Title",
			"description": "Updated Description",
			"points":      150,
			"flag":        "FLAG{updated}",
			"category":    "pwn",
			"difficulty":  "hard",
			"is_hidden":   false,
		}).
		Expect().
		Status(http.StatusOK)

	challenge := h.FindChallengeInList(tokenAdmin, challengeID)
	challenge.Value("title").String().IsEqual("Updated Title")
	challenge.Value("description").String().IsEqual("Updated Description")
	challenge.Value("points").Number().IsEqual(150)
}

func TestChallenge_SubmitInvalidFlag(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]
	_, _, tokenAdmin := h.RegisterAdmin("admin4_" + suffix)

	h.StartCompetition(tokenAdmin)

	challengeID := h.CreateChallenge(tokenAdmin, map[string]interface{}{
		"title":       "Test Challenge",
		"description": "Test invalid flag",
		"flag":        "FLAG{correct}",
		"points":      100,
		"category":    "misc",
	})

	_, _, tokenUser := h.RegisterUserAndLogin("user3_" + suffix)

	h.SubmitFlag(tokenUser, challengeID, "FLAG{wrong}", http.StatusBadRequest)
}

func TestChallenge_Delete(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]
	_, _, tokenAdmin := h.RegisterAdmin("admin5_" + suffix)

	h.StartCompetition(tokenAdmin)

	challengeID := h.CreateChallenge(tokenAdmin, map[string]interface{}{
		"title":       "To Delete",
		"description": "Test delete challenge",
		"flag":        "FLAG{delete}",
		"points":      50,
		"category":    "misc",
	})

	e.DELETE("/api/v1/admin/challenges/{id}", challengeID).
		WithHeader("Authorization", tokenAdmin).
		Expect().
		Status(http.StatusOK)

	h.AssertChallengeMissing(tokenAdmin, challengeID)
}

func TestChallenge_AccessWithoutAuth(t *testing.T) {
	e := setupE2E(t)

	e.GET("/api/v1/challenges").
		Expect().
		Status(http.StatusUnauthorized)
}
