package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// POST /challenges/{id}/submit: when rate limit is exceeded, a submission with type=ratelimited is stored for audit.
func TestSubmission_RateLimitExceeded_StoresRatelimitedSubmission(t *testing.T) {
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_ratelimit_audit")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Rate Limit Audit Challenge", "FLAG{ratelimit_audit}", 100)

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("ratelimit_usr_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.PutAdminSettings(tokenAdmin, map[string]any{
		"submit_limit_per_user":     1,
		"submit_limit_duration_min": 1,
	}, http.StatusOK)
	t.Cleanup(resetAppSettings)
	require.Eventually(t, func() bool {
		resp, err := h.Client().GetAdminSettingsWithResponse(context.Background(), helper.WithBearerToken(tokenAdmin))

		return err == nil && resp != nil && resp.StatusCode() == http.StatusOK &&
			resp.JSON200 != nil && resp.JSON200.SubmitLimitPerUser != nil && *resp.JSON200.SubmitLimitPerUser == 1
	}, 2*time.Second, 50*time.Millisecond)

	getAfterPut := h.GetAdminSettings(tokenAdmin)
	require.NotNil(t, getAfterPut.JSON200)
	require.NotNil(t, getAfterPut.JSON200.SubmitLimitPerUser)
	require.Equal(t, 1, *getAfterPut.JSON200.SubmitLimitPerUser)

	require.Eventually(t, func() bool {
		resp := h.SubmitFlagExpectStatus(tokenUser, challengeID, "FLAG{wrong}", http.StatusOK, http.StatusTooManyRequests)

		return resp != nil && resp.StatusCode() == http.StatusTooManyRequests
	}, 10*time.Second, 200*time.Millisecond)

	ctx := context.Background()
	cID, err := uuid.Parse(challengeID)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		var n int

		err := TestPool.QueryRow(ctx,
			"SELECT COUNT(*) FROM submissions WHERE challenge_id = $1 AND submission_type = $2",
			cID, domain.SubmissionTypeRatelimited,
		).Scan(&n)

		return err == nil && n >= 1
	}, 5*time.Second, 25*time.Millisecond, "ratelimited audit row should appear after async LogSubmission")
}

// GET /admin/submissions/challenge/{challengeID}: submissions exist after a wrong flag submit.
func TestSubmission_AdminListByChallenge_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_subs_ok")

	challengeID := h.CreateBasicChallenge(tokenAdmin, "Sub Challenge", "FLAG{sub}", 100)

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("sub_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.SubmitFlag(tokenUser, challengeID, "FLAG{wrong}", http.StatusOK)

	listResp := h.GetAdminSubmissionsByChallenge(tokenAdmin, challengeID, 1, 50, http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	require.NotNil(t, listResp.JSON200.Data)
	require.GreaterOrEqual(t, len(*listResp.JSON200.Data), 1)

	statsResp := h.GetAdminSubmissionStatsByChallenge(tokenAdmin, challengeID, http.StatusOK)
	require.NotNil(t, statsResp.JSON200)
	require.NotNil(t, statsResp.JSON200.Total)
	require.GreaterOrEqual(t, *statsResp.JSON200.Total, 1)
}

// GET /admin/submissions/challenge/{challengeID}/stats: admin gets submission stats for challenge; returns 200 (total, etc.)
func TestSubmission_AdminStatsByChallenge_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_subs_stats")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Stats Challenge", "FLAG{stats}", 100)

	statsResp := h.GetAdminSubmissionStatsByChallenge(tokenAdmin, challengeID, http.StatusOK)
	require.NotNil(t, statsResp.JSON200)
	require.NotNil(t, statsResp.JSON200.Total)
	require.GreaterOrEqual(t, *statsResp.JSON200.Total, 0)
}

// GET /admin/submissions: non-admin gets 403.
func TestSubmission_AdminList_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("sub_forbid_" + suffix)

	h.GetAdminSubmissions(tokenUser, 1, 50, http.StatusForbidden)
}

// GET /admin/submissions: admin gets list (may be empty).
func TestSubmission_AdminList_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_subs_list")
	h.GetAdminSubmissions(tokenAdmin, 1, 50, http.StatusOK)
}

