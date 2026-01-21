package e2e_test

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestChallenge_DataUploadFlow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]

	_, _, tokenAdmin := h.RegisterAdmin("admin_up_" + suffix)

	h.StartCompetition(tokenAdmin)

	challengeID := h.CreateChallenge(tokenAdmin, map[string]interface{}{
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

	fileID := resp.Value("id").String().Raw()
	resp.Value("filename").String().IsEqual(fileName)
	resp.Value("size").Number().IsEqual(len(fileContent))
	resp.Value("sha256").String().IsEqual(expectedHash)

	emailUser, passUser := h.RegisterUser("user_up_" + suffix)
	loginResp := h.Login(emailUser, passUser, http.StatusOK)
	tokenUser := "Bearer " + loginResp.Value("access_token").String().Raw()

	filesList := h.GetChallengeFiles(tokenUser, challengeID)

	filesList.Length().IsEqual(1)
	uploadedFile := filesList.Value(0).Object()
	uploadedFile.Value("id").String().IsEqual(fileID)
	uploadedFile.Value("filename").String().IsEqual(fileName)

	downloadURL := h.GetFileDownloadURL(tokenUser, fileID)
	contentResp := h.DownloadFileContent(tokenUser, downloadURL)

	assert.Equal(t, fileContent, contentResp)
}
