package repo

import (
	"context"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/google/uuid"
)

// =============================================================================
// Shared
// =============================================================================

type (
	// TransactionManager wraps business logic in a database transaction.
	// The transaction is embedded in the context passed to fn.
	TransactionManager interface {
		Run(ctx context.Context, fn func(context.Context) error) error
		RunSerializable(ctx context.Context, fn func(context.Context) error) error
	}
)

// =============================================================================
// User
// =============================================================================

type (
	UserRepository interface {
		Create(ctx context.Context, user *entity.User) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.User, error)
		GetByEmail(ctx context.Context, email string) (*entity.User, error)
		GetByUsername(ctx context.Context, username string) (*entity.User, error)
		GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.User, error)
		GetAll(ctx context.Context) ([]*entity.User, error)
		Search(ctx context.Context, search *string, limit, offset int) ([]*entity.User, error)
		CountSearch(ctx context.Context, search *string) (int64, error)
		SearchByIP(ctx context.Context, ip string, limit, offset int) ([]*entity.User, error)
		CountSearchByIP(ctx context.Context, ip string) (int64, error)
		UpdateTeamID(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID) error
		SetVerified(ctx context.Context, userID uuid.UUID) error
		SetUnverified(ctx context.Context, userID uuid.UUID) error
		UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
		UpdateAdmin(ctx context.Context, userID uuid.UUID, username, email, role, passwordHash *string, isVerified *bool) error
		UpdateProfile(ctx context.Context, userID uuid.UUID, username, email, passwordHash *string) error
		Delete(ctx context.Context, userID uuid.UUID) error
		Lock(ctx context.Context, userID uuid.UUID) error
		Ban(ctx context.Context, userID uuid.UUID, reason string) error
		Unban(ctx context.Context, userID uuid.UUID) error
	}
)

// =============================================================================
// Team
// =============================================================================

type (
	TeamRepository interface {
		Create(ctx context.Context, team *entity.Team) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Team, error)
		GetByInviteToken(ctx context.Context, inviteToken uuid.UUID) (*entity.Team, error)
		GetByName(ctx context.Context, name string) (*entity.Team, error)
		GetSoloTeamByUserID(ctx context.Context, userID uuid.UUID) (*entity.Team, error)
		GetAll(ctx context.Context) ([]*entity.Team, error)
		Search(ctx context.Context, search *string, limit, offset int) ([]*entity.Team, error)
		CountSearch(ctx context.Context, search *string) (int64, error)
		SearchAdmin(ctx context.Context, search *string, limit, offset int) ([]*entity.Team, error)
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
		Lock(ctx context.Context, teamID uuid.UUID) error
		// AcquireAdvisoryLock acquires a session-level advisory lock for the duration
		// of the current transaction. Used to serialize team-count checks.
		AcquireAdvisoryLock(ctx context.Context, lockKey int64) error
		CreateAuditLog(ctx context.Context, log *entity.TeamAuditLog) error
	}
)

// =============================================================================
// Challenge
// =============================================================================

