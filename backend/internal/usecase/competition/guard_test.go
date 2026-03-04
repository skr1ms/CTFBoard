package competition

import (
	"context"
	"errors"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGuard_Get_Success(t *testing.T) {
	t.Parallel()
	h := NewGuardTestHelper(t)
	comp := h.NewCompetition("flexible", true)
	h.Repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	got, err := h.CreateGuard().Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp, got)
}

func TestGuard_Get_Error(t *testing.T) {
	t.Parallel()
	h := NewGuardTestHelper(t)
	h.Repo.EXPECT().Get(mock.Anything).Return(nil, errors.New("db error")).Once()

	got, err := h.CreateGuard().Get(context.Background())

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGuard_RequireTeamSwitch_Success(t *testing.T) {
	t.Parallel()
	h := NewGuardTestHelper(t)
	comp := h.NewCompetition("flexible", true)
	h.Repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	got, err := h.CreateGuard().RequireTeamSwitch(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp, got)
}

func TestGuard_RequireTeamSwitch_Error(t *testing.T) {
	t.Parallel()
	h := NewGuardTestHelper(t)
	comp := h.NewCompetition("flexible", false)
	h.Repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	got, err := h.CreateGuard().RequireTeamSwitch(context.Background())

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrRosterFrozen))
	assert.Nil(t, got)
}

func TestGuard_RequireTeamSwitchAndTeamsMode_Success(t *testing.T) {
	t.Parallel()
	h := NewGuardTestHelper(t)
	comp := h.NewCompetition("flexible", true)
	h.Repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	got, err := h.CreateGuard().RequireTeamSwitchAndTeamsMode(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp, got)
}

func TestGuard_RequireTeamSwitchAndTeamsMode_Error(t *testing.T) {
	t.Parallel()
	h := NewGuardTestHelper(t)
	comp := h.NewCompetition("solo_only", true)
	h.Repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	got, err := h.CreateGuard().RequireTeamSwitchAndTeamsMode(context.Background())

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrTeamsNotAllowed))
	assert.Nil(t, got)
}

func TestGuard_RequireTeamSwitchAndSoloMode_Success(t *testing.T) {
	t.Parallel()
	h := NewGuardTestHelper(t)
	comp := h.NewCompetition("solo_only", true)
	h.Repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	got, err := h.CreateGuard().RequireTeamSwitchAndSoloMode(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp, got)
}

func TestGuard_RequireTeamSwitchAndSoloMode_Error(t *testing.T) {
	t.Parallel()
	h := NewGuardTestHelper(t)
	comp := h.NewCompetition("teams_only", true)
	h.Repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	got, err := h.CreateGuard().RequireTeamSwitchAndSoloMode(context.Background())

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrSoloModeNotAllowed))
	assert.Nil(t, got)
}
