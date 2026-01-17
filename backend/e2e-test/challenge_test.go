package e2e

import (
	"testing"

	"github.com/google/uuid"
)

func TestChallenge_Lifecycle(t *testing.T) {
	e := setupE2E(t)

	suffix := uuid.New().String()[:8]
	_, _, tokenAdmin := registerAdmin(e, "admin_"+suffix)

	challengeId := createChallenge(e, tokenAdmin, map[string]interface{}{
		"title":       "Test Challenge",
		"description": "Test Description",
		"points":      100,
		"flag":        "FLAG{test}",
		"category":    "web",
		"difficulty":  "easy",
		"is_hidden":   false,
	})

	emailUser, passUser := registerUser(e, "chall_usr_"+suffix)
	tokenUser := login(e, emailUser, passUser)

	challengesResp := e.GET("/api/v1/challenges").
		WithHeader("Authorization", tokenUser).
		Expect().
		Status(200).
		JSON()

	if challengesResp.Raw() == nil {
		t.Fatal("challenges response is null")
	}

	challengesArray := challengesResp.Array()
	challengesArray.Length().Gt(0)

	found := false
	length := int(challengesArray.Length().Raw())
	for i := 0; i < length; i++ {
		challenge := challengesArray.Value(i).Object()
		if challenge.Value("id").String().Raw() == challengeId {
			found = true
			challenge.Value("title").String().IsEqual("Test Challenge")
			challenge.Value("solved").Boolean().IsFalse()
			challenge.Value("solve_count").Number().IsEqual(0)
			break
		}
	}

	if !found {
		t.Fatal("Challenge not found in list")
	}

	e.POST("/api/v1/challenges/{id}/submit", challengeId).
		WithHeader("Authorization", tokenUser).
		WithJSON(map[string]string{
			"flag": "FLAG{test}",
		}).
		Expect().
		Status(200)

	challengesResp2 := e.GET("/api/v1/challenges").
		WithHeader("Authorization", tokenUser).
		Expect().
		Status(200).
		JSON().
		Array()

	length2 := int(challengesResp2.Length().Raw())
	for i := 0; i < length2; i++ {
		challenge := challengesResp2.Value(i).Object()
		if challenge.Value("id").String().Raw() == challengeId {
			challenge.Value("solved").Boolean().IsTrue()
			challenge.Value("solve_count").Number().IsEqual(1)
			break
		}
	}

	e.POST("/api/v1/challenges/{id}/submit", challengeId).
		WithHeader("Authorization", tokenUser).
		WithJSON(map[string]string{
			"flag": "FLAG{test}",
		}).
		Expect().
		Status(409)
}

func TestChallenge_DynamicScoring(t *testing.T) {
	e := setupE2E(t)

	suffix := uuid.New().String()[:8]
	_, _, tokenAdmin := registerAdmin(e, "adm_dyn_"+suffix)

	challengeId := createChallenge(e, tokenAdmin, map[string]interface{}{
		"title":         "Dynamic Challenge",
		"description":   "Dynamic Description",
		"points":        500,
		"initial_value": 500,
		"min_value":     100,
		"decay":         1,
		"flag":          "FLAG{dynamic}",
		"category":      "web",
		"difficulty":    "easy",
		"is_hidden":     false,
	})

	emailUser1, passUser1 := registerUser(e, "solver1_"+suffix)
	tokenUser1 := login(e, emailUser1, passUser1)

	e.POST("/api/v1/challenges/{id}/submit", challengeId).
		WithHeader("Authorization", tokenUser1).
		WithJSON(map[string]string{
			"flag": "FLAG{dynamic}",
		}).
		Expect().
		Status(200)

	// Check challenge details - points should have decreased
	// Formula: (500 - 100) / 2^(1/1) + 100 = 200 + 100 = 300
	challengesResp := e.GET("/api/v1/challenges").
		WithHeader("Authorization", tokenUser1).
		Expect().
		Status(200).
		JSON().
		Array()

	var found bool
	length := int(challengesResp.Length().Raw())
	for i := 0; i < length; i++ {
		challenge := challengesResp.Value(i).Object()
		if challenge.Value("id").String().Raw() == challengeId {
			found = true
			challenge.Value("points").Number().IsEqual(300)
			challenge.Value("solve_count").Number().IsEqual(1)
			break
		}
	}

	if !found {
		t.Fatal("Dynamic challenge not found")
	}

	emailUser2, passUser2 := registerUser(e, "solver2_"+suffix)
	tokenUser2 := login(e, emailUser2, passUser2)

	e.POST("/api/v1/challenges/{id}/submit", challengeId).
		WithHeader("Authorization", tokenUser2).
		WithJSON(map[string]string{
			"flag": "FLAG{dynamic}",
		}).
		Expect().
		Status(200)

	// Check challenge details - points should decrease further
	// Formula: (500 - 100) / 2^(2/1) + 100 = 400 / 4 + 100 = 100 + 100 = 200
	challengesResp2 := e.GET("/api/v1/challenges").
		WithHeader("Authorization", tokenUser2).
		Expect().
		Status(200).
		JSON().
		Array()

	found = false
	length2 := int(challengesResp2.Length().Raw())
	for i := 0; i < length2; i++ {
		challenge := challengesResp2.Value(i).Object()
		if challenge.Value("id").String().Raw() == challengeId {
			found = true
			challenge.Value("points").Number().IsEqual(200)
			challenge.Value("solve_count").Number().IsEqual(2)
			break
		}
	}

	if !found {
		t.Fatal("Dynamic challenge not found in second check")
	}
}

