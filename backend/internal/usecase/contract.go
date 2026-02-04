package usecase

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
)

type (
	UserUseCase interface {
		Register(ctx context.Context, username, email, password string) (*entity.User, error)
		Login(ctx context.Context, email, password string) (*jwt.TokenPair, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.User, error)
		GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error)
	}

	UserProfile struct {
		User   *entity.User
		Solves []*entity.Solve
	}

	TeamUseCase interface {
		Create(ctx context.Context, name string, captainID uuid.UUID, isSolo, confirmReset bool) (*entity.Team, error)
		Join(ctx context.Context, inviteToken, userID uuid.UUID, confirmReset bool) (*entity.Team, error)
		Leave(ctx context.Context, userID uuid.UUID) error
		TransferCaptain(ctx context.Context, captainID, newCaptainID uuid.UUID) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Team, error)
		GetMyTeam(ctx context.Context, userID uuid.UUID) (*entity.Team, []*entity.User, error)
		GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*entity.User, error)
		CreateSoloTeam(ctx context.Context, userID uuid.UUID, confirmReset bool) (*entity.Team, error)
		DisbandTeam(ctx context.Context, captainID uuid.UUID) error
		KickMember(ctx context.Context, captainID, targetUserID uuid.UUID) error
		BanTeam(ctx context.Context, teamID uuid.UUID, reason string) error
		UnbanTeam(ctx context.Context, teamID uuid.UUID) error
		SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error
		SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error
	}

	ChallengeWithTags struct {
		*repo.ChallengeWithSolved
		Tags []*entity.Tag
	}

	ChallengeUseCase interface {
		GetAll(ctx context.Context, teamID, tagID *uuid.UUID) ([]*ChallengeWithTags, error)
		Create(ctx context.Context, title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*entity.Challenge, error)
		Update(ctx context.Context, ID uuid.UUID, title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*entity.Challenge, error)
		Delete(ctx context.Context, ID, actorID uuid.UUID, clientIP string) error
		SubmitFlag(ctx context.Context, challengeID uuid.UUID, flag string, userID uuid.UUID, teamID *uuid.UUID) (bool, error)
	}

	TagUseCase interface {
		Create(ctx context.Context, name, color string) (*entity.Tag, error)
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Tag, error)
		GetAll(ctx context.Context) ([]*entity.Tag, error)
		Update(ctx context.Context, id uuid.UUID, name, color string) (*entity.Tag, error)
		Delete(ctx context.Context, id uuid.UUID) error
	}

	HintUseCase interface {
		Create(ctx context.Context, challengeID uuid.UUID, content string, cost, orderIndex int) (*entity.Hint, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Hint, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*HintWithUnlockStatus, error)
		Update(ctx context.Context, ID uuid.UUID, content string, cost, orderIndex int) (*entity.Hint, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		UnlockHint(ctx context.Context, teamID, hintID uuid.UUID) (*entity.Hint, error)
	}

	HintWithUnlockStatus struct {
		Hint     *entity.Hint
		Unlocked bool
	}

	FileUseCase interface {
		Upload(ctx context.Context, challengeID uuid.UUID, fileType entity.FileType, filename string, reader io.Reader, size int64, contentType string) (*entity.File, error)
		Download(ctx context.Context, path string) (io.ReadCloser, error)
		GetDownloadURL(ctx context.Context, fileID uuid.UUID) (string, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType entity.FileType) ([]*entity.File, error)
		Delete(ctx context.Context, fileID uuid.UUID) error
	}

	AwardUseCase interface {
		Create(ctx context.Context, teamID uuid.UUID, value int, description string, createdBy uuid.UUID) (*entity.Award, error)
		GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Award, error)
	}

	CompetitionUseCase interface {
		Get(ctx context.Context) (*entity.Competition, error)
		Update(ctx context.Context, comp *entity.Competition, actorID uuid.UUID, clientIP string) error
		GetStatus(ctx context.Context) (entity.CompetitionStatus, error)
		IsSubmissionAllowed(ctx context.Context) (bool, error)
	}

	SolveUseCase interface {
		Create(ctx context.Context, solve *entity.Solve) error
		GetScoreboard(ctx context.Context, bracketID *uuid.UUID) ([]*repo.ScoreboardEntry, error)
		GetFirstBlood(ctx context.Context, challengeID uuid.UUID) (*repo.FirstBloodEntry, error)
	}

	StatisticsUseCase interface {
		GetGeneralStats(ctx context.Context) (*entity.GeneralStats, error)
		GetChallengeStats(ctx context.Context) ([]*entity.ChallengeStats, error)
		GetScoreboardHistory(ctx context.Context, limit int) ([]*entity.ScoreboardHistoryEntry, error)
		GetScoreboardGraph(ctx context.Context, topN int) (*entity.ScoreboardGraph, error)
	}

	EmailUseCase interface {
		IsEnabled() bool
		SendVerificationEmail(ctx context.Context, user *entity.User) error
		VerifyEmail(ctx context.Context, tokenStr string) error
		SendPasswordResetEmail(ctx context.Context, email string) error
		ResetPassword(ctx context.Context, tokenStr, newPassword string) error
		ResendVerification(ctx context.Context, userID uuid.UUID) error
	}

	SubmissionUseCase interface {
		LogSubmission(ctx context.Context, sub *entity.Submission) error
		GetByChallenge(ctx context.Context, challengeID uuid.UUID, page, perPage int) ([]*entity.SubmissionWithDetails, int64, error)
		GetByUser(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.SubmissionWithDetails, int64, error)
		GetByTeam(ctx context.Context, teamID uuid.UUID, page, perPage int) ([]*entity.SubmissionWithDetails, int64, error)
		GetAll(ctx context.Context, page, perPage int) ([]*entity.SubmissionWithDetails, int64, error)
		GetStats(ctx context.Context, challengeID uuid.UUID) (*entity.SubmissionStats, error)
	}

	BackupUseCase interface {
		Export(ctx context.Context, opts entity.ExportOptions) (*entity.BackupData, error)
		ExportZIP(ctx context.Context, opts entity.ExportOptions) (io.ReadCloser, error)
		ImportZIP(ctx context.Context, r io.ReaderAt, size int64, opts entity.ImportOptions) (*entity.ImportResult, error)
	}

	PageUseCase interface {
		GetPublishedList(ctx context.Context) ([]*entity.PageListItem, error)
		GetBySlug(ctx context.Context, slug string) (*entity.Page, error)
		Create(ctx context.Context, title, slug, content string, isDraft bool, orderIndex int) (*entity.Page, error)
		Update(ctx context.Context, id uuid.UUID, title, slug, content string, isDraft bool, orderIndex int) (*entity.Page, error)
		Delete(ctx context.Context, id uuid.UUID) error
		GetAllList(ctx context.Context) ([]*entity.Page, error)
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Page, error)
	}

	NotificationUseCase interface {
		CreateGlobal(ctx context.Context, title, content string, notifType entity.NotificationType, isPinned bool) (*entity.Notification, error)
		CreatePersonal(ctx context.Context, userID uuid.UUID, title, content string, notifType entity.NotificationType) (*entity.UserNotification, error)
		GetGlobal(ctx context.Context, page, perPage int) ([]*entity.Notification, error)
		GetUserNotifications(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.UserNotification, error)
		MarkAsRead(ctx context.Context, id, userID uuid.UUID) error
		CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
		Update(ctx context.Context, id uuid.UUID, title, content string, notifType entity.NotificationType, isPinned bool) (*entity.Notification, error)
		Delete(ctx context.Context, id uuid.UUID) error
	}

	BracketUseCase interface {
		Create(ctx context.Context, name, description string, isDefault bool) (*entity.Bracket, error)
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Bracket, error)
		GetAll(ctx context.Context) ([]*entity.Bracket, error)
		Update(ctx context.Context, id uuid.UUID, name, description string, isDefault bool) (*entity.Bracket, error)
		Delete(ctx context.Context, id uuid.UUID) error
	}

	FieldUseCase interface {
		GetByEntityType(ctx context.Context, entityType entity.EntityType) ([]*entity.Field, error)
		Create(ctx context.Context, name string, fieldType entity.FieldType, entityType entity.EntityType, required bool, options []string, orderIndex int) (*entity.Field, error)
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Field, error)
		GetAll(ctx context.Context) ([]*entity.Field, error)
		Update(ctx context.Context, id uuid.UUID, name string, fieldType entity.FieldType, required bool, options []string, orderIndex int) (*entity.Field, error)
		Delete(ctx context.Context, id uuid.UUID) error
	}

	APITokenUseCase interface {
		List(ctx context.Context, userID uuid.UUID) ([]*entity.APIToken, error)
		Create(ctx context.Context, userID uuid.UUID, description string, expiresAt *time.Time) (plaintext string, token *entity.APIToken, err error)
		Delete(ctx context.Context, id, userID uuid.UUID) error
		GetByTokenHash(ctx context.Context, tokenHash string) (*entity.APIToken, error)
		UpdateLastUsedAt(ctx context.Context, id uuid.UUID) error
		ValidateToken(t *entity.APIToken) bool
	}

	CommentUseCase interface {
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Comment, error)
		Create(ctx context.Context, userID, challengeID uuid.UUID, content string) (*entity.Comment, error)
		Delete(ctx context.Context, id, userID uuid.UUID) error
	}

	RatingUseCase interface {
		GetGlobalRatings(ctx context.Context, page, perPage int) ([]*entity.GlobalRating, int64, error)
		GetTeamRating(ctx context.Context, teamID uuid.UUID) (*entity.GlobalRating, []*entity.TeamRating, error)
		GetCTFEvents(ctx context.Context) ([]*entity.CTFEvent, error)
		CreateCTFEvent(ctx context.Context, name string, startTime, endTime time.Time, weight float64) (*entity.CTFEvent, error)
		FinalizeCTFEvent(ctx context.Context, eventID uuid.UUID) error
	}
)
