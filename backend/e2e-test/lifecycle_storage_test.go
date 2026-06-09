package e2e_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestE2E_AdminStorageDeleteAcceptsSlashPathQuery(t *testing.T) {
	s := newE2ESuite(t)

	admin := s.registerAdmin("storage_delete_admin")
	challengeID := s.createChallenge(admin, e2eUID("storage_delete_chal"), "flag{storage_delete}", 100)
	filename := "storage-delete.bin"

	s.uploadChallengeFile(admin, challengeID, filename, "storage delete payload")

	list, err := s.client.GetAdminStorageWithResponse(context.Background(), &openapi.GetAdminStorageParams{Prefix: "files/"}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "list storage", http.StatusOK, list.StatusCode(), list.Body)
	require.NotNil(t, list.JSON200)
	require.NotNil(t, list.JSON200.Objects)

	var storagePath string

	for _, obj := range *list.JSON200.Objects {
		if obj.Path != nil && strings.HasSuffix(*obj.Path, "/"+filename) {
			storagePath = *obj.Path

			break
		}
	}

	require.NotEmpty(t, storagePath)
	require.Contains(t, storagePath, "/")

	deleteURL := GetTestBaseURL() + "/api/v1/admin/storage?path=" + url.QueryEscape(storagePath)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, deleteURL, http.NoBody)
	require.NoError(t, err)
	require.NoError(t, e2eBearer(admin.Token)(context.Background(), req))

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	var auditCount int

	err = TestPool.QueryRow(context.Background(), `
		SELECT COUNT(*)::int
		FROM audit_logs
		WHERE user_id = $1
			AND action = 'delete'
			AND entity_type = 'storage'
			AND entity_id = 'object'
			AND ip <> ''
			AND details->>'path' = $2
	`, admin.UserID, storagePath).Scan(&auditCount)
	require.NoError(t, err)
	require.Equal(t, 1, auditCount)
}
