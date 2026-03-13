package usecase

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
)

const (
	DefaultPerPage    = 20
	DefaultMaxPerPage = 100
)

// =============================================================================
// Shared
// =============================================================================

type (
	Paginated[T any] struct {
		Data       []T   `json:"data"`
		Total      int64 `json:"total"`
		Page       int   `json:"page"`
		PerPage    int   `json:"per_page"`
		TotalPages int   `json:"total_pages"`
	}
)

// =============================================================================
// User
// =============================================================================

type (
	UserProfile struct {
		User   *entity.User
		Solves []*entity.Solve
	}

	UserUseCase interface {
		Register(ctx context.Context, username, email, password string, customFields map[string]string) (*entity.User, error)
		Login(ctx context.Context, email, password string) (*jwt.TokenPair, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.User, error)
		GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error)
		ListUsers(ctx context.Context, search *string, field string, page, perPage int) (*Paginated[*entity.User], error)
		GetUserSolves(ctx context.Context, userID uuid.UUID) ([]*entity.SolveWithDetails, error)
		GetUserFails(ctx context.Context, userID uuid.UUID, page, perPage int) (*Paginated[*entity.SubmissionWithDetails], error)
		GetUserAwards(ctx context.Context, userID uuid.UUID) ([]*entity.Award, error)
		AdminCreate(ctx context.Context, username, email, password, role string) (*entity.User, error)
		AdminUpdate(ctx context.Context, userID uuid.UUID, username, email, role, password *string, isVerified *bool) (*entity.User, error)
		AdminDelete(ctx context.Context, userID, actorID uuid.UUID) error
		BanUser(ctx context.Context, userID uuid.UUID, reason string, actorID uuid.UUID) error
		UnbanUser(ctx context.Context, userID, actorID uuid.UUID) error
		UpdateProfile(ctx context.Context, userID uuid.UUID, username, email, currentPassword, newPassword *string) (*entity.User, error)
		GetMySubmissions(ctx context.Context, userID uuid.UUID, page, perPage int) (*Paginated[*entity.SubmissionWithDetails], error)
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
		Team               *entity.Team
		RequiresConfirm    bool
		ConfirmationReason ConfirmReason
		AffectedData       *TeamCreateAffectedData
	}

	TeamUseCase interface {
		Create(ctx context.Context, name string, captainID uuid.UUID, isSolo, confirmReset bool) (*entity.Team, error)
		TryCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*TeamCreateResult, error)
		ConfirmCreate(ctx context.Context, name string, captainID uuid.UUID, isSolo bool) (*entity.Team, error)
		Join(ctx context.Context, inviteToken, userID uuid.UUID, confirmReset bool) (*entity.Team, error)
		Leave(ctx context.Context, userID uuid.UUID) error
		TransferCaptain(ctx context.Context, captainID, newCaptainID uuid.UUID) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Team, error)
		GetMyTeam(ctx context.Context, userID uuid.UUID) (*entity.Team, []*entity.User, int, bool, error)
		GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*entity.User, error)
		CreateSoloTeam(ctx context.Context, userID uuid.UUID, confirmReset bool) (*entity.Team, error)
		DisbandTeam(ctx context.Context, captainID uuid.UUID) error
		KickMember(ctx context.Context, captainID, targetUserID uuid.UUID) error
		BanTeam(ctx context.Context, teamID uuid.UUID, reason string, banMembers bool, actorID uuid.UUID) error
		UnbanTeam(ctx context.Context, teamID, actorID uuid.UUID) error
		SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error
		SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error
		ListTeams(ctx context.Context, search *string, page, perPage int) (*Paginated[*entity.Team], error)
		AdminListTeams(ctx context.Context, search *string, page, perPage int) (*Paginated[*entity.Team], error)
		GetTeamSolves(ctx context.Context, teamID uuid.UUID) ([]*entity.SolveWithDetails, error)
		GetTeamFails(ctx context.Context, teamID uuid.UUID, page, perPage int) (*Paginated[*entity.SubmissionWithDetails], error)
		GetTeamAwards(ctx context.Context, teamID uuid.UUID) ([]*entity.Award, error)
		AdminUpdate(ctx context.Context, teamID uuid.UUID, name *string, captainID, bracketID *uuid.UUID, isHidden *bool) (*entity.Team, error)
		AdminDelete(ctx context.Context, teamID uuid.UUID) error
		AdminGetMembers(ctx context.Context, teamID uuid.UUID) ([]*entity.User, error)
		AdminAddMember(ctx context.Context, teamID, userID uuid.UUID) error
		AdminRemoveMember(ctx context.Context, teamID, userID uuid.UUID) error
		UpdateMyTeam(ctx context.Context, captainID uuid.UUID, name string) (*entity.Team, error)
		GetInviteToken(ctx context.Context, captainID uuid.UUID) (*entity.Team, error)
		RegenerateInviteToken(ctx context.Context, captainID uuid.UUID) (*entity.Team, error)
	}
)

