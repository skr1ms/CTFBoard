package guard

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// stubCounter is a minimal TeamMemberCounter for testing.
type stubCounter struct {
	count int
	err   error
}

func (s *stubCounter) CountTeamMembers(_ context.Context, _ uuid.UUID) (int, error) {
	return s.count, s.err
}

// ---- EnsureChallengeVisible ----

func TestEnsureChallengeVisible_Visible(t *testing.T) {
	t.Parallel()

	ch := &domain.Challenge{State: domain.ChallengeStateVisible}

	require.NoError(t, EnsureChallengeVisible(ch))
}

func TestEnsureChallengeVisible_Hidden(t *testing.T) {
	t.Parallel()

	ch := &domain.Challenge{State: domain.ChallengeStateHidden}

	require.ErrorIs(t, EnsureChallengeVisible(ch), apperr.ErrChallengeNotFound)
}

// ---- ValidateSubmissionEligibility ----

func activeComp() *domain.Competition {
	start := time.Now().Add(-time.Hour)
	end := time.Now().Add(time.Hour)

	return &domain.Competition{
		StartTime: &start,
		EndTime:   &end,
		Mode:      domain.ModeFlexible,
	}
}

func TestValidateSubmissionEligibility_BannedUser(t *testing.T) {
	t.Parallel()

	user := &domain.User{IsBanned: true}
	team := &domain.Team{ID: uuid.New(), IsSolo: true}

	err := ValidateSubmissionEligibility(context.Background(), user, team, activeComp(), nil)

	require.ErrorIs(t, err, apperr.ErrUserBanned)
}

func TestValidateSubmissionEligibility_UserWasInBannedTeam(t *testing.T) {
	t.Parallel()

	user := &domain.User{WasInBannedTeam: true, Role: domain.RoleUser}
	team := &domain.Team{ID: uuid.New(), IsSolo: true}

	err := ValidateSubmissionEligibility(context.Background(), user, team, activeComp(), nil)

	require.ErrorIs(t, err, apperr.ErrUserWasInBannedTeam)
}

func TestValidateSubmissionEligibility_AdminWasInBannedTeamAllowed(t *testing.T) {
	t.Parallel()

	user := &domain.User{WasInBannedTeam: true, Role: domain.RoleAdmin}
	team := &domain.Team{ID: uuid.New(), IsSolo: true}

	err := ValidateSubmissionEligibility(context.Background(), user, team, activeComp(), nil)

	require.NoError(t, err)
}

func TestValidateSubmissionEligibility_BannedTeam(t *testing.T) {
	t.Parallel()

	user := &domain.User{IsBanned: false}
	team := &domain.Team{ID: uuid.New(), IsBanned: true, IsSolo: false}

	err := ValidateSubmissionEligibility(context.Background(), user, team, activeComp(), nil)

	require.ErrorIs(t, err, apperr.ErrTeamBanned)
}

func TestValidateSubmissionEligibility_NilTeam_OK(t *testing.T) {
	t.Parallel()

	user := &domain.User{IsBanned: false}

	err := ValidateSubmissionEligibility(context.Background(), user, nil, activeComp(), nil)

	require.NoError(t, err)
}

func TestValidateSubmissionEligibility_NilComp_OK(t *testing.T) {
	t.Parallel()

	user := &domain.User{IsBanned: false}
	team := &domain.Team{ID: uuid.New(), IsSolo: true}

	err := ValidateSubmissionEligibility(context.Background(), user, team, nil, nil)

	require.NoError(t, err)
}

func TestValidateSubmissionEligibility_SoloInTeamMode(t *testing.T) {
	t.Parallel()

	user := &domain.User{IsBanned: false}
	team := &domain.Team{ID: uuid.New(), IsSolo: true}

	start := time.Now().Add(-time.Hour)
	comp := &domain.Competition{StartTime: &start, Mode: domain.ModeTeamsOnly}

	err := ValidateSubmissionEligibility(context.Background(), user, team, comp, nil)

	require.ErrorIs(t, err, apperr.ErrTeamModeRequired)
}

