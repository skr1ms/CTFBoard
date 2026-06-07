package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestE2E_BanAppealRestoreFlow(t *testing.T) {
	s := newE2ESuite(t)

	admin := s.registerAdmin("appeal_admin")
	player := s.registerUser("appeal_player")
	s.createSoloTeam(&player)
	challengeID := s.createChallenge(admin, "Appeal challenge "+e2eUID("appeal"), "flag{appeal_ok}", 100)

	s.banUserDirectly(player.UserID)

	s.submitFlag(player, challengeID, "flag{appeal_ok}", false, http.StatusForbidden)

	appeal, err := s.client.PostAppealsWithResponse(context.Background(), openapi.PostAppealsJSONRequestBody{
		Message: "please review this automated e2e ban appeal",
	}, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "create appeal", http.StatusCreated, appeal.StatusCode(), appeal.Body)
	require.NotNil(t, appeal.JSON201)
	require.NotNil(t, appeal.JSON201.ID)

	adminResponse := "approved by e2e"
	review, err := s.client.PatchAdminAppealsIDWithResponse(context.Background(), *appeal.JSON201.ID, openapi.PatchAdminAppealsIDJSONRequestBody{
		Decision:      openapi.ReviewAppealRequestDecisionResolved,
		AdminResponse: &adminResponse,
	}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "review appeal", http.StatusOK, review.StatusCode(), review.Body)

	var isBanned bool

	err = TestPool.QueryRow(context.Background(), "SELECT is_banned FROM users WHERE id = $1", player.UserID).Scan(&isBanned)
	require.NoError(t, err)
	require.False(t, isBanned)
}
