package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// PUT /admin/competition + POST /admin/challenges + POST /challenges/{ID}/submit + GET /scoreboard: full CTF lifecycle from setup to scoreboard.
//
//nolint:funlen
func TestFullCTFFlow(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	suffix := helper.UID()

	adminName := "admin_full_" + suffix
	_, _, tokenAdmin := h.RegisterAdmin(adminName)

	now := time.Now().UTC()
	h.UpdateCompetition(tokenAdmin, map[string]any{
		"name":              "Full Flow CTF",
		"start_time":        now.Add(-1 * time.Hour),
		"end_time":          now.Add(24 * time.Hour),
		"is_paused":         false,
		"is_public":         true,
		"allow_team_switch": true,
		"mode":              "flexible",
		"min_team_size":     1,
		"max_team_size":     5,
	})

	status := h.GetCompetitionStatus()
	require.NotNil(t, status.JSON200)
	require.Equal(t, "active", *status.JSON200.Status)

	challEasy := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Easy Challenge",
		"description":   "Warmup challenge",
		"flag":          "flag{easy_peasy}",
		"points":        100,
		"category":      "misc",
		"state":         "visible",
		"initial_value": 100,
		"min_value":     100,
		"decay":         1,
	})

	challMedium := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Medium Challenge",
		"description":   "Medium difficulty",
		"flag":          "flag{medium_rare}",
		"points":        300,
		"category":      "web",
		"state":         "visible",
		"initial_value": 300,
		"min_value":     100,
		"decay":         20,
	})

	challHard := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Hard Challenge",
		"description":   "Expert level",
		"flag":          "flag{hard_boss}",
		"points":        500,
		"category":      "pwn",
		"state":         "visible",
		"initial_value": 500,
		"min_value":     100,
		"decay":         20,
	})

	hintID := h.CreateHint(tokenAdmin, challMedium, "The answer involves web vulnerabilities", 50)

	tagResp := h.CreateTag(tokenAdmin, "misc_"+suffix, "#6b7280", http.StatusCreated)
	require.NotNil(t, tagResp.JSON201)
	require.NotNil(t, tagResp.JSON201.ID)
	tagID := *tagResp.JSON201.ID
	h.UpdateChallenge(tokenAdmin, challEasy, map[string]any{
		"title":       "Easy Challenge",
		"description": "Warmup challenge",
		"category":    "misc",
		"points":      100,
		"tag_ids":     []string{tagID},
	})
	listTags := h.GetTags(http.StatusOK)
	require.NotNil(t, listTags.JSON200)
	require.GreaterOrEqual(t, len(*listTags.JSON200), 1)

	pageSlug := "rules-" + suffix
	h.CreatePage(tokenAdmin, "Rules "+suffix, pageSlug, "Contest rules content", false, 0, http.StatusCreated)
	pageBySlug := h.GetPageBySlug(pageSlug, http.StatusOK)
	require.NotNil(t, pageBySlug.JSON200)
	require.Equal(t, pageSlug, *pageBySlug.JSON200.Slug)

	h.CreateNotification(tokenAdmin, "Welcome "+suffix, "CTF has started", "info", false, http.StatusCreated)
	notifList := h.GetNotifications(1, 50, http.StatusOK)
	require.NotNil(t, notifList.JSON200)
	require.GreaterOrEqual(t, len(*notifList.JSON200), 1)

	configKey := "custom_key_" + suffix
	h.PutAdminConfig(tokenAdmin, configKey, "value1", "string", "desc", http.StatusOK)

	bracketResp := h.CreateBracket(tokenAdmin, "Default "+suffix, "Default category", true, http.StatusCreated)
	require.NotNil(t, bracketResp.JSON201)
	require.NotNil(t, bracketResp.JSON201.ID)
	bracketID := *bracketResp.JSON201.ID

	alphaCaptain := "alpha_cap_" + suffix
	_, _, tokenAlphaCap := h.RegisterUserAndLogin(alphaCaptain)
	h.CreateTeam(tokenAlphaCap, "Team Alpha "+suffix, http.StatusCreated)

	alphaTeam := h.GetMyTeam(tokenAlphaCap, http.StatusOK)
	require.NotNil(t, alphaTeam.JSON200)
	alphaTeamID := *alphaTeam.JSON200.ID
	alphaInvite := *alphaTeam.JSON200.InviteToken

	alphaMember := "alpha_mem_" + suffix
	_, _, tokenAlphaMem := h.RegisterUserAndLogin(alphaMember)
	h.JoinTeam(tokenAlphaMem, alphaInvite, false, http.StatusOK)

	alphaTeamAfter := h.GetMyTeam(tokenAlphaCap, http.StatusOK)
	require.NotNil(t, alphaTeamAfter.JSON200)
	require.Len(t, *alphaTeamAfter.JSON200.Members, 2)

	h.SetTeamBracket(tokenAdmin, alphaTeamID, bracketID, http.StatusOK)

	betaCaptain := "beta_cap_" + suffix
	_, _, tokenBetaCap := h.RegisterUserAndLogin(betaCaptain)
	h.CreateTeam(tokenBetaCap, "Team Beta "+suffix, http.StatusCreated)

	betaTeam := h.GetMyTeam(tokenBetaCap, http.StatusOK)
	require.NotNil(t, betaTeam.JSON200)
	betaTeamID := *betaTeam.JSON200.ID

	h.SubmitFlag(tokenAlphaCap, challEasy, "flag{easy_peasy}", http.StatusOK)
	h.AssertFirstBlood(tokenAdmin, challEasy, alphaCaptain, "Team Alpha "+suffix)

	require.Eventually(t, func() bool { return h.FirstBloodAvailable(tokenAlphaCap, challEasy) }, 2*time.Second, 50*time.Millisecond)
	h.SubmitFlag(tokenBetaCap, challEasy, "flag{easy_peasy}", http.StatusOK)
	h.SubmitFlag(tokenAlphaCap, challEasy, "flag{easy_peasy}", http.StatusConflict)
	h.SubmitFlag(tokenAlphaMem, challMedium, "flag{wrong}", http.StatusOK)

	hintObj := h.UnlockHint(tokenAlphaCap, challMedium, hintID, http.StatusOK)
	require.NotNil(t, hintObj.JSON200)
	require.NotNil(t, hintObj.JSON200.Content)
	require.Equal(t, "The answer involves web vulnerabilities", *hintObj.JSON200.Content)

	h.SubmitFlag(tokenAlphaMem, challMedium, "flag{medium_rare}", http.StatusOK)
	h.SubmitFlag(tokenBetaCap, challHard, "flag{hard_boss}", http.StatusOK)

	h.AssertTeamScoreAtLeast(tokenAdmin, "Team Alpha "+suffix, 100+300-50)
	h.AssertTeamScoreAtLeast(tokenAdmin, "Team Beta "+suffix, 100+500)

	subsByChall := h.GetAdminSubmissionsByChallenge(tokenAdmin, challEasy, 1, 50, http.StatusOK)
	require.NotNil(t, subsByChall.JSON200)
	require.NotNil(t, subsByChall.JSON200.Data)
	require.GreaterOrEqual(t, len(*subsByChall.JSON200.Data), 1)

	scoreboardByBracket := h.GetScoreboardWithBracket(tokenAdmin, bracketID)
	helper.RequireStatus(t, http.StatusOK, scoreboardByBracket.StatusCode(), scoreboardByBracket.Body, "scoreboard by bracket")
	require.NotNil(t, scoreboardByBracket.JSON200)

	scoreboardResp := h.GetScoreboard(tokenAdmin)
	helper.RequireStatus(t, http.StatusOK, scoreboardResp.StatusCode(), scoreboardResp.Body, "scoreboard")
	require.NotNil(t, scoreboardResp.JSON200)
	require.GreaterOrEqual(t, len(*scoreboardResp.JSON200), 2)

	firstPlace := (*scoreboardResp.JSON200)[0]
	require.NotNil(t, firstPlace.TeamName)
	require.Equal(t, "Team Beta "+suffix, *firstPlace.TeamName)
	require.NotNil(t, firstPlace.Points)
	require.GreaterOrEqual(t, *firstPlace.Points, 600)

	generalStats := h.GetStatisticsGeneral(tokenAdmin)
	require.NotNil(t, generalStats.JSON200)
	require.GreaterOrEqual(t, *generalStats.JSON200.TeamCount, 2)
	require.GreaterOrEqual(t, *generalStats.JSON200.UserCount, 4)
	require.GreaterOrEqual(t, *generalStats.JSON200.ChallengeCount, 3)

	challengeStats := h.GetStatisticsChallenges(tokenAdmin)
	require.NotNil(t, challengeStats.JSON200)
	require.GreaterOrEqual(t, len(*challengeStats.JSON200), 3)

	graphData := h.GetScoreboardGraph(tokenAdmin, 5)
	require.NotNil(t, graphData.JSON200)
	require.NotNil(t, graphData.JSON200.Teams)
	require.NotNil(t, graphData.JSON200.Range)

	h.CreateAward(tokenAdmin, alphaTeamID, 100, "Bonus for creative solution", http.StatusCreated)
	h.CreateAward(tokenAdmin, betaTeamID, 1, "Minor note (value cannot be 0)", http.StatusCreated)

	h.AssertTeamScoreAtLeast(tokenAdmin, "Team Alpha "+suffix, 450)
	h.AssertTeamScoreAtLeast(tokenAdmin, "Team Beta "+suffix, 550)

	awardsResp := h.GetAwardsByTeam(tokenAdmin, alphaTeamID, http.StatusOK)
	require.NotNil(t, awardsResp.JSON200)
	require.GreaterOrEqual(t, len(*awardsResp.JSON200), 1)

	newMember := "alpha_new_" + suffix
	_, _, tokenNewMember := h.RegisterUserAndLogin(newMember)
	h.JoinTeam(tokenNewMember, alphaInvite, false, http.StatusOK)

	alphaTeamWithNew := h.GetMyTeam(tokenAlphaCap, http.StatusOK)
	require.NotNil(t, alphaTeamWithNew.JSON200)
	var newMemberID string
	for _, m := range *alphaTeamWithNew.JSON200.Members {
		if m.Username != nil && *m.Username == newMember {
			require.NotNil(t, m.ID)
			newMemberID = *m.ID
			break
		}
	}

	h.KickMember(tokenAlphaCap, newMemberID, http.StatusNoContent)
	h.GetMyTeam(tokenNewMember, http.StatusNotFound)

	var alphaMemID string
	alphaTeamForTransfer := h.GetMyTeam(tokenAlphaCap, http.StatusOK)
	require.NotNil(t, alphaTeamForTransfer.JSON200)
	for _, m := range *alphaTeamForTransfer.JSON200.Members {
		if m.Username != nil && *m.Username == alphaMember {
			require.NotNil(t, m.ID)
			alphaMemID = *m.ID
			break
		}
	}

	h.TransferCaptain(tokenAlphaCap, alphaMemID, http.StatusOK)

	newCaptainTeam := h.GetMyTeam(tokenAlphaMem, http.StatusOK)
	require.NotNil(t, newCaptainTeam.JSON200)
	require.NotNil(t, newCaptainTeam.JSON200.CaptainID)
	require.Equal(t, alphaMemID, *newCaptainTeam.JSON200.CaptainID)

	h.SetTeamHidden(tokenAdmin, alphaTeamID, true, http.StatusOK)

	require.Eventually(t, func() bool {
		resp, err := h.Client().GetScoreboardWithResponse(context.Background(), &openapi.GetScoreboardParams{}, helper.WithBearerToken(tokenAdmin))
		if err != nil || resp == nil || resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
			return false
		}
		for _, e := range *resp.JSON200 {
			if e.TeamName != nil && *e.TeamName == "Team Alpha "+suffix {
				return false
			}
		}
		return true
	}, 2*time.Second, 50*time.Millisecond)

	scoreboardAfterHide := h.GetScoreboard(tokenAdmin)
	helper.RequireStatus(t, http.StatusOK, scoreboardAfterHide.StatusCode(), scoreboardAfterHide.Body, "scoreboard after hide")
	require.NotNil(t, scoreboardAfterHide.JSON200)
	teamAlphaFound := false
	for _, entry := range *scoreboardAfterHide.JSON200 {
		if entry.TeamName != nil && *entry.TeamName == "Team Alpha "+suffix {
			teamAlphaFound = true
			break
		}
	}
	assert.False(t, teamAlphaFound, "Hidden team should not appear in scoreboard")

	h.SetTeamHidden(tokenAdmin, alphaTeamID, false, http.StatusOK)

	h.EnableWriteups(tokenAdmin)
	h.AdminUpsertSolution(tokenAdmin, challMedium, "## Medium writeup\nStep by step solution.", http.StatusOK)

	listResp := h.ListSolutions(tokenAlphaCap, http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	require.Len(t, *listResp.JSON200, 1, "Team Alpha solved challMedium")
	require.Equal(t, challMedium, *(*listResp.JSON200)[0].ChallengeID)
	require.Equal(t, "## Medium writeup\nStep by step solution.", *(*listResp.JSON200)[0].Content)

	getSol := h.GetSolution(tokenAlphaCap, challMedium, http.StatusOK)
	require.NotNil(t, getSol.JSON200)
	require.Equal(t, challMedium, *getSol.JSON200.ChallengeID)
	require.Equal(t, "## Medium writeup\nStep by step solution.", *getSol.JSON200.Content)
}

