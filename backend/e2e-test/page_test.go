package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// GET /admin/pages: admin gets list of all pages (including drafts).
func TestPage_AdminList_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_pages_list")
	suffix := helper.UID()
	h.CreatePage(tokenAdmin, "Title", "slug-list-"+suffix, "content", false, 0, http.StatusCreated)

	listResp := h.GetAdminPages(tokenAdmin, http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	require.GreaterOrEqual(t, len(*listResp.JSON200), 1)
}

// GET /admin/pages/{ID}: admin gets page by ID; returns 200 and page data.
func TestPage_AdminGetByID_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_pages_get_id")
	suffix := helper.UID()
	slug := "page-byid-" + suffix
	title := "Page By ID " + suffix
	createResp := h.CreatePage(tokenAdmin, title, slug, "body", false, 1, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	got := h.GetAdminPageByID(tokenAdmin, *createResp.JSON201.ID, http.StatusOK)
	require.NotNil(t, got.JSON200)
	require.Equal(t, *createResp.JSON201.ID, *got.JSON200.ID)
	require.NotNil(t, got.JSON200.Title)
	require.Equal(t, title, *got.JSON200.Title)
	require.NotNil(t, got.JSON200.Slug)
	require.Equal(t, slug, *got.JSON200.Slug)
}

// GET /pages/{slug}: returns created page.
func TestPage_GetBySlug_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_pages_slug")

	suffix := helper.UID()
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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.GetPageBySlug("missing-"+helper.UID(), http.StatusNotFound)
}

// POST /admin/pages: non-admin gets 403.
func TestPage_Create_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("page_user_" + suffix)

	h.CreatePage(tokenUser, "Title", "slug-"+suffix, "content", false, 0, http.StatusForbidden)
}

// PUT /admin/pages/{id}: admin updates page.
func TestPage_Update_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_page_upd")
	suffix := helper.UID()
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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_page_del")
	suffix := helper.UID()
	slug := "page-del-" + suffix
	createResp := h.CreatePage(tokenAdmin, "Title", slug, "content", false, 0, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)

	h.DeletePage(tokenAdmin, *createResp.JSON201.ID, http.StatusNoContent)
	h.GetPageBySlug(slug, http.StatusNotFound)
}