// GET /admin/submissions/challenge/{id}: non-admin gets 403.
func TestSubmission_AdminListByChallenge_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_subs_ch")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Ch", "FLAG{x}", 100)
	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("sub_ch_forbid_" + suffix)

	h.GetAdminSubmissionsByChallenge(tokenUser, challengeID, 1, 50, http.StatusForbidden)
}

// GET /admin/submissions/user/{id}: admin gets list by user.
func TestSubmission_AdminListByUser_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_subs_user")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "ChUser", "FLAG{user}", 100)
	suffix := helper.UID()
	email, _, tokenUser := h.RegisterUserAndLogin("sub_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "FLAG{wrong}", http.StatusOK)
	userID := h.GetUserIDByEmail(email)

	listResp := h.GetAdminSubmissionsByUser(tokenAdmin, userID, 1, 50, http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	require.NotNil(t, listResp.JSON200.Data)
	require.GreaterOrEqual(t, len(*listResp.JSON200.Data), 1)
}

// GET /admin/submissions/team/{id}: non-admin gets 403.
func TestSubmission_AdminListByTeam_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_subs_team")
	_ = h.CreateBasicChallenge(tokenAdmin, "ChTeam", "FLAG{team}", 100)
	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("sub_team_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID

	h.GetAdminSubmissionsByTeam(tokenUser, teamID, 1, 50, http.StatusForbidden)
}

// POST /admin/submissions: admin creates a submission record.
func TestSubmission_AdminCreate_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_sub_create")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Create Sub Chall", "flag{create}", 100)

	suffix := helper.UID()
	email, _, tokenUser := h.RegisterUserAndLogin("sub_create_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID
	userID := h.GetUserIDByEmail(email)
	tid := openapi_types.UUID(uuid.MustParse(teamID))

	resp, err := h.Client().PostAdminSubmissionsWithResponse(context.Background(), openapi.AdminCreateSubmissionRequest{
		ChallengeID:   openapi_types.UUID(uuid.MustParse(challengeID)),
		UserID:        openapi_types.UUID(uuid.MustParse(userID)),
		TeamID:        &tid,
		SubmittedFlag: "flag{manual}",
		IsCorrect:     false,
	}, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusCreated, resp.StatusCode(), resp.Body, "admin create submission")
}

// POST /admin/submissions: invalid payload returns 400.
func TestSubmission_AdminCreate_InvalidPayload(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_sub_bad")

	resp, err := h.Client().PostAdminSubmissionsWithResponse(context.Background(), openapi.AdminCreateSubmissionRequest{
		ChallengeID:   openapi_types.UUID(uuid.Nil),
		UserID:        openapi_types.UUID(uuid.Nil),
		SubmittedFlag: "",
		IsCorrect:     false,
	}, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusBadRequest, resp.StatusCode(), resp.Body, "admin create submission bad payload")
}

// GET /admin/submissions/{ID}: admin gets submission by ID.
func TestSubmission_AdminGetByID_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_sub_get")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Get Sub Chall", "flag{get}", 100)

	suffix := helper.UID()
	email, _, tokenUser := h.RegisterUserAndLogin("sub_get_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID
	userID := h.GetUserIDByEmail(email)
	tid := openapi_types.UUID(uuid.MustParse(teamID))

	createResp, err := h.Client().PostAdminSubmissionsWithResponse(context.Background(), openapi.AdminCreateSubmissionRequest{
		ChallengeID:   openapi_types.UUID(uuid.MustParse(challengeID)),
		UserID:        openapi_types.UUID(uuid.MustParse(userID)),
		TeamID:        &tid,
		SubmittedFlag: "flag{manual_get}",
		IsCorrect:     false,
	}, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode())
	require.NotNil(t, createResp.JSON201)
	submissionID := createResp.JSON201.ID

	getResp, err := h.Client().GetAdminSubmissionsIDWithResponse(context.Background(), *submissionID, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, getResp.StatusCode(), getResp.Body, "admin get submission by id")
	require.NotNil(t, getResp.JSON200)
	require.Equal(t, *submissionID, *getResp.JSON200.ID)
}

// GET /admin/submissions/{ID}: not found returns 404.
func TestSubmission_AdminGetByID_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_sub_get_404")

	resp, err := h.Client().GetAdminSubmissionsIDWithResponse(context.Background(), uuid.New().String(), helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNotFound, resp.StatusCode(), resp.Body, "admin get submission not found")
}

