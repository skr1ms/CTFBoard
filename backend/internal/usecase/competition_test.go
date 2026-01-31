package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/usecase/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Get Tests

func TestCompetitionUseCase_Get_Success(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	db, redisClient := redismock.NewClientMock()
	auditLogRepo := mocks.NewMockAuditLogRepository(t)

	comp := &entity.Competition{
		Id:              1,
		Name:            "Test CTF",
		Mode:            "flexible",
		AllowTeamSwitch: true,
	}

	redisClient.ExpectGet("competition").SetErr(redis.Nil)
	competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	redisClient.Regexp().ExpectSet("competition", `.*`, 5*time.Second).SetVal("OK")

	uc := NewCompetitionUseCase(competitionRepo, auditLogRepo, db)

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp.Name, result.Name)
	assert.Equal(t, comp.Mode, result.Mode)
	assert.Equal(t, comp.AllowTeamSwitch, result.AllowTeamSwitch)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Get_Cached_Success(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	db, redisClient := redismock.NewClientMock()
	auditLogRepo := mocks.NewMockAuditLogRepository(t)

	comp := &entity.Competition{
		Id:   1,
		Name: "Test CTF",
	}
	bytes, _ := json.Marshal(comp)

	redisClient.ExpectGet("competition").SetVal(string(bytes))

	uc := NewCompetitionUseCase(competitionRepo, auditLogRepo, db)

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp.Name, result.Name)
	competitionRepo.AssertNotCalled(t, "Get", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Get_NotFound_Error(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	db, redisClient := redismock.NewClientMock()
	auditLogRepo := mocks.NewMockAuditLogRepository(t)

	redisClient.ExpectGet("competition").SetErr(redis.Nil)
	competitionRepo.On("Get", mock.Anything).Return(nil, entityError.ErrCompetitionNotFound)

	uc := NewCompetitionUseCase(competitionRepo, auditLogRepo, db)

	result, err := uc.Get(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, entityError.ErrCompetitionNotFound)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

// Update Tests

func TestCompetitionUseCase_Update_Success(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	db, redisClient := redismock.NewClientMock()

	comp := &entity.Competition{
		Id:              1,
		Name:            "Updated CTF",
		Mode:            "flexible",
		AllowTeamSwitch: true,
		MinTeamSize:     1,
		MaxTeamSize:     5,
	}

	competitionRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Competition) bool {
		return c.Id == comp.Id &&
			c.Name == comp.Name &&
			c.Mode == comp.Mode &&
			c.AllowTeamSwitch == comp.AllowTeamSwitch &&
			c.MinTeamSize == comp.MinTeamSize &&
			c.MaxTeamSize == comp.MaxTeamSize
	})).Return(nil)
	redisClient.ExpectDel("competition").SetVal(1)

	auditLogRepo := mocks.NewMockAuditLogRepository(t)
	auditLogRepo.On("Create", mock.Anything, mock.MatchedBy(func(a *entity.AuditLog) bool {
		return a.Action == entity.AuditActionUpdate && a.EntityType == entity.AuditEntityCompetition
	})).Return(nil)

	compUC := NewCompetitionUseCase(competitionRepo, auditLogRepo, db)
	err := compUC.Update(context.Background(), comp, uuid.New(), "127.0.0.1")

	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_Error(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	db, redisClient := redismock.NewClientMock()
	auditLogRepo := mocks.NewMockAuditLogRepository(t)

	comp := &entity.Competition{
		Id:   1,
		Name: "Updated CTF",
	}

	competitionRepo.On("Update", mock.Anything, comp).Return(errors.New("db error"))

	uc := NewCompetitionUseCase(competitionRepo, auditLogRepo, db)

	err := uc.Update(context.Background(), comp, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

// GetStatus Tests

func TestCompetitionUseCase_GetStatus_Success(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	db, redisClient := redismock.NewClientMock()
	auditLogRepo := mocks.NewMockAuditLogRepository(t)

	startTime := time.Now().Add(-1 * time.Hour)
	comp := &entity.Competition{
		Id:        1,
		Name:      "Test CTF",
		StartTime: &startTime,
	}

	redisClient.ExpectGet("competition").SetErr(redis.Nil)

	competitionRepo.On("Get", mock.Anything).Return(comp, nil)

	redisClient.ExpectSet("competition", mock.Anything, 5*time.Second).SetVal("OK")

	uc := NewCompetitionUseCase(competitionRepo, auditLogRepo, db)

	status, err := uc.GetStatus(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, entity.CompetitionStatusActive, status)
}

func TestCompetitionUseCase_GetStatus_Error(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	db, redisClient := redismock.NewClientMock()
	auditLogRepo := mocks.NewMockAuditLogRepository(t)

	redisClient.ExpectGet("competition").SetErr(redis.Nil)

	competitionRepo.On("Get", mock.Anything).Return(nil, errors.New("db error"))

	uc := NewCompetitionUseCase(competitionRepo, auditLogRepo, db)

	status, err := uc.GetStatus(context.Background())

	assert.Error(t, err)
	assert.Empty(t, status)
}

// IsSubmissionAllowed Tests

func TestCompetitionUseCase_IsSubmissionAllowed_Success(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	db, redisClient := redismock.NewClientMock()
	auditLogRepo := mocks.NewMockAuditLogRepository(t)

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	comp := &entity.Competition{
		Id:        1,
		Name:      "Test CTF",
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	redisClient.ExpectGet("competition").SetErr(redis.Nil)

	competitionRepo.On("Get", mock.Anything).Return(comp, nil)

	redisClient.ExpectSet("competition", mock.Anything, 5*time.Second).SetVal("OK")

	uc := NewCompetitionUseCase(competitionRepo, auditLogRepo, db)

	allowed, err := uc.IsSubmissionAllowed(context.Background())

	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestCompetitionUseCase_IsSubmissionAllowed_NotStarted_Success(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	db, redisClient := redismock.NewClientMock()
	auditLogRepo := mocks.NewMockAuditLogRepository(t)

	startTime := time.Now().Add(1 * time.Hour)
	comp := &entity.Competition{
		Id:        1,
		Name:      "Test CTF",
		StartTime: &startTime,
	}

	redisClient.ExpectGet("competition").SetErr(redis.Nil)

	competitionRepo.On("Get", mock.Anything).Return(comp, nil)

	redisClient.ExpectSet("competition", mock.Anything, 5*time.Second).SetVal("OK")

	uc := NewCompetitionUseCase(competitionRepo, auditLogRepo, db)

	allowed, err := uc.IsSubmissionAllowed(context.Background())

	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestCompetitionUseCase_IsSubmissionAllowed_Ended_Success(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	db, redisClient := redismock.NewClientMock()
	auditLogRepo := mocks.NewMockAuditLogRepository(t)

	startTime := time.Now().Add(-2 * time.Hour)
	endTime := time.Now().Add(-1 * time.Hour)
	comp := &entity.Competition{
		Id:        1,
		Name:      "Test CTF",
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	redisClient.ExpectGet("competition").SetErr(redis.Nil)

	competitionRepo.On("Get", mock.Anything).Return(comp, nil)

	redisClient.ExpectSet("competition", mock.Anything, 5*time.Second).SetVal("OK")

	uc := NewCompetitionUseCase(competitionRepo, auditLogRepo, db)

	allowed, err := uc.IsSubmissionAllowed(context.Background())

	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestCompetitionUseCase_IsSubmissionAllowed_Error(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	db, redisClient := redismock.NewClientMock()
	auditLogRepo := mocks.NewMockAuditLogRepository(t)

	redisClient.ExpectGet("competition").SetErr(redis.Nil)

	competitionRepo.On("Get", mock.Anything).Return(nil, errors.New("db error"))

	uc := NewCompetitionUseCase(competitionRepo, auditLogRepo, db)

	allowed, err := uc.IsSubmissionAllowed(context.Background())

	assert.Error(t, err)
	assert.False(t, allowed)
}
