package competition

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCompetitionUseCase_Get_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateCompetitionUseCase()

	comp := h.NewCompetition("Test CTF", "flexible", true)

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	deps.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	redisClient.Regexp().ExpectSet(cache.KeyCompetition, `.*`, 30*time.Second).SetVal("OK")

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp.Name, result.Name)
	assert.Equal(t, comp.Mode, result.Mode)
	assert.Equal(t, comp.AllowTeamSwitch, result.AllowTeamSwitch)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Get_Cached_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateCompetitionUseCase()

	comp := h.NewCompetition("Test CTF", "flexible", true)
	bytes, err := json.Marshal(comp)
	require.NoError(t, err)

	redisClient.ExpectGet(cache.KeyCompetition).SetVal(string(bytes))

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp.Name, result.Name)
	deps.competitionRepo.AssertNotCalled(t, "Get", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Get_NotFound_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateCompetitionUseCase()

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	deps.competitionRepo.On("Get", mock.Anything).Return(nil, httperr.ErrCompetitionNotFound)

	result, err := uc.Get(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, httperr.ErrCompetitionNotFound)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateCompetitionUseCase()

	comp := h.NewCompetition("Updated CTF", "flexible", true)
	comp.MinTeamSize = 1
	comp.MaxTeamSize = 5

	currentNotStarted := h.NewCompetitionWithTimes("Current", timePtr(time.Now().Add(24*time.Hour)), nil)
	deps.competitionRepo.EXPECT().Get(mock.Anything).Return(currentNotStarted, nil).Once()

	deps.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	deps.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *entity.Competition) bool {
		return c.ID == comp.ID &&
			c.Name == comp.Name &&
			c.Mode == comp.Mode &&
			c.AllowTeamSwitch == comp.AllowTeamSwitch &&
			c.MinTeamSize == comp.MinTeamSize &&
			c.MaxTeamSize == comp.MaxTeamSize
	})).Return(nil).Once()
	deps.auditLogRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(a *entity.AuditLog) bool {
		return a.Action == entity.AuditActionUpdate && a.EntityType == entity.AuditEntityCompetition
	})).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	err := uc.Update(context.Background(), comp, uuid.New(), "127.0.0.1")

	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func timePtr(t time.Time) *time.Time { return &t }

func TestCompetitionUseCase_Update_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateCompetitionUseCase()

	comp := h.NewCompetition("Updated CTF", "flexible", true)

	currentNotStarted := h.NewCompetitionWithTimes("Current", timePtr(time.Now().Add(24*time.Hour)), nil)
	deps.competitionRepo.EXPECT().Get(mock.Anything).Return(currentNotStarted, nil).Once()

	deps.tm.EXPECT().Run(mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

	err := uc.Update(context.Background(), comp, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_ActiveCompetitionRejectsDangerousChanges(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateCompetitionUseCase()

	// Current competition is active
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	currentActive := h.NewCompetitionWithTimes("Active CTF", &startTime, &endTime)
	currentActive.Mode = entity.ModeFlexible
	currentActive.AllowTeamSwitch = true

	deps.competitionRepo.EXPECT().Get(mock.Anything).Return(currentActive, nil).Once()

	// Try to change mode (dangerous when active)
	comp := h.NewCompetition("Updated CTF", "teams_only", true) // different mode

	err := uc.Update(context.Background(), comp, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.ErrorIs(t, err, httperr.ErrCompetitionActiveCannotUpdate)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_GetStatus_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateCompetitionUseCase()

	startTime := time.Now().Add(-1 * time.Hour)
	comp := h.NewCompetitionWithTimes("Test CTF", &startTime, nil)

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	deps.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	redisClient.ExpectSet(cache.KeyCompetition, mock.Anything, 5*time.Second).SetVal("OK")

	status, err := uc.GetStatus(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, entity.CompetitionStatusActive, status)
}

func TestCompetitionUseCase_GetStatus_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateCompetitionUseCase()

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	deps.competitionRepo.On("Get", mock.Anything).Return(nil, errors.New("db error"))

	status, err := uc.GetStatus(context.Background())

	assert.Error(t, err)
	assert.Empty(t, status)
}

func TestCompetitionUseCase_IsSubmissionAllowed_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateCompetitionUseCase()

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	comp := h.NewCompetitionWithTimes("Test CTF", &startTime, &endTime)

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	deps.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	redisClient.ExpectSet(cache.KeyCompetition, mock.Anything, 5*time.Second).SetVal("OK")

	allowed, err := uc.IsSubmissionAllowed(context.Background())

	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestCompetitionUseCase_IsSubmissionAllowed_NotStarted_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateCompetitionUseCase()

	startTime := time.Now().Add(1 * time.Hour)
	comp := h.NewCompetitionWithTimes("Test CTF", &startTime, nil)

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	deps.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	redisClient.ExpectSet(cache.KeyCompetition, mock.Anything, 5*time.Second).SetVal("OK")

	allowed, err := uc.IsSubmissionAllowed(context.Background())

	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestCompetitionUseCase_IsSubmissionAllowed_Ended_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateCompetitionUseCase()

	startTime := time.Now().Add(-2 * time.Hour)
	endTime := time.Now().Add(-1 * time.Hour)
	comp := h.NewCompetitionWithTimes("Test CTF", &startTime, &endTime)

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	deps.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	redisClient.ExpectSet(cache.KeyCompetition, mock.Anything, 5*time.Second).SetVal("OK")

	allowed, err := uc.IsSubmissionAllowed(context.Background())

	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestCompetitionUseCase_IsSubmissionAllowed_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateCompetitionUseCase()

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	deps.competitionRepo.On("Get", mock.Anything).Return(nil, errors.New("db error"))

	allowed, err := uc.IsSubmissionAllowed(context.Background())

	assert.Error(t, err)
	assert.False(t, allowed)
}
