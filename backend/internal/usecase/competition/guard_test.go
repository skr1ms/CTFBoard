package competition

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	compMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mock"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type guardTestDeps struct {
	repo *compMock.MockCompetitionRepository
}

func newGuardTestDeps(t *testing.T) *guardTestDeps {
	t.Helper()

	return &guardTestDeps{repo: compMock.NewMockCompetitionRepository(t)}
}

func (d *guardTestDeps) createGuard() *Guard {
	return NewGuard(d.repo)
}

func newGuardCompetition(mode string, allowTeamSwitch bool) *domain.Competition {
	return &domain.Competition{
		Name:            "CTF",
		Mode:            domain.CompetitionMode(mode),
		AllowTeamSwitch: allowTeamSwitch,
	}
}

func TestGuard_Get_Success(t *testing.T) {
	t.Parallel()
	d := newGuardTestDeps(t)
	comp := newGuardCompetition("flexible", true)
	d.repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	got, err := d.createGuard().Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp, got)
}

func TestGuard_Get_Error(t *testing.T) {
	t.Parallel()
	d := newGuardTestDeps(t)
	d.repo.EXPECT().Get(mock.Anything).Return(nil, errors.New("db error")).Once()

	got, err := d.createGuard().Get(context.Background())

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGuard_RequireTeamSwitch_Success(t *testing.T) {
	t.Parallel()
	d := newGuardTestDeps(t)
	comp := newGuardCompetition("flexible", true)
	d.repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	got, err := d.createGuard().RequireTeamSwitch(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp, got)
}

func TestGuard_RequireTeamSwitch_Error(t *testing.T) {
	t.Parallel()
	d := newGuardTestDeps(t)
	comp := newGuardCompetition("flexible", false)
	d.repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	got, err := d.createGuard().RequireTeamSwitch(context.Background())

	assert.Error(t, err)
	assert.ErrorIs(t, err, httperr.ErrRosterFrozen)
	assert.Nil(t, got)
}

func TestGuard_RequireTeamSwitch_Paused_Error(t *testing.T) {
	t.Parallel()
	d := newGuardTestDeps(t)
	comp := newGuardCompetition("flexible", true)
	comp.IsPaused = true
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)
	comp.StartTime = &past
	comp.EndTime = &future
	d.repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	got, err := d.createGuard().RequireTeamSwitch(context.Background())

	assert.Error(t, err)
	assert.ErrorIs(t, err, httperr.ErrCompetitionPaused)
	assert.Nil(t, got)
}

func TestGuard_RequireTeamSwitchAndTeamsMode_Success(t *testing.T) {
	t.Parallel()
	d := newGuardTestDeps(t)
	comp := newGuardCompetition("flexible", true)
	d.repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	got, err := d.createGuard().RequireTeamSwitchAndTeamsMode(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp, got)
}

func TestGuard_RequireTeamSwitchAndTeamsMode_Error(t *testing.T) {
	t.Parallel()
	d := newGuardTestDeps(t)
	comp := newGuardCompetition("solo_only", true)
	d.repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	got, err := d.createGuard().RequireTeamSwitchAndTeamsMode(context.Background())

	assert.Error(t, err)
	assert.ErrorIs(t, err, httperr.ErrTeamsNotAllowed)
	assert.Nil(t, got)
}

func TestGuard_RequireTeamSwitchAndSoloMode_Success(t *testing.T) {
	t.Parallel()
	d := newGuardTestDeps(t)
	comp := newGuardCompetition("solo_only", true)
	d.repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	got, err := d.createGuard().RequireTeamSwitchAndSoloMode(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp, got)
}

func TestGuard_RequireTeamSwitchAndSoloMode_Error(t *testing.T) {
	t.Parallel()
	d := newGuardTestDeps(t)
	comp := newGuardCompetition("teams_only", true)
	d.repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	got, err := d.createGuard().RequireTeamSwitchAndSoloMode(context.Background())

	assert.Error(t, err)
	assert.ErrorIs(t, err, httperr.ErrSoloModeNotAllowed)
	assert.Nil(t, got)
}
