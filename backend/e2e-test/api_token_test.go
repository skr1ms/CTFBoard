package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
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
