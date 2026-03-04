package v1

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// Get global notifications
// (GET /notifications)
func (h *Server) GetNotifications(w http.ResponseWriter, r *http.Request, params openapi.GetNotificationsParams) {
	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)
	notifs, err := h.admin.NotifUC.GetGlobal(r.Context(), page, perPage)
	if h.OnError(w, r, err, "GetNotifications", "GetGlobal") {
		return
	}
	helper.RenderOK(w, r, response.FromNotificationList(notifs))
}

// Get user notifications
// (GET /user/notifications)
func (h *Server) GetUserNotifications(w http.ResponseWriter, r *http.Request, params openapi.GetUserNotificationsParams) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	userID := user.ID

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	userNotifs, err := h.admin.NotifUC.GetUserNotifications(r.Context(), userID, page, perPage)
	if h.OnError(w, r, err, "GetUserNotifications", "GetUserNotifications") {
		return
	}
	helper.RenderOK(w, r, response.FromUserNotificationList(userNotifs))
}

// Mark notification as read
// (PATCH /user/notifications/{ID}/read)
func (h *Server) PatchUserNotificationsIDRead(w http.ResponseWriter, r *http.Request, ID string) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	userID := user.ID

	notifIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.admin.NotifUC.MarkAsRead(r.Context(), notifIDParsed, userID), "PatchUserNotificationsIDRead", "MarkAsRead") {
		return
	}

	helper.RenderOK(w, r, response.Message("marked as read"))
}

// Create global notification
// (POST /admin/notifications)
func (h *Server) PostAdminNotifications(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.CreateNotificationRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAdminNotifications",
	)
	if !ok {
		return
	}

	title, content, notifType, isPinned := request.CreateNotificationRequestToParams(&req)
	notif, err := h.admin.NotifUC.CreateGlobal(r.Context(), title, content, notifType, isPinned)
	if h.OnError(w, r, err, "PostAdminNotifications", "CreateGlobal") {
		return
	}

	helper.RenderCreated(w, r, response.FromNotification(notif))
}

// Create personal notification
// (POST /admin/notifications/user/{userID})
func (h *Server) PostAdminNotificationsUserUserID(w http.ResponseWriter, r *http.Request, userIDString string) {
	userIDParsed, ok := helper.ParseUUID(w, r, userIDString)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.CreateUserNotificationRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAdminNotificationsUserUserID",
	)
	if !ok {
		return
	}

	title, content, notifType := request.CreateUserNotificationRequestToParams(&req)
	userNotif, err := h.admin.NotifUC.CreatePersonal(r.Context(), userIDParsed, title, content, notifType)
	if h.OnError(w, r, err, "PostAdminNotificationsUserUserID", "CreatePersonal") {
		return
	}

	helper.RenderCreated(w, r, response.FromUserNotification(userNotif))
}

// Update notification
// (PUT /admin/notifications/{ID})
func (h *Server) PutAdminNotificationsID(w http.ResponseWriter, r *http.Request, ID string) {
	notifIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.UpdateNotificationRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PutAdminNotificationsID",
	)
	if !ok {
		return
	}

	title, content, notifType, isPinned := request.UpdateNotificationRequestToParams(&req)
	notif, err := h.admin.NotifUC.Update(r.Context(), notifIDParsed, title, content, notifType, isPinned)
	if h.OnError(w, r, err, "PutAdminNotificationsID", "Update") {
		return
	}

	helper.RenderOK(w, r, response.FromNotification(notif))
}

// Delete notification
// (DELETE /admin/notifications/{ID})
func (h *Server) DeleteAdminNotificationsID(w http.ResponseWriter, r *http.Request, ID string) {
	notifIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.admin.NotifUC.Delete(r.Context(), notifIDParsed), "DeleteAdminNotificationsID", "Delete") {
		return
	}

	helper.RenderNoContent(w, r)
}
