package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /notifications).
func (h *Server) GetNotifications(w http.ResponseWriter, r *http.Request, params openapi.GetNotificationsParams) {
	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	notifs, err := h.admin.NotifUC.GetGlobal(r.Context(), page, perPage)
	if h.OnError(w, r, err, "GetNotifications", "GetGlobal") {
		return
	}

	setPublicCache(w, cacheMedium, false)
	httputil.RenderOK(w, r, response.FromNotificationList(notifs))
}

// (GET /user/notifications).
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

	httputil.RenderOK(w, r, response.FromUserNotificationList(userNotifs))
}

// (PATCH /user/notifications/{ID}/read).
func (h *Server) PatchUserNotificationsIDRead(w http.ResponseWriter, r *http.Request, ID string) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	userID := user.ID

	notifIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.admin.NotifUC.MarkAsRead(r.Context(), notifIDParsed, userID), "PatchUserNotificationsIDRead", "MarkAsRead") {
		return
	}

	httputil.RenderOK(w, r, response.Message("marked as read"))
}

// (POST /admin/notifications).
func (h *Server) PostAdminNotifications(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.CreateNotificationRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	params, err := request.CreateNotificationRequestToParams(&req)
	if h.OnError(w, r, err, "PostAdminNotifications", "CreateNotificationRequestToParams") {
		return
	}

	notif, err := h.admin.NotifUC.CreateGlobal(r.Context(), params)
	if h.OnError(w, r, err, "PostAdminNotifications", "CreateGlobal") {
		return
	}

	httputil.RenderCreated(w, r, response.FromNotification(notif))
}

// (POST /admin/notifications/user/{userID}).
func (h *Server) PostAdminNotificationsUserUserID(w http.ResponseWriter, r *http.Request, userIDString string) {
	userIDParsed, ok := httputil.ParseUUID(w, r, userIDString)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.CreateUserNotificationRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	params, err := request.CreateUserNotificationRequestToParams(&req, userIDParsed)
	if h.OnError(w, r, err, "PostAdminNotificationsUserUserID", "CreateUserNotificationRequestToParams") {
		return
	}

	userNotif, err := h.admin.NotifUC.CreatePersonal(r.Context(), params)
	if h.OnError(w, r, err, "PostAdminNotificationsUserUserID", "CreatePersonal") {
		return
	}

	httputil.RenderCreated(w, r, response.FromUserNotification(userNotif))
}

// (PUT /admin/notifications/{ID}).
func (h *Server) PutAdminNotificationsID(w http.ResponseWriter, r *http.Request, ID string) {
	notifIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.UpdateNotificationRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	params, err := request.UpdateNotificationRequestToParams(&req, notifIDParsed)
	if h.OnError(w, r, err, "PutAdminNotificationsID", "UpdateNotificationRequestToParams") {
		return
	}

	notif, err := h.admin.NotifUC.Update(r.Context(), params)
	if h.OnError(w, r, err, "PutAdminNotificationsID", "Update") {
		return
	}

	httputil.RenderOK(w, r, response.FromNotification(notif))
}

// (DELETE /admin/notifications/{ID}).
func (h *Server) DeleteAdminNotificationsID(w http.ResponseWriter, r *http.Request, ID string) {
	notifIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.admin.NotifUC.Delete(r.Context(), notifIDParsed), "DeleteAdminNotificationsID", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}
