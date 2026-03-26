package usecase

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-httpkit/httputil"
	"github.com/wahrwelt-kit/go-jwtkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

const (
	DefaultPerPage    = 20
	DefaultMaxPerPage = 100
)

// =============================================================================
// Shared
// =============================================================================

type Paginated[T any] = httputil.Paginated[T]

// =============================================================================
// User
// =============================================================================

type (
	UserProfile struct {
		User   *domain.User
		Solves []*domain.Solve
	}

	UserUseCase interface {
		Register(ctx context.Context, username, email, password string, customFields map[string]string) (*domain.User, error)
		Login(ctx context.Context, email, password string) (*jwtkit.TokenPair, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.User, error)
		GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error)
		ListUsers(ctx context.Context, search *string, field string, page, perPage int) (*Paginated[*domain.User], error)
		GetUserSolves(ctx context.Context, userID uuid.UUID) ([]*domain.SolveWithDetails, error)
		GetUserFails(ctx context.Context, userID uuid.UUID, page, perPage int) (*Paginated[*domain.SubmissionWithDetails], error)
		GetUserAwards(ctx context.Context, userID uuid.UUID) ([]*domain.Award, error)
		AdminCreate(ctx context.Context, username, email, password, role string) (*domain.User, error)
		AdminUpdate(ctx context.Context, userID uuid.UUID, username, email, role, password *string, isVerified *bool) (*domain.User, error)
		AdminDelete(ctx context.Context, userID, actorID uuid.UUID) error
		BanUser(ctx context.Context, userID uuid.UUID, reason string, actorID uuid.UUID) error
		UnbanUser(ctx context.Context, userID, actorID uuid.UUID) error
		UpdateProfile(ctx context.Context, userID uuid.UUID, username, email, currentPassword, newPassword *string) (*domain.User, error)
		GetMySubmissions(ctx context.Context, userID uuid.UUID, page, perPage int) (*Paginated[*domain.SubmissionWithDetails], error)
	}
)

// =============================================================================
// Team
// =============================================================================

type ConfirmReason string

const (
	ConfirmReasonSoloTeamReset ConfirmReason = "solo_team_reset"
)

type (
	TeamCreateAffectedData struct {
		SolveCount      int
		Points          int
		HintUnlockCount int
		AwardsTotal     int
	}

	TeamCreateResult struct {
		Team               *domain.Team
		RequiresConfirm    bool
		ConfirmationReason ConfirmReason
		AffectedData       *TeamCreateAffectedData
	}

	TeamUseCase interface {
		Create(ctx context.Context, name string, captainID uuid.UUID, isSolo, confirmReset bool) (*domain.Team, error)
		TryCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*TeamCreateResult, error)
		ConfirmCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*domain.Team, error)
		Join(ctx context.Context, inviteToken, userID uuid.UUID, confirmReset bool) (*domain.Team, error)
		Leave(ctx context.Context, userID uuid.UUID) error
		TransferCaptain(ctx context.Context, captainID, newCaptainID uuid.UUID) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Team, error)
		GetMyTeam(ctx context.Context, userID uuid.UUID) (*domain.Team, []*domain.User, int, bool, error)
		GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*domain.User, error)
		CreateSoloTeam(ctx context.Context, userID uuid.UUID, confirmReset bool) (*domain.Team, error)
		DisbandTeam(ctx context.Context, captainID uuid.UUID) error
		KickMember(ctx context.Context, captainID, targetUserID uuid.UUID) error
		BanTeam(ctx context.Context, teamID uuid.UUID, reason string, banMembers bool, actorID uuid.UUID) error
		UnbanTeam(ctx context.Context, teamID, actorID uuid.UUID) error
		SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error
		SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error
		ListTeams(ctx context.Context, search *string, page, perPage int) (*Paginated[*domain.Team], error)
		AdminListTeams(ctx context.Context, search *string, page, perPage int) (*Paginated[*domain.Team], error)
		GetTeamSolves(ctx context.Context, teamID uuid.UUID) ([]*domain.SolveWithDetails, error)
		GetTeamFails(ctx context.Context, teamID uuid.UUID, page, perPage int) (*Paginated[*domain.SubmissionWithDetails], error)
		GetTeamAwards(ctx context.Context, teamID uuid.UUID) ([]*domain.Award, error)
		AdminUpdate(ctx context.Context, teamID uuid.UUID, name *string, captainID, bracketID *uuid.UUID, isHidden *bool) (*domain.Team, error)
		AdminDelete(ctx context.Context, teamID uuid.UUID) error
		AdminGetMembers(ctx context.Context, teamID uuid.UUID) ([]*domain.User, error)
		AdminAddMember(ctx context.Context, teamID, userID uuid.UUID) error
		AdminRemoveMember(ctx context.Context, teamID, userID uuid.UUID) error
		UpdateMyTeam(ctx context.Context, captainID uuid.UUID, name string) (*domain.Team, error)
		GetInviteToken(ctx context.Context, captainID uuid.UUID) (*domain.Team, error)
		RegenerateInviteToken(ctx context.Context, captainID uuid.UUID) (*domain.Team, error)
	}
)

