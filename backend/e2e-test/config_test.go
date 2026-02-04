package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// PUT /admin/configs/{key} + GET /admin/configs: config is visible to admin.
func TestConfig_UpsertAndList_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_configs_ok")

	suffix := uuid.New().String()[:8]
	key := "k_" + suffix

	h.PutAdminConfig(tokenAdmin, key, "v", "string", "desc", http.StatusOK)

	listResp := h.GetAdminConfigs(tokenAdmin, http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	found := false
	for _, cfg := range *listResp.JSON200 {
		if cfg.Key == key {
			found = true
			break
		}
	}
	require.True(t, found, "upserted config must be in admin configs list")
}

// GET /admin/configs: non-admin gets 403.
func TestConfig_List_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("cfg_user_" + suffix)

	h.GetAdminConfigs(tokenUser, http.StatusForbidden)
}

// GET /admin/configs/{key}: admin gets config by key.
func TestConfig_GetKey_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_config_key")
	suffix := uuid.New().String()[:8]
	key := "k_" + suffix
	h.PutAdminConfig(tokenAdmin, key, "v1", "string", "desc", http.StatusOK)

	resp := h.GetAdminConfigKey(tokenAdmin, key, http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.Equal(t, key, resp.JSON200.Key)
	require.Equal(t, "v1", resp.JSON200.Value)
}

// DELETE /admin/configs/{key}: admin deletes config.
func TestConfig_Delete_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_config_del")
	suffix := uuid.New().String()[:8]
	key := "k_del_" + suffix
	h.PutAdminConfig(tokenAdmin, key, "v", "string", "d", http.StatusOK)

	h.DeleteAdminConfig(tokenAdmin, key, http.StatusNoContent)
	h.GetAdminConfigKey(tokenAdmin, key, http.StatusNotFound)
}
