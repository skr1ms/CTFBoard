package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// GET /notifications + GET /user/notifications: global and user notifications are visible.
func TestNotification_List_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_notifs_list")

	suffix := uuid.New().String()[:8]
	title := "Notif " + suffix

	createResp := h.CreateNotification(tokenAdmin, title, "content", "info", false, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	listResp := h.GetNotifications(1, 50, http.StatusOK)
	require.NotNil(t, listResp.JSON200)

	found := false
	for _, n := range *listResp.JSON200 {
		if n.Title != nil && *n.Title == title {
			found = true
			break
		}
	}
	require.True(t, found, "created global notification must be in /notifications list")

	email, _, tokenUser := h.RegisterUserAndLogin("notif_user_" + suffix)
	userID := h.GetUserIDByEmail(email)

	h.CreateUserNotification(tokenAdmin, userID, "UserNotif "+suffix, "hello", "info", http.StatusCreated)

	userList := h.GetUserNotifications(tokenUser, 1, 50, http.StatusOK)
	require.NotNil(t, userList.JSON200)
	require.GreaterOrEqual(t, len(*userList.JSON200), 1)
}

// POST /admin/notifications: non-admin gets 403.
func TestNotification_Create_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("notif_forbid_" + suffix)

	h.CreateNotification(tokenUser, "x", "x", "info", false, http.StatusForbidden)
}

// PATCH /user/notifications/{id}/read: user marks own notification read.
func TestNotification_MarkRead_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_notif_read")
	suffix := uuid.New().String()[:8]
	email, _, tokenUser := h.RegisterUserAndLogin("notif_read_user_" + suffix)
	userID := h.GetUserIDByEmail(email)
	createResp := h.CreateUserNotification(tokenAdmin, userID, "Personal "+suffix, "content", "info", http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	h.MarkUserNotificationRead(tokenUser, *createResp.JSON201.ID, http.StatusOK)
}

// PUT /admin/notifications/{id}: non-admin gets 403.
func TestNotification_Update_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_notif_upd_f")
	suffix := uuid.New().String()[:8]
	createResp := h.CreateNotification(tokenAdmin, "N "+suffix, "content", "info", false, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	_, _, tokenUser := h.RegisterUserAndLogin("notif_upd_user_" + suffix)

	h.UpdateNotification(tokenUser, *createResp.JSON201.ID, "X", "X", "info", false, http.StatusForbidden)
}

// DELETE /admin/notifications/{id}: non-admin gets 403.
func TestNotification_Delete_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_notif_del_f")
	suffix := uuid.New().String()[:8]
	createResp := h.CreateNotification(tokenAdmin, "N "+suffix, "content", "info", false, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	_, _, tokenUser := h.RegisterUserAndLogin("notif_del_user_" + suffix)

	h.DeleteNotification(tokenUser, *createResp.JSON201.ID, http.StatusForbidden)
}
