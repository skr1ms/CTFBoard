package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// POST /user/tokens + GET /user/tokens: token is created and visible in list.
func TestAPIToken_CreateAndList_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("apitok_user_" + suffix)

	createResp := h.CreateUserToken(tokenUser, "desc "+suffix, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	listResp := h.GetUserTokens(tokenUser, http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	require.Len(t, *listResp.JSON200, 1)
}

// /user/tokens management requires a JWT session, not a long-lived API token.
func TestAPIToken_TokenManagement_WithAPITokenForbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("apitok_self_" + suffix)
	createResp := h.CreateUserToken(tokenUser, "desc "+suffix, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	plainToken := createResp.JSON201.Token
	require.NotEmpty(t, plainToken)

	listResp, err := h.Client().GetUserTokensWithResponse(context.Background(), helper.WithAPIToken(plainToken))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, listResp.StatusCode(), listResp.Body, "list tokens with api token")

	desc := "replacement " + suffix
	createWithAPITokenResp, err := h.Client().PostUserTokensWithResponse(
		context.Background(),
		openapi.PostUserTokensJSONRequestBody{Description: &desc},
		helper.WithAPIToken(plainToken),
	)
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, createWithAPITokenResp.StatusCode(), createWithAPITokenResp.Body, "create token with api token")

	deleteResp, err := h.Client().DeleteUserTokensIDWithResponse(context.Background(), *createResp.JSON201.ID, helper.WithAPIToken(plainToken))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, deleteResp.StatusCode(), deleteResp.Body, "delete token with api token")
}

// POST /user/tokens: without auth returns 401.
func TestAPIToken_Create_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.CreateUserToken("", "desc", http.StatusUnauthorized)
}

// GET /user/tokens: without auth returns 401.
func TestAPIToken_List_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.GetUserTokens("", http.StatusUnauthorized)
}

// DELETE /user/tokens/{id}: user deletes own token.
func TestAPIToken_Delete_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("apitok_del_" + suffix)
	createResp := h.CreateUserToken(tokenUser, "desc", http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	h.DeleteUserToken(tokenUser, *createResp.JSON201.ID, http.StatusNoContent)
	listResp := h.GetUserTokens(tokenUser, http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	require.Empty(t, *listResp.JSON200)
}

// DELETE /user/tokens/{id}: delete with wrong id returns 204 (idempotent).
func TestAPIToken_Delete_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("apitok_del_nf_" + suffix)

	h.DeleteUserToken(tokenUser, uuid.New().String(), http.StatusNoContent)
}
