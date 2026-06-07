package response

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestFromCompetitionStatusIncludesParticipationPolicy(t *testing.T) {
	t.Parallel()

	startedAt := time.Now().Add(-time.Hour)
	comp := &domain.Competition{
		Name:            "CTF",
		Mode:            domain.ModeTeamsOnly,
		StartTime:       &startedAt,
		AllowTeamSwitch: true,
		MinTeamSize:     2,
		MaxTeamSize:     5,
	}

	got := FromCompetitionStatus(comp)

	require.NotNil(t, got.AllowTeamSwitch)
	require.NotNil(t, got.MinTeamSize)
	require.NotNil(t, got.MaxTeamSize)
	assert.True(t, *got.AllowTeamSwitch)
	assert.Equal(t, 2, *got.MinTeamSize)
	assert.Equal(t, 5, *got.MaxTeamSize)
}

func TestTeamResponsesExposeSoloFlag(t *testing.T) {
	t.Parallel()

	team := &domain.Team{
		ID:          uuid.New(),
		Name:        "solo_alice",
		InviteToken: uuid.New(),
		CaptainID:   uuid.New(),
		IsSolo:      true,
		CreatedAt:   time.Now(),
	}

	got := FromTeam(team)
	require.NotNil(t, got.IsSolo)
	assert.True(t, *got.IsSolo)

	gotWithoutToken := FromTeamWithoutToken(team)
	require.NotNil(t, gotWithoutToken.IsSolo)
	assert.True(t, *gotWithoutToken.IsSolo)

	gotWithMembers := FromTeamWithMembers(team, nil, 1, true)
	require.NotNil(t, gotWithMembers.IsSolo)
	assert.True(t, *gotWithMembers.IsSolo)
}
