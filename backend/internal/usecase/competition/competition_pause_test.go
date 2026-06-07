package competition

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestCompetitionUseCase_Update_PauseSetsTimestamp(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(23 * time.Hour)
	currentActive := newTestCompetitionWithTimes("CTF", &startTime, &endTime)
	currentActive.Mode = domain.ModeTeamsOnly
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(withDefaultTeamSizes(currentActive), nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return c.IsPaused && c.PausedAt != nil && time.Since(*c.PausedAt) < time.Second
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: true, Mode: domain.ModeTeamsOnly, AllowTeamSwitch: true}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
}

func TestCompetitionUseCase_Update_UnpauseShiftsEndTime(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-10 * time.Hour)
	endTime := now.Add(14 * time.Hour)
	freezeTime := now.Add(12 * time.Hour)
	pausedAt := now.Add(-2 * time.Hour)

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeTeamsOnly,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(withDefaultTeamSizes(currentPaused), nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		if c.IsPaused || c.PausedAt != nil {
			return false
		}

		shift := time.Since(pausedAt)
		endShift := c.EndTime.Sub(endTime)
		freezeShift := c.FreezeTime.Sub(freezeTime)

		return endShift > shift-time.Second && endShift < shift+time.Second &&
			c.FreezeTime != nil &&
			freezeShift > shift-time.Second && freezeShift < shift+time.Second
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: domain.ModeTeamsOnly}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
}

func TestCompetitionUseCase_Update_UnpauseWithEndTimeInPastForceEnds(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-10 * time.Hour)
	endTime := now.Add(-2 * time.Hour)
	pausedAt := now.Add(-1 * time.Hour)

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeTeamsOnly,
		StartTime: &startTime, EndTime: &endTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(withDefaultTeamSizes(currentPaused), nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return !c.IsPaused && c.PausedAt == nil && c.EndTime.Equal(endTime)
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: domain.ModeTeamsOnly}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
}

func TestCompetitionUseCase_Update_UnpauseWhenPausedBeforeEndTimeShiftsTimes(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-10 * time.Hour)
	endTime := now.Add(-1 * time.Hour)
	freezeTime := now.Add(-2 * time.Hour)
	pausedAt := now.Add(-3 * time.Hour)

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeTeamsOnly,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(withDefaultTeamSizes(currentPaused), nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		if c.IsPaused || c.PausedAt != nil {
			return false
		}

		shift := time.Since(pausedAt)
		endShift := c.EndTime.Sub(endTime)
		freezeShift := c.FreezeTime.Sub(freezeTime)

		return endShift > shift-time.Second && endShift < shift+time.Second &&
			c.FreezeTime != nil &&
			freezeShift > shift-time.Second && freezeShift < shift+time.Second
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: domain.ModeTeamsOnly}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
}

func TestCompetitionUseCase_Update_UnpauseValidationRejectsFreezeAfterEnd(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-10 * time.Hour)
	endTime := now.Add(2 * time.Hour)
	freezeTime := now.Add(1 * time.Hour)
	pausedAt := now.Add(-1 * time.Hour)

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeTeamsOnly,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(withDefaultTeamSizes(currentPaused), nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	adminEndTime := now.Add(30 * time.Minute)
	comp := &domain.Competition{ID: 1, Name: "CTF", EndTime: &adminEndTime, IsPaused: false, Mode: domain.ModeTeamsOnly}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "freeze_time must be before end_time") ||
		strings.Contains(err.Error(), "unpausing shifts freeze_time"),
		"error should mention freeze_time or unpause shift: %s", err.Error())
	d.competitionRepo.AssertNotCalled(t, "Update", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_UnpauseAfterPreStartPause_ClampsToStartTime(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-2 * time.Hour)
	endTime := now.Add(14 * time.Hour)
	freezeTime := now.Add(12 * time.Hour)
	pausedAt := now.Add(-3 * time.Hour)

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeTeamsOnly,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(withDefaultTeamSizes(currentPaused), nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		if c.IsPaused || c.PausedAt != nil {
			return false
		}

		effectiveShift := time.Since(startTime)
		endShift := c.EndTime.Sub(endTime)
		freezeShift := c.FreezeTime.Sub(freezeTime)

		return endShift > effectiveShift-time.Second && endShift < effectiveShift+time.Second &&
			c.FreezeTime != nil &&
			freezeShift > effectiveShift-time.Second && freezeShift < effectiveShift+time.Second
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: domain.ModeTeamsOnly}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_UnpauseKeepsFreezeTimeUnchanged(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-10 * time.Hour)
	endTime := now.Add(14 * time.Hour)
	freezeTime := now.Add(-1 * time.Hour)
	pausedAt := now.Add(-30 * time.Minute)

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeTeamsOnly,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(withDefaultTeamSizes(currentPaused), nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return c.FreezeTime != nil && c.FreezeTime.Equal(freezeTime)
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: domain.ModeTeamsOnly}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
}

func TestCompetitionUseCase_Update_UnpauseWithNilEndTime_ShiftsFreezeTime(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-2 * time.Hour)
	freezeTime := now.Add(1 * time.Hour)
	pausedAt := now.Add(-30 * time.Minute)

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeTeamsOnly,
		StartTime: &startTime, EndTime: nil, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(withDefaultTeamSizes(currentPaused), nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		if c.IsPaused || c.PausedAt != nil || c.EndTime != nil {
			return false
		}

		shift := time.Since(pausedAt)
		freezeShift := c.FreezeTime.Sub(freezeTime)

		return c.FreezeTime != nil &&
			freezeShift > shift-time.Second && freezeShift < shift+time.Second
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: domain.ModeTeamsOnly}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_UnpauseBeforeStartTime_NoShift(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(2 * time.Hour)
	endTime := now.Add(6 * time.Hour)
	freezeTime := now.Add(5 * time.Hour)
	pausedAt := now.Add(-1 * time.Hour)

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeTeamsOnly,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(withDefaultTeamSizes(currentPaused), nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return !c.IsPaused && c.PausedAt == nil &&
			c.EndTime != nil && c.EndTime.Equal(endTime) &&
			c.FreezeTime != nil && c.FreezeTime.Equal(freezeTime)
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: domain.ModeTeamsOnly}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_UnpauseWithChangedEndTime_StillShiftsFreezeTime(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-10 * time.Hour)
	endTime := now.Add(2 * time.Hour)
	freezeTime := now.Add(1 * time.Hour)
	pausedAt := now.Add(-1 * time.Hour)

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeTeamsOnly,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(withDefaultTeamSizes(currentPaused), nil).Once()

	adminEndTime := now.Add(4 * time.Hour)

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		if c.IsPaused || c.PausedAt != nil {
			return false
		}

		if c.EndTime == nil || !c.EndTime.Equal(adminEndTime) {
			return false
		}

		shift := time.Since(pausedAt)
		freezeShift := c.FreezeTime.Sub(freezeTime)

		return c.FreezeTime != nil &&
			freezeShift > shift-time.Second && freezeShift < shift+time.Second
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "CTF", EndTime: &adminEndTime, IsPaused: false, Mode: domain.ModeTeamsOnly}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}
