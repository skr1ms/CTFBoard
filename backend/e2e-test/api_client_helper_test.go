package e2e_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func getStr(m map[string]any, key, def string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return def
}

func getInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

func getIntPtr(m map[string]any, key string) *int {
	switch v := m[key].(type) {
	case int:
		return &v
	case float64:
		i := int(v)
		return &i
	}
	return nil
}

func GetAPIClient(t *testing.T) *openapi.ClientWithResponses {
	t.Helper()
	client, err := openapi.NewClientWithResponses(GetTestBaseURL() + "/api/v1")
	require.NoError(t, err)
	return client
}

func WithBearerToken(token string) openapi.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		if token != "" && !strings.HasPrefix(token, "Bearer ") {
			token = "Bearer " + token
		}
		req.Header.Set("Authorization", token)
		return nil
	}
}

func RequireStatus(t *testing.T, expect, actual int, body []byte, label string) {
	t.Helper()
	require.Equal(t, expect, actual, "%s: %s", label, body)
}

func RequireRegisterCreated(t *testing.T, resp *openapi.PostAuthRegisterResponse) {
	t.Helper()
	RequireStatus(t, http.StatusCreated, resp.StatusCode(), resp.Body, "register")
}

func RequireLoginOK(t *testing.T, resp *openapi.PostAuthLoginResponse) string {
	t.Helper()
	RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "login")
	require.NotNil(t, resp.JSON200)
	return *resp.JSON200.AccessToken
}

func RequireMeOK(t *testing.T, resp *openapi.GetAuthMeResponse) *openapi.ResponseMeResponse {
	t.Helper()
	RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "me")
	require.NotNil(t, resp.JSON200)
	return resp.JSON200
}

func RequireConflict(t *testing.T, resp *openapi.PostAuthRegisterResponse, label string) {
	t.Helper()
	RequireStatus(t, http.StatusConflict, resp.StatusCode(), resp.Body, label)
	require.NotNil(t, resp.JSON409)
	require.NotEmpty(t, *resp.JSON409.Error)
}

func RequireUnauthorized(t *testing.T, resp *openapi.PostAuthLoginResponse, label string) {
	t.Helper()
	RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, label)
	require.NotNil(t, resp.JSON401)
	require.NotEmpty(t, *resp.JSON401.Error)
}

func RequireMeUnauthorized(t *testing.T, resp *openapi.GetAuthMeResponse) {
	t.Helper()
	RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, "me")
	require.NotNil(t, resp.JSON401)
	require.NotEmpty(t, *resp.JSON401.Error)
}

func RequireMyTeamOK(t *testing.T, resp *openapi.GetTeamsMyResponse) string {
	t.Helper()
	RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get my team")
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.ID)
	return *resp.JSON200.ID
}

func RequireAwardsCount(t *testing.T, resp *openapi.GetAdminAwardsTeamTeamIDResponse, count int) {
	t.Helper()
	RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get awards by team")
	require.NotNil(t, resp.JSON200)
	require.Len(t, *resp.JSON200, count)
}

func RequireChallengeFields(t *testing.T, c *openapi.ResponseChallengeResponse, title string, solved *bool, solveCount, points *int) {
	t.Helper()

	require.NotNil(t, c, "challenge is nil")
	if title != "" {
		require.NotNil(t, c.Title)
		require.Equal(t, title, *c.Title)
	}
	if solved != nil {
		require.NotNil(t, c.Solved)
		require.Equal(t, *solved, *c.Solved)
	}
	if solveCount != nil {
		require.NotNil(t, c.SolveCount)
		require.Equal(t, *solveCount, *c.SolveCount)
	}
	if points != nil {
		require.NotNil(t, c.Points)
		require.Equal(t, *points, *c.Points)
	}
}

func (h *E2EHelper) RegisterLoginAndGetMe(ctx context.Context, username, email, password string) *openapi.ResponseMeResponse {
	h.t.Helper()
	regResp := h.RegisterWithClient(ctx, h.client, username, email, password)
	RequireStatus(h.t, http.StatusCreated, regResp.StatusCode(), regResp.Body, "register")
	token := RequireLoginOK(h.t, h.LoginWithClient(ctx, h.client, email, password))
	return RequireMeOK(h.t, h.MeWithClient(ctx, h.client, token))
}

