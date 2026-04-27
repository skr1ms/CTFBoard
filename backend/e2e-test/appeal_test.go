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

// banUser bans via direct SQL so the user's existing JWT stays valid.
// BanUser through the usecase revokes all JWTs, making the token unusable
// in the same second the new one is issued (Unix timestamp precision).
func banUser(t *testing.T, h *helper.E2EHelper, userID string) {
	t.Helper()
	h.BanUserDirectly(userID)
}

// POST /appeals: banned user submits an appeal; returns 201 with decision=pending.
func TestAppeal_CreateSuccess(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	uid := helper.UID()
	_, _, userToken := h.RegisterUserAndLogin("appeal_user_" + uid)

	me := h.GetMe(userToken, http.StatusOK)
	require.NotNil(t, me.JSON200)

	banUser(t, h, *me.JSON200.ID)

	resp := h.CreateAppeal(ctx, userToken, "please unban me", http.StatusCreated)
	require.NotNil(t, resp.JSON201)
	assert.Equal(t, "pending", *resp.JSON201.Decision)
	assert.NotNil(t, resp.JSON201.ID)
	assert.NotNil(t, resp.JSON201.CreatedAt)
}

// POST /appeals: second appeal within the 7-day cooldown returns 429.
func TestAppeal_RateLimited(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	uid := helper.UID()
	_, _, userToken := h.RegisterUserAndLogin("appeal_rl_usr_" + uid)

	me := h.GetMe(userToken, http.StatusOK)
	require.NotNil(t, me.JSON200)
	banUser(t, h, *me.JSON200.ID)

	h.CreateAppeal(ctx, userToken, "first appeal", http.StatusCreated)
	h.CreateAppeal(ctx, userToken, "second appeal within cooldown", http.StatusTooManyRequests)
}

// GET /appeals/me: each user sees only their own appeals, not other users'.
func TestAppeal_GetMyAppeals_ReturnsOwnOnly(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	uid := helper.UID()
	_, _, tok1 := h.RegisterUserAndLogin("appeal_iso_u1_" + uid)
	_, _, tok2 := h.RegisterUserAndLogin("appeal_iso_u2_" + uid)

	me1 := h.GetMe(tok1, http.StatusOK)
	me2 := h.GetMe(tok2, http.StatusOK)

	require.NotNil(t, me1.JSON200)
	require.NotNil(t, me2.JSON200)

	banUser(t, h, *me1.JSON200.ID)
	banUser(t, h, *me2.JSON200.ID)

	h.CreateAppeal(ctx, tok1, "appeal from u1", http.StatusCreated)
	h.CreateAppeal(ctx, tok2, "appeal from u2", http.StatusCreated)

	resp := h.GetMyAppeals(ctx, tok1, http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.Appeals)
	assert.Len(t, *resp.JSON200.Appeals, 1)
	assert.Equal(t, "appeal from u1", *(*resp.JSON200.Appeals)[0].Message)
}

// GET /admin/appeals: pagination returns correct page size and total count.
func TestAppeal_AdminList_Pagination(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	uid := helper.UID()
	_, _, adminToken := h.RegisterAdmin("appeal_pag_adm_" + uid)

	for i := range 3 {
		suffix := "appeal_pag_u" + string(rune('0'+i)) + "_" + uid
		_, _, tok := h.RegisterUserAndLogin(suffix)
		me := h.GetMe(tok, http.StatusOK)
		require.NotNil(t, me.JSON200)
		banUser(t, h, *me.JSON200.ID)
		h.CreateAppeal(ctx, tok, "please unban me", http.StatusCreated)
	}

	page, perPage := 1, 2
	resp := h.AdminListAppeals(ctx, adminToken, nil, &page, &perPage, http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.Appeals)
	assert.Len(t, *resp.JSON200.Appeals, 2)
	require.NotNil(t, resp.JSON200.Meta)
	require.NotNil(t, resp.JSON200.Meta.Total)
	assert.GreaterOrEqual(t, *resp.JSON200.Meta.Total, 3)
}