// PATCH /admin/submissions/{ID}: admin updates isCorrect flag.
func TestSubmission_AdminUpdate_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_sub_patch")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Patch Sub Chall", "flag{patch}", 100)

	suffix := helper.UID()
	email, _, tokenUser := h.RegisterUserAndLogin("sub_patch_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID
	userID := h.GetUserIDByEmail(email)
	tid := openapi_types.UUID(uuid.MustParse(teamID))

	createResp, err := h.Client().PostAdminSubmissionsWithResponse(context.Background(), openapi.AdminCreateSubmissionRequest{
		ChallengeID:   openapi_types.UUID(uuid.MustParse(challengeID)),
		UserID:        openapi_types.UUID(uuid.MustParse(userID)),
		TeamID:        &tid,
		SubmittedFlag: "flag{manual_patch}",
		IsCorrect:     false,
	}, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode())
	submissionID := createResp.JSON201.ID

	isCorrect := true
	patchResp, err := h.Client().PatchAdminSubmissionsIDWithResponse(context.Background(), *submissionID, openapi.AdminUpdateSubmissionRequest{
		IsCorrect: isCorrect,
	}, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, patchResp.StatusCode(), patchResp.Body, "admin patch submission")
}

// PATCH /admin/submissions/{ID}: non-admin gets 403.
func TestSubmission_AdminUpdate_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_sub_patch_f")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Patch Forbid Chall", "flag{pf}", 100)

	suffix := helper.UID()
	email, _, tokenUser := h.RegisterUserAndLogin("sub_patch_f_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID
	userID := h.GetUserIDByEmail(email)
	tid := openapi_types.UUID(uuid.MustParse(teamID))

	createResp, err := h.Client().PostAdminSubmissionsWithResponse(context.Background(), openapi.AdminCreateSubmissionRequest{
		ChallengeID:   openapi_types.UUID(uuid.MustParse(challengeID)),
		UserID:        openapi_types.UUID(uuid.MustParse(userID)),
		TeamID:        &tid,
		SubmittedFlag: "flag{pf_manual}",
		IsCorrect:     false,
	}, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode())
	submissionID := createResp.JSON201.ID

	isCorrect := true
	patchResp, err := h.Client().PatchAdminSubmissionsIDWithResponse(context.Background(), *submissionID, openapi.AdminUpdateSubmissionRequest{
		IsCorrect: isCorrect,
	}, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, patchResp.StatusCode(), patchResp.Body, "admin patch submission forbidden")
}

// DELETE /admin/submissions/{ID}: admin deletes submission.
func TestSubmission_AdminDelete_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_sub_del")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Del Sub Chall", "flag{del_sub}", 100)

	suffix := helper.UID()
	email, _, tokenUser := h.RegisterUserAndLogin("sub_del_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID
	userID := h.GetUserIDByEmail(email)
	tid := openapi_types.UUID(uuid.MustParse(teamID))

	createResp, err := h.Client().PostAdminSubmissionsWithResponse(context.Background(), openapi.AdminCreateSubmissionRequest{
		ChallengeID:   openapi_types.UUID(uuid.MustParse(challengeID)),
		UserID:        openapi_types.UUID(uuid.MustParse(userID)),
		TeamID:        &tid,
		SubmittedFlag: "flag{manual_del}",
		IsCorrect:     false,
	}, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode())
	submissionID := createResp.JSON201.ID

	delResp, err := h.Client().DeleteAdminSubmissionsIDWithResponse(context.Background(), *submissionID, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNoContent, delResp.StatusCode(), delResp.Body, "admin delete submission")
}

// DELETE /admin/submissions/{ID}: not found returns 204 (idempotent delete).
func TestSubmission_AdminDelete_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_sub_del_404")

	resp, err := h.Client().DeleteAdminSubmissionsIDWithResponse(context.Background(), uuid.New().String(), helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNoContent, resp.StatusCode(), resp.Body, "admin delete submission not found")
}

// GET /admin/submissions/team/{teamID}: admin lists team submissions.
func TestSubmission_AdminListByTeam_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_sub_team_ok")
	_ = h.CreateBasicChallenge(tokenAdmin, "Team Sub Chall", "flag{team_sub}", 100)

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("sub_team_ok_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID

	h.GetAdminSubmissionsByTeam(tokenAdmin, teamID, 1, 50, http.StatusOK)
}