type (
	// ChallengeFlags is an alias for entity.ChallengeFlags kept for repo-internal use.
	ChallengeFlags = entity.ChallengeFlags

	// ChallengeRequirement is an alias for entity.ChallengeRequirement.
	ChallengeRequirement = entity.ChallengeRequirement

	// ChallengeSolution is an alias for entity.ChallengeSolution.
	ChallengeSolution = entity.ChallengeSolution

	// ChallengeSolutionEntry is an alias for entity.ChallengeSolutionEntry.
	ChallengeSolutionEntry = entity.ChallengeSolutionEntry

	// ChallengeWithSolved is an alias for entity.ChallengeWithSolved.
	ChallengeWithSolved = entity.ChallengeWithSolved

	ChallengeRepository interface {
		Create(ctx context.Context, c *entity.Challenge) error
		Update(ctx context.Context, c *entity.Challenge) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Challenge, error)
		GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*entity.Challenge, error)
		GetByIDForUpdate(ctx context.Context, ID uuid.UUID) (*entity.Challenge, error)
		GetAll(ctx context.Context, teamID, tagID *uuid.UUID) ([]*ChallengeWithSolved, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		IncrementSolveCount(ctx context.Context, ID uuid.UUID) (int, error)
		DecrementSolveCount(ctx context.Context, ID uuid.UUID) (int, error)
		UpdatePoints(ctx context.Context, ID uuid.UUID, points int) error
		SetTags(ctx context.Context, challengeID uuid.UUID, tagIDs []uuid.UUID) error
		SetRequirements(ctx context.Context, challengeID uuid.UUID, requirementIDs []uuid.UUID) error
		GetFlags(ctx context.Context, ID uuid.UUID) (*ChallengeFlags, error)
		GetRequirements(ctx context.Context, ID uuid.UUID) ([]*ChallengeRequirement, error)
		GetSolution(ctx context.Context, ID uuid.UUID) (*ChallengeSolution, error)
		ListSolutions(ctx context.Context, teamID uuid.UUID) ([]*ChallengeSolutionEntry, error)
		UpsertSolution(ctx context.Context, challengeID uuid.UUID, content string) (*ChallengeSolution, error)
		DeleteSolution(ctx context.Context, challengeID uuid.UUID) error
		GetMissingChallengesByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Challenge, error)
		GetMissingChallengesByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Challenge, error)
	}
)

// =============================================================================
// Tag
// =============================================================================

type (
	TagRepository interface {
		Create(ctx context.Context, tag *entity.Tag) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Tag, error)
		GetByName(ctx context.Context, name string) (*entity.Tag, error)
		GetAll(ctx context.Context) ([]*entity.Tag, error)
		Update(ctx context.Context, tag *entity.Tag) error
		Delete(ctx context.Context, ID uuid.UUID) error
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Tag, error)
		GetByChallengeIDs(ctx context.Context, challengeIDs []uuid.UUID) (map[uuid.UUID][]*entity.Tag, error)
	}
)

// =============================================================================
// Hint
// =============================================================================

type (
	HintRepository interface {
		Create(ctx context.Context, hint *entity.Hint) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Hint, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Hint, error)
		Update(ctx context.Context, hint *entity.Hint) error
		Delete(ctx context.Context, ID uuid.UUID) error
		GetByTeamAndHint(ctx context.Context, teamID, hintID uuid.UUID) (*entity.HintUnlock, error)
		GetByTeamAndHintForUpdate(ctx context.Context, teamID, hintID uuid.UUID) (*entity.HintUnlock, error)
		GetUnlockedHintIDs(ctx context.Context, teamID, challengeID uuid.UUID) ([]uuid.UUID, error)
		GetAll(ctx context.Context, limit, offset int) ([]*entity.HintUnlockWithDetails, error)
		GetAllUnlocks(ctx context.Context) ([]*entity.HintUnlock, error)
		CountAll(ctx context.Context) (int, error)
		CountByTeamID(ctx context.Context, teamID uuid.UUID) (int, error)
		CreateUnlock(ctx context.Context, teamID, hintID uuid.UUID) error
	}
)

// =============================================================================
// File
// =============================================================================

