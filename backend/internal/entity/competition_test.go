package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCompetitionMode_IsValid_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ModeFlexible.IsValid())
	assert.True(t, ModeSoloOnly.IsValid())
	assert.True(t, ModeTeamsOnly.IsValid())
}

func TestCompetitionMode_IsValid_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, CompetitionMode("").IsValid())
	assert.False(t, CompetitionMode("invalid").IsValid())
}

func TestCompetitionMode_AllowsSolo_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ModeSoloOnly.AllowsSolo())
	assert.True(t, ModeFlexible.AllowsSolo())
}

func TestCompetitionMode_AllowsSolo_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, ModeTeamsOnly.AllowsSolo())
}

func TestCompetitionMode_AllowsTeams_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ModeTeamsOnly.AllowsTeams())
	assert.True(t, ModeFlexible.AllowsTeams())
}

func TestCompetitionMode_AllowsTeams_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, ModeSoloOnly.AllowsTeams())
}

func TestCompetition_GetStatus_Success(t *testing.T) {
	t.Parallel()
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	c := &Competition{
		StartTime:  &past,
		EndTime:    &future,
		IsPaused:   false,
		FreezeTime: nil,
	}
	assert.Equal(t, CompetitionStatusActive, c.GetStatus())
}

func TestCompetition_GetStatus_NotStarted(t *testing.T) {
	t.Parallel()
	future := time.Now().Add(time.Hour)
	c := &Competition{StartTime: &future}
	assert.Equal(t, CompetitionStatusNotStarted, c.GetStatus())
}

func TestCompetition_GetStatus_Ended(t *testing.T) {
	t.Parallel()
	now := time.Now()
	past := now.Add(-time.Hour)
	startPast := now.Add(-2 * time.Hour)

	c := &Competition{
		StartTime: &startPast,
		EndTime:   &past,
	}
	assert.Equal(t, CompetitionStatusEnded, c.GetStatus())
}

func TestCompetition_GetStatus_Paused(t *testing.T) {
	t.Parallel()
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	c := &Competition{
		StartTime: &past,
		EndTime:   &future,
		IsPaused:  true,
	}
	assert.Equal(t, CompetitionStatusPaused, c.GetStatus())
}

func TestCompetition_GetStatus_Frozen(t *testing.T) {
	t.Parallel()
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	freezePast := now.Add(-30 * time.Minute)

	c := &Competition{
		StartTime:  &past,
		EndTime:    &future,
		IsPaused:   false,
		FreezeTime: &freezePast,
	}
	assert.Equal(t, CompetitionStatusFrozen, c.GetStatus())
}

func TestCompetition_IsSubmissionAllowed_Success(t *testing.T) {
	t.Parallel()
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	c := &Competition{
		StartTime:  &past,
		EndTime:    &future,
		IsPaused:   false,
		FreezeTime: nil,
	}
	assert.True(t, c.IsSubmissionAllowed())
}

func TestCompetition_IsSubmissionAllowed_Error(t *testing.T) {
	t.Parallel()
	future := time.Now().Add(time.Hour)
	c := &Competition{StartTime: &future}
	assert.False(t, c.IsSubmissionAllowed())

	now := time.Now()
	past := now.Add(-time.Hour)
	future = now.Add(time.Hour)
	paused := &Competition{
		StartTime: &past,
		EndTime:   &future,
		IsPaused:  true,
	}
	assert.False(t, paused.IsSubmissionAllowed())
}
