package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// GET /fields: returns created fields for entity_type.
func TestField_GetFields_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_fields_list")

	suffix := helper.UID()
	name := "field_" + suffix

	createResp := h.CreateField(tokenAdmin, name, "text", "user", false, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	listResp := h.GetFields("user", http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	found := false
	for _, f := range *listResp.JSON200 {
		if f.Name != nil && *f.Name == name {
			found = true
			break
		}
	}
	require.True(t, found, "created field must be in /fields list")
}

// POST /admin/fields: non-admin gets 403.
func TestField_Create_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("field_user_" + suffix)

	h.CreateField(tokenUser, "x_"+suffix, "text", "user", false, http.StatusForbidden)
}

// PUT /admin/fields/{id}: admin updates field.
func TestField_Update_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_field_upd")
	suffix := helper.UID()
	createResp := h.CreateField(tokenAdmin, "field_"+suffix, "text", "user", false, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	h.UpdateField(tokenAdmin, *createResp.JSON201.ID, "field_updated_"+suffix, "text", true, http.StatusOK)
	listResp := h.GetFields("user", http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	found := false
	for _, f := range *listResp.JSON200 {
		if f.Name != nil && *f.Name == "field_updated_"+suffix {
			found = true
			break
		}
	}
	require.True(t, found)
}

// DELETE /admin/fields/{id}: admin deletes field.
func TestField_Delete_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_field_del")
	suffix := helper.UID()
	createResp := h.CreateField(tokenAdmin, "field_del_"+suffix, "text", "user", false, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)

	h.DeleteField(tokenAdmin, *createResp.JSON201.ID, http.StatusNoContent)
}
