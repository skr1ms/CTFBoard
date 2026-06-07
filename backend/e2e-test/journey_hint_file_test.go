package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestE2E_HintAndFileJourney(t *testing.T) {
	s := newE2ESuite(t)

	admin := s.registerAdmin("asset_admin")
	player := s.registerUser("asset_player")
	s.createSoloTeam(&player)

	challengeID := s.createChallenge(admin, "Assets "+e2eUID("assets"), "flag{assets_ok}", 200)
	hintID := s.createHint(admin, challengeID, "read the attached file", 50)
	s.createAward(admin, player.TeamID, 1_000)

	uploadContent := "task attachment content"
	fileID := s.uploadChallengeFile(admin, challengeID, "task.txt", uploadContent)

	unlock, err := s.client.PostChallengesChallengeIDHintsHintIDUnlockWithResponse(context.Background(), challengeID, hintID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "unlock hint", http.StatusOK, unlock.StatusCode(), unlock.Body)
	require.NotNil(t, unlock.JSON200)

	files, err := s.client.GetChallengesChallengeIDFilesWithResponse(context.Background(), challengeID, nil, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "list challenge files", http.StatusOK, files.StatusCode(), files.Body)
	require.NotNil(t, files.JSON200)
	require.Len(t, *files.JSON200, 1)

	require.Equal(t, uploadContent, s.downloadFile(player, fileID))
	s.submitFlag(player, challengeID, "flag{assets_ok}", true, http.StatusOK)
	s.requireTeamScore(admin.Token, player.TeamName, 1_150)
}