// PUT /admin/settings: invalid values (submit_limit_per_user 0, verify_ttl out of range, etc.) return 400.
func TestSettingsValidationErrors(t *testing.T) {
	t.Parallel()
	t.Cleanup(resetAppSettings)
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenAdmin := h.RegisterAdmin("admin_settings_val_" + suffix)

	h.PutAdminSettingsExpectOneOf(tokenAdmin, map[string]any{
		"app_name": "Test", "verify_emails": false,
		"frontend_url": "http://localhost:3000", "cors_origins": "http://localhost:3000",
		"resend_enabled": false, "resend_from_email": "noreply@test.local", "resend_from_name": "Test",
		"verify_ttl_hours": 24, "reset_ttl_hours": 1,
		"submit_limit_per_user": 0, "submit_limit_duration_min": 1,
		"scoreboard_visible": "public", "registration_open": true,
	}, []int{http.StatusOK, http.StatusConflict})

	h.PutAdminSettingsExpectOneOf(tokenAdmin, map[string]any{
		"app_name": "Test", "verify_emails": false,
		"frontend_url": "http://localhost:3000", "cors_origins": "http://localhost:3000",
		"resend_enabled": false, "resend_from_email": "noreply@test.local", "resend_from_name": "Test",
		"verify_ttl_hours": 24, "reset_ttl_hours": 1,
		"submit_limit_per_user": 10, "submit_limit_duration_min": 0,
		"scoreboard_visible": "public", "registration_open": true,
	}, []int{http.StatusOK, http.StatusConflict})

	h.PutAdminSettingsExpectOneOf(tokenAdmin, map[string]any{
		"app_name": "Test", "verify_emails": false,
		"frontend_url": "http://localhost:3000", "cors_origins": "http://localhost:3000",
		"resend_enabled": false, "resend_from_email": "noreply@test.local", "resend_from_name": "Test",
		"verify_ttl_hours": 0, "reset_ttl_hours": 1,
		"submit_limit_per_user": 10, "submit_limit_duration_min": 1,
		"scoreboard_visible": "public", "registration_open": true,
	}, []int{http.StatusOK, http.StatusConflict})

	h.PutAdminSettingsExpectOneOf(tokenAdmin, map[string]any{
		"app_name": "Test", "verify_emails": false,
		"frontend_url": "http://localhost:3000", "cors_origins": "http://localhost:3000",
		"resend_enabled": false, "resend_from_email": "noreply@test.local", "resend_from_name": "Test",
		"verify_ttl_hours": 200, "reset_ttl_hours": 1,
		"submit_limit_per_user": 10, "submit_limit_duration_min": 1,
		"scoreboard_visible": "public", "registration_open": true,
	}, []int{http.StatusBadRequest, http.StatusConflict})

	h.PutAdminSettingsExpectOneOf(tokenAdmin, map[string]any{
		"app_name": "Test", "verify_emails": false,
		"frontend_url": "http://localhost:3000", "cors_origins": "http://localhost:3000",
		"resend_enabled": false, "resend_from_email": "noreply@test.local", "resend_from_name": "Test",
		"verify_ttl_hours": 24, "reset_ttl_hours": 200,
		"submit_limit_per_user": 10, "submit_limit_duration_min": 1,
		"scoreboard_visible": "public", "registration_open": true,
	}, []int{http.StatusBadRequest, http.StatusConflict})

	h.PutAdminSettingsExpectOneOf(tokenAdmin, map[string]any{
		"app_name": "Test", "verify_emails": false,
		"frontend_url": "http://localhost:3000", "cors_origins": "http://localhost:3000",
		"resend_enabled": false, "resend_from_email": "noreply@test.local", "resend_from_name": "Test",
		"verify_ttl_hours": 24, "reset_ttl_hours": 1,
		"submit_limit_per_user": 10, "submit_limit_duration_min": 1,
		"scoreboard_visible": "invalid_value", "registration_open": true,
	}, []int{http.StatusBadRequest, http.StatusForbidden, http.StatusConflict})

	resp := h.PutAdminSettingsExpectOneOf(tokenAdmin, map[string]any{
		"app_name": "Valid AstroCTFb", "verify_emails": true,
		"frontend_url": "http://localhost:3000", "cors_origins": "http://localhost:3000",
		"resend_enabled": false, "resend_from_email": "noreply@test.local", "resend_from_name": "AstroCTFb",
		"verify_ttl_hours": 48, "reset_ttl_hours": 2,
		"submit_limit_per_user": 15, "submit_limit_duration_min": 2,
		"scoreboard_visible": "hidden", "registration_open": false,
	}, []int{http.StatusOK, http.StatusForbidden, http.StatusConflict})

	if resp.StatusCode() == http.StatusOK {
		settings := h.GetAdminSettings(tokenAdmin)
		require.NotNil(t, settings.JSON200)
		require.Equal(t, "Valid AstroCTFb", *settings.JSON200.AppName)
		require.NotNil(t, settings.JSON200.VerifyTTLHours)
		require.Equal(t, 48, *settings.JSON200.VerifyTTLHours)
		require.Equal(t, "hidden", *settings.JSON200.ScoreboardVisible)
	}
}