// =============================================================================
// Challenge
// =============================================================================

type (
	ChallengeWithTags struct {
		*domain.ChallengeWithSolved

		Tags []*domain.Tag
	}

	ChallengeDetail struct {
		Challenge  *domain.Challenge
		Tags       []*domain.Tag
		Files      []*domain.File
		Hints      []*HintWithUnlockStatus
		FirstBlood *domain.FirstBloodEntry
		SolvedByMe bool
		SolveCount int
	}

	ChallengeUseCase interface {
		GetAll(ctx context.Context, teamID, tagID *uuid.UUID) ([]*ChallengeWithTags, error)
		GetByID(ctx context.Context, challengeID uuid.UUID) (*domain.Challenge, error)
		GetDetail(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (*ChallengeDetail, error)
		GetSolves(ctx context.Context, challengeID uuid.UUID) ([]*domain.SolveWithDetails, error)
		GetTags(ctx context.Context, challengeID uuid.UUID) ([]*domain.Tag, error)
		GetRequirements(ctx context.Context, challengeID uuid.UUID) ([]*domain.ChallengeRequirement, error)
		SetRequirements(ctx context.Context, challengeID uuid.UUID, requirementIDs []uuid.UUID) error
		GetSolution(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (*domain.ChallengeSolution, error)
		ListSolutions(ctx context.Context, teamID uuid.UUID) ([]*domain.ChallengeSolutionEntry, error)
		AdminUpsertSolution(ctx context.Context, challengeID uuid.UUID, content string) (*domain.ChallengeSolution, error)
		AdminDeleteSolution(ctx context.Context, challengeID uuid.UUID) error
		GetFlags(ctx context.Context, challengeID uuid.UUID) (*domain.ChallengeFlags, error)
		GetTypes(ctx context.Context) ([]string, error)
		GetMissingChallengesByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.Challenge, error)
		GetMissingChallengesByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Challenge, error)
		Create(ctx context.Context, title, description, category string, points, initialValue, minValue, decay int, flag, connectionInfo string, maxAttempts, position int, state string, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*domain.Challenge, error)
		Update(ctx context.Context, ID uuid.UUID, title, description, category string, points int, initialValue, minValue, decay *int, flag string, connectionInfo *string, maxAttempts, position *int, state string, isRegex, isCaseInsensitive *bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*domain.Challenge, error)
		Delete(ctx context.Context, ID, actorID uuid.UUID, clientIP string) error
		SubmitFlag(ctx context.Context, challengeID uuid.UUID, flag string, userID uuid.UUID, teamID *uuid.UUID, clientIP string) (bool, error)
		InvalidateScoreboardCache(ctx context.Context)
		InvalidateScoreboardCacheForTeam(ctx context.Context, teamID uuid.UUID)
		AdminCreateSolve(ctx context.Context, userID, teamID, challengeID uuid.UUID, skipCompetitionCheck bool) error
		AdminDeleteSolve(ctx context.Context, teamID, challengeID uuid.UUID) error
	}
)

// =============================================================================
// Tag
// =============================================================================

