package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestE2E_NotificationLifecycle_PersonalTeamAndMissingTargets(t *testing.T) {
	s := newE2ESuite(t)

	suffix := e2eUID("notifications")
	admin := s.registerAdmin("notif_admin")
	player := s.registerUser("notif_player")
	other := s.registerUser("notif_other")
	teamCaptain := s.registerUser("notif_captain")
	s.createTeam(&teamCaptain, "Notify "+suffix)

	notifType := openapi.CreateUserNotificationRequestTypeInfo
	personal, err := s.client.PostAdminNotificationsUserUserIDWithResponse(context.Background(), player.UserID, openapi.PostAdminNotificationsUserUserIDJSONRequestBody{
		Title:   "Personal " + suffix,
		Content: "personal notification body",
		Type:    &notifType,
	}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "create personal notification", http.StatusCreated, personal.StatusCode(), personal.Body)
	require.NotNil(t, personal.JSON201)
	require.NotNil(t, personal.JSON201.ID)

	missingUser, err := s.client.PostAdminNotificationsUserUserIDWithResponse(context.Background(), uuid.NewString(), openapi.PostAdminNotificationsUserUserIDJSONRequestBody{
		Title:   "Missing " + suffix,
		Content: "must return not found",
		Type:    &notifType,
	}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "create personal notification for missing user", http.StatusNotFound, missingUser.StatusCode(), missingUser.Body)

	teamNotif, err := s.client.PostAdminNotificationsTeamTeamIDWithResponse(context.Background(), teamCaptain.TeamID, openapi.PostAdminNotificationsTeamTeamIDJSONRequestBody{
		Title:   "Team " + suffix,
		Content: "team notification body",
		Type:    &notifType,
	}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "create team notification", http.StatusCreated, teamNotif.StatusCode(), teamNotif.Body)
	require.NotNil(t, teamNotif.JSON201)
	require.NotNil(t, teamNotif.JSON201.CreatedCount)
	require.Equal(t, 1, *teamNotif.JSON201.CreatedCount)

	playerNotifications, err := s.client.GetUserNotificationsWithResponse(context.Background(), nil, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "list player notifications", http.StatusOK, playerNotifications.StatusCode(), playerNotifications.Body)
	require.NotNil(t, playerNotifications.JSON200)
	require.NotEmpty(t, *playerNotifications.JSON200)

	personalID := *personal.JSON201.ID
	require.Contains(t, userNotificationIDs(*playerNotifications.JSON200), personalID)

	unread, err := s.client.GetUserNotificationsUnreadCountWithResponse(context.Background(), e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "count unread notifications", http.StatusOK, unread.StatusCode(), unread.Body)
	require.NotNil(t, unread.JSON200)
	require.NotNil(t, unread.JSON200.UnreadCount)
	require.GreaterOrEqual(t, *unread.JSON200.UnreadCount, 1)

	foreignRead, err := s.client.PatchUserNotificationsIDReadWithResponse(context.Background(), personalID, e2eBearer(other.Token))
	require.NoError(t, err)
	requireStatus(t, "read foreign notification", http.StatusNotFound, foreignRead.StatusCode(), foreignRead.Body)

	read, err := s.client.PatchUserNotificationsIDReadWithResponse(context.Background(), personalID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "read own notification", http.StatusOK, read.StatusCode(), read.Body)

	teamNotifications, err := s.client.GetUserNotificationsWithResponse(context.Background(), nil, e2eBearer(teamCaptain.Token))
	require.NoError(t, err)
	requireStatus(t, "list team notifications", http.StatusOK, teamNotifications.StatusCode(), teamNotifications.Body)
	require.NotNil(t, teamNotifications.JSON200)
	require.Contains(t, userNotificationTitles(*teamNotifications.JSON200), "Team "+suffix)

	deleteMissing, err := s.client.DeleteAdminNotificationsIDWithResponse(context.Background(), uuid.NewString(), e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "delete missing notification", http.StatusNotFound, deleteMissing.StatusCode(), deleteMissing.Body)
}

func userNotificationIDs(items []openapi.UserNotificationResponse) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item.ID != nil {
			out = append(out, *item.ID)
		}
	}

	return out
}

func userNotificationTitles(items []openapi.UserNotificationResponse) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item.Title != nil {
			out = append(out, *item.Title)
		}
	}

	return out
}
