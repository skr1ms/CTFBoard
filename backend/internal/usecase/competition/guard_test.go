package competition

import (
	"context"
	"errors"
	"testing"

	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGuard_Get_Success(t *testing.T) {
	repo := mocks.NewMockCompetitionRepository(t)
	comp := &entity.Competition{Name: "CTF", Mode: "flexible", AllowTeamSwitch: true}
	repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	g := NewGuard(repo)
	got, err := g.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp, got)
}

func TestGuard_Get_Error(t *testing.T) {
	repo := mocks.NewMockCompetitionRepository(t)
	repo.EXPECT().Get(mock.Anything).Return(nil, errors.New("db error")).Once()

	g := NewGuard(repo)
	got, err := g.Get(context.Background())

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGuard_RequireTeamSwitch_Success(t *testing.T) {
	repo := mocks.NewMockCompetitionRepository(t)
	comp := &entity.Competition{AllowTeamSwitch: true}
	repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	g := NewGuard(repo)
	got, err := g.RequireTeamSwitch(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp, got)
}

func TestGuard_RequireTeamSwitch_Error(t *testing.T) {
	repo := mocks.NewMockCompetitionRepository(t)
	comp := &entity.Competition{AllowTeamSwitch: false}
	repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	g := NewGuard(repo)
	got, err := g.RequireTeamSwitch(context.Background())

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrRosterFrozen))
	assert.Nil(t, got)
}

func TestGuard_RequireTeamSwitchAndTeamsMode_Success(t *testing.T) {
	repo := mocks.NewMockCompetitionRepository(t)
	comp := &entity.Competition{Mode: "flexible", AllowTeamSwitch: true}
	repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	g := NewGuard(repo)
	got, err := g.RequireTeamSwitchAndTeamsMode(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp, got)
}

func TestGuard_RequireTeamSwitchAndTeamsMode_Error(t *testing.T) {
	repo := mocks.NewMockCompetitionRepository(t)
	comp := &entity.Competition{Mode: "solo_only", AllowTeamSwitch: true}
	repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	g := NewGuard(repo)
	got, err := g.RequireTeamSwitchAndTeamsMode(context.Background())

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSoloModeNotAllowed))
	assert.Nil(t, got)
}

func TestGuard_RequireTeamSwitchAndSoloMode_Success(t *testing.T) {
	repo := mocks.NewMockCompetitionRepository(t)
	comp := &entity.Competition{Mode: "solo_only", AllowTeamSwitch: true}
	repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	g := NewGuard(repo)
	got, err := g.RequireTeamSwitchAndSoloMode(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp, got)
}

func TestGuard_RequireTeamSwitchAndSoloMode_Error(t *testing.T) {
	repo := mocks.NewMockCompetitionRepository(t)
	comp := &entity.Competition{Mode: "teams_only", AllowTeamSwitch: true}
	repo.EXPECT().Get(mock.Anything).Return(comp, nil).Once()

	g := NewGuard(repo)
	got, err := g.RequireTeamSwitchAndSoloMode(context.Background())

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSoloModeNotAllowed))
	assert.Nil(t, got)
}
