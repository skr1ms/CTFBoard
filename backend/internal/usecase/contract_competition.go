package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// =============================================================================
// Competition
// =============================================================================

// CompetitionUpdateOptionals carries optional fields for a competition update, using nil to indicate "no change".
type CompetitionUpdateOptionals struct {
	IsPaused                     *bool
	IsPublic                     *bool
	AllowTeamSwitch              *bool
	MinTeamSize                  *int
	MaxTeamSize                  *int
	ClearFreezeTime              *bool
	ClearEndTime                 *bool
	KeepScoreboardFrozenAfterEnd *bool
}

type (
	// CompetitionUseCase manages competition settings, status, and submission gating.
	CompetitionUseCase interface {
		Get(ctx context.Context) (*domain.Competition, error)
		Update(ctx context.Context, comp *domain.Competition, optionals *CompetitionUpdateOptionals, actorID uuid.UUID, clientIP string) error
		GetStatus(ctx context.Context) (domain.CompetitionStatus, error)
		IsSubmissionAllowed(ctx context.Context) (bool, error)
	}

	CompetitionGuard interface {
		Get(ctx context.Context) (*domain.Competition, error)
		RequireTeamSwitch(ctx context.Context) (*domain.Competition, error)
		RequireTeamSwitchAndTeamsMode(ctx context.Context) (*domain.Competition, error)
		RequireTeamSwitchAndSoloMode(ctx context.Context) (*domain.Competition, error)
	}

	// SettingsGetter returns app settings (e.g. via SettingsUseCase for singleflight/cache).
	SettingsGetter interface {
		Get(ctx context.Context) (*domain.Settings, error)
	}
)

// =============================================================================
// Solve
// =============================================================================

type (
	// SolveUseCase handles solve recording, scoreboard retrieval, and first-blood lookups.
	SolveUseCase interface {
		Create(ctx context.Context, solve *domain.Solve) error
		GetScoreboard(ctx context.Context, bracketID *uuid.UUID, forceLive bool) ([]*domain.ScoreboardEntry, error)
		GetFirstBlood(ctx context.Context, challengeID uuid.UUID, forceLive bool) (*domain.FirstBloodEntry, error)
	}
)

// =============================================================================
// Statistics
// =============================================================================

type (
	// StatisticsUseCase provides competition analytics including scoreboards, solve rates, and time-series data.
	StatisticsUseCase interface {
		GetGeneralStats(ctx context.Context, forceLive bool) (*domain.GeneralStats, error)
		GetChallengeStats(ctx context.Context, forceLive bool) ([]*domain.ChallengeStats, error)
		GetChallengeDetailStats(ctx context.Context, challengeID string, forceLive bool) (*domain.ChallengeDetailStats, error)
		GetScoreboardHistory(ctx context.Context, limit int, forceLive bool) ([]*domain.ScoreboardHistoryEntry, error)
		GetScoreboardGraph(ctx context.Context, topN int, forceLive bool) (*domain.ScoreboardGraph, error)
		GetChallengeSolvePercentages(ctx context.Context, forceLive bool) ([]*domain.ChallengeSolvePercentage, error)
		GetScoreDistribution(ctx context.Context, forceLive bool) ([]*domain.ScoreDistributionBucket, error)
		GetSubmissionTimeSeries(ctx context.Context, forceLive bool) (*domain.SubmissionTimeSeriesStats, error)
		GetSubmissionTimeSeriesByType(ctx context.Context, isCorrect, forceLive bool) ([]*domain.RegistrationTimePoint, error)
		GetTeamRegistrationTimeSeries(ctx context.Context) ([]*domain.RegistrationTimePoint, error)
		GetUserRegistrationTimeSeries(ctx context.Context) ([]*domain.RegistrationTimePoint, error)
		GetSolveMatrix(ctx context.Context, forceLive bool) ([]*domain.SolveMatrixRow, error)
	}
)

// =============================================================================
// Submission
// =============================================================================

type (
	// SubmissionUseCase manages flag submission records, querying, and admin corrections.
	SubmissionUseCase interface {
		LogSubmission(ctx context.Context, sub *domain.Submission) error
		LogRateLimited(ctx context.Context, userID, teamID, challengeID uuid.UUID, ip string) error
		AdminCreate(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID, challengeID uuid.UUID, submittedFlag string, isCorrect bool, ip string) (*domain.SubmissionWithDetails, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.SubmissionWithDetails, error)
		GetByChallenge(ctx context.Context, challengeID uuid.UUID, page, perPage int, forceLive bool) (*Paginated[*domain.SubmissionWithDetails], error)
		GetByUser(ctx context.Context, userID uuid.UUID, page, perPage int, forceLive bool) (*Paginated[*domain.SubmissionWithDetails], error)
		GetByTeam(ctx context.Context, teamID uuid.UUID, page, perPage int, forceLive bool) (*Paginated[*domain.SubmissionWithDetails], error)
		GetAll(ctx context.Context, page, perPage int, forceLive bool) (*Paginated[*domain.SubmissionWithDetails], error)
		GetStats(ctx context.Context, challengeID uuid.UUID, forceLive bool) (*domain.SubmissionStats, error)
		Update(ctx context.Context, ID uuid.UUID, isCorrect bool) (*domain.SubmissionWithDetails, error)
		Discard(ctx context.Context, ID uuid.UUID) (*domain.SubmissionWithDetails, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Competition params (dynamic key-value)
// =============================================================================

type (
	// CompetitionParamUseCase manages dynamic key-value competition parameters with typed access helpers.
	CompetitionParamUseCase interface {
		Get(ctx context.Context, key string) (*domain.CompetitionParam, error)
		GetAll(ctx context.Context) ([]*domain.CompetitionParam, error)
		GetByCategory(ctx context.Context, category string) ([]*domain.CompetitionParam, error)
		GetPublic(ctx context.Context) ([]*domain.CompetitionParam, error)
		Set(ctx context.Context, key, value, description string, valueType domain.CompetitionParamValueType, category string, actorID uuid.UUID, clientIP string) error
		SetBatch(ctx context.Context, params []*domain.CompetitionParam, actorID uuid.UUID, clientIP string) error
		Delete(ctx context.Context, key string, actorID uuid.UUID, clientIP string) error
		GetString(ctx context.Context, key, defaultVal string) string
		GetInt(ctx context.Context, key string, defaultVal int) int
		GetBool(ctx context.Context, key string, defaultVal bool) bool
	}
)
