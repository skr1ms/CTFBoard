package repo

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

type SolveForPointsRecalc struct {
	ID           uuid.UUID
	ChallengeID  uuid.UUID
	SolvedAt     time.Time
	InitialValue int
	MinValue     int
	Decay        int
}

// =============================================================================
// Shared
// =============================================================================

type (
	// TransactionManager wraps business logic in a database transaction.
	// The transaction is embedded in the context passed to fn.
	TransactionManager interface {
		Run(ctx context.Context, fn func(context.Context) error) error
		RunSerializable(ctx context.Context, fn func(context.Context) error) error
		ReadOnly(ctx context.Context, fn func(context.Context) error) error
	}
)

// =============================================================================
// User
// =============================================================================

type (
	UserRepository interface {
		Create(ctx context.Context, user *domain.User) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.User, error)
		GetByEmail(ctx context.Context, email string) (*domain.User, error)
		GetByUsername(ctx context.Context, username string) (*domain.User, error)
		GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.User, error)
		GetByTeamIDs(ctx context.Context, teamIDs []uuid.UUID) (map[uuid.UUID][]*domain.User, error)
		GetAll(ctx context.Context) ([]*domain.User, error)
		Search(ctx context.Context, search *string, limit, offset int) ([]*domain.User, error)
		CountSearch(ctx context.Context, search *string) (int64, error)
		SearchByIP(ctx context.Context, ip string, limit, offset int) ([]*domain.User, error)
		CountSearchByIP(ctx context.Context, ip string) (int64, error)
		UpdateTeamID(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID) error
		UpdateTeamIDBatch(ctx context.Context, userIDs []uuid.UUID, teamID *uuid.UUID) error
		FilterIDsByTeamIDNull(ctx context.Context, userIDs []uuid.UUID) ([]uuid.UUID, error)
		FilterIDsByTeamIDNullAndNotBanned(ctx context.Context, userIDs []uuid.UUID) ([]uuid.UUID, error)
		SetVerified(ctx context.Context, userID uuid.UUID) error
		SetUnverified(ctx context.Context, userID uuid.UUID) error
		UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
		UpdateAdmin(ctx context.Context, userID uuid.UUID, username, email, role, passwordHash *string, isVerified *bool) error
		UpdateProfile(ctx context.Context, userID uuid.UUID, username, email, passwordHash *string) error
		Delete(ctx context.Context, userID uuid.UUID) error
		Lock(ctx context.Context, userID uuid.UUID) error
		Ban(ctx context.Context, userID uuid.UUID, reason string) error
		Unban(ctx context.Context, userID uuid.UUID) error
		SetWasInBannedTeamByIDs(ctx context.Context, userIDs []uuid.UUID, value bool) error
		AcquireAdvisoryLock(ctx context.Context, lockKey int64) error
	}
)

// =============================================================================
// Team
// =============================================================================

type (
	TeamRepository interface {
		Create(ctx context.Context, team *domain.Team) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Team, error)
		GetByInviteToken(ctx context.Context, inviteToken uuid.UUID) (*domain.Team, error)
		GetByName(ctx context.Context, name string) (*domain.Team, error)
		GetSoloTeamByUserID(ctx context.Context, userID uuid.UUID) (*domain.Team, error)
		GetAll(ctx context.Context) ([]*domain.Team, error)
		Search(ctx context.Context, search *string, limit, offset int) ([]*domain.Team, error)
		CountSearch(ctx context.Context, search *string) (int64, error)
		SearchAdmin(ctx context.Context, search *string, limit, offset int) ([]*domain.Team, error)
		CountSearchAdmin(ctx context.Context, search *string) (int64, error)
		CountTeamMembers(ctx context.Context, teamID uuid.UUID) (int, error)
		CountActiveTeams(ctx context.Context) (int, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		HardDeleteTeams(ctx context.Context, cutoffDate time.Time) error
		Ban(ctx context.Context, teamID uuid.UUID, reason string) error
		Unban(ctx context.Context, teamID uuid.UUID) error
		SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error
		SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error
		UpdateAdmin(ctx context.Context, teamID uuid.UUID, name *string, captainID, bracketID *uuid.UUID, isHidden *bool) error
		UpdateName(ctx context.Context, teamID uuid.UUID, name string) error
		UpdateCaptain(ctx context.Context, teamID, newCaptainID uuid.UUID) error
		UpdateInviteToken(ctx context.Context, teamID, inviteToken uuid.UUID, expiresAt *time.Time) error
		Lock(ctx context.Context, teamID uuid.UUID) error
		// AcquireAdvisoryLock acquires a session-level advisory lock for the duration
		// of the current transaction. Used to serialize team-count checks.
		AcquireAdvisoryLock(ctx context.Context, lockKey int64) error
		CreateAuditLog(ctx context.Context, log *domain.TeamAuditLog) error
		GetLatestAuditLogByTeamIDAndAction(ctx context.Context, teamID uuid.UUID, action string) (*domain.TeamAuditLog, error)
	}
)

