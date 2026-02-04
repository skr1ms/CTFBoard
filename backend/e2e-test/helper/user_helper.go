package helper

import (
	"context"
	"net/http"
	"time"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

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
	RequireStatus(h.t, http.StatusCreated, resp.StatusCode(), resp.Body, "register")
}

func (h *E2EHelper) RegisterExpectStatus(username, email, password string, expectStatus int) *openapi.PostAuthRegisterResponse {
	h.t.Helper()
	resp, err := h.client.PostAuthRegisterWithResponse(context.Background(), openapi.PostAuthRegisterJSONRequestBody{
		Username: &username,
		Email:    &email,
		Password: &password,
	})
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "register")
	return resp
}

func (h *E2EHelper) Login(email, password string, expectStatus int) *openapi.PostAuthLoginResponse {
	h.t.Helper()
	resp := h.LoginWithClient(context.Background(), h.client, email, password)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "login")
	return resp
}

func (h *E2EHelper) ForgotPassword(email string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.PostAuthForgotPasswordWithResponse(context.Background(), openapi.PostAuthForgotPasswordJSONRequestBody{
		Email: &email,
	})
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "forgot-password")
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
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "resend-verification")
}

func (h *E2EHelper) GetPublicProfile(userID string, expectStatus int) *openapi.GetUsersIDResponse {
	h.t.Helper()
	resp, err := h.client.GetUsersIDWithResponse(context.Background(), userID)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get user profile")
	return resp
}

func (h *E2EHelper) GetUserTokens(token string, expectStatus int) *openapi.GetUserTokensResponse {
	h.t.Helper()
	resp, err := h.client.GetUserTokensWithResponse(context.Background(), WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get user tokens")
	return resp
}

func (h *E2EHelper) CreateUserToken(token, description string, expectStatus int) *openapi.PostUserTokensResponse {
	h.t.Helper()
	desc := description
	resp, err := h.client.PostUserTokensWithResponse(context.Background(), openapi.PostUserTokensJSONRequestBody{
		Description: &desc,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create user token")
	return resp
}

func (h *E2EHelper) DeleteUserToken(token, id string, expectStatus int) *openapi.DeleteUserTokensIDResponse {
	h.t.Helper()
	resp, err := h.client.DeleteUserTokensIDWithResponse(context.Background(), id, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete user token")
	return resp
}

func (h *E2EHelper) RegisterUserAndLogin(username string) (email, password, token string) {
	h.t.Helper()
	email = username + "@example.com"
	password = "password123"
	h.Register(username, email, password)
	token = RequireLoginOK(h.t, h.Login(email, password, http.StatusOK))
	return email, password, "Bearer " + token
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
	email, password, token = h.RegisterUserAndLogin(username)
	meResp := h.MeWithClient(context.Background(), h.client, token)
	RequireStatus(h.t, http.StatusOK, meResp.StatusCode(), meResp.Body, "me")
	require.NotNil(h.t, meResp.JSON200)
	require.NotNil(h.t, meResp.JSON200.ID)
	userID := *meResp.JSON200.ID
	_, err := h.pool.Exec(context.Background(), "UPDATE users SET role = 'admin' WHERE ID = $1", userID)
	require.NoError(h.t, err)
	resp := h.Login(email, password, http.StatusOK)
	require.NotNil(h.t, resp.JSON200)
	token = "Bearer " + *resp.JSON200.AccessToken
	return email, password, token
}

func (h *E2EHelper) SetupCompetition(adminNamePrefix string) (string, string) {
	h.t.Helper()
	suffix := time.Now().Format("150405")
	username := adminNamePrefix + "_" + suffix
	_, _, token := h.RegisterAdmin(username)
	h.StartCompetition(token)
	return username, token
}
