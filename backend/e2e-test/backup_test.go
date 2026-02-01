package e2e_test

import (
	"net/http"
	"testing"
)

// GET /admin/export: admin exports competition as JSON; returns 200 and JSON body.
func TestBackup_ExportJSON(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_backup_export")
	h.CreateBasicChallenge(tokenAdmin, "Export Chall", "flag{export}", 50)

	resp := h.AdminExport(tokenAdmin, false, true)
	resp.Header("Content-Type").Contains("application/json")
	resp.Header("Content-Disposition").Contains("attachment")
	body := resp.Body().Raw()
	if len(body) < 10 {
		t.Error("expected non-empty JSON body")
	}
}

// GET /admin/export/zip: admin exports competition as ZIP; returns 200 and binary body.
func TestBackup_ExportZip(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_backup_zip")
	h.CreateBasicChallenge(tokenAdmin, "Zip Chall", "flag{zip}", 50)

	resp := h.AdminExportZip(tokenAdmin)
	resp.Header("Content-Type").Contains("application/zip")
	resp.Header("Content-Disposition").Contains("attachment")
	body := resp.Body().Raw()
	if len(body) < 4 {
		t.Error("expected non-empty ZIP body")
	}
	if len(body) >= 2 && (body[0] != 'P' || body[1] != 'K') {
		t.Error("expected ZIP magic (PK)")
	}
}

// GET /admin/export/zip then POST /admin/import: export ZIP, re-import with conflict_mode skip; returns 200.
func TestBackup_ExportThenImport(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_backup_roundtrip")
	h.CreateBasicChallenge(tokenAdmin, "Roundtrip Chall", "flag{rt}", 50)

	resp := h.AdminExportZip(tokenAdmin)
	zipBytes := []byte(resp.Body().Raw())
	if len(zipBytes) < 4 {
		t.Fatal("export returned empty or too small body")
	}

	h.AdminImport(tokenAdmin, zipBytes, "backup.zip", "skip", http.StatusOK)
}