// =============================================================================
// Challenge
// =============================================================================

type (
	// ChallengeFlags is an alias for domain.ChallengeFlags kept for repo-internal use.
	ChallengeFlags = domain.ChallengeFlags

	// ChallengeRequirement is an alias for domain.ChallengeRequirement.
	ChallengeRequirement = domain.ChallengeRequirement

	// ChallengeSolution is an alias for domain.ChallengeSolution.
	ChallengeSolution = domain.ChallengeSolution

	// ChallengeSolutionEntry is an alias for domain.ChallengeSolutionEntry.
	ChallengeSolutionEntry = domain.ChallengeSolutionEntry

	// ChallengeWithSolved is an alias for domain.ChallengeWithSolved.
	ChallengeWithSolved = domain.ChallengeWithSolved

	ChallengeRepository interface {
		Create(ctx context.Context, c *domain.Challenge) error
		Update(ctx context.Context, c *domain.Challenge) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Challenge, error)
		GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Challenge, error)
		GetByIDForUpdate(ctx context.Context, ID uuid.UUID) (*domain.Challenge, error)
		GetAll(ctx context.Context, teamID, tagID *uuid.UUID) ([]*ChallengeWithSolved, error)
		GetAllForBackup(ctx context.Context) ([]*ChallengeWithSolved, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		IncrementSolveCount(ctx context.Context, ID uuid.UUID) (int, error)
		DecrementSolveCount(ctx context.Context, ID uuid.UUID) (int, error)
		BatchDecrementSolveCount(ctx context.Context, ids []uuid.UUID) error
		BatchIncrementSolveCount(ctx context.Context, ids []uuid.UUID) error
		BatchUpdatePoints(ctx context.Context, ids []uuid.UUID, points []int) error
		UpdatePoints(ctx context.Context, ID uuid.UUID, points int) error
		SetTags(ctx context.Context, challengeID uuid.UUID, tagIDs []uuid.UUID) error
		SetRequirements(ctx context.Context, challengeID uuid.UUID, requirementIDs []uuid.UUID) error
		GetFlags(ctx context.Context, ID uuid.UUID) (*ChallengeFlags, error)
		GetRequirements(ctx context.Context, ID uuid.UUID) ([]*ChallengeRequirement, error)
		GetAllRequirementPairs(ctx context.Context) ([]*domain.ChallengeRequirementPair, error)
		GetSolution(ctx context.Context, ID uuid.UUID) (*ChallengeSolution, error)
		GetAllSolutions(ctx context.Context) ([]*domain.SolutionBackup, error)
		ListSolutions(ctx context.Context, teamID uuid.UUID) ([]*ChallengeSolutionEntry, error)
		UpsertSolution(ctx context.Context, challengeID uuid.UUID, content string) (*ChallengeSolution, error)
		DeleteSolution(ctx context.Context, challengeID uuid.UUID) error
		GetMissingChallengesByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.Challenge, error)
		GetMissingChallengesByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Challenge, error)
	}
)

// =============================================================================
// Tag
// =============================================================================

type (
	TagRepository interface {
		Create(ctx context.Context, tag *domain.Tag) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Tag, error)
		GetByName(ctx context.Context, name string) (*domain.Tag, error)
		GetAll(ctx context.Context) ([]*domain.Tag, error)
		Update(ctx context.Context, tag *domain.Tag) error
		Delete(ctx context.Context, ID uuid.UUID) error
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Tag, error)
		GetByChallengeIDs(ctx context.Context, challengeIDs []uuid.UUID) (map[uuid.UUID][]*domain.Tag, error)
	}
)

