package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestE2E_SettingsScoreVisibility(t *testing.T) {
	s := newE2ESuite(t)

	defer resetAppSettingsFull()

	admin := s.registerAdmin("settings_admin")
	player := s.registerUser("settings_player")

	s.setScoreVisibility(admin, "public")
	publicResp, err := s.client.GetScoreboardWithResponse(context.Background(), &openapi.GetScoreboardParams{})
	require.NoError(t, err)
	requireStatus(t, "public scoreboard", http.StatusOK, publicResp.StatusCode(), publicResp.Body)

	s.setScoreVisibility(admin, "hidden")
	hiddenUser, err := s.client.GetScoreboardWithResponse(context.Background(), &openapi.GetScoreboardParams{}, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "hidden scoreboard user", http.StatusNotFound, hiddenUser.StatusCode(), hiddenUser.Body)

	hiddenAdmin, err := s.client.GetScoreboardWithResponse(context.Background(), &openapi.GetScoreboardParams{}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "hidden scoreboard admin", http.StatusOK, hiddenAdmin.StatusCode(), hiddenAdmin.Body)
}