type (
	FileRepository interface {
		Create(ctx context.Context, file *entity.File) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.File, error)
		GetByLocation(ctx context.Context, location string) (*entity.File, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType entity.FileType) ([]*entity.File, error)
		GetAll(ctx context.Context) ([]*entity.File, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Solve
// =============================================================================

type (
	// ScoreboardEntry is an alias for entity.ScoreboardEntry.
	ScoreboardEntry = entity.ScoreboardEntry

	// FirstBloodEntry is an alias for entity.FirstBloodEntry.
	FirstBloodEntry = entity.FirstBloodEntry

	SolveRepository interface {
		Create(ctx context.Context, solve *entity.Solve) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Solve, error)
		GetByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) (*entity.Solve, error)
		GetSolvedChallengeIDsByTeam(ctx context.Context, teamID uuid.UUID, challengeIDs []uuid.UUID) ([]uuid.UUID, error)
		GetByTeamAndChallengeForUpdate(ctx context.Context, teamID, challengeID uuid.UUID) (*entity.Solve, error)
		GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Solve, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.SolveWithDetails, error)
		GetByUserIDWithDetails(ctx context.Context, userID uuid.UUID) ([]*entity.SolveWithDetails, error)
		GetByTeamIDWithDetails(ctx context.Context, teamID uuid.UUID) ([]*entity.SolveWithDetails, error)
		GetAll(ctx context.Context) ([]*entity.Solve, error)
		GetScoreboard(ctx context.Context) ([]*ScoreboardEntry, error)
		GetScoreboardFrozen(ctx context.Context, freezeTime time.Time) ([]*ScoreboardEntry, error)
		GetScoreboardByBracket(ctx context.Context, bracketID *uuid.UUID) ([]*ScoreboardEntry, error)
		GetScoreboardByBracketFrozen(ctx context.Context, freezeTime time.Time, bracketID *uuid.UUID) ([]*ScoreboardEntry, error)
		GetFirstBlood(ctx context.Context, challengeID uuid.UUID) (*FirstBloodEntry, error)
		GetTeamScore(ctx context.Context, teamID uuid.UUID) (int, error)
		DeleteByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) error
		DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error
	}
)

// =============================================================================
// Award
// =============================================================================

type (
	AwardRepository interface {
		Create(ctx context.Context, award *entity.Award) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Award, error)
		GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Award, error)
		GetAll(ctx context.Context) ([]*entity.Award, error)
		GetTeamTotalAwards(ctx context.Context, teamID uuid.UUID) (int, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error
	}
)

// =============================================================================
// Competition
// =============================================================================

type (
	CompetitionRepository interface {
		Get(ctx context.Context) (*entity.Competition, error)
		Update(ctx context.Context, competition *entity.Competition) error
	}
)

// =============================================================================
// App settings
// =============================================================================

type (
	SettingsRepository interface {
		Get(ctx context.Context) (*entity.Settings, error)
		Update(ctx context.Context, s *entity.Settings) error
	}
)

// =============================================================================
// Competition params (dynamic key-value)
// =============================================================================

type (
	CompetitionParamRepository interface {
		GetAll(ctx context.Context) ([]*entity.CompetitionParam, error)
		GetByKey(ctx context.Context, key string) (*entity.CompetitionParam, error)
		Upsert(ctx context.Context, p *entity.CompetitionParam) error
		Delete(ctx context.Context, key string) error
	}
)

// =============================================================================
// Submission
// =============================================================================

type (
	SubmissionRepository interface {
		Create(ctx context.Context, sub *entity.Submission) error
		CreateBatch(ctx context.Context, subs []*entity.Submission) error
		GetByIDForUpdate(ctx context.Context, ID uuid.UUID) (*entity.Submission, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.SubmissionWithDetails, error)
		GetByChallenge(ctx context.Context, challengeID uuid.UUID, limit, offset int) ([]*entity.SubmissionWithDetails, error)
		GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.SubmissionWithDetails, error)
		GetByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*entity.SubmissionWithDetails, error)
		GetAll(ctx context.Context, limit, offset int) ([]*entity.SubmissionWithDetails, error)
		GetFailsByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.SubmissionWithDetails, error)
		CountFailsByUser(ctx context.Context, userID uuid.UUID) (int64, error)
		GetFailsByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*entity.SubmissionWithDetails, error)
		CountFailsByTeam(ctx context.Context, teamID uuid.UUID) (int64, error)
		CountByChallenge(ctx context.Context, challengeID uuid.UUID) (int64, error)
		CountByUser(ctx context.Context, userID uuid.UUID) (int64, error)
		CountByTeam(ctx context.Context, teamID uuid.UUID) (int64, error)
		CountAll(ctx context.Context) (int64, error)
		CountFailedByIP(ctx context.Context, ip string, since time.Time) (int64, error)
		GetStats(ctx context.Context, challengeID uuid.UUID) (*entity.SubmissionStats, error)
		Update(ctx context.Context, ID uuid.UUID, isCorrect bool) error
		Delete(ctx context.Context, ID uuid.UUID) error
		DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error
	}
)

