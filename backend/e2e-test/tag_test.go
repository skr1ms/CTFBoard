package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// GET /tags: returns created tags.
func TestTag_GetTags_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_tags_list")

	suffix := uuid.New().String()[:8]
	name := "tag_" + suffix
	color := "#00ff00"

	createResp := h.CreateTag(tokenAdmin, name, color, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	listResp := h.GetTags(http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	found := false
	for _, t := range *listResp.JSON200 {
		if t.Name != nil && *t.Name == name {
			found = true
			break
		}
	}
	require.True(t, found, "created tag must be in /tags list")
}

// POST /admin/tags: non-admin gets 403.
func TestTag_Create_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("tag_user_" + suffix)

	h.CreateTag(tokenUser, "x_"+suffix, "#111111", http.StatusForbidden)
}

// PUT /admin/tags/{id}: admin updates tag.
func TestTag_Update_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_tag_upd")
	suffix := uuid.New().String()[:8]
	createResp := h.CreateTag(tokenAdmin, "tag_"+suffix, "#000000", http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	h.UpdateTag(tokenAdmin, *createResp.JSON201.ID, "tag_updated_"+suffix, "#ff0000", http.StatusOK)
	listResp := h.GetTags(http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	found := false
	for _, tg := range *listResp.JSON200 {
		if tg.Name != nil && *tg.Name == "tag_updated_"+suffix {
			found = true
			break
		}
	}
	require.True(t, found)
}

// PUT /admin/tags/{id}: non-admin gets 403.
func TestTag_Update_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_tag_upd_f")
	suffix := uuid.New().String()[:8]
	createResp := h.CreateTag(tokenAdmin, "tag_"+suffix, "#000", http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	_, _, tokenUser := h.RegisterUserAndLogin("tag_upd_user_" + suffix)

	h.UpdateTag(tokenUser, *createResp.JSON201.ID, "x", "#fff", http.StatusForbidden)
}

// DELETE /admin/tags/{id}: admin deletes tag.
func TestTag_Delete_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_tag_del")
	suffix := uuid.New().String()[:8]
	createResp := h.CreateTag(tokenAdmin, "tag_del_"+suffix, "#000", http.StatusCreated)
	require.NotNil(t, createResp.JSON201)

	h.DeleteTag(tokenAdmin, *createResp.JSON201.ID, http.StatusNoContent)
}

// DELETE /admin/tags/{id}: non-admin gets 403.
func TestTag_Delete_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_tag_del_f")
	suffix := uuid.New().String()[:8]
	createResp := h.CreateTag(tokenAdmin, "tag_"+suffix, "#000", http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	_, _, tokenUser := h.RegisterUserAndLogin("tag_del_user_" + suffix)

	h.DeleteTag(tokenUser, *createResp.JSON201.ID, http.StatusForbidden)
}