// =============================================================================
// Hint
// =============================================================================

type (
	HintRepository interface {
		Create(ctx context.Context, hint *domain.Hint) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Hint, error)
		GetByIDForUpdate(ctx context.Context, ID uuid.UUID) (*domain.Hint, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Hint, error)
		GetByChallengeIDs(ctx context.Context, challengeIDs []uuid.UUID) (map[uuid.UUID][]*domain.Hint, error)
		Update(ctx context.Context, hint *domain.Hint) error
		Delete(ctx context.Context, ID uuid.UUID) error
		GetByTeamAndHint(ctx context.Context, teamID, hintID uuid.UUID) (*domain.HintUnlock, error)
		GetByTeamAndHintForUpdate(ctx context.Context, teamID, hintID uuid.UUID) (*domain.HintUnlock, error)
		GetUnlockedHintIDs(ctx context.Context, teamID, challengeID uuid.UUID) ([]uuid.UUID, error)
		GetAll(ctx context.Context, limit, offset int) ([]*domain.HintUnlockWithDetails, error)
		GetAllUnlocks(ctx context.Context) ([]*domain.HintUnlock, error)
		GetAllUnlocksForBackup(ctx context.Context) ([]*domain.HintUnlock, error)
		CountAll(ctx context.Context) (int, error)
		CountByTeamID(ctx context.Context, teamID uuid.UUID) (int, error)
		CreateUnlock(ctx context.Context, teamID, hintID uuid.UUID) error
		DeleteUnlocksByTeamID(ctx context.Context, teamID uuid.UUID) error
		SoftBanUnlocksByTeamID(ctx context.Context, teamID uuid.UUID) error
		RestoreUnlocksByBannedTeamID(ctx context.Context, teamID uuid.UUID) error
	}
)

// =============================================================================
// File
// =============================================================================

type (
	FileRepository interface {
		Create(ctx context.Context, file *domain.File) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.File, error)
		GetByLocation(ctx context.Context, location string) (*domain.File, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType domain.FileType) ([]*domain.File, error)
		GetAllByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.File, error)
		GetByChallengeIDs(ctx context.Context, challengeIDs []uuid.UUID) (map[uuid.UUID][]*domain.File, error)
		GetAll(ctx context.Context) ([]*domain.File, error)
		ListLocations(ctx context.Context, limit, offset int) ([]string, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Solve
// =============================================================================

type (
	// ScoreboardEntry is an alias for domain.ScoreboardEntry.
	ScoreboardEntry = domain.ScoreboardEntry

	// FirstBloodEntry is an alias for domain.FirstBloodEntry.
	FirstBloodEntry = domain.FirstBloodEntry

	SolveRepository interface {
		Create(ctx context.Context, solve *domain.Solve) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Solve, error)
		GetByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) (*domain.Solve, error)
		GetSolvedChallengeIDsByTeam(ctx context.Context, teamID uuid.UUID, challengeIDs []uuid.UUID) ([]uuid.UUID, error)
		GetByTeamAndChallengeForUpdate(ctx context.Context, teamID, challengeID uuid.UUID) (*domain.Solve, error)
		GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Solve, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.SolveWithDetails, error)
		GetByChallengeIDFrozen(ctx context.Context, challengeID uuid.UUID, freezeTime time.Time) ([]*domain.SolveWithDetails, error)
		GetSolveCountsFrozen(ctx context.Context, freezeTime time.Time) (map[uuid.UUID]int, error)
		GetByUserIDWithDetails(ctx context.Context, userID uuid.UUID) ([]*domain.SolveWithDetails, error)
		GetByTeamIDWithDetails(ctx context.Context, teamID uuid.UUID) ([]*domain.SolveWithDetails, error)
		GetAll(ctx context.Context) ([]*domain.Solve, error)
		GetAllForBackup(ctx context.Context) ([]*domain.Solve, error)
		GetScoreboard(ctx context.Context) ([]*ScoreboardEntry, error)
		GetScoreboardFrozen(ctx context.Context, freezeTime time.Time) ([]*ScoreboardEntry, error)
		GetScoreboardByBracket(ctx context.Context, bracketID *uuid.UUID) ([]*ScoreboardEntry, error)
		GetScoreboardByBracketFrozen(ctx context.Context, freezeTime time.Time, bracketID *uuid.UUID) ([]*ScoreboardEntry, error)
		GetFirstBlood(ctx context.Context, challengeID uuid.UUID) (*FirstBloodEntry, error)
		GetFirstBloodFrozen(ctx context.Context, challengeID uuid.UUID, freezeTime time.Time) (*FirstBloodEntry, error)
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
// Award
// =============================================================================

