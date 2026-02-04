package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// GET /pages/{slug}: returns created page.
func TestPage_GetBySlug_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_pages_slug")

	suffix := uuid.New().String()[:8]
	slug := "page-" + suffix
	title := "Title " + suffix

	createResp := h.CreatePage(tokenAdmin, title, slug, "content", false, 0, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	gotResp := h.GetPageBySlug(slug, http.StatusOK)
	require.NotNil(t, gotResp.JSON200)
	require.NotNil(t, gotResp.JSON200.Slug)
	require.Equal(t, slug, *gotResp.JSON200.Slug)
}

// GET /pages/{slug}: not found returns 404.
func TestPage_GetBySlug_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	h.GetPageBySlug("missing-"+uuid.New().String()[:8], http.StatusNotFound)
}

// POST /admin/pages: non-admin gets 403.
func TestPage_Create_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("page_user_" + suffix)

	h.CreatePage(tokenUser, "Title", "slug-"+suffix, "content", false, 0, http.StatusForbidden)
}

// PUT /admin/pages/{id}: admin updates page.
func TestPage_Update_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_page_upd")
	suffix := uuid.New().String()[:8]
	slug := "page-upd-" + suffix
	createResp := h.CreatePage(tokenAdmin, "Title", slug, "content", false, 0, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	h.UpdatePage(tokenAdmin, *createResp.JSON201.ID, "Title Updated", slug, "content updated", false, 0, http.StatusOK)
	got := h.GetPageBySlug(slug, http.StatusOK)
	require.NotNil(t, got.JSON200)
	require.Equal(t, "Title Updated", *got.JSON200.Title)
}

// DELETE /admin/pages/{id}: admin deletes page.
func TestPage_Delete_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_page_del")
	suffix := uuid.New().String()[:8]
	slug := "page-del-" + suffix
	createResp := h.CreatePage(tokenAdmin, "Title", slug, "content", false, 0, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)

	h.DeletePage(tokenAdmin, *createResp.JSON201.ID, http.StatusNoContent)
	h.GetPageBySlug(slug, http.StatusNotFound)
}