func TestChallenge_CreateHidden(t *testing.T) {
	e := setupE2E(t)

	suffix := uuid.New().String()[:8]
	_, _, tokenAdmin := registerAdmin(e, "admin2_"+suffix)

	challengeId := createChallenge(e, tokenAdmin, map[string]interface{}{
		"title":       "Hidden Challenge",
		"description": "Hidden Description",
		"points":      200,
		"flag":        "FLAG{hidden}",
		"category":    "crypto",
		"difficulty":  "medium",
		"is_hidden":   true,
	})

	emailUser, passUser := registerUser(e, "user2_"+suffix)
	tokenUser := login(e, emailUser, passUser)

	challengesResp := e.GET("/api/v1/challenges").
		WithHeader("Authorization", tokenUser).
		Expect().
		Status(200).
		JSON()

	if challengesResp.Raw() == nil {
		t.Fatal("challenges response is null")
	}

	challengesArray := challengesResp.Array()
	found := false
	length := int(challengesArray.Length().Raw())
	for i := 0; i < length; i++ {
		challenge := challengesArray.Value(i).Object()
		if challenge.Value("id").String().Raw() == challengeId {
			found = true
			break
		}
	}

	if found {
		t.Fatal("Hidden challenge should not be visible to users")
	}
}

func TestChallenge_Update(t *testing.T) {
	e := setupE2E(t)

	suffix := uuid.New().String()[:8]
	_, _, tokenAdmin := registerAdmin(e, "admin3_"+suffix)

	challengeId := createChallenge(e, tokenAdmin, map[string]interface{}{
		"title":       "Original Title",
		"description": "Original Description",
		"points":      100,
		"flag":        "FLAG{original}",
		"category":    "web",
		"difficulty":  "easy",
		"is_hidden":   false,
	})

	e.PUT("/api/v1/admin/challenges/{id}", challengeId).
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
		Status(200)

	challengesResp := e.GET("/api/v1/challenges").
		WithHeader("Authorization", tokenAdmin).
		Expect().
		Status(200).
		JSON().
		Array()

	var found bool
	length := int(challengesResp.Length().Raw())
	for i := 0; i < length; i++ {
		challenge := challengesResp.Value(i).Object()
		if challenge.Value("id").String().Raw() == challengeId {
			challenge.Value("title").String().IsEqual("Updated Title")
			challenge.Value("description").String().IsEqual("Updated Description")
			challenge.Value("points").Number().IsEqual(150)
			found = true
			break
		}
	}

	if !found {
		t.Fatal("Updated challenge not found in list")
	}
}

func TestChallenge_SubmitInvalidFlag(t *testing.T) {
	e := setupE2E(t)

	suffix := uuid.New().String()[:8]
	_, _, tokenAdmin := registerAdmin(e, "admin4_"+suffix)

	challengeId := createChallenge(e, tokenAdmin, map[string]interface{}{
		"title":       "Test Challenge",
		"description": "Test Description",
		"points":      100,
		"flag":        "FLAG{correct}",
		"category":    "web",
		"difficulty":  "easy",
		"is_hidden":   false,
	})

	emailUser, passUser := registerUser(e, "user3_"+suffix)
	tokenUser := login(e, emailUser, passUser)

	e.POST("/api/v1/challenges/{id}/submit", challengeId).
		WithHeader("Authorization", tokenUser).
		WithJSON(map[string]string{
			"flag": "FLAG{wrong}",
		}).
		Expect().
		Status(400)
}

func TestChallenge_Delete(t *testing.T) {
	e := setupE2E(t)

	suffix := uuid.New().String()[:8]
	_, _, tokenAdmin := registerAdmin(e, "admin5_"+suffix)

	challengeId := createChallenge(e, tokenAdmin, map[string]interface{}{
		"title":       "To Delete",
		"description": "Will be deleted",
		"points":      50,
		"flag":        "FLAG{delete}",
		"category":    "web",
		"difficulty":  "easy",
		"is_hidden":   false,
	})

	e.DELETE("/api/v1/admin/challenges/{id}", challengeId).
		WithHeader("Authorization", tokenAdmin).
		Expect().
		Status(200)

	challengesResp := e.GET("/api/v1/challenges").
		WithHeader("Authorization", tokenAdmin).
		Expect().
		Status(200).
		JSON().
		Array()

	var found bool
	length := int(challengesResp.Length().Raw())
	for i := 0; i < length; i++ {
		challenge := challengesResp.Value(i).Object()
		if challenge.Value("id").String().Raw() == challengeId {
			found = true
			break
		}
	}

	if found {
		t.Fatal("Deleted challenge should not be in list")
	}
}

func TestChallenge_AccessWithoutAuth(t *testing.T) {
	e := setupE2E(t)

	e.GET("/api/v1/challenges").
		Expect().
		Status(401)
}