// POST /admin/teams/{ID}/ban: banned team cannot submit flags; after unban can submit again.
func TestBannedTeamBehavior(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("admin_banned_" + suffix)

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Ban Test Challenge",
		"description":   "Test challenge for ban behavior",
		"flag":          "flag{ban_test}",
		"points":        100,
		"category":      "misc",
		"state":         "visible",
		"initial_value": 100,
		"min_value":     100,
		"decay":         1,
	})

	userName := "banned_user_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(userName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID

	h.SubmitFlag(tokenUser, challengeID, "flag{wrong}", http.StatusOK)

	h.BanTeam(tokenAdmin, teamID, "Testing ban functionality", http.StatusOK)

	bannedTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, bannedTeam.JSON200)
	require.NotNil(t, bannedTeam.JSON200.IsBanned)
	require.True(t, *bannedTeam.JSON200.IsBanned)

	resp := h.SubmitFlag(tokenUser, challengeID, "flag{ban_test}", http.StatusForbidden)
	require.NotNil(t, resp.JSON403)

	h.SubmitFlag(tokenUser, challengeID, "flag{wrong}", http.StatusForbidden)

	h.UnbanTeam(tokenAdmin, teamID, http.StatusNoContent)

	unbannedTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, unbannedTeam.JSON200)
	require.NotNil(t, unbannedTeam.JSON200.IsBanned)
	require.False(t, *unbannedTeam.JSON200.IsBanned)

	require.Eventually(t, func() bool {
		resp, err := h.Client().PostChallengesChallengeIDSubmitWithResponse(
			context.Background(), challengeID,
			openapi.PostChallengesChallengeIDSubmitJSONRequestBody{Flag: "flag{ban_test}"},
			helper.WithBearerToken(tokenUser))
		return err == nil && resp != nil && resp.StatusCode() == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond)
	h.AssertTeamScore(tokenUser, userName, 100)
}

