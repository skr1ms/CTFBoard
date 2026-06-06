package repo

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// =============================================================================
// Solve
// =============================================================================

type (
	// ScoreboardEntry is an alias for domain.ScoreboardEntry.
	ScoreboardEntry = domain.ScoreboardEntry

	// FirstBloodEntry is an alias for domain.FirstBloodEntry.
	FirstBloodEntry = domain.FirstBloodEntry

	// SolveRepository provides persistence operations for correct solves and scoreboard queries.
	SolveRepository interface {
		Create(ctx context.Context, solve *domain.Solve) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Solve, error)
		GetByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) (*domain.Solve, error)
		GetSolvedChallengeIDsByTeam(ctx context.Context, teamID uuid.UUID, challengeIDs []uuid.UUID) ([]uuid.UUID, error)
		GetByTeamAndChallengeForUpdate(ctx context.Context, teamID, challengeID uuid.UUID) (*domain.Solve, error)
		GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Solve, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, freezeTime *time.Time) ([]*domain.SolveWithDetails, error)
		GetSolveCounts(ctx context.Context, freezeTime *time.Time) (map[uuid.UUID]int, error)
		GetByUserIDWithDetails(ctx context.Context, userID uuid.UUID) ([]*domain.SolveWithDetails, error)
		GetByTeamIDWithDetails(ctx context.Context, teamID uuid.UUID) ([]*domain.SolveWithDetails, error)
		GetAll(ctx context.Context) ([]*domain.Solve, error)
		GetAllForBackup(ctx context.Context) ([]*domain.Solve, error)
		GetScoreboardByBracket(ctx context.Context, bracketID *uuid.UUID, freezeTime *time.Time) ([]*ScoreboardEntry, error)
		GetFirstBlood(ctx context.Context, challengeID uuid.UUID, freezeTime *time.Time) (*FirstBloodEntry, error)
		GetTeamScore(ctx context.Context, teamID uuid.UUID) (int, error)
		DeleteByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) error
		DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error
		SoftBanByTeamID(ctx context.Context, teamID uuid.UUID) error
		RestoreByBannedTeamID(ctx context.Context, teamID uuid.UUID) error
		SoftBanByTeamIDAndUserID(ctx context.Context, teamID, userID uuid.UUID) error
		RestoreByBannedUserID(ctx context.Context, userID uuid.UUID) error
		GetSolvesForPointsRecalc(ctx context.Context, challengeIDs []uuid.UUID) ([]*SolveForPointsRecalc, error)
		BatchUpdateSolvePoints(ctx context.Context, solveIDs []uuid.UUID, points []int) error
	}
)

// =============================================================================
// Competition
// =============================================================================

type (
	// CompetitionRepository provides persistence operations for the singleton competition record.
	CompetitionRepository interface {
		Get(ctx context.Context) (*domain.Competition, error)
		GetForUpdate(ctx context.Context) (*domain.Competition, error)
		Update(ctx context.Context, competition *domain.Competition) error
	}
)

// =============================================================================
// App settings
// =============================================================================

type (
	// SettingsRepository provides persistence operations for the singleton application settings record.
	SettingsRepository interface {
		Get(ctx context.Context) (*domain.Settings, error)
		GetForUpdate(ctx context.Context) (*domain.Settings, error)
		Update(ctx context.Context, s *domain.Settings) error
		UpdateIfCurrent(ctx context.Context, s *domain.Settings) error
	}
)

// =============================================================================
// Competition params (dynamic key-value)
// =============================================================================

type (
	// CompetitionParamRepository provides persistence operations for dynamic competition key-value parameters.
	CompetitionParamRepository interface {
		GetAll(ctx context.Context) ([]*domain.CompetitionParam, error)
		GetByCategory(ctx context.Context, category string) ([]*domain.CompetitionParam, error)
		GetByKey(ctx context.Context, key string) (*domain.CompetitionParam, error)
		GetByKeyForUpdate(ctx context.Context, key string) (*domain.CompetitionParam, error)
		Upsert(ctx context.Context, p *domain.CompetitionParam) error
		Delete(ctx context.Context, key string) error
	}
)

// =============================================================================
// Submission
// =============================================================================

