package e2e_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// GET /admin/export: admin exports competition as JSON; returns 200 and JSON body.
func TestBackup_ExportJSON(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_backup_export")
	h.CreateBasicChallenge(tokenAdmin, "Export Chall", "flag{export}", 50)

	resp := h.AdminExport(tokenAdmin, false, true)
	require.NotNil(t, resp.HTTPResponse)
	require.True(t, strings.Contains(resp.HTTPResponse.Header.Get("Content-Type"), "application/json"))
	require.True(t, strings.Contains(resp.HTTPResponse.Header.Get("Content-Disposition"), "attachment"))
	require.GreaterOrEqual(t, len(resp.Body), 10, "expected non-empty JSON body")
}

// GET /admin/export/zip: admin exports competition as ZIP; returns 200 and binary body.
func TestBackup_ExportZip(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_backup_zip")
	h.CreateBasicChallenge(tokenAdmin, "Zip Chall", "flag{zip}", 50)

	resp := h.AdminExportZip(tokenAdmin)
	require.NotNil(t, resp.HTTPResponse)
	require.True(t, strings.Contains(resp.HTTPResponse.Header.Get("Content-Type"), "application/zip"))
	require.True(t, strings.Contains(resp.HTTPResponse.Header.Get("Content-Disposition"), "attachment"))
	require.GreaterOrEqual(t, len(resp.Body), 4, "expected non-empty ZIP body")
	require.True(t, len(resp.Body) < 2 || (resp.Body[0] == 'P' && resp.Body[1] == 'K'), "expected ZIP magic (PK)")
}

// GET /admin/export/zip then POST /admin/import: export ZIP, re-import with conflict_mode skip; returns 200.
func TestBackup_ExportThenImport(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_backup_roundtrip")
	h.CreateBasicChallenge(tokenAdmin, "Roundtrip Chall", "flag{rt}", 50)

	resp := h.AdminExportZip(tokenAdmin)
	require.GreaterOrEqual(t, len(resp.Body), 4, "export returned empty or too small body")

	h.AdminImport(tokenAdmin, resp.Body, "backup.zip", "skip", http.StatusOK)
}

// GET /admin/export: non-admin gets 403 Forbidden.
func TestBackup_Export_Forbidden(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, _ = h.SetupCompetition("admin_exp_f")
	_, _, tokenUser := h.RegisterUserAndLogin("nonadmin_exp")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.AdminExportExpectStatus(tokenUser, false, true, http.StatusForbidden)
}

// GET /admin/export/zip: non-admin gets 403 Forbidden.
func TestBackup_ExportZip_Forbidden(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, _ = h.SetupCompetition("admin_zip_f")
	_, _, tokenUser := h.RegisterUserAndLogin("nonadmin_zip")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.AdminExportZipExpectStatus(tokenUser, http.StatusForbidden)
}

// POST /admin/import: non-admin gets 403 Forbidden.
func TestBackup_Import_Forbidden(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, _ = h.SetupCompetition("admin_imp_f")
	_, _, tokenUser := h.RegisterUserAndLogin("nonadmin_imp")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.AdminImport(tokenUser, []byte("not zip"), "x.zip", "skip", http.StatusForbidden)
}
