package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestE2E_ChallengeCommentsBlockedBeforeCompetitionEnd(t *testing.T) {
	s := newE2ESuite(t)

	admin := s.registerAdmin("comments_before_end_admin")
	player := s.registerUser("comments_before_end_player")
	s.createTeam(&player, e2eUID("comments_before_end_team"))
	challengeID := s.createChallenge(admin, e2eUID("comments_before_end_chal"), "flag{comments_before_end}", 100)

	list, err := s.client.GetChallengesChallengeIDCommentsWithResponse(context.Background(), challengeID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "list comments before competition end", http.StatusForbidden, list.StatusCode(), list.Body)

	create, err := s.client.PostChallengesChallengeIDCommentsWithResponse(context.Background(), challengeID, openapi.PostChallengesChallengeIDCommentsJSONRequestBody{
		Content: "blocked before competition end",
	}, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "create comment before competition end", http.StatusForbidden, create.StatusCode(), create.Body)
}