type (
	// SubmissionRepository provides persistence operations for flag submission records.
	SubmissionRepository interface {
		Create(ctx context.Context, sub *domain.Submission) error
		CreateBatch(ctx context.Context, subs []*domain.Submission) error
		GetByIDForUpdate(ctx context.Context, ID uuid.UUID) (*domain.Submission, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.SubmissionWithDetails, error)
		GetByChallenge(ctx context.Context, challengeID uuid.UUID, freezeTime *time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		GetByUser(ctx context.Context, userID uuid.UUID, freezeTime *time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		GetByTeam(ctx context.Context, teamID uuid.UUID, freezeTime *time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		GetAll(ctx context.Context, freezeTime *time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		GetFailsByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		CountFailsByUser(ctx context.Context, userID uuid.UUID) (int64, error)
		GetFailsByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		CountFailsByTeam(ctx context.Context, teamID uuid.UUID) (int64, error)
		CountByChallenge(ctx context.Context, challengeID uuid.UUID, freezeTime *time.Time) (int64, error)
		CountByUser(ctx context.Context, userID uuid.UUID, freezeTime *time.Time) (int64, error)
		CountByTeam(ctx context.Context, teamID uuid.UUID, freezeTime *time.Time) (int64, error)
		CountSubmissionsByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) (int64, error)
		CountSubmissionsByTeamAndChallengeInWindow(ctx context.Context, teamID, challengeID uuid.UUID, windowStart time.Time) (int64, error)
		AcquireAdvisoryLockForSubmit(ctx context.Context, teamID, challengeID uuid.UUID) error
		CountAll(ctx context.Context, freezeTime *time.Time) (int64, error)
		CountFailedByIP(ctx context.Context, ip string, since time.Time) (int64, error)
		GetStats(ctx context.Context, challengeID uuid.UUID, freezeTime *time.Time) (*domain.SubmissionStats, error)
		Update(ctx context.Context, ID uuid.UUID, isCorrect bool) error
		Discard(ctx context.Context, ID uuid.UUID) error
		Delete(ctx context.Context, ID uuid.UUID) error
		DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error
		SoftBanByTeamID(ctx context.Context, teamID uuid.UUID) error
		RestoreByBannedTeamID(ctx context.Context, teamID uuid.UUID) error
		SoftBanByUserID(ctx context.Context, userID uuid.UUID) error
		RestoreByBannedUserID(ctx context.Context, userID uuid.UUID) error
	}
)

// =============================================================================
// Statistics
// =============================================================================

type (
	// StatisticsRepository provides read-only aggregate queries for competition statistics, scoreboards, and solve matrices.
	StatisticsRepository interface {
		GetGeneralStats(ctx context.Context, freezeTime *time.Time) (*domain.GeneralStats, error)
		GetChallengeStats(ctx context.Context, freezeTime *time.Time) ([]*domain.ChallengeStats, error)
		GetChallengeDetailStats(ctx context.Context, challengeID uuid.UUID, freezeTime *time.Time) (*domain.ChallengeDetailStats, error)
		GetScoreboardHistory(ctx context.Context, limit int, freezeTime *time.Time) ([]*domain.ScoreboardHistoryEntry, error)
		GetChallengeSolvePercentages(ctx context.Context, freezeTime *time.Time) ([]*domain.ChallengeSolvePercentage, error)
		GetScoreDistribution(ctx context.Context, freezeTime *time.Time) ([]*domain.ScoreDistributionBucket, error)
		GetSubmissionTimeSeries(ctx context.Context, freezeTime *time.Time) (*domain.SubmissionTimeSeriesStats, error)
		GetSubmissionTimeSeriesByType(ctx context.Context, isCorrect bool, freezeTime *time.Time) ([]*domain.RegistrationTimePoint, error)
		GetTeamRegistrationTimeSeries(ctx context.Context) ([]*domain.RegistrationTimePoint, error)
		GetUserRegistrationTimeSeries(ctx context.Context) ([]*domain.RegistrationTimePoint, error)
		GetSolveMatrix(ctx context.Context, freezeTime *time.Time) ([]*domain.SolveMatrixRow, error)
	}
)
