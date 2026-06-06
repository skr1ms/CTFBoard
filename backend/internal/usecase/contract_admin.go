package usecase

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// =============================================================================
// Backup
// =============================================================================

type (
	// CSVImportResult reports the outcome of a CSV import operation including counts and per-row errors.
	CSVImportResult struct {
		Success       bool
		ImportedCount int
		Errors        []string
		SkippedCount  int
	}

	// BackupUseCase handles full-platform export/import (ZIP, JSON, CSV) and admin reset operations.
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
	PageCreateParams struct {
		Title      string
		Slug       string
		Content    string
		IsDraft    bool
		OrderIndex int
	}

	PageUpdateParams struct {
		Title      string
		Slug       string
		Content    string
		IsDraft    bool
		OrderIndex int
	}

	// PageUseCase manages static content pages including draft/publish lifecycle.
	PageUseCase interface {
		GetPublishedList(ctx context.Context) ([]*domain.PageListItem, error)
		GetBySlug(ctx context.Context, slug string) (*domain.Page, error)
		Create(ctx context.Context, params PageCreateParams) (*domain.Page, error)
		Update(ctx context.Context, ID uuid.UUID, params PageUpdateParams) (*domain.Page, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		GetAllList(ctx context.Context) ([]*domain.Page, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Page, error)
	}
)

// =============================================================================
// Notification
// =============================================================================

type (
	NotificationCreateGlobalParams struct {
		Title    string
		Content  string
		Type     domain.NotificationType
		IsPinned bool
	}

	NotificationCreatePersonalParams struct {
		UserID  uuid.UUID
		Title   string
		Content string
		Type    domain.NotificationType
	}

	NotificationUpdateParams struct {
		ID       uuid.UUID
		Title    string
		Content  string
		Type     domain.NotificationType
		IsPinned bool
	}

	// NotificationUseCase manages global and personal notifications, including read state tracking.
	NotificationUseCase interface {
		CreateGlobal(ctx context.Context, params NotificationCreateGlobalParams) (*domain.Notification, error)
		CreatePersonal(ctx context.Context, params NotificationCreatePersonalParams) (*domain.UserNotification, error)
		GetGlobal(ctx context.Context, page, perPage int) ([]*domain.Notification, error)
		GetUserNotifications(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*domain.UserNotification, error)
		MarkAsRead(ctx context.Context, ID, userID uuid.UUID) error
		CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
		Update(ctx context.Context, params NotificationUpdateParams) (*domain.Notification, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Bracket
// =============================================================================

type (
	// BracketUseCase manages competition brackets used to segment teams in the scoreboard.
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
	// FieldUseCase manages custom registration fields attached to users or teams.
	FieldCreateParams struct {
		Name       string
		FieldType  domain.FieldType
		EntityType domain.EntityType
		Required   bool
		Options    []string
		OrderIndex int
	}

	FieldUpdateParams struct {
		Name       string
		FieldType  domain.FieldType
		Required   bool
		Options    []string
		OrderIndex int
	}

	FieldUseCase interface {
		GetByEntityType(ctx context.Context, entityType domain.EntityType) ([]*domain.Field, error)
		Create(ctx context.Context, params FieldCreateParams) (*domain.Field, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Field, error)
		GetAll(ctx context.Context) ([]*domain.Field, error)
		Update(ctx context.Context, ID uuid.UUID, params FieldUpdateParams) (*domain.Field, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// API Token
// =============================================================================

type (
	// APITokenUseCase manages long-lived API tokens for programmatic access on behalf of users.
	APITokenUseCase interface {
		List(ctx context.Context, userID uuid.UUID) ([]*domain.APIToken, error)
		Create(ctx context.Context, userID uuid.UUID, description string, expiresAt *time.Time) (plaintext string, token *domain.APIToken, err error)
		Delete(ctx context.Context, ID, userID uuid.UUID) error
		AuthenticatePlaintext(ctx context.Context, plaintext string) (*domain.APIToken, error)
		UpdateLastUsedAt(ctx context.Context, ID uuid.UUID) error
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
// Avatar
// =============================================================================

// AvatarUseCase handles upload, deletion, and URL retrieval for user and team avatars.
type AvatarUseCase interface {
	UploadUserAvatar(ctx context.Context, userID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error)
	DeleteUserAvatar(ctx context.Context, userID uuid.UUID) error
	GetUserAvatarURL(ctx context.Context, userID uuid.UUID) (fullURL, thumbURL *string, err error)

	UploadTeamAvatar(ctx context.Context, teamID, callerID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error)
	DeleteTeamAvatar(ctx context.Context, teamID, callerID uuid.UUID) error
	GetTeamAvatarURL(ctx context.Context, teamID uuid.UUID) (fullURL, thumbURL *string, err error)
	GetTeamAvatarURLBatch(ctx context.Context, teamIDs []uuid.UUID) (map[uuid.UUID]string, error)
	GetTeamAvatarStoragePathBatch(ctx context.Context, teamIDs []uuid.UUID) (map[uuid.UUID]string, error)

	AdminUploadUserAvatar(ctx context.Context, userID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error)
	AdminDeleteUserAvatar(ctx context.Context, userID uuid.UUID) error
	AdminUploadTeamAvatar(ctx context.Context, teamID uuid.UUID, file io.Reader, filename string, size int64) (fullURL, thumbURL string, err error)
	AdminDeleteTeamAvatar(ctx context.Context, teamID uuid.UUID) error
}
