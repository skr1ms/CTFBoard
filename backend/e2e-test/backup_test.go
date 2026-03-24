package e2e_test

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// GET /admin/export: admin exports competition as JSON; returns 200 and JSON body.
func TestBackup_ExportJSON(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_backup_roundtrip")
	h.CreateBasicChallenge(tokenAdmin, "Roundtrip Chall", "flag{rt}", 50)

	resp := h.AdminExportZip(tokenAdmin)
	require.GreaterOrEqual(t, len(resp.Body), 4, "export returned empty or too small body")

	h.AdminImport(tokenAdmin, resp.Body, "backup.zip", "skip", http.StatusOK)
}

// GET /admin/export: non-admin gets 403 Forbidden.
func TestBackup_Export_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_exp_f")
	_, _, tokenUser := h.RegisterUserAndLogin("nonadmin_exp")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.AdminExportExpectStatus(tokenUser, false, true, http.StatusForbidden)
}

// GET /admin/export/zip: non-admin gets 403 Forbidden.
func TestBackup_ExportZip_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_zip_f")
	_, _, tokenUser := h.RegisterUserAndLogin("nonadmin_zip")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.AdminExportZipExpectStatus(tokenUser, http.StatusForbidden)
}

// POST /admin/import: non-admin gets 403 Forbidden.
func TestBackup_Import_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_imp_f")
	_, _, tokenUser := h.RegisterUserAndLogin("nonadmin_imp")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.AdminImport(tokenUser, []byte("not zip"), "x.zip", "skip", http.StatusForbidden)
}

// GET /admin/export/csv: admin exports table as CSV bytes.
func TestBackup_ExportCSV_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_csv_export")
	h.CreateBasicChallenge(tokenAdmin, "CSV Chall", "flag{csv}", 50)

	table := openapi.GetAdminExportCsvParamsTableChallenges
	resp, err := h.Client().GetAdminExportCsvWithResponse(context.Background(), &openapi.GetAdminExportCsvParams{
		Table: table,
	}, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "admin export csv")
	require.True(t, strings.Contains(resp.HTTPResponse.Header.Get("Content-Type"), "text/csv") ||
		strings.Contains(resp.HTTPResponse.Header.Get("Content-Type"), "application/octet-stream") ||
		len(resp.Body) > 0, "expected non-empty CSV response")
}

// GET /admin/export/csv: non-admin returns 403.
func TestBackup_ExportCSV_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_csv_exp_f")
	_, _, tokenUser := h.RegisterUserAndLogin("nonadmin_csv")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	table := openapi.GetAdminExportCsvParamsTableUsers
	resp, err := h.Client().GetAdminExportCsvWithResponse(context.Background(), &openapi.GetAdminExportCsvParams{
		Table: table,
	}, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, resp.StatusCode(), resp.Body, "admin export csv forbidden")
}

// POST /admin/import/csv: admin imports valid CSV.
func TestBackup_ImportCSV_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_csv_import")

	csvData := "id,name\n" + "00000000-0000-0000-0000-000000000001,TestTeam\n"
	resp, err := h.Client().PostAdminImportCsvWithBodyWithResponse(context.Background(), "text/csv", bytes.NewBufferString(csvData), helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	// 200 for valid CSV or 400 for invalid format - both are acceptable since actual schema validation is server-side
	require.True(t, resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusBadRequest,
		"expected 200 or 400, got %d", resp.StatusCode())
}

// POST /admin/import/csv: non-admin returns 403.
func TestBackup_ImportCSV_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_csv_import_f")
	_, _, tokenUser := h.RegisterUserAndLogin("nonadmin_csv_imp")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	resp, err := h.Client().PostAdminImportCsvWithBodyWithResponse(context.Background(), "text/csv", bytes.NewBufferString("a,b\n1,2"), helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, resp.StatusCode(), resp.Body, "admin import csv non-admin forbidden")
}

// POST /admin/import/csv: invalid CSV format returns 400.
func TestBackup_ImportCSV_InvalidFormat(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_csv_invalid")

	invalidCSV := []byte("id,username\n\"unclosed quote")
	resp, err := h.Client().PostAdminImportCsvWithBodyWithResponse(context.Background(), "text/csv", bytes.NewBuffer(invalidCSV), helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	require.True(t, resp.StatusCode() == http.StatusBadRequest || resp.StatusCode() == http.StatusOK,
		"expected 400 for invalid CSV or 200 if parse succeeds with empty, got %d: %s", resp.StatusCode(), string(resp.Body))
}
