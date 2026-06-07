package competition

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func TestCompetitionUseCase_Update_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	comp := newTestCompetition("Updated CTF", "solo_only", true)
	comp.MinTeamSize = 1
	comp.MaxTeamSize = 5

	currentNotStarted := newTestCompetitionWithTimes("Current", new(time.Now().Add(24*time.Hour)), nil)
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentNotStarted, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return c.ID == comp.ID &&
			c.Name == comp.Name &&
			c.Mode == comp.Mode &&
			c.AllowTeamSwitch == comp.AllowTeamSwitch &&
			c.MinTeamSize == comp.MinTeamSize &&
			c.MaxTeamSize == comp.MaxTeamSize
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(a *domain.AuditLog) bool {
		return a.Action == domain.AuditActionUpdate && a.EntityType == domain.AuditEntityCompetition
	})).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	comp := newTestCompetition("Updated CTF", "solo_only", true)

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_ActiveCompetitionRejectsDangerousChanges(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	// Current competition is active
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	currentActive := newTestCompetitionWithTimes("Active CTF", &startTime, &endTime)
	currentActive.Mode = domain.ModeTeamsOnly
	currentActive.AllowTeamSwitch = true

	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentActive, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	comp := newTestCompetition("Updated CTF", "solo_only", true)
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrCompetitionActiveCannotUpdate)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_PartialUpdatePreservesBooleans(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(23 * time.Hour)
	currentPaused := newTestCompetitionWithTimes("CTF", &startTime, &endTime)
	currentPaused.Mode = domain.ModeTeamsOnly
	currentPaused.IsPaused = true
	currentPaused.IsPublic = true
	currentPaused.AllowTeamSwitch = false
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return c.Name == "Updated Name" && c.IsPaused && c.IsPublic && !c.AllowTeamSwitch
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "Updated Name", Mode: domain.ModeTeamsOnly}
	optionals := &usecase.CompetitionUpdateOptionals{}
	err := uc.Update(context.Background(), comp, optionals, uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
}