// GET /admin/appeals?decision=pending: filter returns only pending appeals, resolved ones excluded.
func TestAppeal_AdminList_FilterByDecision(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	uid := helper.UID()
	_, _, adminToken := h.RegisterAdmin("appeal_flt_adm_" + uid)

	_, _, tok1 := h.RegisterUserAndLogin("appeal_flt_u1_" + uid)
	me1 := h.GetMe(tok1, http.StatusOK)
	require.NotNil(t, me1.JSON200)
	banUser(t, h, *me1.JSON200.ID)
	h.CreateAppeal(ctx, tok1, "pending appeal", http.StatusCreated)

	_, _, tok2 := h.RegisterUserAndLogin("appeal_flt_u2_" + uid)
	me2 := h.GetMe(tok2, http.StatusOK)
	require.NotNil(t, me2.JSON200)
	banUser(t, h, *me2.JSON200.ID)
	created := h.CreateAppeal(ctx, tok2, "resolved appeal", http.StatusCreated)
	require.NotNil(t, created.JSON201)

	adminResp := "approved"
	h.AdminReviewAppeal(ctx, adminToken, *created.JSON201.ID, openapi.ReviewAppealRequestDecisionResolved, &adminResp, http.StatusOK)

	dec := openapi.GetAdminAppealsParamsDecisionPending
	resp := h.AdminListAppeals(ctx, adminToken, &dec, nil, nil, http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.Appeals)

	for _, a := range *resp.JSON200.Appeals {
		assert.Equal(t, "pending", *a.Decision)
	}
}

// PATCH /admin/appeals/{id}: resolved decision unbans the user in the DB.
func TestAppeal_AdminReview_Resolve_UnbansUser(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	uid := helper.UID()
	_, _, adminToken := h.RegisterAdmin("appeal_res_adm_" + uid)
	_, _, userToken := h.RegisterUserAndLogin("appeal_res_usr_" + uid)

	me := h.GetMe(userToken, http.StatusOK)
	require.NotNil(t, me.JSON200)
	userID := *me.JSON200.ID

	banUser(t, h, userID)

	created := h.CreateAppeal(ctx, userToken, "please unban", http.StatusCreated)
	require.NotNil(t, created.JSON201)

	adminResp := "appeal approved"
	reviewResp := h.AdminReviewAppeal(ctx, adminToken, *created.JSON201.ID, openapi.ReviewAppealRequestDecisionResolved, &adminResp, http.StatusOK)
	require.NotNil(t, reviewResp.JSON200)
	assert.Equal(t, "resolved", *reviewResp.JSON200.Decision)
	assert.Equal(t, adminResp, *reviewResp.JSON200.AdminResponse)

	// Verify via DB - resolve triggers Unban which revokes all JWTs,
	// so /auth/me with the old token is unreliable immediately after.
	var isBanned bool

	err := TestPool.QueryRow(ctx, "SELECT is_banned FROM users WHERE id = $1", userID).Scan(&isBanned)
	require.NoError(t, err)
	assert.False(t, isBanned)
}

// GET /auth/me: banned user with no prior appeals sees can_appeal=true, has_pending_appeal=false.
func TestAppeal_GetMe_BannedUser_CanAppeal(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	uid := helper.UID()
	_, _, userToken := h.RegisterUserAndLogin("appeal_me_can_" + uid)

	me := h.GetMe(userToken, http.StatusOK)
	require.NotNil(t, me.JSON200)
	banUser(t, h, *me.JSON200.ID)

	resp := h.GetMe(userToken, http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.BanStatus)
	require.NotNil(t, resp.JSON200.BanStatus.CanAppeal)
	require.NotNil(t, resp.JSON200.BanStatus.HasPendingAppeal)
	assert.True(t, *resp.JSON200.BanStatus.CanAppeal)
	assert.False(t, *resp.JSON200.BanStatus.HasPendingAppeal)
}

// GET /auth/me: after submitting an appeal, has_pending_appeal=true and can_appeal=false.
func TestAppeal_GetMe_BannedUser_HasPendingAppeal(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	uid := helper.UID()
	_, _, userToken := h.RegisterUserAndLogin("appeal_me_pend_" + uid)

	me := h.GetMe(userToken, http.StatusOK)
	require.NotNil(t, me.JSON200)
	banUser(t, h, *me.JSON200.ID)

	h.CreateAppeal(ctx, userToken, "please unban", http.StatusCreated)

	resp := h.GetMe(userToken, http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.BanStatus)
	require.NotNil(t, resp.JSON200.BanStatus.CanAppeal)
	require.NotNil(t, resp.JSON200.BanStatus.HasPendingAppeal)
	assert.False(t, *resp.JSON200.BanStatus.CanAppeal)
	assert.True(t, *resp.JSON200.BanStatus.HasPendingAppeal)
}