type (
	AwardRepository interface {
		Create(ctx context.Context, award *domain.Award) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Award, error)
		GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.Award, error)
		GetAll(ctx context.Context) ([]*domain.Award, error)
		GetAllForBackup(ctx context.Context) ([]*domain.Award, error)
		GetTeamTotalAwards(ctx context.Context, teamID uuid.UUID) (int, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error
		SoftBanByTeamID(ctx context.Context, teamID uuid.UUID) error
		RestoreByBannedTeamID(ctx context.Context, teamID uuid.UUID) error
	}
)

// =============================================================================
// Competition
// =============================================================================

type (
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
	SubmissionRepository interface {
		Create(ctx context.Context, sub *domain.Submission) error
		CreateBatch(ctx context.Context, subs []*domain.Submission) error
		GetByIDForUpdate(ctx context.Context, ID uuid.UUID) (*domain.Submission, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.SubmissionWithDetails, error)
		GetByChallenge(ctx context.Context, challengeID uuid.UUID, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		GetByChallengeFrozen(ctx context.Context, challengeID uuid.UUID, freezeTime time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		GetByUserFrozen(ctx context.Context, userID uuid.UUID, freezeTime time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		GetByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		GetByTeamFrozen(ctx context.Context, teamID uuid.UUID, freezeTime time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		GetAll(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		GetAllFrozen(ctx context.Context, freezeTime time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		GetFailsByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		CountFailsByUser(ctx context.Context, userID uuid.UUID) (int64, error)
		GetFailsByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*domain.SubmissionWithDetails, error)
		CountFailsByTeam(ctx context.Context, teamID uuid.UUID) (int64, error)
		CountByChallenge(ctx context.Context, challengeID uuid.UUID) (int64, error)
		CountByChallengeFrozen(ctx context.Context, challengeID uuid.UUID, freezeTime time.Time) (int64, error)
		CountByUser(ctx context.Context, userID uuid.UUID) (int64, error)
		CountByUserFrozen(ctx context.Context, userID uuid.UUID, freezeTime time.Time) (int64, error)
		CountByTeam(ctx context.Context, teamID uuid.UUID) (int64, error)
		CountByTeamFrozen(ctx context.Context, teamID uuid.UUID, freezeTime time.Time) (int64, error)
		CountSubmissionsByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) (int64, error)
		AcquireAdvisoryLockForSubmit(ctx context.Context, teamID, challengeID uuid.UUID) error
		CountAll(ctx context.Context) (int64, error)
		CountAllFrozen(ctx context.Context, freezeTime time.Time) (int64, error)
		CountFailedByIP(ctx context.Context, ip string, since time.Time) (int64, error)
		GetStats(ctx context.Context, challengeID uuid.UUID) (*domain.SubmissionStats, error)
		GetStatsFrozen(ctx context.Context, challengeID uuid.UUID, freezeTime time.Time) (*domain.SubmissionStats, error)
		Update(ctx context.Context, ID uuid.UUID, isCorrect bool) error
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
	StatisticsRepository interface {
		GetGeneralStats(ctx context.Context) (*domain.GeneralStats, error)
		GetGeneralStatsFrozen(ctx context.Context, freezeTime time.Time) (*domain.GeneralStats, error)
		GetChallengeStats(ctx context.Context) ([]*domain.ChallengeStats, error)
		GetChallengeStatsFrozen(ctx context.Context, freezeTime time.Time) ([]*domain.ChallengeStats, error)
		GetChallengeDetailStats(ctx context.Context, challengeID uuid.UUID) (*domain.ChallengeDetailStats, error)
		GetChallengeDetailStatsFrozen(ctx context.Context, challengeID uuid.UUID, freezeTime time.Time) (*domain.ChallengeDetailStats, error)
		GetScoreboardHistory(ctx context.Context, limit int) ([]*domain.ScoreboardHistoryEntry, error)
		GetScoreboardHistoryFrozen(ctx context.Context, freezeTime time.Time, limit int) ([]*domain.ScoreboardHistoryEntry, error)
		GetChallengeSolvePercentages(ctx context.Context) ([]*domain.ChallengeSolvePercentage, error)
		GetChallengeSolvePercentagesFrozen(ctx context.Context, freezeTime time.Time) ([]*domain.ChallengeSolvePercentage, error)
		GetScoreDistribution(ctx context.Context) ([]*domain.ScoreDistributionBucket, error)
		GetScoreDistributionFrozen(ctx context.Context, freezeTime time.Time) ([]*domain.ScoreDistributionBucket, error)
		GetSubmissionTimeSeries(ctx context.Context) (*domain.SubmissionTimeSeriesStats, error)
		GetSubmissionTimeSeriesFrozen(ctx context.Context, freezeTime time.Time) (*domain.SubmissionTimeSeriesStats, error)
		GetSubmissionTimeSeriesByType(ctx context.Context, isCorrect bool) ([]*domain.RegistrationTimePoint, error)
		GetSubmissionTimeSeriesByTypeFrozen(ctx context.Context, isCorrect bool, freezeTime time.Time) ([]*domain.RegistrationTimePoint, error)
		GetTeamRegistrationTimeSeries(ctx context.Context) ([]*domain.RegistrationTimePoint, error)
		GetUserRegistrationTimeSeries(ctx context.Context) ([]*domain.RegistrationTimePoint, error)
		GetSolveMatrix(ctx context.Context) ([]*domain.SolveMatrixRow, error)
		GetSolveMatrixFrozen(ctx context.Context, freezeTime time.Time) ([]*domain.SolveMatrixRow, error)
	}
)

// =============================================================================
// Verification token (email)
// =============================================================================

type (
	VerificationTokenRepository interface {
		Create(ctx context.Context, token *domain.VerificationToken) error
		GetByToken(ctx context.Context, token string) (*domain.VerificationToken, error)
		MarkUsed(ctx context.Context, ID uuid.UUID) error
		DeleteExpired(ctx context.Context) error
		DeleteByUserAndType(ctx context.Context, userID uuid.UUID, tokenType domain.TokenType) error
	}
)

// =============================================================================
// Audit log
// =============================================================================

type (
	AuditLogRepository interface {
		Create(ctx context.Context, log *domain.AuditLog) error
	}
)

// =============================================================================
// Tracking
// =============================================================================

type (
	TrackingRepository interface {
		Create(ctx context.Context, entry *domain.TrackingEntry) error
		GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.TrackingEntry, error)
		CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
		CreateChallengeOpen(ctx context.Context, entry *domain.ChallengeOpen) error
		GetChallengeOpensByChallenge(ctx context.Context, challengeID uuid.UUID, limit, offset int) ([]*domain.ChallengeOpen, error)
		CountChallengeOpensByChallenge(ctx context.Context, challengeID uuid.UUID) (int, error)
	}
)

// =============================================================================
// Notification
// =============================================================================

type (
	NotificationRepository interface {
		Create(ctx context.Context, notif *domain.Notification) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Notification, error)
		GetAll(ctx context.Context, limit, offset int) ([]*domain.Notification, error)
		Update(ctx context.Context, notif *domain.Notification) error
		Delete(ctx context.Context, ID uuid.UUID) error
		CreateUserNotification(ctx context.Context, userNotif *domain.UserNotification) error
		GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.UserNotification, error)
		GetUserNotificationByID(ctx context.Context, ID, userID uuid.UUID) (*domain.UserNotification, error)
		MarkAsRead(ctx context.Context, ID, userID uuid.UUID) error
		CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
		DeleteUserNotification(ctx context.Context, ID, userID uuid.UUID) error
	}
)

// =============================================================================
// Page
// =============================================================================

type (
	PageRepository interface {
		Create(ctx context.Context, page *domain.Page) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Page, error)
		GetBySlug(ctx context.Context, slug string) (*domain.Page, error)
		GetPublishedList(ctx context.Context) ([]*domain.PageListItem, error)
		GetAllList(ctx context.Context) ([]*domain.Page, error)
		Update(ctx context.Context, page *domain.Page) error
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Bracket
// =============================================================================

type (
	BracketRepository interface {
		Create(ctx context.Context, bracket *domain.Bracket) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Bracket, error)
		GetByName(ctx context.Context, name string) (*domain.Bracket, error)
		GetAll(ctx context.Context) ([]*domain.Bracket, error)
		Update(ctx context.Context, bracket *domain.Bracket) error
		Delete(ctx context.Context, ID uuid.UUID) error
		ClearAllDefaults(ctx context.Context) error
	}
)

// =============================================================================
// Field
// =============================================================================

type (
	FieldRepository interface {
		Create(ctx context.Context, field *domain.Field) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Field, error)
		GetByEntityType(ctx context.Context, entityType domain.EntityType) ([]*domain.Field, error)
		GetAll(ctx context.Context) ([]*domain.Field, error)
		Update(ctx context.Context, field *domain.Field) error
		Delete(ctx context.Context, ID uuid.UUID) error
	}

	FieldValueRepository interface {
		GetByEntityID(ctx context.Context, entityID uuid.UUID) ([]*domain.FieldValue, error)
		GetAll(ctx context.Context) ([]*domain.FieldValue, error)
		SetValues(ctx context.Context, entityID uuid.UUID, values map[string]string) error
		DeleteByEntityID(ctx context.Context, entityID uuid.UUID) error
	}
)

// =============================================================================
// API Token
// =============================================================================

type (
	APITokenRepository interface {
		Create(ctx context.Context, token *domain.APIToken) error
		GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.APIToken, error)
		GetByTokenHash(ctx context.Context, tokenHash string) (*domain.APIToken, error)
		Delete(ctx context.Context, ID, userID uuid.UUID) error
		UpdateLastUsedAt(ctx context.Context, ID uuid.UUID, at time.Time) error
	}
)

