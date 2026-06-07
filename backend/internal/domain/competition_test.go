package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCompetitionMode_IsValid_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ModeSoloOnly.IsValid())
	assert.True(t, ModeTeamsOnly.IsValid())
	assert.Equal(t, ModeTeamsOnly, DefaultCompetitionMode)
}

func TestCompetitionMode_IsValid_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, CompetitionMode("").IsValid())
	assert.False(t, CompetitionMode("flexible").IsValid())
	assert.False(t, CompetitionMode("invalid").IsValid())
}

func TestCompetitionMode_AllowsSolo_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ModeSoloOnly.AllowsSolo())
}

func TestCompetitionMode_AllowsSolo_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, ModeTeamsOnly.AllowsSolo())
	assert.False(t, CompetitionMode("").AllowsSolo())
	assert.False(t, CompetitionMode("invalid").AllowsSolo())
}

func TestCompetitionMode_AllowsTeams_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, ModeTeamsOnly.AllowsTeams())
}

func TestCompetitionMode_AllowsTeams_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, ModeSoloOnly.AllowsTeams())
	assert.False(t, CompetitionMode("").AllowsTeams())
	assert.False(t, CompetitionMode("flexible").AllowsTeams())
	assert.False(t, CompetitionMode("invalid").AllowsTeams())
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

func TestCompetition_GetStatus_PausedBlocksEnded(t *testing.T) {
	t.Parallel()

	now := time.Now()
	startPast := now.Add(-2 * time.Hour)
	endPast := now.Add(-1 * time.Hour)
	pausedAt := now.Add(-90 * time.Minute)

	c := &Competition{
		StartTime: &startPast,
		EndTime:   &endPast,
		IsPaused:  true,
		PausedAt:  &pausedAt,
	}
	assert.Equal(t, CompetitionStatusPaused, c.GetStatus())
}

func TestCompetition_GetStatus_PausedBlocksFrozen(t *testing.T) {
	t.Parallel()

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	freezePast := now.Add(-30 * time.Minute)

	c := &Competition{
		StartTime:  &past,
		EndTime:    &future,
		IsPaused:   true,
		FreezeTime: &freezePast,
	}
	assert.Equal(t, CompetitionStatusPaused, c.GetStatus())
}

func TestCompetition_GetStatus_AfterUnpauseFreezeInFuture_Active(t *testing.T) {
	t.Parallel()

	now := time.Now()
	startPast := now.Add(-2 * time.Hour)
	freezeFuture := now.Add(30 * time.Minute)
	endFuture := now.Add(time.Hour)

	c := &Competition{
		StartTime:  &startPast,
		EndTime:    &endFuture,
		FreezeTime: &freezeFuture,
		IsPaused:   false,
	}
	assert.Equal(t, CompetitionStatusActive, c.GetStatus())
	assert.False(t, c.IsFreezeActive())
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

func TestCompetition_IsSubmissionAllowed_Frozen_Allowed(t *testing.T) {
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

func TestCompetition_IsFreezeActive_NotStarted(t *testing.T) {
	t.Parallel()

	future := time.Now().Add(time.Hour)
	c := &Competition{StartTime: &future, FreezeTime: &future}
	assert.False(t, c.IsFreezeActive())
}

func TestCompetition_IsFreezeActive_NoFreezeTime(t *testing.T) {
	t.Parallel()

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	c := &Competition{StartTime: &past, EndTime: &future, FreezeTime: nil}
	assert.False(t, c.IsFreezeActive())
}

func TestCompetition_IsFreezeActive_BeforeFreeze(t *testing.T) {
	t.Parallel()

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	freezeFuture := now.Add(30 * time.Minute)
	c := &Competition{StartTime: &past, EndTime: &future, FreezeTime: &freezeFuture}
	assert.False(t, c.IsFreezeActive())
}

func TestCompetition_IsFreezeActive_AfterFreeze(t *testing.T) {
	t.Parallel()

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	freezePast := now.Add(-30 * time.Minute)
	c := &Competition{StartTime: &past, EndTime: &future, FreezeTime: &freezePast}
	assert.True(t, c.IsFreezeActive())
}

func TestCompetition_IsFreezeActive_PausedButPastFreeze(t *testing.T) {
	t.Parallel()

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	freezePast := now.Add(-30 * time.Minute)
	c := &Competition{
		StartTime:  &past,
		EndTime:    &future,
		IsPaused:   true,
		FreezeTime: &freezePast,
	}
	assert.Equal(t, CompetitionStatusPaused, c.GetStatus())
	assert.True(t, c.IsFreezeActive())
}

func TestCompetition_IsFreezeActive_AfterEnd_Inactive(t *testing.T) {
	t.Parallel()

	now := time.Now()
	startPast := now.Add(-2 * time.Hour)
	endPast := now.Add(-time.Hour)
	freezePast := now.Add(-90 * time.Minute)
	c := &Competition{
		StartTime:  &startPast,
		EndTime:    &endPast,
		IsPaused:   false,
		FreezeTime: &freezePast,
	}
	assert.Equal(t, CompetitionStatusEnded, c.GetStatus())
	assert.False(t, c.IsFreezeActive())
}

func TestCompetition_IsFreezeActive_PausedAndEnded_StillActive(t *testing.T) {
	t.Parallel()

	now := time.Now()
	startPast := now.Add(-2 * time.Hour)
	endPast := now.Add(-time.Hour)
	freezePast := now.Add(-90 * time.Minute)
	pausedAt := now.Add(-30 * time.Minute)
	c := &Competition{
		StartTime:  &startPast,
		EndTime:    &endPast,
		IsPaused:   true,
		PausedAt:   &pausedAt,
		FreezeTime: &freezePast,
	}
	assert.Equal(t, CompetitionStatusPaused, c.GetStatus())
	assert.True(t, c.IsFreezeActive())
}

func TestCompetition_GetStatusAt_MatchesGetStatus(t *testing.T) {
	t.Parallel()

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	c := &Competition{StartTime: &past, EndTime: &future, IsPaused: false}
	assert.Equal(t, c.GetStatus(), c.GetStatusAt(now))
}

func TestCompetition_IsSubmissionAllowedAt_MatchesIsSubmissionAllowed(t *testing.T) {
	t.Parallel()

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	c := &Competition{StartTime: &past, EndTime: &future, IsPaused: false}
	assert.Equal(t, c.IsSubmissionAllowed(), c.IsSubmissionAllowedAt(now))
}

func TestCompetition_IsEffectivelyEnded_Ended(t *testing.T) {
	t.Parallel()

	now := time.Now()
	startPast := now.Add(-2 * time.Hour)
	endPast := now.Add(-time.Hour)
	c := &Competition{StartTime: &startPast, EndTime: &endPast, IsPaused: false}
	assert.True(t, c.IsEffectivelyEnded(now))
}

func TestCompetition_IsEffectivelyEnded_PausedButEndTimePassed(t *testing.T) {
	t.Parallel()

	now := time.Now()
	startPast := now.Add(-2 * time.Hour)
	endPast := now.Add(-time.Hour)
	c := &Competition{StartTime: &startPast, EndTime: &endPast, IsPaused: true}
	assert.True(t, c.IsEffectivelyEnded(now))
}

func TestCompetition_IsEffectivelyEnded_Active(t *testing.T) {
	t.Parallel()

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	c := &Competition{StartTime: &past, EndTime: &future, IsPaused: false}
	assert.False(t, c.IsEffectivelyEnded(now))
}
