package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// =============================================================================
// Challenge
// =============================================================================

type (
	// ChallengeRepository provides persistence operations for challenges, flags, requirements, and solutions.
	ChallengeRepository interface {
		Create(ctx context.Context, c *domain.Challenge) error
		Update(ctx context.Context, c *domain.Challenge) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Challenge, error)
		GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Challenge, error)
		GetByIDForUpdate(ctx context.Context, ID uuid.UUID) (*domain.Challenge, error)
		GetAll(ctx context.Context, teamID, tagID *uuid.UUID) ([]*domain.ChallengeWithSolved, error)
		GetAllForBackup(ctx context.Context) ([]*domain.ChallengeWithSolved, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		IncrementSolveCount(ctx context.Context, ID uuid.UUID) (int, error)
		DecrementSolveCount(ctx context.Context, ID uuid.UUID) (int, error)
		BatchDecrementSolveCount(ctx context.Context, ids []uuid.UUID) error
		BatchIncrementSolveCount(ctx context.Context, ids []uuid.UUID) error
		BatchUpdatePoints(ctx context.Context, ids []uuid.UUID, points []int) error
		RecalculateSolveCounts(ctx context.Context, ids []uuid.UUID) error
		UpdatePoints(ctx context.Context, ID uuid.UUID, points int) error
		SetTags(ctx context.Context, challengeID uuid.UUID, tagIDs []uuid.UUID) error
		SetRequirements(ctx context.Context, challengeID uuid.UUID, requirementIDs []uuid.UUID) error
		GetFlags(ctx context.Context, ID uuid.UUID) (*domain.ChallengeFlags, error)
		GetRequirements(ctx context.Context, ID uuid.UUID) ([]*domain.ChallengeRequirement, error)
		GetRequirementsForEnforcement(ctx context.Context, ID uuid.UUID) ([]*domain.ChallengeRequirement, error)
		GetAllRequirementPairs(ctx context.Context) ([]*domain.ChallengeRequirementPair, error)
		AcquireRequirementsLock(ctx context.Context) error
		GetSolution(ctx context.Context, ID uuid.UUID) (*domain.ChallengeSolution, error)
		GetAllSolutions(ctx context.Context) ([]*domain.SolutionBackup, error)
		ListSolutions(ctx context.Context) ([]*domain.ChallengeSolutionEntry, error)
		UpsertSolution(ctx context.Context, challengeID uuid.UUID, content, state string) (*domain.ChallengeSolution, error)
		DeleteSolution(ctx context.Context, challengeID uuid.UUID) error
		GetMissingChallengesByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.Challenge, error)
		GetMissingChallengesByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Challenge, error)
		GetAllDynamicIDs(ctx context.Context) ([]uuid.UUID, error)
	}
)

// =============================================================================
// Tag
// =============================================================================

type (
	// TagRepository provides persistence operations for challenge tags.
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
// Topic
// =============================================================================

type (
	// TopicRepository provides persistence operations for organizer challenge topics.
	TopicRepository interface {
		Create(ctx context.Context, topic *domain.Topic) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Topic, error)
		GetByName(ctx context.Context, name string) (*domain.Topic, error)
		GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Topic, error)
		GetAll(ctx context.Context) ([]*domain.Topic, error)
		Update(ctx context.Context, topic *domain.Topic) error
		Delete(ctx context.Context, ID uuid.UUID) error
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Topic, error)
		GetByChallengeIDs(ctx context.Context, challengeIDs []uuid.UUID) (map[uuid.UUID][]*domain.Topic, error)
		SetByChallengeID(ctx context.Context, challengeID uuid.UUID, topicIDs []uuid.UUID) error
	}
)

// =============================================================================
// Hint
// =============================================================================

type (
	// HintRepository provides persistence operations for hints and team hint unlocks.
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
		AcquireAdvisoryLock(ctx context.Context, lockKey int64) error
	}
)

// =============================================================================
// File
// =============================================================================

type (
	// FileRepository provides persistence operations for challenge and page file metadata.
	FileRepository interface {
		Create(ctx context.Context, file *domain.File) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.File, error)
		GetByLocation(ctx context.Context, location string) (*domain.File, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType domain.FileType) ([]*domain.File, error)
		GetAllByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.File, error)
		GetByChallengeIDs(ctx context.Context, challengeIDs []uuid.UUID) (map[uuid.UUID][]*domain.File, error)
		GetByPageID(ctx context.Context, pageID uuid.UUID) ([]*domain.File, error)
		GetAll(ctx context.Context) ([]*domain.File, error)
		ListLocations(ctx context.Context, limit, offset int) ([]string, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}
)

// =============================================================================
// Comment
// =============================================================================

type (
	// CommentRepository provides persistence operations for challenge comments.
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
	// RatingRepository provides persistence operations for challenge difficulty ratings.
	RatingRepository interface {
		Upsert(ctx context.Context, rating *domain.Rating) error
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*domain.Rating, error)
		GetByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) (*domain.Rating, error)
		GetAll(ctx context.Context) ([]*domain.Rating, error)
		DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error
		SoftBanByTeamID(ctx context.Context, teamID uuid.UUID) error
		RestoreByBannedTeamID(ctx context.Context, teamID uuid.UUID) error
	}
)
