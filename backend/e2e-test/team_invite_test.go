package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// POST /teams/me/invite: captain can regenerate invite token; new token is returned.
func TestTeamInvite_Regenerate_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _ = h.SetupCompetition("inv_ok_" + suffix)
	_, _, tokenCap := h.RegisterUserAndLogin("inv_cap_" + suffix)
	h.CreateTeam(tokenCap, "InvTeam_"+suffix, http.StatusCreated)

	teamBefore := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, teamBefore.JSON200)
	require.NotNil(t, teamBefore.JSON200.InviteToken)
	oldToken := *teamBefore.JSON200.InviteToken

	resp, err := h.Client().PostTeamsMeInviteWithResponse(context.Background(), helper.WithBearerToken(tokenCap))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode(), "regenerate invite should succeed: %s", string(resp.Body))
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.InviteToken)
	require.NotEqual(t, oldToken, *resp.JSON200.InviteToken, "new invite token should differ from old one")
}

// POST /teams/me/invite: non-captain member cannot regenerate invite token.
func TestTeamInvite_Regenerate_NotCaptain_Error(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _ = h.SetupCompetition("inv_nocap_" + suffix)

	_, _, tokenCap := h.RegisterUserAndLogin("inv_cap2_" + suffix)
	h.CreateTeam(tokenCap, "InvTeam2_"+suffix, http.StatusCreated)

	teamResp := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, teamResp.JSON200)
	require.NotNil(t, teamResp.JSON200.InviteToken)
	inviteToken := *teamResp.JSON200.InviteToken

	_, _, tokenMember := h.RegisterUserAndLogin("inv_mem_" + suffix)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	resp, err := h.Client().PostTeamsMeInviteWithResponse(context.Background(), helper.WithBearerToken(tokenMember))
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode(), "non-captain should get 403: %s", string(resp.Body))
}

// POST /teams/me/invite: old invite code is invalidated after regeneration.
func TestTeamInvite_OldCodeInvalid(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _ = h.SetupCompetition("inv_old_" + suffix)

	_, _, tokenCap := h.RegisterUserAndLogin("inv_cap3_" + suffix)
	h.CreateTeam(tokenCap, "InvTeam3_"+suffix, http.StatusCreated)

	teamBefore := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, teamBefore.JSON200)
	require.NotNil(t, teamBefore.JSON200.InviteToken)
	oldToken := *teamBefore.JSON200.InviteToken

	resp, err := h.Client().PostTeamsMeInviteWithResponse(context.Background(), helper.WithBearerToken(tokenCap))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())

	_, _, tokenNewUser := h.RegisterUserAndLogin("inv_new_" + suffix)
	h.JoinTeam(tokenNewUser, oldToken, false, http.StatusNotFound)
}
