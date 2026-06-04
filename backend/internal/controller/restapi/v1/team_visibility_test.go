package v1

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestTeamStatsVisibleToViewer_PublicTeam(t *testing.T) {
	t.Parallel()

	team := &domain.Team{ID: uuid.New(), IsHidden: false}

	assert.True(t, teamStatsVisibleToViewer(team, nil))
}

func TestTeamStatsVisibleToViewer_HiddenTeamGuest(t *testing.T) {
	t.Parallel()

	team := &domain.Team{ID: uuid.New(), IsHidden: true}

	assert.False(t, teamStatsVisibleToViewer(team, nil))
}

func TestTeamStatsVisibleToViewer_HiddenTeamOwner(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	team := &domain.Team{ID: teamID, IsHidden: true}
	viewer := &domain.User{TeamID: &teamID, Role: domain.RoleUser}

	assert.True(t, teamStatsVisibleToViewer(team, viewer))
}

func TestTeamStatsVisibleToViewer_HiddenTeamOtherUser(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	otherTeamID := uuid.New()
	team := &domain.Team{ID: teamID, IsHidden: true}
	viewer := &domain.User{TeamID: &otherTeamID, Role: domain.RoleUser}

	assert.False(t, teamStatsVisibleToViewer(team, viewer))
}

func TestTeamStatsVisibleToViewer_HiddenTeamAdmin(t *testing.T) {
	t.Parallel()

	team := &domain.Team{ID: uuid.New(), IsHidden: true}
	viewer := &domain.User{Role: domain.RoleAdmin}

	assert.True(t, teamStatsVisibleToViewer(team, viewer))
}
