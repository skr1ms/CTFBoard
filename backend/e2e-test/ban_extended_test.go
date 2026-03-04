package e2e_test

import (
	"net/http"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// POST /admin/teams/{ID}/ban: banned team cannot unlock hints; after unban can unlock again.
func TestBannedTeam_CannotUnlockHint(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, tokenAdmin := h.SetupCompetition("admin_ban_hint_" + suffix)

	challID := h.CreateBasicChallenge(tokenAdmin, "Hint Ban Challenge "+suffix, "flag{hint_ban_"+suffix+"}", 100)
	hintID := h.CreateHint(tokenAdmin, challID, "This is a ban test hint", 0)

	userName := "ban_hint_user_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(userName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	myTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	teamID := *myTeam.JSON200.ID

	t.Log("=== Before ban: hint unlock succeeds ===")
	h.UnlockHint(tokenUser, challID, hintID, http.StatusOK)

	t.Log("=== Admin bans team ===")
	h.BanTeam(tokenAdmin, teamID, "Testing hint ban", http.StatusOK)

	// Re-try unlock on a DIFFERENT hint to confirm ban blocks, not "already unlocked"
	hintID2 := h.CreateHint(tokenAdmin, challID, "Second hint for ban test", 0)
	t.Log("=== After ban: hint unlock blocked ===")
	h.UnlockHint(tokenUser, challID, hintID2, http.StatusForbidden)

	t.Log("=== Admin unbans team ===")
	h.UnbanTeam(tokenAdmin, teamID, http.StatusOK)

	t.Log("=== After unban: hint unlock succeeds again ===")
	h.UnlockHint(tokenUser, challID, hintID2, http.StatusOK)
}

// POST /admin/teams/{ID}/ban: banned team cannot get file download URL; after unban can download.
func TestBannedTeam_CannotDownloadFile(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, tokenAdmin := h.SetupCompetition("admin_ban_file_" + suffix)

	challID := h.CreateBasicChallenge(tokenAdmin, "File Ban Challenge "+suffix, "flag{file_ban_"+suffix+"}", 100)

	uploadResp := h.UploadChallengeFile(tokenAdmin, challID, "secret.txt", "secret content")
	require.NotNil(t, uploadResp.JSON201)
	fileID := uploadResp.JSON201.ID

	userName := "ban_file_user_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(userName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	myTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	teamID := *myTeam.JSON200.ID

	t.Log("=== Before ban: file download URL accessible ===")
	h.GetFilesIDDownloadExpectStatus(tokenUser, fileID, http.StatusOK)

	t.Log("=== Admin bans team ===")
	h.BanTeam(tokenAdmin, teamID, "Testing file ban", http.StatusOK)

	t.Log("=== After ban: file download blocked ===")
	h.GetFilesIDDownloadExpectStatus(tokenUser, fileID, http.StatusForbidden)

	t.Log("=== Admin unbans team ===")
	h.UnbanTeam(tokenAdmin, teamID, http.StatusOK)

	t.Log("=== After unban: file download accessible again ===")
	h.GetFilesIDDownloadExpectStatus(tokenUser, fileID, http.StatusOK)
}

// POST /admin/teams/{ID}/ban: all three actions (submit + hint + file) are blocked simultaneously.
func TestBannedTeam_AllActionsBlocked(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, tokenAdmin := h.SetupCompetition("admin_ban_all_" + suffix)

	challID := h.CreateBasicChallenge(tokenAdmin, "All Ban Challenge "+suffix, "flag{all_ban_"+suffix+"}", 100)
	hintID := h.CreateHint(tokenAdmin, challID, "Hint for all-ban test", 0)
	uploadResp := h.UploadChallengeFile(tokenAdmin, challID, "file.txt", "file content")
	require.NotNil(t, uploadResp.JSON201)
	fileID := uploadResp.JSON201.ID

	userName := "ban_all_user_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(userName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	myTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	teamID := *myTeam.JSON200.ID

	// Confirm pre-ban: all actions work
	h.SubmitFlag(tokenUser, challID, "flag{wrong}", http.StatusOK)
	h.UnlockHint(tokenUser, challID, hintID, http.StatusOK)
	h.GetFilesIDDownloadExpectStatus(tokenUser, fileID, http.StatusOK)

	t.Log("=== Ban team ===")
	h.BanTeam(tokenAdmin, teamID, "All actions ban test", http.StatusOK)

	t.Log("=== All three actions must be blocked ===")
	h.SubmitFlag(tokenUser, challID, "flag{all_ban_"+suffix+"}", http.StatusForbidden)
	hintID2 := h.CreateHint(tokenAdmin, challID, "Second hint for all-ban", 0)
	h.UnlockHint(tokenUser, challID, hintID2, http.StatusForbidden)
	h.GetFilesIDDownloadExpectStatus(tokenUser, fileID, http.StatusForbidden)
}
