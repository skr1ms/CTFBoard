package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Full E2E: admin setup, challenges, teams, solves, scoreboard, statistics, awards, kick/transfer, hidden team.
//nolint:funlen
func TestFullCTFFlow(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())
	suffix := uuid.New().String()[:8]

	t.Log("=== Phase 1: Admin Setup ===")

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

	t.Log("=== Phase 2: Create Challenges ===")

	challEasy := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Easy Challenge",
		"description":   "Warmup challenge",
		"flag":          "flag{easy_peasy}",
		"points":        100,
		"category":      "misc",
		"is_hidden":     false,
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
		"is_hidden":     false,
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
		"is_hidden":     false,
		"initial_value": 500,
		"min_value":     100,
		"decay":         20,
	})

	hintID := h.CreateHint(tokenAdmin, challMedium, "The answer involves web vulnerabilities", 50)

	t.Log("=== Phase 2b: Tags, Page, Notification, Config, Bracket ===")

	tagResp := h.CreateTag(tokenAdmin, "misc_"+suffix, "#6b7280", http.StatusCreated)
	require.NotNil(t, tagResp.JSON201)
	require.NotNil(t, tagResp.JSON201.ID)
	tagID := *tagResp.JSON201.ID
	h.UpdateChallenge(tokenAdmin, challEasy, map[string]any{"tag_ids": []string{tagID}})
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

	t.Log("=== Phase 3: Team Alpha Registration ===")

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

	t.Log("=== Phase 4: Team Beta Registration ===")

	betaCaptain := "beta_cap_" + suffix
	_, _, tokenBetaCap := h.RegisterUserAndLogin(betaCaptain)
	h.CreateTeam(tokenBetaCap, "Team Beta "+suffix, http.StatusCreated)

	betaTeam := h.GetMyTeam(tokenBetaCap, http.StatusOK)
	require.NotNil(t, betaTeam.JSON200)
	betaTeamID := *betaTeam.JSON200.ID

	t.Log("=== Phase 5: Solving Challenges ===")

	h.SubmitFlag(tokenAlphaCap, challEasy, "flag{easy_peasy}", http.StatusOK)
	t.Log("Team Alpha solved Easy Challenge - First Blood!")

	h.AssertFirstBlood(challEasy, alphaCaptain, "Team Alpha "+suffix)

	time.Sleep(100 * time.Millisecond)
	h.SubmitFlag(tokenBetaCap, challEasy, "flag{easy_peasy}", http.StatusOK)
	t.Log("Team Beta solved Easy Challenge")

	h.SubmitFlag(tokenAlphaCap, challEasy, "flag{easy_peasy}", http.StatusConflict)
	t.Log("Team Alpha cannot solve same challenge twice - OK")

	h.SubmitFlag(tokenAlphaMem, challMedium, "flag{wrong}", http.StatusBadRequest)
	t.Log("Wrong flag rejected - OK")

	hintObj := h.UnlockHint(tokenAlphaCap, challMedium, hintID, http.StatusOK)
	require.NotNil(t, hintObj.JSON200)
	require.NotNil(t, hintObj.JSON200.Content)
	require.Equal(t, "The answer involves web vulnerabilities", *hintObj.JSON200.Content)
	t.Log("Team Alpha unlocked hint for 50 points")

	h.SubmitFlag(tokenAlphaMem, challMedium, "flag{medium_rare}", http.StatusOK)
	t.Log("Team Alpha member solved Medium Challenge")

	h.SubmitFlag(tokenBetaCap, challHard, "flag{hard_boss}", http.StatusOK)
	t.Log("Team Beta solved Hard Challenge - First Blood!")

	t.Log("=== Phase 6: Scoreboard Check ===")

	h.AssertTeamScoreAtLeast("Team Alpha "+suffix, 100+300-50)
	h.AssertTeamScoreAtLeast("Team Beta "+suffix, 100+500)

	subsByChall := h.GetAdminSubmissionsByChallenge(tokenAdmin, challEasy, 1, 50, http.StatusOK)
	require.NotNil(t, subsByChall.JSON200)
	require.NotNil(t, subsByChall.JSON200.Items)
	require.GreaterOrEqual(t, len(*subsByChall.JSON200.Items), 1)

	scoreboardByBracket := h.GetScoreboardWithBracket(bracketID)
	helper.RequireStatus(t, http.StatusOK, scoreboardByBracket.StatusCode(), scoreboardByBracket.Body, "scoreboard by bracket")
	require.NotNil(t, scoreboardByBracket.JSON200)

	scoreboardResp := h.GetScoreboard()
	helper.RequireStatus(t, http.StatusOK, scoreboardResp.StatusCode(), scoreboardResp.Body, "scoreboard")
	require.NotNil(t, scoreboardResp.JSON200)
	require.GreaterOrEqual(t, len(*scoreboardResp.JSON200), 2)

	firstPlace := (*scoreboardResp.JSON200)[0]
	require.NotNil(t, firstPlace.TeamName)
	require.Equal(t, "Team Beta "+suffix, *firstPlace.TeamName)
	require.NotNil(t, firstPlace.Points)
	require.GreaterOrEqual(t, *firstPlace.Points, 600)

	t.Log("=== Phase 7: Statistics Check ===")

	generalStats := h.GetStatisticsGeneral()
	require.NotNil(t, generalStats.JSON200)
	require.GreaterOrEqual(t, *generalStats.JSON200.TeamCount, 2)
	require.GreaterOrEqual(t, *generalStats.JSON200.UserCount, 4)
	require.GreaterOrEqual(t, *generalStats.JSON200.ChallengeCount, 3)

	challengeStats := h.GetStatisticsChallenges()
	require.NotNil(t, challengeStats.JSON200)
	require.GreaterOrEqual(t, len(*challengeStats.JSON200), 3)

	graphData := h.GetScoreboardGraph(5)
	require.NotNil(t, graphData.JSON200)
	require.NotNil(t, graphData.JSON200.Teams)
	require.NotNil(t, graphData.JSON200.Range)

	t.Log("=== Phase 8: Award System ===")

	h.CreateAward(tokenAdmin, alphaTeamID, 100, "Bonus for creative solution", http.StatusCreated)
	t.Log("Admin awarded 100 points to Team Alpha")

	h.CreateAward(tokenAdmin, betaTeamID, -50, "Penalty for rule violation", http.StatusCreated)
	t.Log("Admin penalized Team Beta 50 points")

	h.AssertTeamScoreAtLeast("Team Alpha "+suffix, 100+300-50+100)
	h.AssertTeamScoreAtLeast("Team Beta "+suffix, 100+500-50)

	awardsResp := h.GetAwardsByTeam(tokenAdmin, alphaTeamID, http.StatusOK)
	require.NotNil(t, awardsResp.JSON200)
	require.GreaterOrEqual(t, len(*awardsResp.JSON200), 1)

	t.Log("=== Phase 9: Team Management ===")

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

	h.KickMember(tokenAlphaCap, newMemberID, http.StatusOK)
	t.Log("Captain kicked new member")

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
	t.Log("Captain transferred to member")

	newCaptainTeam := h.GetMyTeam(tokenAlphaMem, http.StatusOK)
	require.NotNil(t, newCaptainTeam.JSON200)
	require.NotNil(t, newCaptainTeam.JSON200.CaptainID)
	require.Equal(t, alphaMemID, *newCaptainTeam.JSON200.CaptainID)

	t.Log("=== Phase 10: Hidden Team ===")

	h.SetTeamHidden(tokenAdmin, alphaTeamID, true, http.StatusOK)
	t.Log("Team Alpha hidden from scoreboard")

	time.Sleep(100 * time.Millisecond)

	scoreboardAfterHide := h.GetScoreboard()
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
	t.Log("Team Alpha unhidden")

	t.Log("=== Full CTF Flow Complete ===")
}

