package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// POST /challenges/{challengeID}/comments + GET /challenges/{challengeID}/comments: allowed only after competition ended.
func TestComment_CreateAndList_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_comments_ok")

	now := time.Now().UTC()
	h.UpdateCompetition(tokenAdmin, map[string]any{
		"name":              "Test CTF",
		"start_time":        now.Add(-2 * time.Hour).Format(time.RFC3339),
		"end_time":          now.Add(-1 * time.Hour).Format(time.RFC3339),
		"is_paused":         false,
		"allow_team_switch": true,
		"mode":              "flexible",
	})

	challengeID := h.CreateBasicChallenge(tokenAdmin, "Comment Challenge", "FLAG{comment}", 100)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("comment_user_" + suffix)

	content := "hello " + suffix
	createResp := h.CreateComment(tokenUser, challengeID, content, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	listResp := h.GetChallengeComments(tokenUser, challengeID, http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	found := false
	for _, c := range *listResp.JSON200 {
		if c.Content != nil && *c.Content == content {
			found = true
			break
		}
	}
	require.True(t, found, "created comment must be in list")
}

// POST /challenges/{challengeID}/comments: while competition active returns 403.
func TestComment_Create_Forbidden_WhenActive(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_comments_forbid")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Comment Challenge 2", "FLAG{comment2}", 100)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("comment_user2_" + suffix)

	h.CreateComment(tokenUser, challengeID, "x", http.StatusForbidden)
}

// DELETE /comments/{id}: author deletes own comment.
func TestComment_Delete_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_comments_del")
	now := time.Now().UTC()
	h.UpdateCompetition(tokenAdmin, map[string]any{
		"name":              "Test CTF",
		"start_time":        now.Add(-2 * time.Hour).Format(time.RFC3339),
		"end_time":          now.Add(-1 * time.Hour).Format(time.RFC3339),
		"is_paused":         false,
		"allow_team_switch": true,
		"mode":              "flexible",
	})
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Comment Del Ch", "FLAG{del}", 100)
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("comment_del_user_" + suffix)
	createResp := h.CreateComment(tokenUser, challengeID, "to delete", http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	h.DeleteComment(tokenUser, *createResp.JSON201.ID, http.StatusNoContent)
}

// DELETE /comments/{id}: wrong id returns 404 or 403 (comments disabled during active competition).
func TestComment_Delete_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("comment_del_nf_" + suffix)

	h.DeleteComment(tokenUser, uuid.New().String(), http.StatusForbidden)
}
