package helper

import (
	"context"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func (h *E2EHelper) GetNotifications(page, perPage, expectStatus int) *openapi.GetNotificationsResponse {
	h.t.Helper()
	params := &openapi.GetNotificationsParams{Page: &page, PerPage: &perPage}
	resp, err := h.client.GetNotificationsWithResponse(context.Background(), params)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get notifications")
	return resp
}

func (h *E2EHelper) CreateNotification(token, title, content, typ string, pinned bool, expectStatus int) *openapi.PostAdminNotificationsResponse {
	h.t.Helper()
	tp := openapi.RequestCreateNotificationRequestType(typ)
	resp, err := h.client.PostAdminNotificationsWithResponse(context.Background(), openapi.PostAdminNotificationsJSONRequestBody{
		Title:    title,
		Content:  content,
		Type:     &tp,
		IsPinned: &pinned,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create notification")
	return resp
}

func (h *E2EHelper) CreateUserNotification(token, userID, title, content, typ string, expectStatus int) *openapi.PostAdminNotificationsUserUserIDResponse {
	h.t.Helper()
	tp := openapi.RequestCreateUserNotificationRequestType(typ)
	resp, err := h.client.PostAdminNotificationsUserUserIDWithResponse(context.Background(), userID, openapi.PostAdminNotificationsUserUserIDJSONRequestBody{
		Title:   title,
		Content: content,
		Type:    &tp,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create user notification")
	return resp
}

func (h *E2EHelper) GetUserNotifications(token string, page, perPage, expectStatus int) *openapi.GetUserNotificationsResponse {
	h.t.Helper()
	params := &openapi.GetUserNotificationsParams{Page: &page, PerPage: &perPage}
	resp, err := h.client.GetUserNotificationsWithResponse(context.Background(), params, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get user notifications")
	return resp
}

func (h *E2EHelper) MarkUserNotificationRead(token, id string, expectStatus int) *openapi.PatchUserNotificationsIDReadResponse {
	h.t.Helper()
	resp, err := h.client.PatchUserNotificationsIDReadWithResponse(context.Background(), id, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "mark user notification read")
	return resp
}

func (h *E2EHelper) UpdateNotification(token, id, title, content, typ string, pinned bool, expectStatus int) *openapi.PutAdminNotificationsIDResponse {
	h.t.Helper()
	tp := openapi.RequestUpdateNotificationRequestType(typ)
	resp, err := h.client.PutAdminNotificationsIDWithResponse(context.Background(), id, openapi.PutAdminNotificationsIDJSONRequestBody{
		Title:    title,
		Content:  content,
		Type:     &tp,
		IsPinned: &pinned,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "update notification")
	return resp
}

func (h *E2EHelper) DeleteNotification(token, id string, expectStatus int) *openapi.DeleteAdminNotificationsIDResponse {
	h.t.Helper()
	resp, err := h.client.DeleteAdminNotificationsIDWithResponse(context.Background(), id, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete notification")
	return resp
}
