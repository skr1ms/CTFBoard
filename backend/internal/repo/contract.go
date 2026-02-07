package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
)

type (
	Transaction interface {
		Commit(ctx context.Context) error
		Rollback(ctx context.Context) error
	}

	UserRepository interface {
		Create(ctx context.Context, user *entity.User) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.User, error)
		GetByEmail(ctx context.Context, email string) (*entity.User, error)
		GetByUsername(ctx context.Context, username string) (*entity.User, error)
		GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.User, error)
		GetAll(ctx context.Context) ([]*entity.User, error)
		UpdateTeamID(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID) error
		SetVerified(ctx context.Context, userID uuid.UUID) error
		UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	}

	ChallengeRepository interface {
		Create(ctx context.Context, challenge *entity.Challenge) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Challenge, error)
		GetAll(ctx context.Context, teamID, tagID *uuid.UUID) ([]*ChallengeWithSolved, error)
		Update(ctx context.Context, challenge *entity.Challenge) error
		Delete(ctx context.Context, ID uuid.UUID) error
		IncrementSolveCount(ctx context.Context, ID uuid.UUID) (int, error)
		UpdatePoints(ctx context.Context, ID uuid.UUID, points int) error
	}

	TagRepository interface {
		Create(ctx context.Context, tag *entity.Tag) error
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Tag, error)
		GetByName(ctx context.Context, name string) (*entity.Tag, error)
		GetAll(ctx context.Context) ([]*entity.Tag, error)
		Update(ctx context.Context, tag *entity.Tag) error
		Delete(ctx context.Context, id uuid.UUID) error
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Tag, error)
		GetByChallengeIDs(ctx context.Context, challengeIDs []uuid.UUID) (map[uuid.UUID][]*entity.Tag, error)
		SetChallengeTags(ctx context.Context, challengeID uuid.UUID, tagIDs []uuid.UUID) error
	}

	ChallengeWithSolved struct {
		Challenge *entity.Challenge
		Solved    bool
	}

	TeamRepository interface {
		Create(ctx context.Context, team *entity.Team) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Team, error)
		GetByInviteToken(ctx context.Context, inviteToken uuid.UUID) (*entity.Team, error)
		GetByName(ctx context.Context, name string) (*entity.Team, error)
		GetSoloTeamByUserID(ctx context.Context, userID uuid.UUID) (*entity.Team, error)
		GetAll(ctx context.Context) ([]*entity.Team, error)
		CountTeamMembers(ctx context.Context, teamID uuid.UUID) (int, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		HardDeleteTeams(ctx context.Context, cutoffDate time.Time) error
		Ban(ctx context.Context, teamID uuid.UUID, reason string) error
		Unban(ctx context.Context, teamID uuid.UUID) error
		SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error
		SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error
	}

	SolveRepository interface {
		Create(ctx context.Context, solve *entity.Solve) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Solve, error)
		GetByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) (*entity.Solve, error)
		GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Solve, error)
		GetAll(ctx context.Context) ([]*entity.Solve, error)
		GetScoreboard(ctx context.Context) ([]*ScoreboardEntry, error)
		GetScoreboardFrozen(ctx context.Context, freezeTime time.Time) ([]*ScoreboardEntry, error)
		GetScoreboardByBracket(ctx context.Context, bracketID *uuid.UUID) ([]*ScoreboardEntry, error)
		GetScoreboardByBracketFrozen(ctx context.Context, freezeTime time.Time, bracketID *uuid.UUID) ([]*ScoreboardEntry, error)
		GetFirstBlood(ctx context.Context, challengeID uuid.UUID) (*FirstBloodEntry, error)
		GetTeamScore(ctx context.Context, teamID uuid.UUID) (int, error)
	}

	CompetitionRepository interface {
		Get(ctx context.Context) (*entity.Competition, error)
		Update(ctx context.Context, competition *entity.Competition) error
	}

	AppSettingsRepository interface {
		Get(ctx context.Context) (*entity.AppSettings, error)
		Update(ctx context.Context, s *entity.AppSettings) error
	}

	ConfigRepository interface {
		GetAll(ctx context.Context) ([]*entity.Config, error)
		GetByKey(ctx context.Context, key string) (*entity.Config, error)
		Upsert(ctx context.Context, cfg *entity.Config) error
		Delete(ctx context.Context, key string) error
	}

	HintRepository interface {
		Create(ctx context.Context, hint *entity.Hint) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Hint, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Hint, error)
		Update(ctx context.Context, hint *entity.Hint) error
		Delete(ctx context.Context, ID uuid.UUID) error
	}

	HintUnlockRepository interface {
		GetByTeamAndHint(ctx context.Context, teamID, hintID uuid.UUID) (*entity.HintUnlock, error)
		GetUnlockedHintIDs(ctx context.Context, teamID, challengeID uuid.UUID) ([]uuid.UUID, error)
	}

	AwardRepository interface {
		Create(ctx context.Context, award *entity.Award) error
		GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Award, error)
		GetAll(ctx context.Context) ([]*entity.Award, error)
		GetTeamTotalAwards(ctx context.Context, teamID uuid.UUID) (int, error)
	}

	FileRepository interface {
		Create(ctx context.Context, file *entity.File) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.File, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType entity.FileType) ([]*entity.File, error)
		GetAll(ctx context.Context) ([]*entity.File, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}

	TxRepository interface {
		BeginTx(ctx context.Context) (Transaction, error)
		BeginSerializableTx(ctx context.Context) (Transaction, error)
		RunTransaction(ctx context.Context, fn func(context.Context, Transaction) error) error

		GetChallengeByIDTx(ctx context.Context, tx Transaction, ID uuid.UUID) (*entity.Challenge, error)
		DeleteChallengeTx(ctx context.Context, tx Transaction, challengeID uuid.UUID) error
		IncrementChallengeSolveCountTx(ctx context.Context, tx Transaction, ID uuid.UUID) (int, error)
		UpdateChallengePointsTx(ctx context.Context, tx Transaction, ID uuid.UUID, points int) error

		CreateUserTx(ctx context.Context, tx Transaction, user *entity.User) error
		UpdateUserTeamIDTx(ctx context.Context, tx Transaction, userID uuid.UUID, teamID *uuid.UUID) error

		CreateTeamTx(ctx context.Context, tx Transaction, team *entity.Team) error
		GetTeamByIDTx(ctx context.Context, tx Transaction, ID uuid.UUID) (*entity.Team, error)
		GetSoloTeamByUserIDTx(ctx context.Context, tx Transaction, userID uuid.UUID) (*entity.Team, error)

		CreateSolveTx(ctx context.Context, tx Transaction, solve *entity.Solve) error
		GetSolveByTeamAndChallengeTx(ctx context.Context, tx Transaction, teamID, challengeID uuid.UUID) (*entity.Solve, error)
		GetTeamScoreTx(ctx context.Context, tx Transaction, teamID uuid.UUID) (int, error)

		CreateHintUnlockTx(ctx context.Context, tx Transaction, teamID, hintID uuid.UUID) error
		GetHintUnlockByTeamAndHintTx(ctx context.Context, tx Transaction, teamID, hintID uuid.UUID) (*entity.HintUnlock, error)

		CreateAwardTx(ctx context.Context, tx Transaction, award *entity.Award) error

		LockTeamTx(ctx context.Context, tx Transaction, teamID uuid.UUID) error
		LockUserTx(ctx context.Context, tx Transaction, userID uuid.UUID) error

		DeleteSolvesByTeamIDTx(ctx context.Context, tx Transaction, teamID uuid.UUID) error

		GetTeamByNameTx(ctx context.Context, tx Transaction, name string) (*entity.Team, error)
		GetTeamByInviteTokenTx(ctx context.Context, tx Transaction, inviteToken uuid.UUID) (*entity.Team, error)
		GetUsersByTeamIDTx(ctx context.Context, tx Transaction, teamID uuid.UUID) ([]*entity.User, error)
		DeleteTeamTx(ctx context.Context, tx Transaction, teamID uuid.UUID) error
		SoftDeleteTeamTx(ctx context.Context, tx Transaction, teamID uuid.UUID) error
		UpdateTeamCaptainTx(ctx context.Context, tx Transaction, teamID, newCaptainID uuid.UUID) error
		CreateTeamAuditLogTx(ctx context.Context, tx Transaction, log *entity.TeamAuditLog) error
		CreateAuditLogTx(ctx context.Context, tx Transaction, log *entity.AuditLog) error
	}

	VerificationTokenRepository interface {
		Create(ctx context.Context, token *entity.VerificationToken) error
		GetByToken(ctx context.Context, token string) (*entity.VerificationToken, error)
		MarkUsed(ctx context.Context, ID uuid.UUID) error
		DeleteExpired(ctx context.Context) error
		DeleteByUserAndType(ctx context.Context, userID uuid.UUID, tokenType entity.TokenType) error
	}

	AuditLogRepository interface {
		Create(ctx context.Context, log *entity.AuditLog) error
	}

	ScoreboardEntry struct {
		TeamID   uuid.UUID
		TeamName string
		Points   int
		SolvedAt time.Time
	}

	FirstBloodEntry struct {
		UserID   uuid.UUID
		Username string
		TeamID   uuid.UUID
		TeamName string
		SolvedAt time.Time
	}
	StatisticsRepository interface {
		GetGeneralStats(ctx context.Context) (*entity.GeneralStats, error)
		GetChallengeStats(ctx context.Context) ([]*entity.ChallengeStats, error)
		GetChallengeDetailStats(ctx context.Context, challengeID uuid.UUID) (*entity.ChallengeDetailStats, error)
		GetScoreboardHistory(ctx context.Context, limit int) ([]*entity.ScoreboardHistoryEntry, error)
	}

	SubmissionRepository interface {
		Create(ctx context.Context, sub *entity.Submission) error
		GetByChallenge(ctx context.Context, challengeID uuid.UUID, limit, offset int) ([]*entity.SubmissionWithDetails, error)
		GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.SubmissionWithDetails, error)
		GetByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*entity.SubmissionWithDetails, error)
		GetAll(ctx context.Context, limit, offset int) ([]*entity.SubmissionWithDetails, error)
		CountByChallenge(ctx context.Context, challengeID uuid.UUID) (int64, error)
		CountByUser(ctx context.Context, userID uuid.UUID) (int64, error)
		CountByTeam(ctx context.Context, teamID uuid.UUID) (int64, error)
		CountAll(ctx context.Context) (int64, error)
		CountFailedByIP(ctx context.Context, ip string, since time.Time) (int64, error)
		GetStats(ctx context.Context, challengeID uuid.UUID) (*entity.SubmissionStats, error)
	}

	NotificationRepository interface {
		Create(ctx context.Context, notif *entity.Notification) error
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Notification, error)
		GetAll(ctx context.Context, limit, offset int) ([]*entity.Notification, error)
		Update(ctx context.Context, notif *entity.Notification) error
		Delete(ctx context.Context, id uuid.UUID) error

		CreateUserNotification(ctx context.Context, userNotif *entity.UserNotification) error
		GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.UserNotification, error)
		MarkAsRead(ctx context.Context, id, userID uuid.UUID) error
		CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
		DeleteUserNotification(ctx context.Context, id, userID uuid.UUID) error
	}

	PageRepository interface {
		Create(ctx context.Context, page *entity.Page) error
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Page, error)
		GetBySlug(ctx context.Context, slug string) (*entity.Page, error)
		GetPublishedList(ctx context.Context) ([]*entity.PageListItem, error)
		GetAllList(ctx context.Context) ([]*entity.Page, error)
		Update(ctx context.Context, page *entity.Page) error
		Delete(ctx context.Context, id uuid.UUID) error
	}

	BracketRepository interface {
		Create(ctx context.Context, bracket *entity.Bracket) error
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Bracket, error)
		GetByName(ctx context.Context, name string) (*entity.Bracket, error)
		GetAll(ctx context.Context) ([]*entity.Bracket, error)
		Update(ctx context.Context, bracket *entity.Bracket) error
		Delete(ctx context.Context, id uuid.UUID) error
	}

	FieldRepository interface {
		Create(ctx context.Context, field *entity.Field) error
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Field, error)
		GetByEntityType(ctx context.Context, entityType entity.EntityType) ([]*entity.Field, error)
		GetAll(ctx context.Context) ([]*entity.Field, error)
		Update(ctx context.Context, field *entity.Field) error
		Delete(ctx context.Context, id uuid.UUID) error
	}

	FieldValueRepository interface {
		GetByEntityID(ctx context.Context, entityID uuid.UUID) ([]*entity.FieldValue, error)
		SetValues(ctx context.Context, entityID uuid.UUID, values map[string]string) error
	}

	APITokenRepository interface {
		Create(ctx context.Context, token *entity.APIToken) error
		GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.APIToken, error)
		GetByTokenHash(ctx context.Context, tokenHash string) (*entity.APIToken, error)
		Delete(ctx context.Context, id, userID uuid.UUID) error
		UpdateLastUsedAt(ctx context.Context, id uuid.UUID, at time.Time) error
	}

	CommentRepository interface {
		Create(ctx context.Context, comment *entity.Comment) error
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Comment, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID) ([]*entity.Comment, error)
		Update(ctx context.Context, comment *entity.Comment) error
		Delete(ctx context.Context, id uuid.UUID) error
	}

	RatingRepository interface {
		CreateCTFEvent(ctx context.Context, event *entity.CTFEvent) error
		GetCTFEventByID(ctx context.Context, id uuid.UUID) (*entity.CTFEvent, error)
		GetAllCTFEvents(ctx context.Context) ([]*entity.CTFEvent, error)
		CreateTeamRating(ctx context.Context, r *entity.TeamRating) error
		GetTeamRatingsByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.TeamRating, error)
		UpsertGlobalRating(ctx context.Context, r *entity.GlobalRating) error
		GetGlobalRatings(ctx context.Context, limit, offset int) ([]*entity.GlobalRating, error)
		CountGlobalRatings(ctx context.Context) (int64, error)
		GetGlobalRatingByTeamID(ctx context.Context, teamID uuid.UUID) (*entity.GlobalRating, error)
	}

	BackupRepository interface {
		EraseAllTablesTx(ctx context.Context, tx Transaction) error
		ImportCompetitionTx(ctx context.Context, tx Transaction, comp *entity.Competition) error
		ImportChallengesTx(ctx context.Context, tx Transaction, data *entity.BackupData) error
		ImportTeamsTx(ctx context.Context, tx Transaction, data *entity.BackupData, opts entity.ImportOptions) error
		ImportUsersTx(ctx context.Context, tx Transaction, data *entity.BackupData, opts entity.ImportOptions) error
		ImportAwardsTx(ctx context.Context, tx Transaction, data *entity.BackupData) error
		ImportSolvesTx(ctx context.Context, tx Transaction, data *entity.BackupData) error
		ImportFileMetadataTx(ctx context.Context, tx Transaction, data *entity.BackupData) error
	}
)