func (h *E2EHelper) RegisterWithClient(ctx context.Context, client *openapi.ClientWithResponses, username, email, password string) *openapi.PostAuthRegisterResponse {
	h.t.Helper()
	resp, err := client.PostAuthRegisterWithResponse(ctx, openapi.PostAuthRegisterJSONRequestBody{
		Username: &username,
		Email:    &email,
		Password: &password,
	})
	require.NoError(h.t, err)
	return resp
}

func (h *E2EHelper) LoginWithClient(ctx context.Context, client *openapi.ClientWithResponses, email, password string) *openapi.PostAuthLoginResponse {
	h.t.Helper()
	resp, err := client.PostAuthLoginWithResponse(ctx, openapi.PostAuthLoginJSONRequestBody{
		Email:    &email,
		Password: password,
	})
	require.NoError(h.t, err)
	return resp
}

func (h *E2EHelper) MeWithClient(ctx context.Context, client *openapi.ClientWithResponses, token string) *openapi.GetAuthMeResponse {
	h.t.Helper()
	resp, err := client.GetAuthMeWithResponse(ctx, WithBearerToken(token))
	require.NoError(h.t, err)
	return resp
}

func (h *E2EHelper) Register(username, email, password string) {
	h.t.Helper()
	resp := h.RegisterWithClient(context.Background(), h.client, username, email, password)
	require.Equal(h.t, http.StatusCreated, resp.StatusCode(), "register: %s", resp.Body)
}

func (h *E2EHelper) RegisterExpectStatus(username, email, password string, expectStatus int) *openapi.PostAuthRegisterResponse {
	h.t.Helper()
	resp, err := h.client.PostAuthRegisterWithResponse(context.Background(), openapi.PostAuthRegisterJSONRequestBody{
		Username: &username,
		Email:    &email,
		Password: &password,
	})
	require.NoError(h.t, err)
	require.Equal(h.t, expectStatus, resp.StatusCode(), "register: %s", resp.Body)
	return resp
}

func (h *E2EHelper) Login(email, password string, expectStatus int) *openapi.PostAuthLoginResponse {
	h.t.Helper()
	resp := h.LoginWithClient(context.Background(), h.client, email, password)
	require.Equal(h.t, expectStatus, resp.StatusCode(), "login: %s", resp.Body)
	return resp
}

func (h *E2EHelper) ForgotPassword(email string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.PostAuthForgotPasswordWithResponse(context.Background(), openapi.PostAuthForgotPasswordJSONRequestBody{
		Email: &email,
	})
	require.NoError(h.t, err)
	require.Equal(h.t, expectStatus, resp.StatusCode(), "forgot-password: %s", resp.Body)
}

func (h *E2EHelper) ResetPassword(token, newPassword string) {
	h.t.Helper()
	h.ResetPasswordExpectStatus(token, newPassword, http.StatusOK)
}

func (h *E2EHelper) ResetPasswordExpectStatus(token, newPassword string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.PostAuthResetPasswordWithResponse(context.Background(), openapi.PostAuthResetPasswordJSONRequestBody{
		Token:       token,
		NewPassword: &newPassword,
	})
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "reset-password")
}

func (h *E2EHelper) VerifyEmail(token string) {
	h.t.Helper()
	h.VerifyEmailExpectStatus(token, http.StatusOK)
}

func (h *E2EHelper) VerifyEmailExpectStatus(token string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.GetAuthVerifyEmailWithResponse(context.Background(), &openapi.GetAuthVerifyEmailParams{Token: token})
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "verify-email")
}

func (h *E2EHelper) ResendVerification(token string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.PostAuthResendVerificationWithResponse(context.Background(), WithBearerToken(token))
	require.NoError(h.t, err)
	require.Equal(h.t, expectStatus, resp.StatusCode(), "resend-verification: %s", resp.Body)
}

func (h *E2EHelper) GetPublicProfile(userID string, expectStatus int) *openapi.GetUsersIDResponse {
	h.t.Helper()
	resp, err := h.client.GetUsersIDWithResponse(context.Background(), userID)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get user profile")
	return resp
}