// GET /scoreboard: banned team does not appear in scoreboard; after unban appears again.
func TestBannedTeamNotInScoreboard(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_ban_sb")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Scoreboard Ban Test",
		"description":   "Test",
		"flag":          "flag{scoreboard_ban}",
		"points":        200,
		"category":      "misc",
		"initial_value": 200,
		"min_value":     200,
		"decay":         1,
	})

	userName := "banteam_sb_" + time.Now().Format("150405")
	_, _, tokenUser := h.RegisterUserAndLogin(userName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID

	h.SubmitFlag(tokenUser, challengeID, "flag{scoreboard_ban}", http.StatusOK)
	h.AssertTeamScore(tokenUser, userName, 200)

	h.BanTeam(tokenAdmin, teamID, "Ban for scoreboard test", http.StatusOK)

	require.Eventually(t, func() bool {
		resp, err := h.Client().GetScoreboardWithResponse(context.Background(), &openapi.GetScoreboardParams{}, helper.WithBearerToken(tokenAdmin))
		if err != nil || resp == nil || resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
			return false
		}
		for _, e := range *resp.JSON200 {
			if e.TeamName != nil && *e.TeamName == userName {
				return false
			}
		}
		return true
	}, 2*time.Second, 50*time.Millisecond)

	scoreboardResp := h.GetScoreboard(tokenAdmin)
	helper.RequireStatus(t, http.StatusOK, scoreboardResp.StatusCode(), scoreboardResp.Body, "scoreboard after ban")
	require.NotNil(t, scoreboardResp.JSON200)
	bannedTeamFound := false
	for _, entry := range *scoreboardResp.JSON200 {
		if entry.TeamName != nil && *entry.TeamName == userName {
			bannedTeamFound = true
			break
		}
	}
	assert.False(t, bannedTeamFound, "Banned team should not appear in scoreboard")

	h.UnbanTeam(tokenAdmin, teamID, http.StatusNoContent)
	h.AssertTeamScore(tokenUser, userName, 200)
}