type (
	TagUseCase interface {
		Create(ctx context.Context, name, color string) (*domain.Tag, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Tag, error)
		GetAll(ctx context.Context) ([]*domain.Tag, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Tag, error)
		Update(ctx context.Context, ID uuid.UUID, name, color string) (*domain.Tag, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Hint
// =============================================================================

type (
	HintWithUnlockStatus struct {
		Hint     *domain.Hint
		Unlocked bool
	}

	HintUseCase interface {
		Create(ctx context.Context, challengeID uuid.UUID, title, content string, cost, orderIndex int) (*domain.Hint, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Hint, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*HintWithUnlockStatus, error)
		Update(ctx context.Context, ID uuid.UUID, title, content string, cost, orderIndex int) (*domain.Hint, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		UnlockHint(ctx context.Context, userID, teamID, challengeID, hintID uuid.UUID) (*domain.Hint, error)
		GetAllUnlocks(ctx context.Context, page, perPage int) (*Paginated[*domain.HintUnlockWithDetails], error)
	}
)

// =============================================================================
// File
// =============================================================================

type (
	FileUseCase interface {
		Upload(ctx context.Context, challengeID uuid.UUID, fileType domain.FileType, filename string, reader io.Reader, size int64, contentType string) (*domain.File, error)
		Download(ctx context.Context, path string) (io.ReadCloser, error)
		GetDownloadURL(ctx context.Context, fileID uuid.UUID) (string, error)
		GetDownloadURLWithAccess(ctx context.Context, fileID uuid.UUID, teamID *uuid.UUID, isAdmin bool) (string, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType domain.FileType) ([]*domain.File, error)
		GetByChallengeIDWithAccess(ctx context.Context, challengeID uuid.UUID, fileType domain.FileType, teamID *uuid.UUID, isAdmin bool) ([]*domain.File, error)
		VerifyDownloadTokenAndGetFile(ctx context.Context, path, token string) (*domain.File, error)
		Delete(ctx context.Context, fileID uuid.UUID) error
	}
)

// =============================================================================
// Award
// =============================================================================

type (
	AwardUseCase interface {
		Create(ctx context.Context, teamID uuid.UUID, value int, description string, createdBy uuid.UUID) (*domain.Award, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Award, error)
		GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.Award, error)
		GetAll(ctx context.Context) ([]*domain.Award, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Competition
// =============================================================================

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
// Email
// =============================================================================

type (
	EmailUseCase interface {
		IsEnabled() bool
		SendVerificationEmail(ctx context.Context, user *domain.User) error
		VerifyEmail(ctx context.Context, tokenStr string) error
		SendPasswordResetEmail(ctx context.Context, email string) error
		ResetPassword(ctx context.Context, tokenStr, newPassword string) error
		ResendVerification(ctx context.Context, userID uuid.UUID) error
	}
)

// =============================================================================
// Submission
// =============================================================================

type (
	SubmissionUseCase interface {
		LogSubmission(ctx context.Context, sub *domain.Submission) error
		AdminCreate(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID, challengeID uuid.UUID, submittedFlag string, isCorrect bool, ip string) (*domain.SubmissionWithDetails, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.SubmissionWithDetails, error)
		GetByChallenge(ctx context.Context, challengeID uuid.UUID, page, perPage int, forceLive bool) (*Paginated[*domain.SubmissionWithDetails], error)
		GetByUser(ctx context.Context, userID uuid.UUID, page, perPage int, forceLive bool) (*Paginated[*domain.SubmissionWithDetails], error)
		GetByTeam(ctx context.Context, teamID uuid.UUID, page, perPage int, forceLive bool) (*Paginated[*domain.SubmissionWithDetails], error)
		GetAll(ctx context.Context, page, perPage int, forceLive bool) (*Paginated[*domain.SubmissionWithDetails], error)
		GetStats(ctx context.Context, challengeID uuid.UUID, forceLive bool) (*domain.SubmissionStats, error)
		Update(ctx context.Context, ID uuid.UUID, isCorrect bool) (*domain.SubmissionWithDetails, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}

	// SubmissionBatcher queues submission log entries for asynchronous batch flush.
	SubmissionBatcher interface {
		Enqueue(sub *domain.Submission)
		Stop()
	}
)

// =============================================================================
// Tracking
// =============================================================================

type (
	TrackingUseCase interface {
		Track(ctx context.Context, userID uuid.UUID, ip, userAgent string) error
		TrackChallengeOpen(ctx context.Context, userID, challengeID uuid.UUID, ip string) error
		GetByUser(ctx context.Context, userID uuid.UUID, page, perPage int) (*Paginated[*domain.TrackingEntry], error)
	}
)

// =============================================================================
// Backup
// =============================================================================

type (
	CSVImportResult struct {
		Success       bool
		ImportedCount int
		Errors        []string
		SkippedCount  int
	}

	BackupUseCase interface {
		Export(ctx context.Context, opts domain.ExportOptions) (*domain.BackupData, error)
		ExportZIP(ctx context.Context, opts domain.ExportOptions) (io.ReadCloser, error)
		ImportZIP(ctx context.Context, r io.ReaderAt, size int64, opts domain.ImportOptions) (*domain.ImportResult, error)
		Reset(ctx context.Context, opts domain.AdminResetOptions) error
		ExportCSV(ctx context.Context, tableName string) ([]byte, error)
		ImportCSV(ctx context.Context, tableName string, data []byte) (*CSVImportResult, error)
	}
)

// =============================================================================
// Page
// =============================================================================

type (
	PageUseCase interface {
		GetPublishedList(ctx context.Context) ([]*domain.PageListItem, error)
		GetBySlug(ctx context.Context, slug string) (*domain.Page, error)
		Create(ctx context.Context, title, slug, content string, isDraft bool, orderIndex int) (*domain.Page, error)
		Update(ctx context.Context, ID uuid.UUID, title, slug, content string, isDraft bool, orderIndex int) (*domain.Page, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		GetAllList(ctx context.Context) ([]*domain.Page, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Page, error)
	}
)

// =============================================================================
// Notification
// =============================================================================

type (
	NotificationUseCase interface {
		CreateGlobal(ctx context.Context, title, content string, notifType domain.NotificationType, isPinned bool) (*domain.Notification, error)
		CreatePersonal(ctx context.Context, userID uuid.UUID, title, content string, notifType domain.NotificationType) (*domain.UserNotification, error)
		GetGlobal(ctx context.Context, page, perPage int) ([]*domain.Notification, error)
		GetUserNotifications(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*domain.UserNotification, error)
		MarkAsRead(ctx context.Context, ID, userID uuid.UUID) error
		CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
		Update(ctx context.Context, ID uuid.UUID, title, content string, notifType domain.NotificationType, isPinned bool) (*domain.Notification, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Bracket
// =============================================================================

type (
	BracketUseCase interface {
		Create(ctx context.Context, name, description string, isDefault bool) (*domain.Bracket, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Bracket, error)
		GetAll(ctx context.Context) ([]*domain.Bracket, error)
		Update(ctx context.Context, ID uuid.UUID, name, description string, isDefault bool) (*domain.Bracket, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Field
// =============================================================================

type (
	FieldUseCase interface {
		GetByEntityType(ctx context.Context, entityType domain.EntityType) ([]*domain.Field, error)
		Create(ctx context.Context, name string, fieldType domain.FieldType, entityType domain.EntityType, required bool, options []string, orderIndex int) (*domain.Field, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Field, error)
		GetAll(ctx context.Context) ([]*domain.Field, error)
		Update(ctx context.Context, ID uuid.UUID, name string, fieldType domain.FieldType, required bool, options []string, orderIndex int) (*domain.Field, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// API Token
// =============================================================================

type (
	APITokenUseCase interface {
		List(ctx context.Context, userID uuid.UUID) ([]*domain.APIToken, error)
		Create(ctx context.Context, userID uuid.UUID, description string, expiresAt *time.Time) (plaintext string, token *domain.APIToken, err error)
		Delete(ctx context.Context, ID, userID uuid.UUID) error
		GetByTokenHash(ctx context.Context, tokenHash string) (*domain.APIToken, error)
		UpdateLastUsedAt(ctx context.Context, ID uuid.UUID) error
		ValidateToken(t *domain.APIToken) bool
	}
)

// =============================================================================
// Comment
// =============================================================================

type (
	CommentUseCase interface {
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Comment, error)
		Create(ctx context.Context, userID, challengeID uuid.UUID, content string) (*domain.Comment, error)
		Delete(ctx context.Context, ID, userID uuid.UUID, isAdmin bool) error
	}
)

// =============================================================================
// Rating
// =============================================================================

type (
	RatingUseCase interface {
		PutRating(ctx context.Context, challengeID, userID, teamID uuid.UUID, value int, review string) (*domain.Rating, error)
		GetRatingsByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Rating, error)
	}
)

// =============================================================================
// Settings
// =============================================================================

type (
	SettingsUseCase interface {
		Get(ctx context.Context) (*domain.Settings, error)
		Update(ctx context.Context, s *domain.Settings, actorID uuid.UUID, clientIP string) error
	}
)

// =============================================================================
// Competition params (dynamic key-value)
// =============================================================================

type (
	CompetitionParamUseCase interface {
		Get(ctx context.Context, key string) (*domain.CompetitionParam, error)
		GetAll(ctx context.Context) ([]*domain.CompetitionParam, error)
		GetByCategory(ctx context.Context, category string) ([]*domain.CompetitionParam, error)
		Set(ctx context.Context, key, value, description string, valueType domain.CompetitionParamValueType, category string, actorID uuid.UUID, clientIP string) error
		SetBatch(ctx context.Context, params []*domain.CompetitionParam, actorID uuid.UUID, clientIP string) error
		Delete(ctx context.Context, key string, actorID uuid.UUID, clientIP string) error
		GetString(ctx context.Context, key, defaultVal string) string
		GetInt(ctx context.Context, key string, defaultVal int) int
		GetBool(ctx context.Context, key string, defaultVal bool) bool
	}
)

// =============================================================================
// OAuth
// =============================================================================

type (
	OAuthUseCase interface {
		GetAuthURL(ctx context.Context, provider string) (authURL, state string, err error)
		ValidateState(cookieState, queryState string) bool
		HandleCallback(ctx context.Context, provider, code string) (*jwtkit.TokenPair, error)
	}
)

// =============================================================================
// Avatar
// =============================================================================

type AvatarUseCase interface {
	UploadUserAvatar(ctx context.Context, userID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error)
	DeleteUserAvatar(ctx context.Context, userID uuid.UUID) error
	GetUserAvatarURL(ctx context.Context, userID uuid.UUID) (fullURL, thumbURL *string, err error)

	UploadTeamAvatar(ctx context.Context, teamID, callerID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error)
	DeleteTeamAvatar(ctx context.Context, teamID, callerID uuid.UUID) error
	GetTeamAvatarURL(ctx context.Context, teamID uuid.UUID) (fullURL, thumbURL *string, err error)

	AdminUploadUserAvatar(ctx context.Context, userID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error)
	AdminDeleteUserAvatar(ctx context.Context, userID uuid.UUID) error
	AdminUploadTeamAvatar(ctx context.Context, teamID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error)
	AdminDeleteTeamAvatar(ctx context.Context, teamID uuid.UUID) error
}

// =============================================================================
// Cleanup
// =============================================================================

// Cleaner describes the cleanup use case used by the standalone cleanup command.
type Cleaner interface {
	CleanupDeletedTeams(ctx context.Context, olderThan time.Duration) error
	CleanupOrphanedStorageFiles(ctx context.Context, prefix string) (int, error)
	CleanupOrphanedAvatars(ctx context.Context) (int, error)
	CleanupOldTracking(ctx context.Context, olderThan time.Duration) error
}

// =============================================================================
// Helpers
// =============================================================================

func NewPaginated[T any](data []T, total int64, page, perPage int) *Paginated[T] {
	return httputil.NewPaginated(data, total, page, perPage)
}

func FetchPage[T any](
	ctx context.Context,
	page, perPage int,
	fetchFn func(ctx context.Context, limit, offset int) ([]T, error),
	countFn func(ctx context.Context) (int64, error),
) (*Paginated[T], error) {
	return httputil.FetchPage(ctx, page, perPage, fetchFn, countFn)
}