func TestValidateSubmissionEligibility_TeamInSoloMode(t *testing.T) {
	t.Parallel()

	user := &domain.User{IsBanned: false}
	team := &domain.Team{ID: uuid.New(), IsSolo: false}

	start := time.Now().Add(-time.Hour)
	comp := &domain.Competition{StartTime: &start, Mode: domain.ModeSoloOnly}

	err := ValidateSubmissionEligibility(context.Background(), user, team, comp, nil)

	require.ErrorIs(t, err, apperr.ErrSoloModeRequired)
}

func TestValidateSubmissionEligibility_BelowMinTeamSize(t *testing.T) {
	t.Parallel()

	user := &domain.User{IsBanned: false}
	team := &domain.Team{ID: uuid.New(), IsSolo: false}

	start := time.Now().Add(-time.Hour)
	comp := &domain.Competition{StartTime: &start, Mode: domain.ModeFlexible, MinTeamSize: 3}
	counter := &stubCounter{count: 1}

	err := ValidateSubmissionEligibility(context.Background(), user, team, comp, counter)

	require.ErrorIs(t, err, apperr.ErrTeamBelowMinSize)
}

func TestValidateSubmissionEligibility_MinTeamSizeMet(t *testing.T) {
	t.Parallel()

	user := &domain.User{IsBanned: false}
	team := &domain.Team{ID: uuid.New(), IsSolo: false}

	start := time.Now().Add(-time.Hour)
	comp := &domain.Competition{StartTime: &start, Mode: domain.ModeFlexible, MinTeamSize: 2}
	counter := &stubCounter{count: 3}

	err := ValidateSubmissionEligibility(context.Background(), user, team, comp, counter)

	require.NoError(t, err)
}

func TestValidateSubmissionEligibility_CounterError(t *testing.T) {
	t.Parallel()

	user := &domain.User{IsBanned: false}
	team := &domain.Team{ID: uuid.New(), IsSolo: false}

	start := time.Now().Add(-time.Hour)
	comp := &domain.Competition{StartTime: &start, Mode: domain.ModeFlexible, MinTeamSize: 2}
	counter := &stubCounter{count: 0, err: errors.New("db error")}

	err := ValidateSubmissionEligibility(context.Background(), user, team, comp, counter)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestValidateSubmissionEligibility_NilUser_OK(t *testing.T) {
	t.Parallel()

	team := &domain.Team{ID: uuid.New(), IsSolo: true}

	err := ValidateSubmissionEligibility(context.Background(), nil, team, activeComp(), nil)

	require.NoError(t, err)
}

// ---- ValidateTeamSwitchState ----

func TestValidateTeamSwitchState_Ended(t *testing.T) {
	t.Parallel()

	start := time.Now().Add(-2 * time.Hour)
	end := time.Now().Add(-time.Hour)
	comp := &domain.Competition{StartTime: &start, EndTime: &end, AllowTeamSwitch: true}

	require.ErrorIs(t, ValidateTeamSwitchState(comp), apperr.ErrCompetitionEnded)
}

func TestValidateTeamSwitchState_Paused(t *testing.T) {
	t.Parallel()

	start := time.Now().Add(-time.Hour)
	comp := &domain.Competition{StartTime: &start, IsPaused: true, AllowTeamSwitch: true}

	require.ErrorIs(t, ValidateTeamSwitchState(comp), apperr.ErrCompetitionPaused)
}

func TestValidateTeamSwitchState_RosterFrozen(t *testing.T) {
	t.Parallel()

	start := time.Now().Add(-time.Hour)
	comp := &domain.Competition{StartTime: &start, AllowTeamSwitch: false}

	require.ErrorIs(t, ValidateTeamSwitchState(comp), apperr.ErrRosterFrozen)
}

func TestValidateTeamSwitchState_Active_AllowSwitch(t *testing.T) {
	t.Parallel()

	start := time.Now().Add(-time.Hour)
	end := time.Now().Add(time.Hour)
	comp := &domain.Competition{StartTime: &start, EndTime: &end, AllowTeamSwitch: true}

	require.NoError(t, ValidateTeamSwitchState(comp))
}

func TestValidateTeamSwitchState_NotStarted_AllowSwitch(t *testing.T) {
	t.Parallel()

	// StartTime in future -> not started
	start := time.Now().Add(time.Hour)
	comp := &domain.Competition{StartTime: &start, AllowTeamSwitch: true}

	require.NoError(t, ValidateTeamSwitchState(comp))
}
