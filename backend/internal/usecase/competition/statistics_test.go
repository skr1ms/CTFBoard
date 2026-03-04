package competition

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStatisticsUseCase_GetGeneralStats_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	stats := &entity.GeneralStats{
		UserCount:      100,
		TeamCount:      20,
		ChallengeCount: 15,
		SolveCount:     50,
	}

	redisClient.ExpectGet("stats:general").SetErr(redis.Nil)
	deps.statsRepo.On("GetGeneralStats", mock.Anything).Return(stats, nil)
	redisClient.Regexp().ExpectSet("stats:general", `.*`, 5*time.Minute).SetVal("OK")

	result, err := uc.GetGeneralStats(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, stats.UserCount, result.UserCount)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetGeneralStats_Cached(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	stats := &entity.GeneralStats{UserCount: 100}
	bytes, err := json.Marshal(stats)
	require.NoError(t, err)
	redisClient.ExpectGet("stats:general").SetVal(string(bytes))

	result, err := uc.GetGeneralStats(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 100, result.UserCount)
	deps.statsRepo.AssertNotCalled(t, "GetGeneralStats", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetGeneralStats_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	redisClient.ExpectGet("stats:general").SetErr(redis.Nil)
	deps.statsRepo.On("GetGeneralStats", mock.Anything).Return(nil, errors.New("db error"))

	result, err := uc.GetGeneralStats(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetChallengeStats_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	stats := []*entity.ChallengeStats{
		{ID: uuid.New(), Title: "Chall 1", SolveCount: 10},
	}

	redisClient.ExpectGet("stats:challenges").SetErr(redis.Nil)
	deps.statsRepo.On("GetChallengeStats", mock.Anything).Return(stats, nil)
	redisClient.Regexp().ExpectSet("stats:challenges", `.*`, 5*time.Minute).SetVal("OK")

	result, err := uc.GetChallengeStats(context.Background())

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Chall 1", result[0].Title)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetScoreboardHistory_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	history := []*entity.ScoreboardHistoryEntry{
		{TeamID: uuid.New(), Points: 100, Timestamp: time.Now()},
	}

	redisClient.ExpectGet("stats:history:10").SetErr(redis.Nil)
	deps.statsRepo.On("GetScoreboardHistory", mock.Anything, 10).Return(history, nil)
	redisClient.Regexp().ExpectSet("stats:history:10", `.*`, 30*time.Second).SetVal("OK")

	result, err := uc.GetScoreboardHistory(context.Background(), 10)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetChallengeStats_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	redisClient.ExpectGet("stats:challenges").SetErr(redis.Nil)
	deps.statsRepo.On("GetChallengeStats", mock.Anything).Return(nil, errors.New("db error"))

	result, err := uc.GetChallengeStats(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetScoreboardHistory_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	redisClient.ExpectGet("stats:history:10").SetErr(redis.Nil)
	deps.statsRepo.On("GetScoreboardHistory", mock.Anything, 10).Return(nil, errors.New("db error"))

	result, err := uc.GetScoreboardHistory(context.Background(), 10)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetScoreboardGraph_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	history := []*entity.ScoreboardHistoryEntry{
		{TeamID: uuid.New(), TeamName: "Team1", Points: 100, Timestamp: time.Now()},
	}

	redisClient.ExpectGet("stats:graph:10").SetErr(redis.Nil)
	deps.statsRepo.On("GetScoreboardHistory", mock.Anything, 10).Return(history, nil)
	redisClient.Regexp().ExpectSet("stats:graph:10", `.*`, 30*time.Second).SetVal("OK")

	result, err := uc.GetScoreboardGraph(context.Background(), 10)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetScoreboardGraph_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	redisClient.ExpectGet("stats:graph:10").SetErr(redis.Nil)
	deps.statsRepo.On("GetScoreboardHistory", mock.Anything, 10).Return(nil, errors.New("db error"))

	result, err := uc.GetScoreboardGraph(context.Background(), 10)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetChallengeDetailStats_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	challengeID := uuid.New()
	stats := &entity.ChallengeDetailStats{
		ID:         challengeID,
		Title:      "Challenge 1",
		Category:   "Web",
		Points:     100,
		SolveCount: 5,
		TotalTeams: 10,
	}

	redisClient.ExpectGet("stats:challenge:" + challengeID.String()).SetErr(redis.Nil)
	deps.statsRepo.On("GetChallengeDetailStats", mock.Anything, challengeID).Return(stats, nil)
	redisClient.Regexp().ExpectSet("stats:challenge:"+challengeID.String(), `.*`, time.Minute).SetVal("OK")

	result, err := uc.GetChallengeDetailStats(context.Background(), challengeID.String())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Challenge 1", result.Title)
	assert.Equal(t, 5, result.SolveCount)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetChallengeDetailStats_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	challengeID := uuid.New()
	redisClient.ExpectGet("stats:challenge:" + challengeID.String()).SetErr(redis.Nil)
	deps.statsRepo.On("GetChallengeDetailStats", mock.Anything, challengeID).Return(nil, errors.New("db error"))

	result, err := uc.GetChallengeDetailStats(context.Background(), challengeID.String())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetTeamRegistrationTimeSeries_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	data := []*entity.RegistrationTimePoint{{Date: "2025-01-01", Count: 5}}
	redisClient.ExpectGet("stats:team_registration").SetErr(redis.Nil)
	deps.statsRepo.On("GetTeamRegistrationTimeSeries", mock.Anything).Return(data, nil)
	redisClient.Regexp().ExpectSet("stats:team_registration", `.*`, 5*time.Minute).SetVal("OK")

	result, err := uc.GetTeamRegistrationTimeSeries(context.Background())

	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "2025-01-01", result[0].Date)
	assert.Equal(t, 5, result[0].Count)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetTeamRegistrationTimeSeries_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	redisClient.ExpectGet("stats:team_registration").SetErr(redis.Nil)
	deps.statsRepo.On("GetTeamRegistrationTimeSeries", mock.Anything).Return(nil, errors.New("db error"))

	result, err := uc.GetTeamRegistrationTimeSeries(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetUserRegistrationTimeSeries_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	data := []*entity.RegistrationTimePoint{{Date: "2025-01-01", Count: 10}}
	redisClient.ExpectGet("stats:user_registration").SetErr(redis.Nil)
	deps.statsRepo.On("GetUserRegistrationTimeSeries", mock.Anything).Return(data, nil)
	redisClient.Regexp().ExpectSet("stats:user_registration", `.*`, 5*time.Minute).SetVal("OK")

	result, err := uc.GetUserRegistrationTimeSeries(context.Background())

	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, 10, result[0].Count)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetUserRegistrationTimeSeries_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	redisClient.ExpectGet("stats:user_registration").SetErr(redis.Nil)
	deps.statsRepo.On("GetUserRegistrationTimeSeries", mock.Anything).Return(nil, errors.New("db error"))

	result, err := uc.GetUserRegistrationTimeSeries(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetSolveMatrix_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	matrix := []*entity.SolveMatrixRow{
		{TeamID: uuid.New(), TeamName: "T1", ChallengeID: uuid.New(), ChallengeTitle: "C1", Solved: true},
	}
	redisClient.ExpectGet("stats:solve_matrix").SetErr(redis.Nil)
	deps.statsRepo.On("GetSolveMatrix", mock.Anything).Return(matrix, nil)
	redisClient.Regexp().ExpectSet("stats:solve_matrix", `.*`, 30*time.Second).SetVal("OK")

	result, err := uc.GetSolveMatrix(context.Background())

	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "T1", result[0].TeamName)
	assert.True(t, result[0].Solved)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetSolveMatrix_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	redisClient.ExpectGet("stats:solve_matrix").SetErr(redis.Nil)
	deps.statsRepo.On("GetSolveMatrix", mock.Anything).Return(nil, errors.New("db error"))

	result, err := uc.GetSolveMatrix(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetSubmissionTimeSeriesByType_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	data := []*entity.RegistrationTimePoint{{Date: "2025-01-01", Count: 3}}
	redisClient.ExpectGet("stats:submission_timeseries:true").SetErr(redis.Nil)
	deps.statsRepo.On("GetSubmissionTimeSeriesByType", mock.Anything, true).Return(data, nil)
	redisClient.Regexp().ExpectSet("stats:submission_timeseries:true", `.*`, 5*time.Minute).SetVal("OK")

	result, err := uc.GetSubmissionTimeSeriesByType(context.Background(), true)

	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, 3, result[0].Count)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetSubmissionTimeSeriesByType_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateStatisticsUseCase()

	redisClient.ExpectGet("stats:submission_timeseries:false").SetErr(redis.Nil)
	deps.statsRepo.On("GetSubmissionTimeSeriesByType", mock.Anything, false).Return(nil, errors.New("db error"))

	result, err := uc.GetSubmissionTimeSeriesByType(context.Background(), false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}