// =============================================================================
// Challenge
// =============================================================================

type (
	ChallengeWithTags struct {
		*entity.ChallengeWithSolved
		Tags []*entity.Tag
	}

	ChallengeDetail struct {
		Challenge  *entity.Challenge
		Tags       []*entity.Tag
		Files      []*entity.File
		Hints      []*HintWithUnlockStatus
		FirstBlood *entity.FirstBloodEntry
		SolvedByMe bool
		SolveCount int
	}

	ChallengeUseCase interface {
		GetAll(ctx context.Context, teamID, tagID *uuid.UUID) ([]*ChallengeWithTags, error)
		GetByID(ctx context.Context, challengeID uuid.UUID) (*entity.Challenge, error)
		GetDetail(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (*ChallengeDetail, error)
		GetSolves(ctx context.Context, challengeID uuid.UUID) ([]*entity.SolveWithDetails, error)
		GetTags(ctx context.Context, challengeID uuid.UUID) ([]*entity.Tag, error)
		GetRequirements(ctx context.Context, challengeID uuid.UUID) ([]*entity.ChallengeRequirement, error)
		SetRequirements(ctx context.Context, challengeID uuid.UUID, requirementIDs []uuid.UUID) error
		GetSolution(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (*entity.ChallengeSolution, error)
		ListSolutions(ctx context.Context, teamID uuid.UUID) ([]*entity.ChallengeSolutionEntry, error)
		AdminUpsertSolution(ctx context.Context, challengeID uuid.UUID, content string) (*entity.ChallengeSolution, error)
		AdminDeleteSolution(ctx context.Context, challengeID uuid.UUID) error
		GetFlags(ctx context.Context, challengeID uuid.UUID) (*entity.ChallengeFlags, error)
		GetTypes(ctx context.Context) ([]string, error)
		GetMissingChallengesByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Challenge, error)
		GetMissingChallengesByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Challenge, error)
		Create(ctx context.Context, title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*entity.Challenge, error)
		Update(ctx context.Context, ID uuid.UUID, title, description, category string, points int, initialValue, minValue, decay *int, flag string, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*entity.Challenge, error)
		Delete(ctx context.Context, ID, actorID uuid.UUID, clientIP string) error
		SubmitFlag(ctx context.Context, challengeID uuid.UUID, flag string, userID uuid.UUID, teamID *uuid.UUID) (bool, error)
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
		Create(ctx context.Context, name, color string) (*entity.Tag, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Tag, error)
		GetAll(ctx context.Context) ([]*entity.Tag, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Tag, error)
		Update(ctx context.Context, ID uuid.UUID, name, color string) (*entity.Tag, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Hint
// =============================================================================

type (
	HintWithUnlockStatus struct {
		Hint     *entity.Hint
		Unlocked bool
	}

	HintUseCase interface {
		Create(ctx context.Context, challengeID uuid.UUID, content string, cost, orderIndex int) (*entity.Hint, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Hint, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*HintWithUnlockStatus, error)
		Update(ctx context.Context, ID uuid.UUID, content string, cost, orderIndex int) (*entity.Hint, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		UnlockHint(ctx context.Context, userID, teamID, challengeID, hintID uuid.UUID) (*entity.Hint, error)
		GetAllUnlocks(ctx context.Context, page, perPage int) (*Paginated[*entity.HintUnlockWithDetails], error)
	}
)

// =============================================================================
// File
// =============================================================================

type (
	FileUseCase interface {
		Upload(ctx context.Context, challengeID uuid.UUID, fileType entity.FileType, filename string, reader io.Reader, size int64, contentType string) (*entity.File, error)
		Download(ctx context.Context, path string) (io.ReadCloser, error)
		GetDownloadURL(ctx context.Context, fileID uuid.UUID) (string, error)
		GetDownloadURLWithAccess(ctx context.Context, fileID uuid.UUID, teamID *uuid.UUID, isAdmin bool) (string, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType entity.FileType) ([]*entity.File, error)
		GetByChallengeIDWithAccess(ctx context.Context, challengeID uuid.UUID, fileType entity.FileType, teamID *uuid.UUID, isAdmin bool) ([]*entity.File, error)
		VerifyDownloadTokenAndGetFile(ctx context.Context, path, token string) (*entity.File, error)
		Delete(ctx context.Context, fileID uuid.UUID) error
	}
)

// =============================================================================
// Award
// =============================================================================

type (
	AwardUseCase interface {
		Create(ctx context.Context, teamID uuid.UUID, value int, description string, createdBy uuid.UUID) (*entity.Award, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Award, error)
		GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Award, error)
		GetAll(ctx context.Context) ([]*entity.Award, error)
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
		Get(ctx context.Context) (*entity.Competition, error)
		Update(ctx context.Context, comp *entity.Competition, optionals *CompetitionUpdateOptionals, actorID uuid.UUID, clientIP string) error
		GetStatus(ctx context.Context) (entity.CompetitionStatus, error)
		IsSubmissionAllowed(ctx context.Context) (bool, error)
	}

	CompetitionGuard interface {
		Get(ctx context.Context) (*entity.Competition, error)
		RequireTeamSwitch(ctx context.Context) (*entity.Competition, error)
		RequireTeamSwitchAndTeamsMode(ctx context.Context) (*entity.Competition, error)
		RequireTeamSwitchAndSoloMode(ctx context.Context) (*entity.Competition, error)
	}

	// SettingsGetter returns app settings (e.g. via SettingsUseCase for singleflight/cache).
	SettingsGetter interface {
		Get(ctx context.Context) (*entity.Settings, error)
	}
)

// =============================================================================
// Solve
// =============================================================================

type (
	SolveUseCase interface {
		Create(ctx context.Context, solve *entity.Solve) error
		GetScoreboard(ctx context.Context, bracketID *uuid.UUID, forceLive bool) ([]*entity.ScoreboardEntry, error)
		GetFirstBlood(ctx context.Context, challengeID uuid.UUID, forceLive bool) (*entity.FirstBloodEntry, error)
	}
)

// =============================================================================
// Statistics
// =============================================================================

type (
	StatisticsUseCase interface {
		GetGeneralStats(ctx context.Context, forceLive bool) (*entity.GeneralStats, error)
		GetChallengeStats(ctx context.Context, forceLive bool) ([]*entity.ChallengeStats, error)
		GetChallengeDetailStats(ctx context.Context, challengeID string, forceLive bool) (*entity.ChallengeDetailStats, error)
		GetScoreboardHistory(ctx context.Context, limit int, forceLive bool) ([]*entity.ScoreboardHistoryEntry, error)
		GetScoreboardGraph(ctx context.Context, topN int, forceLive bool) (*entity.ScoreboardGraph, error)
		GetChallengeSolvePercentages(ctx context.Context, forceLive bool) ([]*entity.ChallengeSolvePercentage, error)
		GetScoreDistribution(ctx context.Context, forceLive bool) ([]*entity.ScoreDistributionBucket, error)
		GetSubmissionTimeSeries(ctx context.Context, forceLive bool) (*entity.SubmissionTimeSeriesStats, error)
		GetSubmissionTimeSeriesByType(ctx context.Context, isCorrect, forceLive bool) ([]*entity.RegistrationTimePoint, error)
		GetTeamRegistrationTimeSeries(ctx context.Context) ([]*entity.RegistrationTimePoint, error)
		GetUserRegistrationTimeSeries(ctx context.Context) ([]*entity.RegistrationTimePoint, error)
		GetSolveMatrix(ctx context.Context, forceLive bool) ([]*entity.SolveMatrixRow, error)
	}
)

// =============================================================================
// Email
// =============================================================================

type (
	EmailUseCase interface {
		IsEnabled() bool
		SendVerificationEmail(ctx context.Context, user *entity.User) error
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
		LogSubmission(ctx context.Context, sub *entity.Submission) error
		AdminCreate(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID, challengeID uuid.UUID, submittedFlag string, isCorrect bool, ip string) (*entity.SubmissionWithDetails, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.SubmissionWithDetails, error)
		GetByChallenge(ctx context.Context, challengeID uuid.UUID, page, perPage int, forceLive bool) (*Paginated[*entity.SubmissionWithDetails], error)
		GetByUser(ctx context.Context, userID uuid.UUID, page, perPage int, forceLive bool) (*Paginated[*entity.SubmissionWithDetails], error)
		GetByTeam(ctx context.Context, teamID uuid.UUID, page, perPage int, forceLive bool) (*Paginated[*entity.SubmissionWithDetails], error)
		GetAll(ctx context.Context, page, perPage int, forceLive bool) (*Paginated[*entity.SubmissionWithDetails], error)
		GetStats(ctx context.Context, challengeID uuid.UUID, forceLive bool) (*entity.SubmissionStats, error)
		Update(ctx context.Context, ID uuid.UUID, isCorrect bool) (*entity.SubmissionWithDetails, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}

	// SubmissionBatcher queues submission log entries for asynchronous batch flush.
	SubmissionBatcher interface {
		Enqueue(sub *entity.Submission)
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
		GetByUser(ctx context.Context, userID uuid.UUID, page, perPage int) (*Paginated[*entity.TrackingEntry], error)
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
		Export(ctx context.Context, opts entity.ExportOptions) (*entity.BackupData, error)
		ExportZIP(ctx context.Context, opts entity.ExportOptions) (io.ReadCloser, error)
		ImportZIP(ctx context.Context, r io.ReaderAt, size int64, opts entity.ImportOptions) (*entity.ImportResult, error)
		Reset(ctx context.Context, opts entity.AdminResetOptions) error
		ExportCSV(ctx context.Context, tableName string) ([]byte, error)
		ImportCSV(ctx context.Context, tableName string, data []byte) (*CSVImportResult, error)
	}
)

// =============================================================================
// Page
// =============================================================================

type (
	PageUseCase interface {
		GetPublishedList(ctx context.Context) ([]*entity.PageListItem, error)
		GetBySlug(ctx context.Context, slug string) (*entity.Page, error)
		Create(ctx context.Context, title, slug, content string, isDraft bool, orderIndex int) (*entity.Page, error)
		Update(ctx context.Context, ID uuid.UUID, title, slug, content string, isDraft bool, orderIndex int) (*entity.Page, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		GetAllList(ctx context.Context) ([]*entity.Page, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Page, error)
	}
)

// =============================================================================
// Notification
// =============================================================================

type (
	NotificationUseCase interface {
		CreateGlobal(ctx context.Context, title, content string, notifType entity.NotificationType, isPinned bool) (*entity.Notification, error)
		CreatePersonal(ctx context.Context, userID uuid.UUID, title, content string, notifType entity.NotificationType) (*entity.UserNotification, error)
		GetGlobal(ctx context.Context, page, perPage int) ([]*entity.Notification, error)
		GetUserNotifications(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.UserNotification, error)
		MarkAsRead(ctx context.Context, ID, userID uuid.UUID) error
		CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
		Update(ctx context.Context, ID uuid.UUID, title, content string, notifType entity.NotificationType, isPinned bool) (*entity.Notification, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Bracket
// =============================================================================

type (
	BracketUseCase interface {
		Create(ctx context.Context, name, description string, isDefault bool) (*entity.Bracket, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Bracket, error)
		GetAll(ctx context.Context) ([]*entity.Bracket, error)
		Update(ctx context.Context, ID uuid.UUID, name, description string, isDefault bool) (*entity.Bracket, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Field
// =============================================================================

type (
	FieldUseCase interface {
		GetByEntityType(ctx context.Context, entityType entity.EntityType) ([]*entity.Field, error)
		Create(ctx context.Context, name string, fieldType entity.FieldType, entityType entity.EntityType, required bool, options []string, orderIndex int) (*entity.Field, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Field, error)
		GetAll(ctx context.Context) ([]*entity.Field, error)
		Update(ctx context.Context, ID uuid.UUID, name string, fieldType entity.FieldType, required bool, options []string, orderIndex int) (*entity.Field, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// API Token
// =============================================================================

type (
	APITokenUseCase interface {
		List(ctx context.Context, userID uuid.UUID) ([]*entity.APIToken, error)
		Create(ctx context.Context, userID uuid.UUID, description string, expiresAt *time.Time) (plaintext string, token *entity.APIToken, err error)
		Delete(ctx context.Context, ID, userID uuid.UUID) error
		GetByTokenHash(ctx context.Context, tokenHash string) (*entity.APIToken, error)
		UpdateLastUsedAt(ctx context.Context, ID uuid.UUID) error
		ValidateToken(t *entity.APIToken) bool
	}
)

// =============================================================================
// Comment
// =============================================================================

type (
	CommentUseCase interface {
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Comment, error)
		Create(ctx context.Context, userID, challengeID uuid.UUID, content string) (*entity.Comment, error)
		Delete(ctx context.Context, ID, userID uuid.UUID, isAdmin bool) error
	}
)

// =============================================================================
// Settings
// =============================================================================

type (
	SettingsUseCase interface {
		Get(ctx context.Context) (*entity.Settings, error)
		Update(ctx context.Context, s *entity.Settings, actorID uuid.UUID, clientIP string) error
	}
)

// =============================================================================
// Competition params (dynamic key-value)
// =============================================================================

type (
	CompetitionParamUseCase interface {
		Get(ctx context.Context, key string) (*entity.CompetitionParam, error)
		GetAll(ctx context.Context) ([]*entity.CompetitionParam, error)
		GetByCategory(ctx context.Context, category string) ([]*entity.CompetitionParam, error)
		Set(ctx context.Context, key, value, description string, valueType entity.CompetitionParamValueType, actorID uuid.UUID, clientIP string) error
		SetBatch(ctx context.Context, params []*entity.CompetitionParam, actorID uuid.UUID, clientIP string) error
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
		HandleCallback(ctx context.Context, provider, code string) (*jwt.TokenPair, error)
	}
)

// =============================================================================
// Cleanup
// =============================================================================

// Cleaner describes the cleanup use case used by the standalone cleanup command.
type Cleaner interface {
	CleanupDeletedTeams(ctx context.Context, olderThan time.Duration) error
	CleanupOrphanedStorageFiles(ctx context.Context, prefix string) (int, error)
}

// =============================================================================
// Helpers
// =============================================================================

func NewPaginated[T any](data []T, total int64, page, perPage int) *Paginated[T] {
	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}
	return &Paginated[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}
}

// FetchPage runs two goroutines in parallel: one to fetch a page of items and
// one to count the total matching rows. It returns a Paginated result or the
// first error encountered.
func FetchPage[T any](
	ctx context.Context,
	page, perPage int,
	fetchFn func(ctx context.Context, limit, offset int) ([]T, error),
	countFn func(ctx context.Context) (int64, error),
) (*Paginated[T], error) {
	offset := (page - 1) * perPage
	var items []T
	var total int64

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		items, err = fetchFn(gCtx, perPage, offset)
		return err
	})
	g.Go(func() error {
		var err error
		total, err = countFn(gCtx)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return NewPaginated(items, total, page, perPage), nil
}
