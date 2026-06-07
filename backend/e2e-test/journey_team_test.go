package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestE2E_TeamInviteLifecycle(t *testing.T) {
	s := newE2ESuite(t)

	captain := s.registerUser("team_captain")
	member := s.registerUser("team_member")
	s.createTeam(&captain, "Team "+e2eUID("invite"))

	invite := s.inviteToken(captain)
	confirm := false
	join, err := s.client.PostTeamsJoinWithResponse(context.Background(), openapi.PostTeamsJoinJSONRequestBody{
		InviteToken:  invite,
		ConfirmReset: &confirm,
	}, e2eBearer(member.Token))
	require.NoError(t, err)
	requireStatus(t, "join team", http.StatusOK, join.StatusCode(), join.Body)

	me, err := s.client.GetTeamsMyWithResponse(context.Background(), e2eBearer(member.Token))
	require.NoError(t, err)
	requireStatus(t, "member team", http.StatusOK, me.StatusCode(), me.Body)
	require.NotNil(t, me.JSON200)
	require.NotNil(t, me.JSON200.ID)
	require.Equal(t, captain.TeamID, *me.JSON200.ID)
}