// =============================================================================
// Statistics
// =============================================================================

type (
	StatisticsRepository interface {
		GetGeneralStats(ctx context.Context) (*entity.GeneralStats, error)
		GetChallengeStats(ctx context.Context) ([]*entity.ChallengeStats, error)
		GetChallengeDetailStats(ctx context.Context, challengeID uuid.UUID) (*entity.ChallengeDetailStats, error)
		GetScoreboardHistory(ctx context.Context, limit int) ([]*entity.ScoreboardHistoryEntry, error)
		GetScoreboardHistoryFrozen(ctx context.Context, freezeTime time.Time, limit int) ([]*entity.ScoreboardHistoryEntry, error)
		GetChallengeSolvePercentages(ctx context.Context) ([]*entity.ChallengeSolvePercentage, error)
		GetChallengeSolvePercentagesFrozen(ctx context.Context, freezeTime time.Time) ([]*entity.ChallengeSolvePercentage, error)
		GetScoreDistribution(ctx context.Context) ([]*entity.ScoreDistributionBucket, error)
		GetScoreDistributionFrozen(ctx context.Context, freezeTime time.Time) ([]*entity.ScoreDistributionBucket, error)
		GetSubmissionTimeSeries(ctx context.Context) (*entity.SubmissionTimeSeriesStats, error)
		GetSubmissionTimeSeriesFrozen(ctx context.Context, freezeTime time.Time) (*entity.SubmissionTimeSeriesStats, error)
		GetSubmissionTimeSeriesByType(ctx context.Context, isCorrect bool) ([]*entity.RegistrationTimePoint, error)
		GetSubmissionTimeSeriesByTypeFrozen(ctx context.Context, isCorrect bool, freezeTime time.Time) ([]*entity.RegistrationTimePoint, error)
		GetTeamRegistrationTimeSeries(ctx context.Context) ([]*entity.RegistrationTimePoint, error)
		GetUserRegistrationTimeSeries(ctx context.Context) ([]*entity.RegistrationTimePoint, error)
		GetSolveMatrix(ctx context.Context) ([]*entity.SolveMatrixRow, error)
	}
)

// =============================================================================
// Verification token (email)
// =============================================================================

type (
	VerificationTokenRepository interface {
		Create(ctx context.Context, token *entity.VerificationToken) error
		GetByToken(ctx context.Context, token string) (*entity.VerificationToken, error)
		MarkUsed(ctx context.Context, ID uuid.UUID) error
		DeleteExpired(ctx context.Context) error
		DeleteByUserAndType(ctx context.Context, userID uuid.UUID, tokenType entity.TokenType) error
	}
)

// =============================================================================
// Audit log
// =============================================================================

type (
	AuditLogRepository interface {
		Create(ctx context.Context, log *entity.AuditLog) error
	}
)

// =============================================================================
// Tracking
// =============================================================================

type (
	TrackingRepository interface {
		Create(ctx context.Context, entry *entity.TrackingEntry) error
		GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.TrackingEntry, error)
		CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
		CreateChallengeOpen(ctx context.Context, entry *entity.ChallengeOpen) error
		GetChallengeOpensByChallenge(ctx context.Context, challengeID uuid.UUID, limit, offset int) ([]*entity.ChallengeOpen, error)
		CountChallengeOpensByChallenge(ctx context.Context, challengeID uuid.UUID) (int, error)
	}
)

// =============================================================================
// Notification
// =============================================================================

