package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// GET /admin/users: admin gets paginated user list.
func TestAdminUsers_List_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenAdmin := h.RegisterAdmin("admin_list_users")
	uid := helper.UID()
	h.Register("listed_"+uid+"_1", "listed_"+uid+"_1@test.com", "ValidPass1")
	h.Register("listed_"+uid+"_2", "listed_"+uid+"_2@test.com", "ValidPass1")

	page, perPage := 1, 20
	params := &openapi.GetAdminUsersParams{Page: &page, PerPage: &perPage}
	resp, err := h.Client().GetAdminUsersWithResponse(context.Background(), params, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "admin list users")
	require.NotNil(t, resp.JSON200)
}

// GET /admin/users: non-admin returns 403 Forbidden.
func TestAdminUsers_List_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenUser := h.RegisterUserAndLogin("non_admin_list")
	page, perPage := 1, 20
	params := &openapi.GetAdminUsersParams{Page: &page, PerPage: &perPage}
	resp, err := h.Client().GetAdminUsersWithResponse(context.Background(), params, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, resp.StatusCode(), resp.Body, "non-admin list users")
}

// POST /admin/users: admin creates user with role.
func TestAdminUsers_Create_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenAdmin := h.RegisterAdmin("admin_create_user")

	uid := helper.UID()
	role := "user"
	body := openapi.AdminCreateUserRequest{
		Email:    "created_" + uid + "@test.com",
		Password: "ValidPass1",
		Username: "created_" + uid,
		Role:     &role,
	}
	resp, err := h.Client().PostAdminUsersWithResponse(context.Background(), body, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusCreated, resp.StatusCode(), resp.Body, "admin create user")
}

// POST /admin/users: non-admin returns 403 Forbidden.
func TestAdminUsers_Create_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenUser := h.RegisterUserAndLogin("non_admin_create")

	uid := helper.UID()
	role := "user"
	body := openapi.AdminCreateUserRequest{
		Email:    "forbidden_" + uid + "@test.com",
		Password: "ValidPass1",
		Username: "forbidden_" + uid,
		Role:     &role,
	}
	resp, err := h.Client().PostAdminUsersWithResponse(context.Background(), body, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, resp.StatusCode(), resp.Body, "non-admin create user")
}

// PATCH /admin/users/{ID}: admin updates user email.
func TestAdminUsers_Update_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenAdmin := h.RegisterAdmin("admin_update_user")
	uid := helper.UID()
	h.Register("update_"+uid, "update_"+uid+"@test.com", "ValidPass1")
	userID := h.GetUserIDByEmail("update_" + uid + "@test.com")

	newEmail := "updated_" + uid + "@test.com"
	body := openapi.AdminUpdateUserRequest{Email: &newEmail}
	resp, err := h.Client().PatchAdminUsersIDWithResponse(context.Background(), userID, body, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "admin update user")
}

// PATCH /admin/users/{ID}: user not found returns 404.
func TestAdminUsers_Update_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenAdmin := h.RegisterAdmin("admin_update_notfound")

	newEmail := "notfound@test.com"
	body := openapi.AdminUpdateUserRequest{Email: &newEmail}
	resp, err := h.Client().PatchAdminUsersIDWithResponse(context.Background(), uuid.New().String(), body, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNotFound, resp.StatusCode(), resp.Body, "admin update user not found")
}

// DELETE /admin/users/{ID}: admin deletes user.
func TestAdminUsers_Delete_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenAdmin := h.RegisterAdmin("admin_delete_user")
	uid := helper.UID()
	h.Register("delete_"+uid, "delete_"+uid+"@test.com", "ValidPass1")
	userID := h.GetUserIDByEmail("delete_" + uid + "@test.com")

	resp, err := h.Client().DeleteAdminUsersIDWithResponse(context.Background(), userID, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNoContent, resp.StatusCode(), resp.Body, "admin delete user")
}

// DELETE /admin/users/{ID}: user not found -> 404.
func TestAdminUsers_Delete_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenAdmin := h.RegisterAdmin("admin_delete_notfound")

	resp, err := h.Client().DeleteAdminUsersIDWithResponse(context.Background(), uuid.New().String(), helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNoContent, resp.StatusCode(), resp.Body, "admin delete user not found returns 204 (idempotent)")
}

// GET /admin/users/{ID}/tracking: admin gets IP tracking for user.
func TestAdminUsers_GetTracking_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenAdmin := h.RegisterAdmin("admin_tracking_admin")
	uid := helper.UID()
	h.Register("tracking_"+uid, "tracking_"+uid+"@test.com", "ValidPass1")
	userID := h.GetUserIDByEmail("tracking_" + uid + "@test.com")

	page, perPage := 1, 20
	params := &openapi.GetAdminUsersIDTrackingParams{Page: &page, PerPage: &perPage}
	resp, err := h.Client().GetAdminUsersIDTrackingWithResponse(context.Background(), userID, params, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "admin get user tracking")
}

// GET /admin/users/{ID}/tracking: non-admin returns 403.
func TestAdminUsers_GetTracking_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenUser := h.RegisterUserAndLogin("tracking_non_admin")
	uid := helper.UID()
	h.Register("tracking_f_"+uid, "tracking_f_"+uid+"@test.com", "ValidPass1")
	userID := h.GetUserIDByEmail("tracking_f_" + uid + "@test.com")

	page, perPage := 1, 20
	params := &openapi.GetAdminUsersIDTrackingParams{Page: &page, PerPage: &perPage}
	resp, err := h.Client().GetAdminUsersIDTrackingWithResponse(context.Background(), userID, params, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, resp.StatusCode(), resp.Body, "non-admin get tracking")
}

