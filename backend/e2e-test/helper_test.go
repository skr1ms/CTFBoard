package e2e_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type E2EHelper struct {
	t    *testing.T
	e    *httpexpect.Expect
	pool *pgxpool.Pool
}

func NewE2EHelper(t *testing.T, e *httpexpect.Expect, pool *pgxpool.Pool) *E2EHelper {
	t.Helper()
	return &E2EHelper{
		t:    t,
		e:    e,
		pool: pool,
	}
}

// API Wrappers

func (h *E2EHelper) Register(username, email, password string) {
	h.t.Helper()
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
	h.t.Helper()
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
	h.t.Helper()
	h.e.POST("/api/v1/auth/forgot-password").
		WithJSON(map[string]string{
			"email": email,
		}).
		Expect().
		Status(expectStatus)
}

func (h *E2EHelper) ResetPassword(token, newPassword string) {
	h.t.Helper()
	h.e.POST("/api/v1/auth/reset-password").
		WithJSON(map[string]string{
			"token":        token,
			"new_password": newPassword,
		}).
		Expect().
		Status(http.StatusOK)
}

func (h *E2EHelper) VerifyEmail(token string) {
	h.t.Helper()
	h.e.GET("/api/v1/auth/verify-email").
		WithQuery("token", token).
		Expect().
		Status(http.StatusOK).
		JSON().Object().Value("message").IsEqual("email verified successfully")
}

// DB Helpers

func (h *E2EHelper) GetuserIDByEmail(email string) string {
	h.t.Helper()
	ctx := context.Background()
	var ID string
	err := h.pool.QueryRow(ctx, "SELECT ID FROM users WHERE email = $1", email).Scan(&ID)
	require.NoError(h.t, err, "failed to find user by email")
	return ID
}

func (h *E2EHelper) AssertUserVerified(email string, expected bool) {
	h.t.Helper()
	ctx := context.Background()
	var isVerified bool
	err := h.pool.QueryRow(ctx, "SELECT is_verified FROM users WHERE email = $1", email).Scan(&isVerified)
	require.NoError(h.t, err)
	assert.Equal(h.t, expected, isVerified, "user verification status mismatch")
}

func (h *E2EHelper) InjectToken(userID string, tokenType entity.TokenType, knownToken string) {
	h.t.Helper()
	ctx := context.Background()
	hashedToken := h.hashToken(knownToken)

	var count int
	err := h.pool.QueryRow(ctx, "SELECT count(*) FROM verification_tokens WHERE user_id = $1 AND type = $2", userID, tokenType).Scan(&count)
	require.NoError(h.t, err)
	require.Equal(h.t, 1, count, "token row should exist before injection")

	_, err = h.pool.Exec(ctx, "UPDATE verification_tokens SET token = $1 WHERE user_id = $2 AND type = $3", hashedToken, userID, tokenType)
	require.NoError(h.t, err, "failed to inject token")
}