type (
	NotificationRepository interface {
		Create(ctx context.Context, notif *entity.Notification) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Notification, error)
		GetAll(ctx context.Context, limit, offset int) ([]*entity.Notification, error)
		Update(ctx context.Context, notif *entity.Notification) error
		Delete(ctx context.Context, ID uuid.UUID) error
		CreateUserNotification(ctx context.Context, userNotif *entity.UserNotification) error
		GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.UserNotification, error)
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
		Create(ctx context.Context, page *entity.Page) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Page, error)
		GetBySlug(ctx context.Context, slug string) (*entity.Page, error)
		GetPublishedList(ctx context.Context) ([]*entity.PageListItem, error)
		GetAllList(ctx context.Context) ([]*entity.Page, error)
		Update(ctx context.Context, page *entity.Page) error
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Bracket
// =============================================================================

type (
	BracketRepository interface {
		Create(ctx context.Context, bracket *entity.Bracket) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Bracket, error)
		GetByName(ctx context.Context, name string) (*entity.Bracket, error)
		GetAll(ctx context.Context) ([]*entity.Bracket, error)
		Update(ctx context.Context, bracket *entity.Bracket) error
		Delete(ctx context.Context, ID uuid.UUID) error
		ClearAllDefaults(ctx context.Context) error
	}
)

// =============================================================================
// Field
// =============================================================================

type (
	FieldRepository interface {
		Create(ctx context.Context, field *entity.Field) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Field, error)
		GetByEntityType(ctx context.Context, entityType entity.EntityType) ([]*entity.Field, error)
		GetAll(ctx context.Context) ([]*entity.Field, error)
		Update(ctx context.Context, field *entity.Field) error
		Delete(ctx context.Context, ID uuid.UUID) error
	}

	FieldValueRepository interface {
		GetByEntityID(ctx context.Context, entityID uuid.UUID) ([]*entity.FieldValue, error)
		SetValues(ctx context.Context, entityID uuid.UUID, values map[string]string) error
	}
)

// =============================================================================
// API Token
// =============================================================================

type (
	APITokenRepository interface {
		Create(ctx context.Context, token *entity.APIToken) error
		GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.APIToken, error)
		GetByTokenHash(ctx context.Context, tokenHash string) (*entity.APIToken, error)
		Delete(ctx context.Context, ID, userID uuid.UUID) error
		UpdateLastUsedAt(ctx context.Context, ID uuid.UUID, at time.Time) error
	}
)

// =============================================================================
// Comment
// =============================================================================

type (
	CommentRepository interface {
		Create(ctx context.Context, comment *entity.Comment) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Comment, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Comment, error)
		Update(ctx context.Context, comment *entity.Comment) error
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// OAuth
// =============================================================================

type (
	OAuthAccountRepository interface {
		Create(ctx context.Context, acc *entity.OAuthAccount) error
		Upsert(ctx context.Context, acc *entity.OAuthAccount) error
		GetByProvider(ctx context.Context, provider, providerUserID string) (*entity.OAuthAccount, error)
		GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.OAuthAccount, error)
	}
)

// =============================================================================
// Backup
// =============================================================================

type (
	BackupRepository interface {
		EraseAllTables(ctx context.Context) error
		EraseTables(ctx context.Context, tables []string) error
		ImportCompetition(ctx context.Context, comp *entity.Competition) error
		ImportChallenges(ctx context.Context, data *entity.BackupData) error
		ImportTeams(ctx context.Context, data *entity.BackupData, opts entity.ImportOptions) error
		ImportUsers(ctx context.Context, data *entity.BackupData, opts entity.ImportOptions) error
		ImportAwards(ctx context.Context, data *entity.BackupData) error
		ImportSolves(ctx context.Context, data *entity.BackupData) error
		ImportHintUnlocks(ctx context.Context, data *entity.BackupData) error
		ImportFileMetadata(ctx context.Context, data *entity.BackupData) error
		ImportCSV(ctx context.Context, tableName string, header []string, rows [][]string) (int, []string, error)
	}
)
