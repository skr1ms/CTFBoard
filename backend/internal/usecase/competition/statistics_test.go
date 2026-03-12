package competition

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
)

func TestStatisticsUseCase_GetGeneralStats_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	stats := &entity.GeneralStats{
		UserCount:      100,
		TeamCount:      20,
		ChallengeCount: 15,
		SolveCount:     50,
	}

	redisClient.ExpectGet("stats:general").SetErr(redis.Nil)
	d.statsRepo.On("GetGeneralStats", mock.Anything).Return(stats, nil)
	redisClient.Regexp().ExpectSet("stats:general", `.*`, 5*time.Minute).SetVal("OK")

	result, err := uc.GetGeneralStats(context.Background(), false)

	assert.NoError(t, err)
	assert.Equal(t, stats.UserCount, result.UserCount)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetGeneralStats_Cached(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	stats := &entity.GeneralStats{UserCount: 100}
	bytes, err := json.Marshal(stats)
	require.NoError(t, err)
	redisClient.ExpectGet("stats:general").SetVal(string(bytes))

	result, err := uc.GetGeneralStats(context.Background(), false)

	assert.NoError(t, err)
	assert.Equal(t, 100, result.UserCount)
	d.statsRepo.AssertNotCalled(t, "GetGeneralStats", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetGeneralStats_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	redisClient.ExpectGet("stats:general").SetErr(redis.Nil)
	d.statsRepo.On("GetGeneralStats", mock.Anything).Return(nil, errors.New("db error"))

	result, err := uc.GetGeneralStats(context.Background(), false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetChallengeStats_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	stats := []*entity.ChallengeStats{
		{ID: uuid.New(), Title: "Chall 1", SolveCount: 10},
	}

	redisClient.ExpectGet("stats:challenges").SetErr(redis.Nil)
	d.statsRepo.On("GetChallengeStats", mock.Anything).Return(stats, nil)
	redisClient.Regexp().ExpectSet("stats:challenges", `.*`, 5*time.Minute).SetVal("OK")

	result, err := uc.GetChallengeStats(context.Background(), false)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Chall 1", result[0].Title)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetScoreboardHistory_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	history := []*entity.ScoreboardHistoryEntry{
		{TeamID: uuid.New(), Points: 100, Timestamp: time.Now()},
	}

	redisClient.ExpectGet("stats:history:10").SetErr(redis.Nil)
	d.statsRepo.On("GetScoreboardHistory", mock.Anything, 10).Return(history, nil)
	redisClient.Regexp().ExpectSet("stats:history:10", `.*`, 30*time.Second).SetVal("OK")

	result, err := uc.GetScoreboardHistory(context.Background(), 10, false)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetChallengeStats_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	redisClient.ExpectGet("stats:challenges").SetErr(redis.Nil)
	d.statsRepo.On("GetChallengeStats", mock.Anything).Return(nil, errors.New("db error"))

	result, err := uc.GetChallengeStats(context.Background(), false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetScoreboardHistory_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	redisClient.ExpectGet("stats:history:10").SetErr(redis.Nil)
	d.statsRepo.On("GetScoreboardHistory", mock.Anything, 10).Return(nil, errors.New("db error"))

	result, err := uc.GetScoreboardHistory(context.Background(), 10, false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetScoreboardGraph_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	history := []*entity.ScoreboardHistoryEntry{
		{TeamID: uuid.New(), TeamName: "Team1", Points: 100, Timestamp: time.Now()},
	}

	redisClient.ExpectGet("stats:graph:10").SetErr(redis.Nil)
	d.statsRepo.On("GetScoreboardHistory", mock.Anything, 10).Return(history, nil)
	redisClient.Regexp().ExpectSet("stats:graph:10", `.*`, 30*time.Second).SetVal("OK")

	result, err := uc.GetScoreboardGraph(context.Background(), 10, false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetScoreboardGraph_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	redisClient.ExpectGet("stats:graph:10").SetErr(redis.Nil)
	d.statsRepo.On("GetScoreboardHistory", mock.Anything, 10).Return(nil, errors.New("db error"))

	result, err := uc.GetScoreboardGraph(context.Background(), 10, false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetChallengeDetailStats_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

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
	d.statsRepo.On("GetChallengeDetailStats", mock.Anything, challengeID).Return(stats, nil)
	redisClient.Regexp().ExpectSet("stats:challenge:"+challengeID.String(), `.*`, time.Minute).SetVal("OK")

	result, err := uc.GetChallengeDetailStats(context.Background(), challengeID.String(), false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Challenge 1", result.Title)
	assert.Equal(t, 5, result.SolveCount)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetChallengeDetailStats_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	challengeID := uuid.New()
	redisClient.ExpectGet("stats:challenge:" + challengeID.String()).SetErr(redis.Nil)
	d.statsRepo.On("GetChallengeDetailStats", mock.Anything, challengeID).Return(nil, errors.New("db error"))

	result, err := uc.GetChallengeDetailStats(context.Background(), challengeID.String(), false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetTeamRegistrationTimeSeries_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	data := []*entity.RegistrationTimePoint{{Date: "2025-01-01", Count: 5}}
	redisClient.ExpectGet("stats:team_registration").SetErr(redis.Nil)
	d.statsRepo.On("GetTeamRegistrationTimeSeries", mock.Anything).Return(data, nil)
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
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	redisClient.ExpectGet("stats:team_registration").SetErr(redis.Nil)
	d.statsRepo.On("GetTeamRegistrationTimeSeries", mock.Anything).Return(nil, errors.New("db error"))

	result, err := uc.GetTeamRegistrationTimeSeries(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetUserRegistrationTimeSeries_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	data := []*entity.RegistrationTimePoint{{Date: "2025-01-01", Count: 10}}
	redisClient.ExpectGet("stats:user_registration").SetErr(redis.Nil)
	d.statsRepo.On("GetUserRegistrationTimeSeries", mock.Anything).Return(data, nil)
	redisClient.Regexp().ExpectSet("stats:user_registration", `.*`, 5*time.Minute).SetVal("OK")

	result, err := uc.GetUserRegistrationTimeSeries(context.Background())

	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, 10, result[0].Count)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetUserRegistrationTimeSeries_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	redisClient.ExpectGet("stats:user_registration").SetErr(redis.Nil)
	d.statsRepo.On("GetUserRegistrationTimeSeries", mock.Anything).Return(nil, errors.New("db error"))

	result, err := uc.GetUserRegistrationTimeSeries(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetSolveMatrix_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	matrix := []*entity.SolveMatrixRow{
		{TeamID: uuid.New(), TeamName: "T1", ChallengeID: uuid.New(), ChallengeTitle: "C1", Solved: true},
	}
	redisClient.ExpectGet("stats:solve_matrix").SetErr(redis.Nil)
	d.statsRepo.On("GetSolveMatrix", mock.Anything).Return(matrix, nil)
	redisClient.Regexp().ExpectSet("stats:solve_matrix", `.*`, 30*time.Second).SetVal("OK")

	result, err := uc.GetSolveMatrix(context.Background(), false)

	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "T1", result[0].TeamName)
	assert.True(t, result[0].Solved)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetSolveMatrix_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	redisClient.ExpectGet("stats:solve_matrix").SetErr(redis.Nil)
	d.statsRepo.On("GetSolveMatrix", mock.Anything).Return(nil, errors.New("db error"))

	result, err := uc.GetSolveMatrix(context.Background(), false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetSubmissionTimeSeriesByType_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	data := []*entity.RegistrationTimePoint{{Date: "2025-01-01", Count: 3}}
	redisClient.ExpectGet("stats:submission_timeseries:true").SetErr(redis.Nil)
	d.statsRepo.On("GetSubmissionTimeSeriesByType", mock.Anything, true).Return(data, nil)
	redisClient.Regexp().ExpectSet("stats:submission_timeseries:true", `.*`, 5*time.Minute).SetVal("OK")

	result, err := uc.GetSubmissionTimeSeriesByType(context.Background(), true, false)

	assert.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, 3, result[0].Count)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetSubmissionTimeSeriesByType_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createStatisticsUseCase()

	redisClient.ExpectGet("stats:submission_timeseries:false").SetErr(redis.Nil)
	d.statsRepo.On("GetSubmissionTimeSeriesByType", mock.Anything, false).Return(nil, errors.New("db error"))

	result, err := uc.GetSubmissionTimeSeriesByType(context.Background(), false, false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}