// GET /admin/users/{ID}/missing-challenges: admin gets unsolved challenges for user.
func TestAdminUsers_GetMissingChallenges_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("adm_missing_ch")
	uid := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("missing_" + uid)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	userID := h.GetUserIDByEmail("missing_" + uid + "@example.com")
	h.InvalidateUserCache(userID)

	h.CreateBasicChallenge(tokenAdmin, "Missing Chall", "flag{miss}", 100)

	resp, err := h.Client().GetAdminUsersIDMissingChallengesWithResponse(context.Background(), userID, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "admin get missing challenges")
	require.NotNil(t, resp.JSON200)
	assert.NotEmpty(t, *resp.JSON200)
}

// GET /admin/users/{ID}/missing-challenges: user not found returns 200 with empty list.
func TestAdminUsers_GetMissingChallenges_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_missing_notfound")

	resp, err := h.Client().GetAdminUsersIDMissingChallengesWithResponse(context.Background(), uuid.New().String(), helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "admin missing challenges empty list for unknown user")
}

// GET /users/me/solves: authed user gets own solves.
func TestUsers_MeSolves_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("me_solves_admin")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Me Solves Chall", "flag{meSolves}", 100)
	_, _, tokenUser := h.RegisterUserAndLogin("me_solves_user")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{meSolves}", http.StatusOK)

	resp, err := h.Client().GetUsersMeSolvesWithResponse(context.Background(), helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get me solves")
}

// GET /users/me/solves: no auth returns 401.
func TestUsers_MeSolves_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resp, err := h.Client().GetUsersMeSolvesWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, "me solves no auth")
}

// GET /users/{ID}/solves: admin can view any user's solves; regular user can only view own.
func TestUsers_IDSolves_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("id_solves_admin")
	uid := helper.UID()
	h.Register("id_solves_"+uid, "id_solves_"+uid+"@test.com", "ValidPass1")
	targetID := h.GetUserIDByEmail("id_solves_" + uid + "@test.com")

	// Admin can view any user's solves.
	resp, err := h.Client().GetUsersIDSolvesWithResponse(context.Background(), targetID, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "admin get user id solves")

	// Regular user can view own solves.
	_, _, tokenUser := h.RegisterUserAndLogin("id_solves_self_" + uid)
	meResp := h.GetMe(tokenUser, http.StatusOK)
	require.NotNil(t, meResp.JSON200)
	selfID := *meResp.JSON200.ID
	selfResp, err := h.Client().GetUsersIDSolvesWithResponse(context.Background(), selfID, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, selfResp.StatusCode(), selfResp.Body, "user get own solves")
}

// GET /users/{ID}/solves: user not found -> 404.
func TestUsers_IDSolves_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("id_solves_notfound")

	resp, err := h.Client().GetUsersIDSolvesWithResponse(context.Background(), uuid.New().String(), helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNotFound, resp.StatusCode(), resp.Body, "get user id solves not found")
}

// GET /users/me/fails: authed gets own failed submissions.
func TestUsers_MeFails_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("me_fails_admin")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Me Fails Chall", "flag{meFails}", 100)
	_, _, tokenUser := h.RegisterUserAndLogin("me_fails_user")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{wrong}", http.StatusOK)

	page, perPage := 1, 20
	params := &openapi.GetUsersMeFailsParams{Page: &page, PerPage: &perPage}
	resp, err := h.Client().GetUsersMeFailsWithResponse(context.Background(), params, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get me fails")
}

// GET /users/me/fails: no auth returns 401.
func TestUsers_MeFails_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	page, perPage := 1, 20
	params := &openapi.GetUsersMeFailsParams{Page: &page, PerPage: &perPage}
	resp, err := h.Client().GetUsersMeFailsWithResponse(context.Background(), params)
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, "me fails no auth")
}

// GET /users/me/submissions: authed gets own submissions.
func TestUsers_MeSubmissions_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenUser := h.RegisterUserAndLogin("me_submissions_user")

	page, perPage := 1, 20
	params := &openapi.GetUsersMeSubmissionsParams{Page: &page, PerPage: &perPage}
	resp, err := h.Client().GetUsersMeSubmissionsWithResponse(context.Background(), params, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get me submissions")
}

// GET /users/me/submissions: no auth returns 401.
func TestUsers_MeSubmissions_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	page, perPage := 1, 20
	params := &openapi.GetUsersMeSubmissionsParams{Page: &page, PerPage: &perPage}
	resp, err := h.Client().GetUsersMeSubmissionsWithResponse(context.Background(), params)
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, "me submissions no auth")
}

// GET /users/me/awards: authed gets own awards.
func TestUsers_MeAwards_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenUser := h.RegisterUserAndLogin("me_awards_user")

	resp, err := h.Client().GetUsersMeAwardsWithResponse(context.Background(), helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get me awards")
}

// GET /users/me/awards: no auth returns 401.
func TestUsers_MeAwards_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resp, err := h.Client().GetUsersMeAwardsWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, "me awards no auth")
}

// GET /users/{ID}/awards: authed gets user's awards.
func TestUsers_IDAwards_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("id_awards_admin")
	uid := helper.UID()
	h.Register("id_awards_"+uid, "id_awards_"+uid+"@test.com", "ValidPass1")
	targetID := h.GetUserIDByEmail("id_awards_" + uid + "@test.com")

	resp, err := h.Client().GetUsersIDAwardsWithResponse(context.Background(), targetID, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get user id awards")
}

// GET /users/{ID}/awards: user not found returns 404.
func TestUsers_IDAwards_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("id_awards_notfound")

	resp, err := h.Client().GetUsersIDAwardsWithResponse(context.Background(), uuid.New().String(), helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNotFound, resp.StatusCode(), resp.Body, "get user id awards not found")
}
