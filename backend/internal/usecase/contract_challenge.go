package usecase

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// =============================================================================
// Challenge
// =============================================================================

type (
	// ChallengeWithTags is a challenge enriched with its associated tags and whether prerequisites are met.
	ChallengeWithTags struct {
		*domain.ChallengeWithSolved

		Tags            []*domain.Tag
		RequirementsMet *bool
	}

	// ChallengeDetail is the full challenge detail view used on the challenge detail page.
	ChallengeDetail struct {
		Challenge  *domain.Challenge
		Tags       []*domain.Tag
		Files      []*domain.File
		Hints      []*HintWithUnlockStatus
		FirstBlood *domain.FirstBloodEntry
		SolvedByMe bool
		SolveCount int
	}

	ChallengeCreateParams struct {
		Title             string
		Description       string
		Category          string
		Points            int
		InitialValue      int
		MinValue          int
		Decay             int
		Flag              string
		Attribution       string
		ConnectionInfo    string
		MaxAttempts       int
		MaxAttemptsWindow time.Duration
		Position          int
		NextChallengeID   *uuid.UUID
		State             string
		IsRegex           bool
		IsCaseInsensitive bool
		FlagFormatRegex   *string
		TagIDs            []uuid.UUID
	}

	ChallengeUpdateParams struct {
		Title              string
		Description        string
		Category           string
		Points             int
		InitialValue       *int
		MinValue           *int
		Decay              *int
		Flag               string
		Attribution        *string
		ConnectionInfo     *string
		MaxAttempts        *int
		MaxAttemptsWindow  *time.Duration
		Position           *int
		NextChallengeID    *uuid.UUID
		NextChallengeSet   bool
		State              string
		IsRegex            *bool
		IsCaseInsensitive  *bool
		FlagFormatRegex    *string
		FlagFormatRegexSet bool
		TagIDs             []uuid.UUID
	}

	ChallengeSubmitParams struct {
		ChallengeID uuid.UUID
		Flag        string
		UserID      uuid.UUID
		TeamID      *uuid.UUID
		ClientIP    string
	}

	ChallengeSolutionUpsertParams struct {
		Content string
		State   string
	}

	// ChallengeReadUseCase exposes participant-facing challenge reads.
	ChallengeReadUseCase interface {
		GetAll(ctx context.Context, teamID, tagID *uuid.UUID) ([]*ChallengeWithTags, error)
		GetByID(ctx context.Context, challengeID uuid.UUID) (*domain.Challenge, error)
		GetDetail(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (*ChallengeDetail, error)
		GetSolves(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*domain.SolveWithDetails, error)
		GetTags(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*domain.Tag, error)
		GetRequirements(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID, isAdmin bool) ([]*domain.ChallengeRequirement, error)
		GetSolution(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID, isAdmin bool) (*domain.ChallengeSolution, error)
		ListSolutions(ctx context.Context, teamID *uuid.UUID, isAdmin bool) ([]*domain.ChallengeSolutionEntry, error)
		GetTypes(ctx context.Context) ([]string, error)
	}

	// ChallengeSubmitUseCase exposes flag submission.
	ChallengeSubmitUseCase interface {
		SubmitFlag(ctx context.Context, params ChallengeSubmitParams) (bool, error)
	}

	// ChallengeAdminUseCase exposes administrative challenge mutations and views.
	ChallengeAdminUseCase interface {
		SetRequirements(ctx context.Context, challengeID uuid.UUID, requirementIDs []uuid.UUID) error
		AdminUpsertSolution(ctx context.Context, challengeID uuid.UUID, params ChallengeSolutionUpsertParams) (*domain.ChallengeSolution, error)
		AdminDeleteSolution(ctx context.Context, challengeID uuid.UUID) error
		GetFlags(ctx context.Context, challengeID uuid.UUID) (*domain.ChallengeFlags, error)
		GetMissingChallengesByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.Challenge, error)
		GetMissingChallengesByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Challenge, error)
		Create(ctx context.Context, params ChallengeCreateParams) (*domain.Challenge, error)
		Update(ctx context.Context, ID uuid.UUID, params ChallengeUpdateParams) (*domain.Challenge, error)
		Delete(ctx context.Context, ID, actorID uuid.UUID, clientIP string) error
		RecalcAllDynamicPoints(ctx context.Context) error
	}
)

// =============================================================================
// Tag
// =============================================================================