// =============================================================================
// Comment
// =============================================================================

type (
	CommentRepository interface {
		Create(ctx context.Context, comment *domain.Comment) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Comment, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Comment, error)
		GetAll(ctx context.Context) ([]*domain.Comment, error)
		Update(ctx context.Context, comment *domain.Comment) error
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Rating
// =============================================================================

type (
	RatingRepository interface {
		Upsert(ctx context.Context, rating *domain.Rating) error
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Rating, error)
		GetByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) (*domain.Rating, error)
		GetAll(ctx context.Context) ([]*domain.Rating, error)
	}
)

// =============================================================================
// OAuth
// =============================================================================

type (
	OAuthAccountRepository interface {
		Create(ctx context.Context, acc *domain.OAuthAccount) error
		Upsert(ctx context.Context, acc *domain.OAuthAccount) error
		GetByProvider(ctx context.Context, provider, providerUserID string) (*domain.OAuthAccount, error)
		GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.OAuthAccount, error)
	}
)

// =============================================================================
// Backup
// =============================================================================

type (
	BackupRepository interface {
		EraseAllTables(ctx context.Context) error
		EraseTables(ctx context.Context, tables []string) error
		ImportCompetition(ctx context.Context, comp *domain.Competition) error
		ImportTags(ctx context.Context, data *domain.BackupData) error
		ImportChallenges(ctx context.Context, data *domain.BackupData) error
		ImportChallengeTags(ctx context.Context, data *domain.BackupData) error
		ImportUsers(ctx context.Context, data *domain.BackupData, opts domain.ImportOptions) error
		ImportTeams(ctx context.Context, data *domain.BackupData, opts domain.ImportOptions) error
		UpdateUserTeamIDs(ctx context.Context, data *domain.BackupData) error
		ImportAwards(ctx context.Context, data *domain.BackupData) error
		ImportSolves(ctx context.Context, data *domain.BackupData) error
		ImportHintUnlocks(ctx context.Context, data *domain.BackupData) error
		ImportFileMetadata(ctx context.Context, data *domain.BackupData) error
		ImportBrackets(ctx context.Context, data *domain.BackupData) error
		ImportChallengeRequirements(ctx context.Context, data *domain.BackupData) error
		ImportSolutions(ctx context.Context, data *domain.BackupData) error
		ImportRatings(ctx context.Context, data *domain.BackupData) error
		ImportComments(ctx context.Context, data *domain.BackupData) error
		ImportFields(ctx context.Context, data *domain.BackupData) error
		ImportFieldValues(ctx context.Context, data *domain.BackupData) error
		ImportCSV(ctx context.Context, tableName string, header []string, rows [][]string) (int, []string, error)
	}
)
