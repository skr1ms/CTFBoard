package competition

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestCompetitionUseCase_Update_InvalidTimesAfterMergeReturnsError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(2 * time.Hour)
	endTime := time.Now().Add(24 * time.Hour)
	currentNotStarted := newTestCompetitionWithTimes("CTF", &startTime, &endTime)
	currentNotStarted.Mode = domain.ModeTeamsOnly
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentNotStarted, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	invalidEndTime := time.Now().Add(1 * time.Hour)
	comp := &domain.Competition{ID: 1, Name: "CTF", EndTime: &invalidEndTime, Mode: domain.ModeTeamsOnly}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "end_time must be after start_time")
	d.competitionRepo.AssertNotCalled(t, "Update", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_RejectsFlexibleMode(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	currentNotStarted := newTestCompetition("CTF", "teams_only", true)
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentNotStarted, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	comp := &domain.Competition{ID: 1, Name: "CTF", Mode: domain.CompetitionMode("flexible")}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be solo_only or teams_only")
	d.competitionRepo.AssertNotCalled(t, "Update", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_FreezeTimeEqualEndTime_ReturnsError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(2 * time.Hour)
	sameTime := time.Now().Add(24 * time.Hour)
	currentNotStarted := newTestCompetitionWithTimes("CTF", &startTime, &sameTime)
	currentNotStarted.Mode = domain.ModeTeamsOnly
	currentNotStarted.FreezeTime = nil
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentNotStarted, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	comp := &domain.Competition{ID: 1, Name: "CTF", StartTime: &startTime, EndTime: &sameTime, FreezeTime: &sameTime, Mode: domain.ModeTeamsOnly}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "freeze_time must be before end_time")
	d.competitionRepo.AssertNotCalled(t, "Update", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_EndedAllowsUpdate(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(-48 * time.Hour)
	endTime := time.Now().Add(-1 * time.Hour)
	currentEnded := newTestCompetitionWithTimes("CTF", &startTime, &endTime)
	currentEnded.Mode = domain.ModeTeamsOnly
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentEnded, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return c.Name == "Updated"
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "Updated", Mode: domain.ModeTeamsOnly}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_ActiveRejectsTeamSizeChange(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	currentActive := newTestCompetitionWithTimes("Active CTF", &startTime, &endTime)
	currentActive.Mode = domain.ModeTeamsOnly
	currentActive.MinTeamSize = 1
	currentActive.MaxTeamSize = 5
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(withDefaultTeamSizes(currentActive), nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	comp := newTestCompetition("Updated CTF", "teams_only", true)
	comp.MinTeamSize = 2
	comp.MaxTeamSize = 6
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrCompetitionActiveCannotUpdate)
	d.competitionRepo.AssertNotCalled(t, "Update", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_ActiveAllowsEndTimeChange_ForceEnd(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-1 * time.Hour)
	endTime := now.Add(23 * time.Hour)
	currentActive := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeTeamsOnly,
		StartTime: &startTime, EndTime: &endTime,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(withDefaultTeamSizes(currentActive), nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return c.EndTime != nil && c.EndTime.Before(now)
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	forceEndTime := now.Add(-1 * time.Minute)
	comp := &domain.Competition{ID: 1, Name: "CTF", StartTime: &startTime, EndTime: &forceEndTime, Mode: domain.ModeTeamsOnly}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
}