// PATCH /admin/appeals/{id}: rejected decision leaves the user banned.
func TestAppeal_AdminReview_Reject(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	uid := helper.UID()
	_, _, adminToken := h.RegisterAdmin("appeal_rej_adm_" + uid)
	_, _, userToken := h.RegisterUserAndLogin("appeal_rej_usr_" + uid)

	me := h.GetMe(userToken, http.StatusOK)
	require.NotNil(t, me.JSON200)
	userID := *me.JSON200.ID

	banUser(t, h, userID)

	created := h.CreateAppeal(ctx, userToken, "please unban", http.StatusCreated)
	require.NotNil(t, created.JSON201)

	adminResp := "appeal denied"
	reviewResp := h.AdminReviewAppeal(ctx, adminToken, *created.JSON201.ID, openapi.ReviewAppealRequestDecisionRejected, &adminResp, http.StatusOK)
	require.NotNil(t, reviewResp.JSON200)
	assert.Equal(t, "rejected", *reviewResp.JSON200.Decision)

	var isBanned bool

	err := TestPool.QueryRow(ctx, "SELECT is_banned FROM users WHERE id = $1", userID).Scan(&isBanned)
	require.NoError(t, err)
	assert.True(t, isBanned)
}

// PATCH /admin/appeals/{id}: non-admin user gets 403 Forbidden.
func TestAppeal_AdminReview_NonAdmin_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	uid := helper.UID()
	_, _, regularToken := h.RegisterUserAndLogin("appeal_nadm_reg_" + uid)
	_, _, userToken := h.RegisterUserAndLogin("appeal_nadm_usr_" + uid)

	me := h.GetMe(userToken, http.StatusOK)
	require.NotNil(t, me.JSON200)
	banUser(t, h, *me.JSON200.ID)

	created := h.CreateAppeal(ctx, userToken, "please unban", http.StatusCreated)
	require.NotNil(t, created.JSON201)

	h.AdminReviewAppeal(ctx, regularToken, *created.JSON201.ID, openapi.ReviewAppealRequestDecisionResolved, nil, http.StatusForbidden)
}

// PATCH /admin/appeals/{id}: unknown appeal ID returns 404 Not Found.
func TestAppeal_AdminReview_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	uid := helper.UID()
	_, _, adminToken := h.RegisterAdmin("appeal_nf_adm_" + uid)

	h.AdminReviewAppeal(ctx, adminToken, uuid.New().String(), openapi.ReviewAppealRequestDecisionResolved, nil, http.StatusNotFound)
}

// PATCH /admin/appeals/{id}: reviewing an already-reviewed appeal returns 403 Forbidden.
func TestAppeal_AdminReview_AlreadyReviewed(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	uid := helper.UID()
	_, _, adminToken := h.RegisterAdmin("appeal_ar_adm_" + uid)
	_, _, userToken := h.RegisterUserAndLogin("appeal_ar_usr_" + uid)

	me := h.GetMe(userToken, http.StatusOK)
	require.NotNil(t, me.JSON200)
	banUser(t, h, *me.JSON200.ID)

	created := h.CreateAppeal(ctx, userToken, "first appeal", http.StatusCreated)
	require.NotNil(t, created.JSON201)
	appealID := *created.JSON201.ID

	h.AdminReviewAppeal(ctx, adminToken, appealID, openapi.ReviewAppealRequestDecisionRejected, nil, http.StatusOK)
	h.AdminReviewAppeal(ctx, adminToken, appealID, openapi.ReviewAppealRequestDecisionResolved, nil, http.StatusForbidden)
}

// POST /appeals + GET /appeals/me: unauthenticated requests return 401 Unauthorized.
func TestAppeal_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	resp, err := h.Client().PostAppealsWithResponse(ctx, openapi.PostAppealsJSONRequestBody{Message: "test"})
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode())

	meResp, err := h.Client().GetAppealsMeWithResponse(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, meResp.StatusCode())
}
