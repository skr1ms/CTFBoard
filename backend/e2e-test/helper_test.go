package e2e_test

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type E2EHelper struct {
	t  *testing.T
	e  *httpexpect.Expect
	db *sql.DB
}

func NewE2EHelper(t *testing.T, e *httpexpect.Expect, db *sql.DB) *E2EHelper {
	return &E2EHelper{
		t:  t,
		e:  e,
		db: db,
	}
}

// API Wrappers

func (h *E2EHelper) Register(username, email, password string) {
	h.e.POST("/api/v1/auth/register").
		WithJSON(map[string]string{
			"username": username,
			"email":    email,
			"password": password,
		}).
		Expect().
		Status(http.StatusCreated)
}

func (h *E2EHelper) Login(email, password string, expectStatus int) *httpexpect.Object {
	return h.e.POST("/api/v1/auth/login").
		WithJSON(map[string]string{
			"email":    email,
			"password": password,
		}).
		Expect().
		Status(expectStatus).
		JSON().Object()
}

func (h *E2EHelper) ForgotPassword(email string, expectStatus int) {
	h.e.POST("/api/v1/auth/forgot-password").
		WithJSON(map[string]string{
			"email": email,
		}).
		Expect().
		Status(expectStatus)
}

func (h *E2EHelper) ResetPassword(token, newPassword string) {
	h.e.POST("/api/v1/auth/reset-password").
		WithJSON(map[string]string{
			"token":        token,
			"new_password": newPassword,
		}).
		Expect().
		Status(http.StatusOK)
}

func (h *E2EHelper) VerifyEmail(token string) {
	h.e.GET("/api/v1/auth/verify-email").
		WithQuery("token", token).
		Expect().
		Status(http.StatusOK).
		JSON().Object().Value("message").IsEqual("email verified successfully")
}

// DB Helpers

func (h *E2EHelper) GetUserIDByEmail(email string) string {
	var id string
	err := h.db.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&id)
	require.NoError(h.t, err, "failed to find user by email")
	return id
}

func (h *E2EHelper) AssertUserVerified(email string, expected bool) {
	var isVerified bool
	err := h.db.QueryRow("SELECT is_verified FROM users WHERE email = ?", email).Scan(&isVerified)
	require.NoError(h.t, err)
	assert.Equal(h.t, expected, isVerified, "user verification status mismatch")
}

func (h *E2EHelper) InjectToken(userID string, tokenType entity.TokenType, knownToken string) {
	hashedToken := h.hashToken(knownToken)

	var count int
	err := h.db.QueryRow("SELECT count(*) FROM verification_tokens WHERE user_id = ? AND type = ?", userID, tokenType).Scan(&count)
	require.NoError(h.t, err)
	require.Equal(h.t, 1, count, "token row should exist before injection")

	_, err = h.db.Exec("UPDATE verification_tokens SET token = ? WHERE user_id = ? AND type = ?", hashedToken, userID, tokenType)
	require.NoError(h.t, err, "failed to inject token")
}

func (h *E2EHelper) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Challenge Helpers

func (h *E2EHelper) RegisterUserAndLogin(username string) (email, password, token string) {
	email = username + "@example.com"
	password = "password123"
	h.Register(username, email, password)

	resp := h.Login(email, password, http.StatusOK)
	token = "Bearer " + resp.Value("access_token").String().Raw()
	return
}

func (h *E2EHelper) RegisterUser(username string) (email, password string) {
	email = username + "@example.com"
	password = "password123"
	h.Register(username, email, password)
	return
}

func (h *E2EHelper) RegisterAdmin(username string) (email, password, token string) {
	email, password, token = h.RegisterUserAndLogin(username)

	meResp := h.e.GET("/api/v1/auth/me").
		WithHeader("Authorization", token).
		Expect().Status(200).JSON().Object()
	userId := meResp.Value("id").String().Raw()

	_, err := h.db.Exec("UPDATE users SET role = 'admin' WHERE id = ?", userId)
	require.NoError(h.t, err)

	resp := h.Login(email, password, http.StatusOK)
	token = "Bearer " + resp.Value("access_token").String().Raw()
	return
}

func (h *E2EHelper) CreateChallenge(token string, data map[string]interface{}) string {
	resp := h.e.POST("/api/v1/admin/challenges").
		WithHeader("Authorization", token).
		WithJSON(data).
		Expect().
		Status(http.StatusCreated).
		JSON().Object()

	return resp.Value("id").String().Raw()
}

func (h *E2EHelper) SubmitFlag(token, challengeId, flag string, expectStatus int) {
	h.e.POST("/api/v1/challenges/{id}/submit", challengeId).
		WithHeader("Authorization", token).
		WithJSON(map[string]string{"flag": flag}).
		Expect().
		Status(expectStatus)
}

func (h *E2EHelper) FindChallengeInList(token, challengeId string) *httpexpect.Object {
	challenges := h.e.GET("/api/v1/challenges").
		WithHeader("Authorization", token).
		Expect().
		Status(http.StatusOK).
		JSON().Array()

	for i := 0; i < int(challenges.Length().Raw()); i++ {
		c := challenges.Value(i).Object()
		if c.Value("id").String().Raw() == challengeId {
			return c
		}
	}
	h.t.Fatalf("Challenge %s not found in list", challengeId)
	return nil
}

