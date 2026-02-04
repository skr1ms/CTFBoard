package e2e_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GET /files/{ID}/download: non-existent file returns 404.
func TestFiles_DownloadPublic_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, GetTestBaseURL()+"/api/v1/files/download/nonexistent-file-id", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// POST /admin/challenges/{ID}/files upload + GET /challenges/{ID}/files + GET /files/{ID}/download: admin uploads file; user lists and downloads; content and sha256 match.
func TestChallenge_DataUploadFlow(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_up")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Upload Test Challenge",
		"description": "Challenge with file",
		"points":      100,
		"flag":        "FLAG{upload_test}",
		"category":    "misc",
		"difficulty":  "medium",
	})

	fileName := "secret_task.txt"
	fileContent := "This is a secret task description file."
	fileHash := sha256.Sum256([]byte(fileContent))
	expectedHash := hex.EncodeToString(fileHash[:])

	resp := h.UploadChallengeFile(tokenAdmin, challengeID, fileName, fileContent)
	require.NotNil(t, resp.JSON201)
	fileID, ok := (*resp.JSON201)["id"].(string)
	require.True(t, ok, "id")
	filename, ok := (*resp.JSON201)["filename"].(string)
	require.True(t, ok, "filename")
	require.Equal(t, fileName, filename)
	sizeVal, ok := (*resp.JSON201)["size"].(float64)
	require.True(t, ok, "size")
	require.Equal(t, len(fileContent), int(sizeVal))
	shaVal, ok := (*resp.JSON201)["sha256"].(string)
	require.True(t, ok, "sha256")
	require.Equal(t, expectedHash, shaVal)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("user_up_" + suffix)

	filesList := h.GetChallengeFiles(tokenUser, challengeID)
	require.NotNil(t, filesList.JSON200)
	require.Len(t, *filesList.JSON200, 1)
	uploadedFile := (*filesList.JSON200)[0]
	idVal, ok := uploadedFile["id"].(string)
	require.True(t, ok, "id")
	require.Equal(t, fileID, idVal)
	filenameVal, ok := uploadedFile["filename"].(string)
	require.True(t, ok, "filename")
	require.Equal(t, fileName, filenameVal)

	downloadURL := h.GetFileDownloadURL(tokenUser, fileID)

	contentResp := h.DownloadFileContent(tokenUser, downloadURL)

	assert.Equal(t, fileContent, contentResp)
}

// DELETE /admin/files/{ID}: admin deletes file; GET /challenges/{ID}/files no longer returns it.
func TestFile_Delete_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_file_del")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "File Del Chal", "description": "desc", "points": 50, "flag": "FLAG{del}", "category": "misc",
	})
	resp := h.UploadChallengeFile(tokenAdmin, challengeID, "todel.txt", "content")
	require.NotNil(t, resp.JSON201)
	fileID, ok := (*resp.JSON201)["id"].(string)
	require.True(t, ok, "id")

	h.DeleteChallengeFile(tokenAdmin, fileID, http.StatusNoContent)

	filesList := h.GetChallengeFiles(tokenAdmin, challengeID)
	require.NotNil(t, filesList.JSON200)
	for _, f := range *filesList.JSON200 {
		if id, ok := f["id"].(string); ok && id == fileID {
			t.Fatal("file should be gone after delete")
		}
	}
}

// DELETE /admin/files/{ID}: non-existent file returns 404.
func TestFile_Delete_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_file_del_err")

	h.DeleteChallengeFile(tokenAdmin, "00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}

// GET /challenges/{ID}/files: non-existent challenge returns 200 with empty array.
func TestChallenge_GetChallengeFiles_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("files_404_" + uuid.New().String()[:8])
	h.CreateSoloTeam(token, http.StatusCreated)
	resp := h.GetChallengeFilesExpectStatus(token, "00000000-0000-0000-0000-000000000000", http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.Empty(t, *resp.JSON200)
}

// GET /challenges/{ID}/hints: non-existent challenge returns 200 with empty array.
func TestChallenge_GetHints_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("hints_404_" + uuid.New().String()[:8])
	h.CreateSoloTeam(token, http.StatusCreated)
	resp := h.GetChallengesChallengeIDHintsExpectStatus(token, "00000000-0000-0000-0000-000000000000", http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.Empty(t, *resp.JSON200)
}

// GET /files/{ID}/download: non-existent file returns 404.
func TestFile_GetDownload_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("filedl_404_" + uuid.New().String()[:8])
	h.CreateSoloTeam(token, http.StatusCreated)
	h.GetFilesIDDownloadExpectStatus(token, "00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}

// POST /admin/challenges/{ID}/files: non-existent challenge returns 500.
func TestChallenge_UploadFile_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_upload_404")
	h.UploadChallengeFileExpectStatus(tokenAdmin, "00000000-0000-0000-0000-000000000000", "a.txt", "content", http.StatusInternalServerError)
}

// POST /admin/challenges/{ID}/hints: non-existent challenge returns 500.
func TestChallenge_CreateHint_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_hint_create_404")
	h.CreateHintExpectStatus(tokenAdmin, "00000000-0000-0000-0000-000000000000", "hint", 0, http.StatusInternalServerError)
}
