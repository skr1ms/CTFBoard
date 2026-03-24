package e2e_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// GET /files/{ID}/download: non-existent file returns 404.
func TestFiles_DownloadPublic_NotFound(t *testing.T) {
	t.Parallel()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, GetTestBaseURL()+"/api/v1/files/download/nonexistent-file-id", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// POST /admin/challenges/{ID}/files upload + GET /challenges/{ID}/files + GET /files/{ID}/download: admin uploads file; user lists and downloads; content and sha256 match.
func TestChallenge_DataUploadFlow(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

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
	fileID := resp.JSON201.ID
	require.Equal(t, fileName, resp.JSON201.Filename)
	require.Equal(t, int64(len(fileContent)), resp.JSON201.Size)
	require.Equal(t, expectedHash, resp.JSON201.Sha256)

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("user_up_" + suffix)

	filesList := h.GetChallengeFiles(tokenUser, challengeID)
	require.NotNil(t, filesList.JSON200)
	require.Len(t, *filesList.JSON200, 1)
	uploadedFile := (*filesList.JSON200)[0]
	require.NotNil(t, uploadedFile.ID)
	require.Equal(t, fileID, *uploadedFile.ID)
	require.NotNil(t, uploadedFile.Filename)
	require.Equal(t, fileName, *uploadedFile.Filename)

	downloadURL := h.GetFileDownloadURL(tokenUser, fileID)

	contentResp := h.DownloadFileContent(tokenUser, downloadURL)

	assert.Equal(t, fileContent, contentResp)
}

// DELETE /admin/files/{ID}: admin deletes file; GET /challenges/{ID}/files no longer returns it.
func TestFile_Delete_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_file_del")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "File Del Chal", "description": "desc", "points": 50, "flag": "FLAG{del}", "category": "misc",
	})
	resp := h.UploadChallengeFile(tokenAdmin, challengeID, "todel.txt", "content")
	require.NotNil(t, resp.JSON201)
	fileID := resp.JSON201.ID

	h.DeleteChallengeFile(tokenAdmin, fileID, http.StatusNoContent)

	filesList := h.GetChallengeFiles(tokenAdmin, challengeID)
	require.NotNil(t, filesList.JSON200)
	for _, f := range *filesList.JSON200 {
		if f.ID != nil && *f.ID == fileID {
			t.Fatal("file should be gone after delete")
		}
	}
}

// DELETE /admin/files/{ID}: non-existent file returns 404.
func TestFile_Delete_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_file_del_err")

	h.DeleteChallengeFile(tokenAdmin, "00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}

// GET /challenges/{ID}/files: non-existent challenge returns 200 with empty array.
func TestChallenge_GetChallengeFiles_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("files_404_admin")
	_, _, token := h.RegisterUserAndLogin("files_404_" + helper.UID())
	h.CreateSoloTeam(token, http.StatusCreated)
	resp := h.GetChallengeFilesExpectStatus(token, "00000000-0000-0000-0000-000000000000", http.StatusNotFound)
	require.Nil(t, resp.JSON200)
}

// GET /challenges/{ID}/hints: non-existent challenge returns 200 with empty array.
func TestChallenge_GetHints_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("hints_404_admin")
	_, _, token := h.RegisterUserAndLogin("hints_404_" + helper.UID())
	h.CreateSoloTeam(token, http.StatusCreated)
	resp := h.GetChallengesChallengeIDHintsExpectStatus(token, "00000000-0000-0000-0000-000000000000", http.StatusNotFound)
	require.Nil(t, resp.JSON200)
}

// GET /files/{ID}/download: non-existent file returns 404.
func TestFile_GetDownload_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("filedl_404_admin")
	_, _, token := h.RegisterUserAndLogin("filedl_404_" + helper.UID())
	h.CreateSoloTeam(token, http.StatusCreated)
	h.GetFilesIDDownloadExpectStatus(token, "00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}

// POST /admin/challenges/{ID}/files: non-existent challenge returns 404.
func TestChallenge_UploadFile_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_upload_404")
	h.UploadChallengeFileExpectStatus(tokenAdmin, "00000000-0000-0000-0000-000000000000", "a.txt", "content", http.StatusNotFound)
}

// POST /admin/challenges/{ID}/hints: non-existent challenge returns 404.
func TestChallenge_CreateHint_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_hint_create_404")
	h.CreateHintExpectStatus(tokenAdmin, "00000000-0000-0000-0000-000000000000", "hint", 0, http.StatusNotFound)
}
