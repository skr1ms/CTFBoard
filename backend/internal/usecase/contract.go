package usecase

import (
	"context"
	"io"

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
	}

	ChallengeUseCase interface {
		GetAll(ctx context.Context, teamID *uuid.UUID) ([]*repo.ChallengeWithSolved, error)
		Create(ctx context.Context, title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden, isRegex, isCaseInsensitive bool) (*entity.Challenge, error)
		Update(ctx context.Context, ID uuid.UUID, title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden, isRegex, isCaseInsensitive bool) (*entity.Challenge, error)
		Delete(ctx context.Context, ID, actorID uuid.UUID, clientIP string) error
		SubmitFlag(ctx context.Context, challengeID uuid.UUID, flag string, userID uuid.UUID, teamID *uuid.UUID) (bool, error)
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
		GetScoreboard(ctx context.Context) ([]*repo.ScoreboardEntry, error)
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

	BackupUseCase interface {
		Export(ctx context.Context, opts entity.ExportOptions) (*entity.BackupData, error)
		ExportZIP(ctx context.Context, opts entity.ExportOptions) (io.ReadCloser, error)
		ImportZIP(ctx context.Context, r io.ReaderAt, size int64, opts entity.ImportOptions) (*entity.ImportResult, error)
	}
)