func (h *E2EHelper) CreateChallenge(token string, data map[string]any) string {
	h.t.Helper()
	id := h.CreateChallengeExpectStatus(token, data, http.StatusCreated)
	require.NotEmpty(h.t, id, "create challenge returned empty id")
	return id
}

func (h *E2EHelper) CreateChallengeExpectStatus(token string, data map[string]any, expectStatus int) string {
	h.t.Helper()
	req := openapi.PostAdminChallengesJSONRequestBody{
		Category:    getStr(data, "category", "misc"),
		Description: getStr(data, "description", ""),
		Flag:        getStr(data, "flag", ""),
		Points:      getInt(data, "points"),
		Title:       getStr(data, "title", ""),
	}
	if v, ok := data["is_hidden"].(bool); ok {
		req.IsHidden = &v
	}
	if v, ok := data["is_regex"].(bool); ok {
		req.IsRegex = &v
	}
	if v, ok := data["is_case_insensitive"].(bool); ok {
		req.IsCaseInsensitive = &v
	}
	req.InitialValue = getIntPtr(data, "initial_value")
	req.MinValue = getIntPtr(data, "min_value")
	req.Decay = getIntPtr(data, "decay")
	resp, err := h.client.PostAdminChallengesWithResponse(context.Background(), req, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create challenge")
	if resp.JSON201 != nil && resp.JSON201.ID != nil {
		return *resp.JSON201.ID
	}
	return ""
}

func (h *E2EHelper) CreateBasicChallenge(token, title, flag string, points int) string {
	h.t.Helper()
	hidden := false
	return h.CreateChallenge(token, map[string]any{
		"title":         title,
		"description":   "Standard basic challenge",
		"flag":          flag,
		"points":        points,
		"category":      "misc",
		"is_hidden":     hidden,
		"initial_value": points,
		"min_value":     points,
		"decay":         1,
	})
}

func (h *E2EHelper) UpdateChallenge(token, challengeID string, data map[string]any) {
	h.t.Helper()
	h.UpdateChallengeExpectStatus(token, challengeID, data, http.StatusOK)
}

func (h *E2EHelper) UpdateChallengeExpectStatus(token, challengeID string, data map[string]any, expectStatus int) {
	h.t.Helper()
	req := openapi.PutAdminChallengesIDJSONRequestBody{
		Category:    getStr(data, "category", "misc"),
		Description: getStr(data, "description", ""),
		Points:      getInt(data, "points"),
		Title:       getStr(data, "title", ""),
	}
	if v, ok := data["flag"].(string); ok {
		req.Flag = &v
	}
	if v, ok := data["is_hidden"].(bool); ok {
		req.IsHidden = &v
	}
	resp, err := h.client.PutAdminChallengesIDWithResponse(context.Background(), challengeID, req, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "update challenge")
}

func (h *E2EHelper) DeleteChallenge(token, challengeID string) {
	h.t.Helper()
	h.DeleteChallengeExpectStatus(token, challengeID, http.StatusNoContent)
}

func (h *E2EHelper) DeleteChallengeExpectStatus(token, challengeID string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.DeleteAdminChallengesIDWithResponse(context.Background(), challengeID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete challenge")
}

func (h *E2EHelper) SubmitFlag(token, challengeID, flag string, expectStatus int) *openapi.PostChallengesIDSubmitResponse {
	h.t.Helper()
	resp, err := h.client.PostChallengesIDSubmitWithResponse(context.Background(), challengeID, openapi.PostChallengesIDSubmitJSONRequestBody{Flag: flag}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "submit flag")
	return resp
}

func (h *E2EHelper) GetChallengesExpectStatus(token string, expectStatus int) *openapi.GetChallengesResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesWithResponse(context.Background(), WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get challenges")
	return resp
}

func (h *E2EHelper) FindChallengeInList(token, challengeID string) *openapi.ResponseChallengeResponse {
	h.t.Helper()
	resp := h.GetChallengesExpectStatus(token, http.StatusOK)
	require.NotNil(h.t, resp.JSON200)
	for i := range *resp.JSON200 {
		c := &(*resp.JSON200)[i]
		if c.ID != nil && *c.ID == challengeID {
			return c
		}
	}
	h.t.Fatalf("Challenge %s not found in list", challengeID)
	return nil
}

func (h *E2EHelper) AssertChallengeMissing(token, challengeID string) {
	h.t.Helper()
	resp := h.GetChallengesExpectStatus(token, http.StatusOK)
	require.NotNil(h.t, resp.JSON200)
	for i := range *resp.JSON200 {
		c := &(*resp.JSON200)[i]
		if c.ID != nil && *c.ID == challengeID {
			h.t.Fatalf("Challenge %s should NOT be in list", challengeID)
		}
	}
}

func (h *E2EHelper) GetCompetitionStatus() *openapi.GetCompetitionStatusResponse {
	h.t.Helper()
	resp, err := h.client.GetCompetitionStatusWithResponse(context.Background())
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "competition status")
	return resp
}

func parseTimeField(data map[string]any, key string) *time.Time {
	switch v := data[key].(type) {
	case string:
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			return &t
		}
	case time.Time:
		return &v
	}
	return nil
}

