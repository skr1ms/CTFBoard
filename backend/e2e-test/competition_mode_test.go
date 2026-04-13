package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

func setCompetitionMode(mode string) {
	ctx := context.Background()

	_, err := TestPool.Exec(ctx,
		`UPDATE competition SET mode = $1, allow_team_switch = true, updated_at = NOW() WHERE id = 1`,
		mode)
	if err != nil {
		panic("setCompetitionMode: " + err.Error())
	}

	_ = TestRedis.Del(ctx, "competition")
}

func resetCompetitionModeToFlexible() {
	ctx := context.Background()
	now := time.Now().UTC()

	_, err := TestPool.Exec(ctx,
		`UPDATE competition SET mode = 'flexible', is_paused = false,
		 start_time = $1, end_time = $2, freeze_time = NULL,
		 allow_team_switch = true, updated_at = NOW() WHERE id = 1`,
		now.Add(-1*time.Hour), now.Add(24*time.Hour))
	if err != nil {
		panic("resetCompetitionModeToFlexible: " + err.Error())
	}

	_ = TestRedis.Del(ctx, "competition")
}

func activateCompetition() {
	now := time.Now().UTC()
	setCompetitionTimes(now.Add(-1*time.Hour), now.Add(24*time.Hour), nil)
}

// POST /auth/register: solo_only mode auto-creates a solo team so the user can submit flags immediately.
func TestMode_SoloOnly_AutoTeamOnRegistration(t *testing.T) {
	t.Cleanup(resetCompetitionModeToFlexible)

	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	suffix := helper.UID()

	setCompetitionMode("solo_only")
	activateCompetition()

	_, _, tokenAdmin := h.RegisterAdmin("admin_solo_auto_" + suffix)
	challID := h.CreateBasicChallenge(tokenAdmin, "Solo Auto Challenge "+suffix, "flag{solo_auto_"+suffix+"}", 100)

	userName := "solo_auto_user_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(userName)

	myTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, myTeam.JSON200, "user should have auto-created solo team")
	require.NotNil(t, myTeam.JSON200.ID)

	resp := h.SubmitFlag(tokenUser, challID, "flag{solo_auto_"+suffix+"}", http.StatusOK)
	require.NotNil(t, resp.JSON200)
	assert.True(t, resp.JSON200.Correct, "correct flag should be accepted")
}

// POST /teams: solo_only mode returns 403 when creating a regular (non-solo) team.
func TestMode_SoloOnly_CannotCreateRegularTeam(t *testing.T) {
	t.Cleanup(resetCompetitionModeToFlexible)

	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	suffix := helper.UID()

	setCompetitionMode("solo_only")
	activateCompetition()

	userName := "solo_noteam_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(userName)
	h.CreateTeam(tokenUser, "RegularTeam_"+suffix, http.StatusForbidden)
}

// POST /teams/join: solo_only mode returns 403 when joining a team via invite token.
func TestMode_SoloOnly_CannotJoinTeam(t *testing.T) {
	t.Cleanup(resetCompetitionModeToFlexible)

	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	suffix := helper.UID()

	setCompetitionMode("solo_only")
	activateCompetition()

	_, _, tokenUser := h.RegisterUserAndLogin("solo_join_" + suffix)
	h.JoinTeam(tokenUser, uuid.New().String(), false, http.StatusForbidden)
}

// POST /challenges/{ID}/submit: solo_only mode returns 200 correct=false for a wrong flag.
func TestMode_SoloOnly_WrongFlagReturnsIncorrect(t *testing.T) {
	t.Cleanup(resetCompetitionModeToFlexible)

	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	suffix := helper.UID()

	setCompetitionMode("solo_only")
	activateCompetition()

	_, _, tokenAdmin := h.RegisterAdmin("admin_solo_wrong_" + suffix)
	challID := h.CreateBasicChallenge(tokenAdmin, "Solo Wrong Challenge "+suffix, "flag{correct_"+suffix+"}", 50)

	_, _, tokenUser := h.RegisterUserAndLogin("solo_wrong_" + suffix)
	resp := h.SubmitFlag(tokenUser, challID, "flag{wrong}", http.StatusOK)
	require.NotNil(t, resp.JSON200)
	assert.False(t, resp.JSON200.Correct, "wrong flag should not be accepted")
}

// POST /challenges/{ID}/submit: teams_only mode returns 403 without a team.
func TestMode_TeamsOnly_CannotSubmitWithoutTeam(t *testing.T) {
	t.Cleanup(resetCompetitionModeToFlexible)

	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	suffix := helper.UID()

	setCompetitionMode("teams_only")
	activateCompetition()

	_, _, tokenAdmin := h.RegisterAdmin("admin_teams_nosubmit_" + suffix)
	challID := h.CreateBasicChallenge(tokenAdmin, "Teams Only Submit "+suffix, "flag{teams_nosubmit_"+suffix+"}", 100)

	_, _, tokenUser := h.RegisterUserAndLogin("teams_nosubmit_" + suffix)
	h.SubmitFlagExpectStatus(tokenUser, challID, "flag{teams_nosubmit_"+suffix+"}", http.StatusForbidden, http.StatusNotFound)
}

// POST /teams + POST /challenges/{ID}/submit: teams_only mode allows submission after creating a team.
func TestMode_TeamsOnly_SubmitAfterCreateTeam(t *testing.T) {
	t.Cleanup(resetCompetitionModeToFlexible)

	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	suffix := helper.UID()

	setCompetitionMode("teams_only")
	activateCompetition()

	_, _, tokenAdmin := h.RegisterAdmin("admin_teams_submit_" + suffix)
	challID := h.CreateBasicChallenge(tokenAdmin, "Teams Submit "+suffix, "flag{teams_submit_"+suffix+"}", 100)

	_, _, tokenUser := h.RegisterUserAndLogin("teams_submit_" + suffix)
	h.CreateTeam(tokenUser, "Team_"+suffix, http.StatusCreated)

	resp := h.SubmitFlag(tokenUser, challID, "flag{teams_submit_"+suffix+"}", http.StatusOK)
	require.NotNil(t, resp.JSON200)
	assert.True(t, resp.JSON200.Correct, "correct flag should be accepted after joining team")
}

// POST /teams/solo: teams_only mode returns 403.
func TestMode_TeamsOnly_CannotCreateSoloTeam(t *testing.T) {
	t.Cleanup(resetCompetitionModeToFlexible)

	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	suffix := helper.UID()

	setCompetitionMode("teams_only")
	activateCompetition()

	_, _, tokenUser := h.RegisterUserAndLogin("teams_nosolo_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusForbidden)
}

// POST /challenges/{ID}/submit: teams_only mode rejects solo team on submission (TOCTOU - mode changed after solo team was created).
func TestMode_TeamsOnly_SoloTeamCannotSubmit(t *testing.T) {
	t.Cleanup(resetCompetitionModeToFlexible)

	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	suffix := helper.UID()

	setCompetitionMode("flexible")
	activateCompetition()

	_, _, tokenAdmin := h.RegisterAdmin("admin_teams_solo_" + suffix)
	challID := h.CreateBasicChallenge(tokenAdmin, "Teams Solo Submit "+suffix, "flag{teams_solo_"+suffix+"}", 100)

	_, _, tokenUser := h.RegisterUserAndLogin("teams_solo_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	myTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)

	setCompetitionMode("teams_only")

	h.SubmitFlag(tokenUser, challID, "flag{teams_solo_"+suffix+"}", http.StatusForbidden)
}