func (h *E2EHelper) AssertChallengeMissing(token, challengeId string) {
	challenges := h.e.GET("/api/v1/challenges").
		WithHeader("Authorization", token).
		Expect().
		Status(http.StatusOK).
		JSON().Array()

	for i := 0; i < int(challenges.Length().Raw()); i++ {
		if challenges.Value(i).Object().Value("id").String().Raw() == challengeId {
			h.t.Fatalf("Challenge %s should NOT be in list", challengeId)
		}
	}
}

// Competition Helpers

func (h *E2EHelper) GetCompetitionStatus() *httpexpect.Object {
	return h.e.GET("/api/v1/competition/status").
		Expect().
		Status(http.StatusOK).
		JSON().Object()
}

func (h *E2EHelper) UpdateCompetition(token string, data map[string]interface{}) {
	h.e.PUT("/api/v1/admin/competition").
		WithHeader("Authorization", token).
		WithJSON(data).
		Expect().
		Status(http.StatusOK)
}

// First Blood Helpers

func (h *E2EHelper) GetFirstBlood(challengeId string, expectStatus int) *httpexpect.Object {
	return h.e.GET("/api/v1/challenges/{id}/first-blood", challengeId).
		Expect().
		Status(expectStatus).
		JSON().
		Object()
}

func (h *E2EHelper) AssertFirstBlood(challengeId, expectedUsername string) {
	resp := h.GetFirstBlood(challengeId, http.StatusOK)

	resp.Value("username").String().IsEqual(expectedUsername)
	resp.Value("team_name").String().IsEqual(expectedUsername)

	resp.ContainsKey("user_id")
	resp.ContainsKey("team_id")
	resp.ContainsKey("solved_at")
}

// Hint Helpers

func (h *E2EHelper) CreateHint(token, challengeID, content string, cost int) string {
	resp := h.e.POST("/api/v1/admin/challenges/{id}/hints", challengeID).
		WithHeader("Authorization", token).
		WithJSON(map[string]interface{}{
			"content":     content,
			"cost":        cost,
			"order_index": 1,
		}).
		Expect().
		Status(http.StatusCreated).
		JSON().Object()

	return resp.Value("id").String().Raw()
}

func (h *E2EHelper) UnlockHint(token, challengeID, hintID string, expectStatus int) *httpexpect.Object {
	req := h.e.POST("/api/v1/challenges/{cid}/hints/{hid}/unlock", challengeID, hintID).
		WithHeader("Authorization", token)

	if expectStatus != http.StatusOK {
		req.Expect().Status(expectStatus)
		return nil
	}

	return req.Expect().Status(http.StatusOK).JSON().Object()
}

func (h *E2EHelper) GetHintFromList(token, challengeID, hintID string) *httpexpect.Object {
	hintsArr := h.e.GET("/api/v1/challenges/{id}/hints", challengeID).
		WithHeader("Authorization", token).
		Expect().
		Status(http.StatusOK).
		JSON().Array()

	for i := 0; i < int(hintsArr.Length().Raw()); i++ {
		hint := hintsArr.Value(i).Object()
		if hint.Value("id").String().Raw() == hintID {
			return hint
		}
	}
	h.t.Fatalf("Hint %s not found in challenge %s list", hintID, challengeID)
	return nil
}

func (h *E2EHelper) GetScoreboard() *httpexpect.Response {
	return h.e.GET("/api/v1/scoreboard").Expect()
}

func (h *E2EHelper) AssertTeamScore(teamName string, expectedPoints int) {
	scoreboard := h.GetScoreboard().
		Status(http.StatusOK).
		JSON().Array()

	found := false
	for _, val := range scoreboard.Iter() {
		obj := val.Object()
		if obj.Value("team_name").String().Raw() == teamName {
			obj.Value("points").Number().IsEqual(expectedPoints)
			found = true
			break
		}
	}

	if !found {
		h.t.Fatalf("Team %s not found in scoreboard", teamName)
	}
}

// Profile Helpers

func (h *E2EHelper) GetMe(token string) *httpexpect.Object {
	return h.e.GET("/api/v1/auth/me").
		WithHeader("Authorization", token).
		Expect().
		Status(http.StatusOK).
		JSON().
		Object()
}

func (h *E2EHelper) GetPublicProfile(userID string, expectStatus int) *httpexpect.Object {
	req := h.e.GET("/api/v1/users/{id}", userID)

	if expectStatus != http.StatusOK {
		req.Expect().Status(expectStatus)
		return nil
	}

	return req.Expect().
		Status(http.StatusOK).
		JSON().
		Object()
}

// Team Helpers

func (h *E2EHelper) GetMyTeam(token string, expectStatus int) *httpexpect.Object {
	return h.e.GET("/api/v1/teams/my").
		WithHeader("Authorization", token).
		Expect().
		Status(expectStatus).
		JSON().
		Object()
}

func (h *E2EHelper) JoinTeam(token, inviteToken string, expectStatus int) {
	h.e.POST("/api/v1/teams/join").
		WithHeader("Authorization", token).
		WithJSON(map[string]string{
			"invite_token": inviteToken,
		}).
		Expect().
		Status(expectStatus)
}