func buildCompetitionBody(data map[string]any) openapi.PutAdminCompetitionJSONRequestBody {
	body := openapi.PutAdminCompetitionJSONRequestBody{Name: getStr(data, "name", "Test CTF")}
	if v, ok := data["is_public"].(bool); ok {
		body.IsPublic = &v
	}
	if v, ok := data["flag_regex"].(string); ok {
		body.FlagRegex = &v
	}
	if v, ok := data["is_paused"].(bool); ok {
		body.IsPaused = &v
	}
	if v, ok := data["allow_team_switch"].(bool); ok {
		body.AllowTeamSwitch = &v
	}
	if v, ok := data["mode"].(string); ok {
		body.Mode = &v
	}
	body.StartTime = parseTimeField(data, "start_time")
	body.EndTime = parseTimeField(data, "end_time")
	body.FreezeTime = parseTimeField(data, "freeze_time")
	return body
}

func (h *E2EHelper) UpdateCompetition(token string, data map[string]any) {
	h.t.Helper()
	h.PutAdminCompetitionExpectStatus(token, data, http.StatusOK)
}

func (h *E2EHelper) PutAdminCompetitionExpectStatus(token string, data map[string]any, expectStatus int) {
	h.t.Helper()
	body := buildCompetitionBody(data)
	resp, err := h.client.PutAdminCompetitionWithResponse(context.Background(), body, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "update competition")
}

func (h *E2EHelper) SetCompetitionRegex(token, regex string) {
	h.t.Helper()
	now := time.Now().UTC()
	h.UpdateCompetition(token, map[string]any{
		"name":              "Test CTF",
		"is_public":         true,
		"flag_regex":        regex,
		"start_time":        now.Add(-1 * time.Hour).Format(time.RFC3339),
		"end_time":          now.Add(24 * time.Hour).Format(time.RFC3339),
		"is_paused":         false,
		"allow_team_switch": true,
		"mode":              "flexible",
	})
}

func (h *E2EHelper) StartCompetition(adminToken string) {
	h.t.Helper()
	now := time.Now().UTC()
	h.UpdateCompetition(adminToken, map[string]any{
		"name":              "Test CTF",
		"start_time":        now.Add(-1 * time.Hour).Format(time.RFC3339),
		"end_time":          now.Add(24 * time.Hour).Format(time.RFC3339),
		"is_paused":         false,
		"allow_team_switch": true,
		"mode":              "flexible",
	})
}

func (h *E2EHelper) GetFirstBlood(challengeID string, expectStatus int) *openapi.GetChallengesIDFirstBloodResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesIDFirstBloodWithResponse(context.Background(), challengeID)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "first-blood")
	return resp
}

func (h *E2EHelper) AssertFirstBlood(challengeID, expectedUsername, expectedTeamName string) {
	h.t.Helper()
	resp := h.GetFirstBlood(challengeID, http.StatusOK)
	require.NotNil(h.t, resp.JSON200)
	require.NotNil(h.t, resp.JSON200.Username, "username")
	require.Equal(h.t, expectedUsername, *resp.JSON200.Username)
	require.NotNil(h.t, resp.JSON200.TeamName, "team_name")
	require.Equal(h.t, expectedTeamName, *resp.JSON200.TeamName)
	require.NotNil(h.t, resp.JSON200.UserID)
	require.NotNil(h.t, resp.JSON200.TeamID)
	require.NotNil(h.t, resp.JSON200.SolvedAt)
}

