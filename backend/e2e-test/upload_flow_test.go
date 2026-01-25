package e2e_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestChallenge_DataUploadFlow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition
	_, tokenAdmin := h.SetupCompetition("admin_up")

	// 2. Create Challenge
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Upload Test Challenge",
		"description": "Challenge with file",
		"points":      100,
		"flag":        "FLAG{upload_test}",
		"category":    "misc",
		"difficulty":  "medium",
	})

	// 3. Prepare File Data and Hash
	fileName := "secret_task.txt"
	fileContent := "This is a secret task description file."
	fileHash := sha256.Sum256([]byte(fileContent))
	expectedHash := hex.EncodeToString(fileHash[:])

	// 4. Admin Uploads File to Challenge
	resp := h.UploadChallengeFile(tokenAdmin, challengeID, fileName, fileContent)

	fileID := resp.Value("id").String().Raw()
	resp.Value("filename").String().IsEqual(fileName)
	resp.Value("size").Number().IsEqual(len(fileContent))
	resp.Value("sha256").String().IsEqual(expectedHash)

	// 5. Regular User Registers and Logins
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("user_up_" + suffix)

	// 6. User Fetches Files for the Challenge
	filesList := h.GetChallengeFiles(tokenUser, challengeID)

	filesList.Length().IsEqual(1)
	uploadedFile := filesList.Value(0).Object()
	uploadedFile.Value("id").String().IsEqual(fileID)
	uploadedFile.Value("filename").String().IsEqual(fileName)

	// 7. User Obtains Download URL
	downloadURL := h.GetFileDownloadURL(tokenUser, fileID)

	// 8. User Downloads and Verifies Content
	contentResp := h.DownloadFileContent(tokenUser, downloadURL)

	assert.Equal(t, fileContent, contentResp)
}
