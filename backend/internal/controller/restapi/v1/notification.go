package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Get global notifications
// (GET /notifications)
func (h *Server) GetNotifications(w http.ResponseWriter, r *http.Request, params openapi.GetNotificationsParams) {
	page := 1
	if params.Page != nil {
		page = *params.Page
	}
	perPage := 20
	if params.PerPage != nil {
		perPage = *params.PerPage
	}

	notifs, err := h.admin.NotifUC.GetGlobal(r.Context(), page, perPage)
	if h.OnError(w, r, err, "GetNotifications", "GetGlobal") {
		return
	}

	res := make([]openapi.ResponseNotificationResponse, len(notifs))
	for i, n := range notifs {
		res[i] = response.FromNotification(n)
	}
	helper.RenderOK(w, r, res)
}

// Get user notifications
// (GET /user/notifications)
func (h *Server) GetUserNotifications(w http.ResponseWriter, r *http.Request, params openapi.GetUserNotificationsParams) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	userID := user.ID

	page := 1
	if params.Page != nil {
		page = *params.Page
	}
	perPage := 20
	if params.PerPage != nil {
		perPage = *params.PerPage
	}

	userNotifs, err := h.admin.NotifUC.GetUserNotifications(r.Context(), userID, page, perPage)
	if h.OnError(w, r, err, "GetUserNotifications", "GetUserNotifications") {
		return
	}

	res := make([]openapi.ResponseUserNotificationResponse, len(userNotifs))
	for i, un := range userNotifs {
		res[i] = response.FromUserNotification(un)
	}
	helper.RenderOK(w, r, res)
}

// Mark notification as read
// (PATCH /user/notifications/{ID}/read)
func (h *Server) PatchUserNotificationsIDRead(w http.ResponseWriter, r *http.Request, id string) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	userID := user.ID

	notifID, ok := helper.ParseUUID(w, r, id)
	if !ok {
		return
	}

	if h.OnError(w, r, h.admin.NotifUC.MarkAsRead(r.Context(), notifID, userID), "PatchUserNotificationsIDRead", "MarkAsRead") {
		return
	}

	helper.RenderOK(w, r, map[string]string{"message": "marked as read"})
}

// Create global notification
// (POST /admin/notifications)
func (h *Server) PostAdminNotifications(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.RequestCreateNotificationRequest](w, r, h.infra.Validator, h.infra.Logger, "PostAdminNotifications")
	if !ok {
		return
	}

	title := req.Title
	content := req.Content
	notifType := entity.NotificationInfo
	if req.Type != nil {
		notifType = entity.NotificationType(*req.Type)
	}
	isPinned := false
	if req.IsPinned != nil {
		isPinned = *req.IsPinned
	}

	notif, err := h.admin.NotifUC.CreateGlobal(r.Context(), title, content, notifType, isPinned)
	if h.OnError(w, r, err, "PostAdminNotifications", "CreateGlobal") {
		return
	}

	helper.RenderCreated(w, r, response.FromNotification(notif))
}

// Create personal notification
// (POST /admin/notifications/user/{userID})
func (h *Server) PostAdminNotificationsUserUserID(w http.ResponseWriter, r *http.Request, userIDStr string) {
	userID, ok := helper.ParseUUID(w, r, userIDStr)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.RequestCreateUserNotificationRequest](w, r, h.infra.Validator, h.infra.Logger, "PostAdminNotificationsUserUserID")
	if !ok {
		return
	}

	title := req.Title
	content := req.Content
	notifType := entity.NotificationInfo
	if req.Type != nil {
		notifType = entity.NotificationType(*req.Type)
	}

	userNotif, err := h.admin.NotifUC.CreatePersonal(r.Context(), userID, title, content, notifType)
	if h.OnError(w, r, err, "PostAdminNotificationsUserUserID", "CreatePersonal") {
		return
	}

	helper.RenderCreated(w, r, response.FromUserNotification(userNotif))
}

// Update notification
// (PUT /admin/notifications/{ID})
func (h *Server) PutAdminNotificationsID(w http.ResponseWriter, r *http.Request, id string) {
	notifID, ok := helper.ParseUUID(w, r, id)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.RequestUpdateNotificationRequest](w, r, h.infra.Validator, h.infra.Logger, "PutAdminNotificationsID")
	if !ok {
		return
	}

	title := req.Title
	content := req.Content
	notifType := entity.NotificationInfo
	if req.Type != nil {
		notifType = entity.NotificationType(*req.Type)
	}
	isPinned := false
	if req.IsPinned != nil {
		isPinned = *req.IsPinned
	}

	notif, err := h.admin.NotifUC.Update(r.Context(), notifID, title, content, notifType, isPinned)
	if h.OnError(w, r, err, "PutAdminNotificationsID", "Update") {
		return
	}

	helper.RenderOK(w, r, response.FromNotification(notif))
}

// Delete notification
// (DELETE /admin/notifications/{ID})
func (h *Server) DeleteAdminNotificationsID(w http.ResponseWriter, r *http.Request, id string) {
	notifID, ok := helper.ParseUUID(w, r, id)
	if !ok {
		return
	}

	if h.OnError(w, r, h.admin.NotifUC.Delete(r.Context(), notifID), "DeleteAdminNotificationsID", "Delete") {
		return
	}

	helper.RenderNoContent(w, r)
}