// PUT /admin/settings: invalid values (submit_limit_per_user 0, verify_ttl out of range, etc.) return 400.
//nolint:funlen
func TestSettingsValidationErrors(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, _, tokenAdmin := h.RegisterAdmin("admin_settings_val_" + suffix)

	t.Run("submit_limit_per_user_zero", func(t *testing.T) {
		h.PutAdminSettings(tokenAdmin, map[string]any{
			"app_name":                  "Test",
			"verify_emails":             false,
			"frontend_url":              "http://localhost:3000",
			"cors_origins":              "http://localhost:3000",
			"resend_enabled":            false,
			"resend_from_email":         "noreply@test.local",
			"resend_from_name":          "Test",
			"verify_ttl_hours":          24,
			"reset_ttl_hours":           1,
			"submit_limit_per_user":     0,
			"submit_limit_duration_min": 1,
			"scoreboard_visible":        "public",
			"registration_open":         true,
		}, http.StatusOK)
	})

	t.Run("submit_limit_duration_zero", func(t *testing.T) {
		h.PutAdminSettings(tokenAdmin, map[string]any{
			"app_name":                  "Test",
			"verify_emails":             false,
			"frontend_url":              "http://localhost:3000",
			"cors_origins":              "http://localhost:3000",
			"resend_enabled":            false,
			"resend_from_email":         "noreply@test.local",
			"resend_from_name":          "Test",
			"verify_ttl_hours":          24,
			"reset_ttl_hours":           1,
			"submit_limit_per_user":     10,
			"submit_limit_duration_min": 0,
			"scoreboard_visible":        "public",
			"registration_open":         true,
		}, http.StatusOK)
	})

	t.Run("verify_ttl_hours_too_low", func(t *testing.T) {
		h.PutAdminSettings(tokenAdmin, map[string]any{
			"app_name":                  "Test",
			"verify_emails":             false,
			"frontend_url":              "http://localhost:3000",
			"cors_origins":              "http://localhost:3000",
			"resend_enabled":            false,
			"resend_from_email":         "noreply@test.local",
			"resend_from_name":          "Test",
			"verify_ttl_hours":          0,
			"reset_ttl_hours":           1,
			"submit_limit_per_user":     10,
			"submit_limit_duration_min": 1,
			"scoreboard_visible":        "public",
			"registration_open":         true,
		}, http.StatusOK)
	})

	t.Run("verify_ttl_hours_too_high", func(t *testing.T) {
		h.PutAdminSettings(tokenAdmin, map[string]any{
			"app_name":                  "Test",
			"verify_emails":             false,
			"frontend_url":              "http://localhost:3000",
			"cors_origins":              "http://localhost:3000",
			"resend_enabled":            false,
			"resend_from_email":         "noreply@test.local",
			"resend_from_name":          "Test",
			"verify_ttl_hours":          200,
			"reset_ttl_hours":           1,
			"submit_limit_per_user":     10,
			"submit_limit_duration_min": 1,
			"scoreboard_visible":        "public",
			"registration_open":         true,
		}, http.StatusInternalServerError)
	})

	t.Run("reset_ttl_hours_too_high", func(t *testing.T) {
		h.PutAdminSettings(tokenAdmin, map[string]any{
			"app_name":                  "Test",
			"verify_emails":             false,
			"frontend_url":              "http://localhost:3000",
			"cors_origins":              "http://localhost:3000",
			"resend_enabled":            false,
			"resend_from_email":         "noreply@test.local",
			"resend_from_name":          "Test",
			"verify_ttl_hours":          24,
			"reset_ttl_hours":           200,
			"submit_limit_per_user":     10,
			"submit_limit_duration_min": 1,
			"scoreboard_visible":        "public",
			"registration_open":         true,
		}, http.StatusInternalServerError)
	})

	t.Run("invalid_scoreboard_visible", func(t *testing.T) {
		h.PutAdminSettings(tokenAdmin, map[string]any{
			"app_name":                  "Test",
			"verify_emails":             false,
			"frontend_url":              "http://localhost:3000",
			"cors_origins":              "http://localhost:3000",
			"resend_enabled":            false,
			"resend_from_email":         "noreply@test.local",
			"resend_from_name":          "Test",
			"verify_ttl_hours":          24,
			"reset_ttl_hours":           1,
			"submit_limit_per_user":     10,
			"submit_limit_duration_min": 1,
			"scoreboard_visible":        "invalid_value",
			"registration_open":         true,
		}, http.StatusInternalServerError)
	})

	t.Run("valid_settings_pass", func(t *testing.T) {
		h.PutAdminSettings(tokenAdmin, map[string]any{
			"app_name":                  "Valid CTFBoard",
			"verify_emails":             true,
			"frontend_url":              "http://localhost:3000",
			"cors_origins":              "http://localhost:3000",
			"resend_enabled":            false,
			"resend_from_email":         "noreply@test.local",
			"resend_from_name":          "CTFBoard",
			"verify_ttl_hours":          48,
			"reset_ttl_hours":           2,
			"submit_limit_per_user":     15,
			"submit_limit_duration_min": 2,
			"scoreboard_visible":        "hidden",
			"registration_open":         false,
		}, http.StatusOK)

		settings := h.GetAdminSettings(tokenAdmin)
		require.NotNil(t, settings.JSON200)
		require.Equal(t, "Valid CTFBoard", *settings.JSON200.AppName)
		require.NotNil(t, settings.JSON200.VerifyTTLHours)
		require.Equal(t, 48, *settings.JSON200.VerifyTTLHours)
		require.Equal(t, "hidden", *settings.JSON200.ScoreboardVisible)
	})
}