func (h *E2EHelper) CreateHint(token, challengeID, content string, cost int) string {
	h.t.Helper()
	return h.CreateHintExpectStatus(token, challengeID, content, cost, http.StatusCreated)
}

func (h *E2EHelper) CreateHintExpectStatus(token, challengeID, content string, cost, expectStatus int) string {
	h.t.Helper()
	orderIndex := 1
	resp, err := h.client.PostAdminChallengesChallengeIDHintsWithResponse(context.Background(), challengeID, openapi.PostAdminChallengesChallengeIDHintsJSONRequestBody{
		Content:    content,
		Cost:       &cost,
		OrderIndex: &orderIndex,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create hint")
	if resp.JSON201 != nil && resp.JSON201.ID != nil {
		return *resp.JSON201.ID
	}
	return ""
}

func (h *E2EHelper) UnlockHint(token, challengeID, hintID string, expectStatus int) *openapi.PostChallengesChallengeIDHintsHintIDUnlockResponse {
	h.t.Helper()
	resp, err := h.client.PostChallengesChallengeIDHintsHintIDUnlockWithResponse(context.Background(), challengeID, hintID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "unlock hint")
	return resp
}

func (h *E2EHelper) GetChallengesChallengeIDHintsExpectStatus(token, challengeID string, expectStatus int) *openapi.GetChallengesChallengeIDHintsResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesChallengeIDHintsWithResponse(context.Background(), challengeID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get hints")
	return resp
}

func (h *E2EHelper) GetHintFromList(token, challengeID, hintID string) *openapi.ResponseHintResponse {
	h.t.Helper()
	resp := h.GetChallengesChallengeIDHintsExpectStatus(token, challengeID, http.StatusOK)
	require.NotNil(h.t, resp.JSON200)
	for i := range *resp.JSON200 {
		c := &(*resp.JSON200)[i]
		if c.ID != nil && *c.ID == hintID {
			return c
		}
	}
	h.t.Fatalf("Hint %s not found in challenge %s list", hintID, challengeID)
	return nil
}

func (h *E2EHelper) UpdateHint(token, hintID, content string, cost, expectStatus int) *openapi.PutAdminHintsIDResponse {
	h.t.Helper()
	resp, err := h.client.PutAdminHintsIDWithResponse(context.Background(), hintID, openapi.PutAdminHintsIDJSONRequestBody{
		Content: content,
		Cost:    &cost,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "update hint")
	return resp
}

func (h *E2EHelper) DeleteHint(token, hintID string, expectStatus int) *openapi.DeleteAdminHintsIDResponse {
	h.t.Helper()
	resp, err := h.client.DeleteAdminHintsIDWithResponse(context.Background(), hintID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete hint")
	return resp
}

func (h *E2EHelper) DeleteChallengeFile(token, fileID string, expectStatus int) *openapi.DeleteAdminFilesIDResponse {
	h.t.Helper()
	resp, err := h.client.DeleteAdminFilesIDWithResponse(context.Background(), fileID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete file")
	return resp
}

func (h *E2EHelper) GetScoreboard() *openapi.GetScoreboardResponse {
	h.t.Helper()
	resp, err := h.client.GetScoreboardWithResponse(context.Background())
	require.NoError(h.t, err)
	return resp
}

func (h *E2EHelper) AssertTeamScore(teamName string, expectedPoints int) {
	h.t.Helper()
	resp := h.GetScoreboard()
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "scoreboard")
	require.NotNil(h.t, resp.JSON200)
	for _, entry := range *resp.JSON200 {
		if entry.TeamName != nil && *entry.TeamName == teamName {
			require.NotNil(h.t, entry.Points, "team %s has nil points", teamName)
			require.Equal(h.t, expectedPoints, *entry.Points, "team %s points", teamName)
			return
		}
	}
	h.t.Fatalf("Team %s not found in scoreboard", teamName)
}

func (h *E2EHelper) GetMyTeam(token string, expectStatus int) *openapi.GetTeamsMyResponse {
	h.t.Helper()
	resp, err := h.client.GetTeamsMyWithResponse(context.Background(), WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get my team")
	return resp
}

func (h *E2EHelper) CreateTeam(token, name string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.PostTeamsWithResponse(context.Background(), openapi.PostTeamsJSONRequestBody{Name: name}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create team")
}

func (h *E2EHelper) CreateSoloTeam(token string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.PostTeamsSoloWithResponse(context.Background(), openapi.PostTeamsSoloJSONRequestBody{Name: "Solo"}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create solo team")
}

func (h *E2EHelper) JoinTeam(token, inviteToken string, confirmReset bool, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.PostTeamsJoinWithResponse(context.Background(), openapi.PostTeamsJoinJSONRequestBody{
		InviteToken:  inviteToken,
		ConfirmReset: &confirmReset,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "join team")
}

func (h *E2EHelper) LeaveTeam(token string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.PostTeamsLeaveWithResponse(context.Background(), WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "leave team")
}

func (h *E2EHelper) GetTeamByID(token, teamID string, expectStatus int) *openapi.GetTeamsIDResponse {
	h.t.Helper()
	resp, err := h.client.GetTeamsIDWithResponse(context.Background(), teamID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get team by id")
	return resp
}

func (h *E2EHelper) DisbandTeam(token string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.DeleteTeamsMeWithResponse(context.Background(), WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "disband team")
}

func (h *E2EHelper) TransferCaptain(token, newCaptainID string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.PostTeamsTransferCaptainWithResponse(context.Background(), openapi.PostTeamsTransferCaptainJSONRequestBody{
		NewCaptainID: newCaptainID,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "transfer captain")
}

func (h *E2EHelper) KickMember(token, memberID string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.DeleteTeamsMembersIDWithResponse(context.Background(), memberID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "kick member")
}

func (h *E2EHelper) UploadChallengeFile(token, challengeID, fileName, content string) *openapi.PostAdminChallengesChallengeIDFilesResponse {
	h.t.Helper()
	return h.UploadChallengeFileExpectStatus(token, challengeID, fileName, content, http.StatusCreated)
}

func (h *E2EHelper) UploadChallengeFileExpectStatus(token, challengeID, fileName, content string, expectStatus int) *openapi.PostAdminChallengesChallengeIDFilesResponse {
	h.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", fileName)
	require.NoError(h.t, err)
	_, err = part.Write([]byte(content))
	require.NoError(h.t, err)
	contentType := w.FormDataContentType()
	require.NoError(h.t, w.Close())
	resp, err := h.client.PostAdminChallengesChallengeIDFilesWithBodyWithResponse(context.Background(), challengeID, contentType, &buf, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "upload challenge file")
	return resp
}

func (h *E2EHelper) GetChallengeFiles(token, challengeID string) *openapi.GetChallengesChallengeIDFilesResponse {
	h.t.Helper()
	return h.GetChallengeFilesExpectStatus(token, challengeID, http.StatusOK)
}

func (h *E2EHelper) GetChallengeFilesExpectStatus(token, challengeID string, expectStatus int) *openapi.GetChallengesChallengeIDFilesResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesChallengeIDFilesWithResponse(context.Background(), challengeID, nil, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get challenge files")
	return resp
}

func (h *E2EHelper) GetFileDownloadURL(token, fileID string) string {
	h.t.Helper()
	resp := h.GetFilesIDDownloadExpectStatus(token, fileID, http.StatusOK)
	require.NotNil(h.t, resp.JSON200)
	url, ok := (*resp.JSON200)["url"]
	require.True(h.t, ok, "url key in response")
	return url
}

func (h *E2EHelper) GetFilesIDDownloadExpectStatus(token, fileID string, expectStatus int) *openapi.GetFilesIDDownloadResponse {
	h.t.Helper()
	resp, err := h.client.GetFilesIDDownloadWithResponse(context.Background(), fileID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get file download url")
	return resp
}

func (h *E2EHelper) DownloadFileContent(token, url string) string {
	h.t.Helper()
	downloadURL := url
	if len(downloadURL) > 0 && downloadURL[0] == '/' {
		downloadURL = GetTestBaseURL() + downloadURL
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, downloadURL, nil)
	require.NoError(h.t, err)
	require.NoError(h.t, WithBearerToken(token)(context.Background(), req))
	rsp, err := http.DefaultClient.Do(req)
	require.NoError(h.t, err)
	defer rsp.Body.Close()
	require.Equal(h.t, http.StatusOK, rsp.StatusCode)
	body, err := io.ReadAll(rsp.Body)
	require.NoError(h.t, err)
	return string(body)
}

func (h *E2EHelper) CreateAward(token, teamID string, value int, description string, expectStatus int) *openapi.PostAdminAwardsResponse {
	h.t.Helper()
	resp, err := h.client.PostAdminAwardsWithResponse(context.Background(), openapi.PostAdminAwardsJSONRequestBody{
		TeamID:      teamID,
		Value:       value,
		Description: description,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create award")
	return resp
}

func (h *E2EHelper) GetAwardsByTeam(token, teamID string, expectStatus int) *openapi.GetAdminAwardsTeamTeamIDResponse {
	h.t.Helper()
	resp, err := h.client.GetAdminAwardsTeamTeamIDWithResponse(context.Background(), teamID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get awards by team")
	return resp
}

func (h *E2EHelper) BanTeam(token, teamID, reason string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.PostAdminTeamsIDBanWithResponse(context.Background(), teamID, openapi.PostAdminTeamsIDBanJSONRequestBody{Reason: reason}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "ban team")
}

func (h *E2EHelper) UnbanTeam(token, teamID string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.DeleteAdminTeamsIDBanWithResponse(context.Background(), teamID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "unban team")
}

func (h *E2EHelper) SetTeamHidden(token, teamID string, hidden bool, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.PatchAdminTeamsIDHiddenWithResponse(context.Background(), teamID, openapi.PatchAdminTeamsIDHiddenJSONRequestBody{Hidden: &hidden}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "set team hidden")
}

func (h *E2EHelper) GetStatisticsGeneral() *openapi.GetStatisticsGeneralResponse {
	h.t.Helper()
	resp, err := h.client.GetStatisticsGeneralWithResponse(context.Background())
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "statistics general")
	return resp
}

func (h *E2EHelper) GetStatisticsChallenges() *openapi.GetStatisticsChallengesResponse {
	h.t.Helper()
	resp, err := h.client.GetStatisticsChallengesWithResponse(context.Background())
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "statistics challenges")
	return resp
}

func (h *E2EHelper) GetStatisticsChallengesId(id string) *openapi.GetStatisticsChallengesIDResponse {
	h.t.Helper()
	parsed, err := uuid.Parse(id)
	require.NoError(h.t, err)
	resp, err := h.client.GetStatisticsChallengesIDWithResponse(context.Background(), parsed)
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "statistics challenge detail")
	return resp
}

func (h *E2EHelper) GetStatisticsChallengesIdExpectStatus(id string, expectStatus int) {
	h.t.Helper()
	parsed, err := uuid.Parse(id)
	require.NoError(h.t, err)
	resp, err := h.client.GetStatisticsChallengesIDWithResponse(context.Background(), parsed)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "statistics challenge detail")
}

func (h *E2EHelper) GetStatisticsScoreboard(limit int) *openapi.GetStatisticsScoreboardResponse {
	h.t.Helper()
	resp, err := h.client.GetStatisticsScoreboardWithResponse(context.Background())
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "statistics scoreboard")
	return resp
}

func (h *E2EHelper) GetScoreboardGraph(top int) *openapi.GetScoreboardGraphResponse {
	h.t.Helper()
	params := (*openapi.GetScoreboardGraphParams)(nil)
	if top > 0 {
		params = &openapi.GetScoreboardGraphParams{Top: &top}
	}
	resp, err := h.client.GetScoreboardGraphWithResponse(context.Background(), params)
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "scoreboard graph")
	return resp
}

func (h *E2EHelper) GetAdminCompetition(token string) *openapi.GetAdminCompetitionResponse {
	h.t.Helper()
	return h.GetAdminCompetitionExpectStatus(token, http.StatusOK)
}

func (h *E2EHelper) GetAdminCompetitionExpectStatus(token string, expectStatus int) *openapi.GetAdminCompetitionResponse {
	h.t.Helper()
	resp, err := h.client.GetAdminCompetitionWithResponse(context.Background(), WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin competition")
	return resp
}

func (h *E2EHelper) GetAdminSettings(token string) *openapi.GetAdminSettingsResponse {
	h.t.Helper()
	return h.GetAdminSettingsExpectStatus(token, http.StatusOK)
}

func (h *E2EHelper) GetAdminSettingsExpectStatus(token string, expectStatus int) *openapi.GetAdminSettingsResponse {
	h.t.Helper()
	resp, err := h.client.GetAdminSettingsWithResponse(context.Background(), WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin settings")
	return resp
}

func (h *E2EHelper) PutAdminSettings(token string, body map[string]any, expectStatus int) *openapi.PutAdminSettingsResponse {
	h.t.Helper()
	req := openapi.PutAdminSettingsJSONRequestBody{
		AppName:         getStr(body, "app_name", ""),
		CorsOrigins:     getStr(body, "cors_origins", ""),
		FrontendURL:     getStr(body, "frontend_url", ""),
		ResendFromEmail: getStr(body, "resend_from_email", ""),
		ResendFromName:  getStr(body, "resend_from_name", ""),
	}
	if v := getInt(body, "submit_limit_per_user"); v != 0 {
		req.SubmitLimitPerUser = &v
	}
	if v := getInt(body, "submit_limit_duration_min"); v != 0 {
		req.SubmitLimitDurationMin = &v
	}
	if v := getInt(body, "verify_ttl_hours"); v != 0 {
		req.VerifyTTLHours = &v
	}
	if v := getInt(body, "reset_ttl_hours"); v != 0 {
		req.ResetTTLHours = &v
	}
	if v, ok := body["verify_emails"].(bool); ok {
		req.VerifyEmails = &v
	}
	if v, ok := body["registration_open"].(bool); ok {
		req.RegistrationOpen = &v
	}
	if v, ok := body["resend_enabled"].(bool); ok {
		req.ResendEnabled = &v
	}
	if v, ok := body["scoreboard_visible"].(string); ok {
		req.ScoreboardVisible = (*openapi.RequestUpdateAppSettingsRequestScoreboardVisible)(&v)
	}
	resp, err := h.client.PutAdminSettingsWithResponse(context.Background(), req, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "put admin settings")
	return resp
}

func (h *E2EHelper) AdminExport(token string, includeUsers, includeAwards bool) *openapi.GetAdminExportResponse {
	h.t.Helper()
	return h.AdminExportExpectStatus(token, includeUsers, includeAwards, http.StatusOK)
}

func (h *E2EHelper) AdminExportExpectStatus(token string, includeUsers, includeAwards bool, expectStatus int) *openapi.GetAdminExportResponse {
	h.t.Helper()
	resp, err := h.client.GetAdminExportWithResponse(context.Background(), &openapi.GetAdminExportParams{
		IncludeUsers:  &includeUsers,
		IncludeAwards: &includeAwards,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin export")
	return resp
}

func (h *E2EHelper) AdminExportZip(token string) *openapi.GetAdminExportZipResponse {
	h.t.Helper()
	return h.AdminExportZipExpectStatus(token, http.StatusOK)
}

func (h *E2EHelper) AdminExportZipExpectStatus(token string, expectStatus int) *openapi.GetAdminExportZipResponse {
	h.t.Helper()
	resp, err := h.client.GetAdminExportZipWithResponse(context.Background(), nil, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin export zip")
	return resp
}

func (h *E2EHelper) AdminImport(token string, fileContent []byte, fileName, conflictMode string, expectStatus int) *openapi.PostAdminImportResponse {
	h.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", fileName)
	require.NoError(h.t, err)
	_, err = part.Write(fileContent)
	require.NoError(h.t, err)
	if conflictMode != "" {
		require.NoError(h.t, w.WriteField("conflict_mode", conflictMode))
	}
	contentType := w.FormDataContentType()
	require.NoError(h.t, w.Close())
	resp, err := h.client.PostAdminImportWithBodyWithResponse(context.Background(), contentType, &buf, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin import")
	return resp
}
