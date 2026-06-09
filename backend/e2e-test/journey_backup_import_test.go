package e2e_test

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"testing"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestE2E_BackupImportJobSmoke(t *testing.T) {
	s := newE2ESuite(t)
	admin := s.registerAdmin("backup_admin")
	player := s.registerUser("backup_player")
	s.createTeam(&player, "backup-"+e2eUID("team"))

	challengeID := s.createChallenge(admin, "Backup import "+e2eUID("challenge"), "flag{backup_import}", 150)
	filePayload := "backup import file payload"
	fileID := s.uploadChallengeFile(admin, challengeID, "backup.txt", filePayload)
	require.Equal(t, filePayload, s.downloadFile(player, fileID))

	includeFiles := true
	exportResp, err := s.client.GetAdminExportZipWithResponse(
		context.Background(),
		&openapi.GetAdminExportZipParams{IncludeFiles: &includeFiles},
		e2eBearer(admin.Token),
	)
	require.NoError(t, err)
	requireStatus(t, "export backup zip", http.StatusOK, exportResp.StatusCode(), exportResp.Body)
	require.NotEmpty(t, exportResp.Body)

	accepted := s.startBackupImportJob(admin, exportResp.Body)
	completed := s.waitBackupImportJob(admin, *accepted.ID)

	require.NotNil(t, completed.Result)
	require.NotNil(t, completed.Result.Success)
	require.True(t, *completed.Result.Success)
	require.NotNil(t, completed.Result.SkippedCount)
	require.Zero(t, *completed.Result.SkippedCount)
	require.Nil(t, completed.Error)
	require.Equal(t, filePayload, s.downloadFile(player, fileID))
}

func (s *e2eSuite) startBackupImportJob(admin e2eActor, archive []byte) *openapi.ImportJobResponse {
	s.t.Helper()

	var buf bytes.Buffer

	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", "backup.zip")
	require.NoError(s.t, err)
	_, err = part.Write(archive)
	require.NoError(s.t, err)

	require.NoError(s.t, writer.WriteField("conflict_mode", "overwrite"))
	require.NoError(s.t, writer.WriteField("validate_files", "true"))
	require.NoError(s.t, writer.WriteField("preserve_admin_roles", "true"))
	require.NoError(s.t, writer.WriteField("erase_existing", "false"))
	require.NoError(s.t, writer.Close())

	resp, err := s.client.PostAdminImportWithBodyWithResponse(
		context.Background(),
		writer.FormDataContentType(),
		&buf,
		e2eBearer(admin.Token),
	)
	require.NoError(s.t, err)
	requireStatus(s.t, "start backup import", http.StatusAccepted, resp.StatusCode(), resp.Body)
	require.NotNil(s.t, resp.JSON202)
	require.NotNil(s.t, resp.JSON202.ID)

	return resp.JSON202
}

func (s *e2eSuite) waitBackupImportJob(admin e2eActor, jobID openapi_types.UUID) *openapi.ImportJobResponse {
	s.t.Helper()

	deadline := time.Now().Add(15 * time.Second)

	var last *openapi.ImportJobResponse

	for time.Now().Before(deadline) {
		resp, err := s.client.GetAdminImportJobsIDWithResponse(context.Background(), jobID, e2eBearer(admin.Token))
		require.NoError(s.t, err)
		requireStatus(s.t, "get backup import job", http.StatusOK, resp.StatusCode(), resp.Body)
		require.NotNil(s.t, resp.JSON200)
		last = resp.JSON200

		if last.Status != nil && *last.Status == openapi.ImportJobResponseStatusFailed {
			errText := ""

			if last.Error != nil {
				errText = *last.Error
			}

			s.t.Fatalf("backup import job failed: %s", errText)
		}

		if last.Status != nil && *last.Status == openapi.ImportJobResponseStatusCompleted {
			require.NotNil(s.t, last.Phase)
			require.Equal(s.t, openapi.ImportJobResponsePhaseFinished, *last.Phase)

			return last
		}

		time.Sleep(200 * time.Millisecond)
	}

	s.t.Fatalf("backup import job did not complete before deadline: %s", formatBackupImportJob(last))

	return nil
}

func formatBackupImportJob(job *openapi.ImportJobResponse) string {
	if job == nil {
		return "<nil>"
	}

	return fmt.Sprintf("status=%v phase=%v error=%v", job.Status, job.Phase, job.Error)
}