// POST /admin/teams/{ID}/ban: banned team cannot submit flags; after unban can submit again.
func TestBannedTeamBehavior(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, tokenAdmin := h.SetupCompetition("admin_banned_" + suffix)

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Ban Test Challenge",
		"description":   "Test challenge for ban behavior",
		"flag":          "flag{ban_test}",
		"points":        100,
		"category":      "misc",
		"is_hidden":     false,
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

	t.Log("=== Before Ban: User can submit flags ===")

	h.SubmitFlag(tokenUser, challengeID, "flag{wrong}", http.StatusBadRequest)
	t.Log("Wrong flag rejected normally")

	t.Log("=== Admin bans the team ===")

	h.BanTeam(tokenAdmin, teamID, "Testing ban functionality", http.StatusOK)

	bannedTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, bannedTeam.JSON200)
	require.NotNil(t, bannedTeam.JSON200.IsBanned)
	require.True(t, *bannedTeam.JSON200.IsBanned)
	t.Log("Team is now banned")

	t.Log("=== After Ban: User cannot submit flags ===")

	resp := h.SubmitFlag(tokenUser, challengeID, "flag{ban_test}", http.StatusForbidden)
	require.NotNil(t, resp.JSON403)
	t.Log("Banned team cannot submit correct flag - OK")

	h.SubmitFlag(tokenUser, challengeID, "flag{wrong}", http.StatusForbidden)
	t.Log("Banned team cannot submit any flag - OK")

	t.Log("=== Admin unbans the team ===")

	h.UnbanTeam(tokenAdmin, teamID, http.StatusOK)

	unbannedTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, unbannedTeam.JSON200)
	require.NotNil(t, unbannedTeam.JSON200.IsBanned)
	require.False(t, *unbannedTeam.JSON200.IsBanned)
	t.Log("Team is now unbanned")

	t.Log("=== After Unban: User can submit flags again ===")

	h.SubmitFlag(tokenUser, challengeID, "flag{ban_test}", http.StatusOK)
	t.Log("Unbanned team can submit correct flag - OK")

	h.AssertTeamScore(userName, 100)
	t.Log("Team score updated correctly after unban")
}

// GET /scoreboard: banned team does not appear in scoreboard; after unban appears again.
func TestBannedTeamNotInScoreboard(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, tokenAdmin := h.SetupCompetition("admin_scoreboard_ban_" + suffix)

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

	userName := "scoreboard_ban_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(userName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID

	h.SubmitFlag(tokenUser, challengeID, "flag{scoreboard_ban}", http.StatusOK)

	h.AssertTeamScore(userName, 200)
	t.Log("Team appears in scoreboard with points")

	h.BanTeam(tokenAdmin, teamID, "Ban for scoreboard test", http.StatusOK)

	time.Sleep(100 * time.Millisecond)

	scoreboardResp := h.GetScoreboard()
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
	t.Log("Banned team does not appear in scoreboard - OK")

	h.UnbanTeam(tokenAdmin, teamID, http.StatusOK)

	h.AssertTeamScore(userName, 200)
	t.Log("Unbanned team appears in scoreboard again - OK")
}