type (
	// TagUseCase manages challenge classification tags.
	TagUseCase interface {
		Create(ctx context.Context, name, color string) (*domain.Tag, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Tag, error)
		GetAll(ctx context.Context) ([]*domain.Tag, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*domain.Tag, error)
		Update(ctx context.Context, ID uuid.UUID, name, color string) (*domain.Tag, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Topic
// =============================================================================

type (
	// TopicUseCase manages organizer-facing challenge topics.
	TopicUseCase interface {
		Create(ctx context.Context, name string) (*domain.Topic, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Topic, error)
		GetAll(ctx context.Context) ([]*domain.Topic, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Topic, error)
		Update(ctx context.Context, ID uuid.UUID, name string) (*domain.Topic, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		SetByChallengeID(ctx context.Context, challengeID uuid.UUID, topicIDs []uuid.UUID) error
	}
)

// =============================================================================
// Hint
// =============================================================================

type (
	// HintWithUnlockStatus is a hint paired with whether the requesting team has already unlocked it.
	HintWithUnlockStatus struct {
		Hint     *domain.Hint
		Unlocked bool
	}

	// HintUseCase manages challenge hints and team unlock operations.
	HintUseCase interface {
		Create(ctx context.Context, challengeID uuid.UUID, title, content string, cost, orderIndex int) (*domain.Hint, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Hint, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*HintWithUnlockStatus, error)
		Update(ctx context.Context, ID uuid.UUID, title, content string, cost, orderIndex int) (*domain.Hint, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		UnlockHint(ctx context.Context, userID, teamID, challengeID, hintID uuid.UUID) (*domain.Hint, error)
		GetAllUnlocks(ctx context.Context, page, perPage int) (*Paginated[*domain.UnlockWithDetails], error)
	}
)

// =============================================================================
// File
// =============================================================================

type (
	// FileUseCase handles file uploads, downloads, and access-controlled URL generation for challenges and pages.
	FileUseCase interface {
		Upload(ctx context.Context, challengeID uuid.UUID, fileType domain.FileType, filename string, reader io.Reader, size int64, contentType string) (*domain.File, error)
		UploadPageFile(ctx context.Context, pageID uuid.UUID, filename string, reader io.Reader, size int64, contentType string) (*domain.File, error)
		Download(ctx context.Context, path string) (io.ReadCloser, error)
		GetDownloadURLWithAccess(ctx context.Context, fileID uuid.UUID, teamID *uuid.UUID, isAdmin bool) (string, error)
		BuildDownloadURLs(ctx context.Context, files []*domain.File, teamID *uuid.UUID, isAdmin bool) (map[string]string, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType domain.FileType) ([]*domain.File, error)
		GetByChallengeIDWithAccess(ctx context.Context, challengeID uuid.UUID, fileType domain.FileType, teamID *uuid.UUID, isAdmin bool) ([]*domain.File, error)
		GetByPageID(ctx context.Context, pageID uuid.UUID) ([]*domain.File, error)
		VerifyDownloadTokenAndGetFile(ctx context.Context, path, token string, teamID *uuid.UUID) (*domain.File, error)
		Delete(ctx context.Context, fileID uuid.UUID) error
	}

	// StorageAdminUseCase handles direct administrative object-storage operations.
	StorageAdminDeleteParams struct {
		Path     string
		ActorID  uuid.UUID
		ClientIP string
	}

	StorageAdminListParams struct {
		Prefix string
		Cursor string
		Limit  int
	}

	StorageAdminListResult struct {
		Paths      []string
		NextCursor string
	}

	StorageAdminUseCase interface {
		List(ctx context.Context, params StorageAdminListParams) (*StorageAdminListResult, error)
		Delete(ctx context.Context, params StorageAdminDeleteParams) error
	}
)

// =============================================================================
// Award
// =============================================================================

type (
	// AwardUseCase manages bonus point awards granted to teams by admins.
	AwardUseCase interface {
		Create(ctx context.Context, teamID uuid.UUID, value int, description string, createdBy uuid.UUID) (*domain.Award, error)
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Award, error)
		GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.Award, error)
		GetAll(ctx context.Context) ([]*domain.Award, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Comment
// =============================================================================

type (
	// CommentUseCase manages participant comments on challenges.
	CommentUseCase interface {
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*domain.Comment, error)
		Create(ctx context.Context, userID, challengeID uuid.UUID, content string) (*domain.Comment, error)
		Delete(ctx context.Context, ID, userID uuid.UUID, isAdmin bool) error
	}
)

// =============================================================================
// Rating
// =============================================================================

type (
	// RatingUseCase manages participant difficulty ratings and reviews for challenges.
	RatingUseCase interface {
		PutRating(ctx context.Context, challengeID, userID, teamID uuid.UUID, value int, review string) (*domain.Rating, error)
		GetRatingsByChallengeID(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*domain.Rating, error)
	}
)