func (h *E2EHelper) hashToken(token string) string {
	h.t.Helper()
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Challenge Helpers

func (h *E2EHelper) RegisterUserAndLogin(username string) (email, password, token string) {
	h.t.Helper()
	email = username + "@example.com"
	password = "password123"
	h.Register(username, email, password)

	resp := h.Login(email, password, http.StatusOK)
	token = "Bearer " + resp.Value("access_token").String().Raw()
	return email, password, token
}

func (h *E2EHelper) RegisterUser(username string) (email, password string) {
	h.t.Helper()
	email = username + "@example.com"
	password = "password123"
	h.Register(username, email, password)
	return email, password
}

func (h *E2EHelper) RegisterAdmin(username string) (email, password, token string) {
	h.t.Helper()
	ctx := context.Background()
	email, password, token = h.RegisterUserAndLogin(username)

	meResp := h.e.GET("/api/v1/auth/me").
		WithHeader("Authorization", token).
		Expect().Status(200).JSON().Object()
	userID := meResp.Value("id").String().Raw()

	_, err := h.pool.Exec(ctx, "UPDATE users SET role = 'admin' WHERE ID = $1", userID)
	require.NoError(h.t, err)

	resp := h.Login(email, password, http.StatusOK)
	token = "Bearer " + resp.Value("access_token").String().Raw()
	return email, password, token
}

func (h *E2EHelper) CreateChallenge(token string, data map[string]any) string {
	h.t.Helper()
	resp := h.e.POST("/api/v1/admin/challenges").
		WithHeader("Authorization", token).
		WithJSON(data).
		Expect().
		Status(http.StatusCreated).
		JSON().Object()

	return resp.Value("id").String().Raw()
}

func (h *E2EHelper) CreateBasicChallenge(token, title, flag string, points int) string {
	h.t.Helper()
	return h.CreateChallenge(token, map[string]any{
		"title":       title,
		"description": "Standard basic challenge",
		"flag":        flag,
		"points":      points,
		"category":    "misc",
		"is_hidden":   false,
	})
}

func (h *E2EHelper) SubmitFlag(token, challengeID, flag string, expectStatus int) *httpexpect.Object {
	h.t.Helper()
	return h.e.POST("/api/v1/challenges/{ID}/submit", challengeID).
		WithHeader("Authorization", token).
		WithJSON(map[string]string{"flag": flag}).
		Expect().
		Status(expectStatus).
		JSON().Object()
}

func (h *E2EHelper) FindChallengeInList(token, challengeID string) *httpexpect.Object {
	h.t.Helper()
	challenges := h.e.GET("/api/v1/challenges").
		WithHeader("Authorization", token).
		Expect().
		Status(http.StatusOK).
		JSON().Array()

	for i := 0; i < int(challenges.Length().Raw()); i++ {
		c := challenges.Value(i).Object()
		if c.Value("id").String().Raw() == challengeID {
			return c
		}
	}
	h.t.Fatalf("Challenge %s not found in list", challengeID)
	return nil
}

func (h *E2EHelper) AssertChallengeMissing(token, challengeID string) {
	h.t.Helper()
	challenges := h.e.GET("/api/v1/challenges").
		WithHeader("Authorization", token).
		Expect().
		Status(http.StatusOK).
		JSON().Array()

	for i := 0; i < int(challenges.Length().Raw()); i++ {
		if challenges.Value(i).Object().Value("id").String().Raw() == challengeID {
			h.t.Fatalf("Challenge %s should NOT be in list", challengeID)
		}
	}
}

// Competition Helpers

func (h *E2EHelper) GetCompetitionStatus() *httpexpect.Object {
	h.t.Helper()
	return h.e.GET("/api/v1/competition/status").
		Expect().
		Status(http.StatusOK).
		JSON().Object()
}

func (h *E2EHelper) UpdateCompetition(token string, data map[string]any) {
	h.t.Helper()
	h.e.PUT("/api/v1/admin/competition").
		WithHeader("Authorization", token).
		WithJSON(data).
		Expect().
		Status(http.StatusOK)
}

func (h *E2EHelper) SetCompetitionRegex(token, regex string) {
	h.t.Helper()
	now := time.Now().UTC()
	h.UpdateCompetition(token, map[string]any{
		"name":              "Test CTF",
		"is_public":         true,
		"flag_regex":        regex,
		"start_time":        now.Add(-1 * time.Hour),
		"end_time":          now.Add(24 * time.Hour),
		"is_paused":         false,
		"allow_team_switch": true,
		"mode":              "flexible",
	})
}

func (h *E2EHelper) StartCompetition(adminToken string) {
	h.t.Helper()
	now := time.Now().UTC()
	resp := h.e.PUT("/api/v1/admin/competition").
		WithHeader("Authorization", adminToken).
		WithJSON(map[string]any{
			"name":              "Test CTF",
			"start_time":        now.Add(-1 * time.Hour),
			"end_time":          now.Add(24 * time.Hour),
			"is_paused":         false,
			"allow_team_switch": true,
			"mode":              "flexible",
		}).
		Expect()

	resp.Status(http.StatusOK)
}

func (h *E2EHelper) SetupCompetition(adminNamePrefix string) (string, string) {
	h.t.Helper()
	suffix := time.Now().Format("150405")
	username := adminNamePrefix + "_" + suffix
	_, _, token := h.RegisterAdmin(username)
	h.StartCompetition(token)
	return username, token
}

func (h *E2EHelper) UpdateChallenge(token, challengeID string, data map[string]any) {
	h.e.PUT("/api/v1/admin/challenges/{ID}", challengeID).
		WithHeader("Authorization", token).
		WithJSON(data).
		Expect().
		Status(http.StatusOK)
}

func (h *E2EHelper) DeleteChallenge(token, challengeID string) {
	h.e.DELETE("/api/v1/admin/challenges/{ID}", challengeID).
		WithHeader("Authorization", token).
		Expect().
		Status(http.StatusNoContent)
}

// First Blood Helpers

func (h *E2EHelper) GetFirstBlood(challengeID string, expectStatus int) *httpexpect.Object {
	return h.e.GET("/api/v1/challenges/{ID}/first-blood", challengeID).
		Expect().
		Status(expectStatus).
		JSON().
		Object()
}

func (h *E2EHelper) AssertFirstBlood(challengeID, expectedUsername string) {
	h.t.Helper()
	resp := h.GetFirstBlood(challengeID, http.StatusOK)

	resp.Value("username").String().IsEqual(expectedUsername)
	resp.Value("team_name").String().IsEqual(expectedUsername)

	resp.ContainsKey("user_id")
	resp.ContainsKey("team_id")
	resp.ContainsKey("solved_at")
}

// Hint Helpers

func (h *E2EHelper) CreateHint(token, challengeID, content string, cost int) string {
	resp := h.e.POST("/api/v1/admin/challenges/{ID}/hints", challengeID).
		WithHeader("Authorization", token).
		WithJSON(map[string]any{
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
	req := h.e.POST("/api/v1/challenges/{cID}/hints/{hID}/unlock", challengeID, hintID).
		WithHeader("Authorization", token)

	if expectStatus != http.StatusOK {
		req.Expect().Status(expectStatus)
		return nil
	}

	return req.Expect().Status(http.StatusOK).JSON().Object()
}

func (h *E2EHelper) GetHintFromList(token, challengeID, hintID string) *httpexpect.Object {
	hintsArr := h.e.GET("/api/v1/challenges/{ID}/hints", challengeID).
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
	h.t.Helper()
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
	req := h.e.GET("/api/v1/users/{ID}", userID)

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

func (h *E2EHelper) JoinTeam(token, inviteToken string, confirmReset bool, expectStatus int) {
	h.e.POST("/api/v1/teams/join").
		WithHeader("Authorization", token).
		WithJSON(map[string]any{
			"invite_token":  inviteToken,
			"confirm_reset": confirmReset,
		}).
		Expect().
		Status(expectStatus)
}

func (h *E2EHelper) CreateTeam(token, name string, expectStatus int) {
	h.e.POST("/api/v1/teams").
		WithHeader("Authorization", token).
		WithJSON(map[string]string{
			"name": name,
		}).
		Expect().
		Status(expectStatus)
}

func (h *E2EHelper) CreateSoloTeam(token string, expectStatus int) {
	h.e.POST("/api/v1/teams/solo").
		WithHeader("Authorization", token).
		WithJSON(map[string]any{}).
		Expect().
		Status(expectStatus)
}

// File Helpers

func (h *E2EHelper) UploadChallengeFile(token, challengeID, fileName, content string) *httpexpect.Object {
	return h.e.POST("/api/v1/admin/challenges/{ID}/files", challengeID).
		WithHeader("Authorization", token).
		WithMultipart().
		WithFile("file", fileName, strings.NewReader(content)).
		Expect().
		Status(http.StatusCreated).
		JSON().Object()
}

func (h *E2EHelper) GetChallengeFiles(token, challengeID string) *httpexpect.Array {
	return h.e.GET("/api/v1/challenges/{ID}/files", challengeID).
		WithHeader("Authorization", token).
		Expect().
		Status(http.StatusOK).
		JSON().Array()
}

func (h *E2EHelper) GetFileDownloadURL(token, fileID string) string {
	resp := h.e.GET("/api/v1/files/{ID}/download", fileID).
		WithHeader("Authorization", token).
		Expect().
		Status(http.StatusOK).
		JSON().Object()

	return resp.Value("url").String().Raw()
}

func (h *E2EHelper) DownloadFileContent(token, url string) string {
	return h.e.GET(url).
		WithHeader("Authorization", token).
		Expect().
		Status(http.StatusOK).
		Body().Raw()
}
