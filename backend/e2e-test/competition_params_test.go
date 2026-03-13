package e2e_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// PUT /admin/configs/{key} + GET /admin/configs: config is visible to admin.
func TestConfig_UpsertAndList_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("cfg_user_" + suffix)

	h.GetAdminConfigs(tokenUser, http.StatusForbidden)
}

// GET /admin/configs/{key}: admin gets config by key.
func TestConfig_GetKey_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_config_del")
	suffix := uuid.New().String()[:8]
	key := "k_del_" + suffix
	h.PutAdminConfig(tokenAdmin, key, "v", "string", "d", http.StatusOK)

	h.DeleteAdminConfig(tokenAdmin, key, http.StatusNoContent)
	h.GetAdminConfigKey(tokenAdmin, key, http.StatusNotFound)
}

// PUT /admin/configs/{key}: invalid value_type returns 400.
func TestConfig_Put_InvalidValueType_Returns400(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_config_inv_vt")
	key := "k_inv_vt_" + uuid.New().String()[:8]
	h.PutAdminConfig(tokenAdmin, key, "v", "invalid_type", "d", http.StatusBadRequest)
}

func TestConfig_BatchUpdate_ValuesMatch(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	_, tokenAdmin := h.SetupCompetition("admin_config_batch")
	vt := openapi.BatchSetConfigItemValueTypeString
	batch := openapi.BatchSetConfigRequest{
		Configs: []openapi.BatchSetConfigItem{
			{Key: "ctf_name", Value: "BatchCTF", ValueType: &vt},
			{Key: "theme_color_primary", Value: "#111111", ValueType: &vt},
		},
	}
	h.PutAdminConfigsBatch(tokenAdmin, batch, http.StatusOK)
	resp1 := h.GetAdminConfigKey(tokenAdmin, "ctf_name", http.StatusOK)
	resp2 := h.GetAdminConfigKey(tokenAdmin, "theme_color_primary", http.StatusOK)
	require.NotNil(t, resp1.JSON200)
	require.Equal(t, "BatchCTF", resp1.JSON200.Value)
	require.NotNil(t, resp2.JSON200)
	require.Equal(t, "#111111", resp2.JSON200.Value)
}

func TestConfig_Categories_ReturnsAllCategories(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	_, tokenAdmin := h.SetupCompetition("admin_config_categories")
	resp := h.GetAdminConfigsCategories(tokenAdmin, http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.GreaterOrEqual(t, len(*resp.JSON200), 1)
	seen := make(map[string]int)
	for _, item := range *resp.JSON200 {
		require.NotEmpty(t, item.Name)
		require.GreaterOrEqual(t, item.Count, 0)
		seen[item.Name] = item.Count
	}
	require.Contains(t, seen, "general")
	require.Contains(t, seen, "theme")
}

func TestConfig_GetByCategory_ThemeOnly(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	_, tokenAdmin := h.SetupCompetition("admin_config_category")
	resp := h.GetAdminConfigsCategory(tokenAdmin, "theme", http.StatusOK)
	require.NotNil(t, resp.JSON200)
	for _, item := range *resp.JSON200 {
		require.NotNil(t, item.Category, "item %q should have category", item.Key)
		require.Equal(t, "theme", *item.Category, "item %q should have category theme", item.Key)
	}
}

func TestConfig_Public_NoToken_WhitelistKeys(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	h.SetupCompetition("admin_config_public")
	resp := h.GetConfigsPublic(http.StatusOK)
	require.NotNil(t, resp.JSON200)
	allowed := map[string]bool{
		"ctf_name": true, "ctf_description": true, "ctf_logo": true,
		"tos_url": true, "privacy_url": true,
	}
	for _, k := range []string{"theme_color_primary", "theme_dark_mode", "social_github"} {
		allowed[k] = true
	}
	for _, item := range *resp.JSON200 {
		require.True(t, allowed[item.Key] || strings.HasPrefix(item.Key, "theme_") || strings.HasPrefix(item.Key, "social_"),
			"public config key %q must be in whitelist", item.Key)
	}
}