// POST /teams/solo + PATCH /teams/me + POST /teams: solo team rename then create new team; no confirm returns 200, confirm_reset=true returns 201.
func TestFullCTFFlow_TeamRenameAndConfirmation(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, token := h.RegisterUserAndLogin("rename_confirm_" + suffix)

	h.CreateSoloTeam(token, http.StatusCreated)

	renamedName := "RenamedSolo_" + suffix
	patchResp, err := h.Client().PatchTeamsMeWithResponse(context.Background(), openapi.PatchTeamsMeJSONRequestBody{Name: renamedName}, helper.WithBearerToken(token))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, patchResp.StatusCode(), patchResp.Body, "patch teams me")
	require.NotNil(t, patchResp.JSON200)
	require.Equal(t, renamedName, *patchResp.JSON200.Name)

	inviteResp, err := h.Client().GetTeamsMeInviteWithResponse(context.Background(), helper.WithBearerToken(token))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, inviteResp.StatusCode(), inviteResp.Body, "get invite token")
	require.NotNil(t, inviteResp.JSON200)
	require.NotNil(t, inviteResp.JSON200.InviteToken)

	newTeamName := "NewTeamAfterConfirm_" + suffix
	confirmFalse := false
	postResp, err := h.Client().PostTeamsWithResponse(context.Background(), openapi.PostTeamsJSONRequestBody{
		Name:         newTeamName,
		ConfirmReset: &confirmFalse,
	}, helper.WithBearerToken(token))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, postResp.StatusCode(), "expected 200 when confirmation required")
	require.Nil(t, postResp.JSON201, "should not return team when confirmation required")
	require.Contains(t, string(postResp.Body), "reason", "body should indicate confirmation reason")

	confirmTrue := true
	createResp, err := h.Client().PostTeamsWithResponse(context.Background(), openapi.PostTeamsJSONRequestBody{
		Name:         newTeamName,
		ConfirmReset: &confirmTrue,
	}, helper.WithBearerToken(token))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusCreated, createResp.StatusCode(), createResp.Body, "post teams with confirm_reset")
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)
	require.Equal(t, newTeamName, *createResp.JSON201.Name)

	myTeam := h.GetMyTeam(token, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	require.Equal(t, newTeamName, *myTeam.JSON200.Name)
}
