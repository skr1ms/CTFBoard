package helper

import (
	"context"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

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
