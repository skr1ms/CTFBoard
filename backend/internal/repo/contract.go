package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/skr1ms/CTFBoard/internal/entity"
)

type (
	PgxTx interface {
		Begin(ctx context.Context) (pgx.Tx, error)
		Commit(ctx context.Context) error
		Rollback(ctx context.Context) error
		Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
		Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
		QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
		Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error)
		CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
		SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
		LargeObjects() pgx.LargeObjects
		Conn() *pgx.Conn
	}

	UserRepository interface {
		Create(ctx context.Context, user *entity.User) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.User, error)
		GetByEmail(ctx context.Context, email string) (*entity.User, error)
		GetByUsername(ctx context.Context, username string) (*entity.User, error)
		GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.User, error)
		UpdateTeamID(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID) error
		SetVerified(ctx context.Context, userID uuid.UUID) error
		UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	}

	ChallengeRepository interface {
		Create(ctx context.Context, challenge *entity.Challenge) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Challenge, error)
		GetAll(ctx context.Context, teamID *uuid.UUID) ([]*ChallengeWithSolved, error)
		Update(ctx context.Context, challenge *entity.Challenge) error
		Delete(ctx context.Context, ID uuid.UUID) error
		IncrementSolveCount(ctx context.Context, ID uuid.UUID) (int, error)
		UpdatePoints(ctx context.Context, ID uuid.UUID, points int) error
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
		CountTeamMembers(ctx context.Context, teamID uuid.UUID) (int, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		HardDeleteTeams(ctx context.Context, cutoffDate time.Time) error
	}

	SolveRepository interface {
		Create(ctx context.Context, solve *entity.Solve) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.Solve, error)
		GetByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) (*entity.Solve, error)
		GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Solve, error)
		GetScoreboard(ctx context.Context) ([]*ScoreboardEntry, error)
		GetScoreboardFrozen(ctx context.Context, freezeTime time.Time) ([]*ScoreboardEntry, error)
		GetFirstBlood(ctx context.Context, challengeID uuid.UUID) (*FirstBloodEntry, error)
		GetTeamScore(ctx context.Context, teamID uuid.UUID) (int, error)
	}

	CompetitionRepository interface {
		Get(ctx context.Context) (*entity.Competition, error)
		Update(ctx context.Context, competition *entity.Competition) error
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
		GetTeamTotalAwards(ctx context.Context, teamID uuid.UUID) (int, error)
	}

	FileRepository interface {
		Create(ctx context.Context, file *entity.File) error
		GetByID(ctx context.Context, ID uuid.UUID) (*entity.File, error)
		GetByChallengeID(ctx context.Context, challengeID uuid.UUID, fileType entity.FileType) ([]*entity.File, error)
		Delete(ctx context.Context, ID uuid.UUID) error
	}

	TxRepository interface {
		BeginTx(ctx context.Context) (pgx.Tx, error)
		BeginSerializableTx(ctx context.Context) (pgx.Tx, error)
		RunTransaction(ctx context.Context, fn func(context.Context, pgx.Tx) error) error

		GetChallengeByIDTx(ctx context.Context, tx pgx.Tx, ID uuid.UUID) (*entity.Challenge, error)
		IncrementChallengeSolveCountTx(ctx context.Context, tx pgx.Tx, ID uuid.UUID) (int, error)
		UpdateChallengePointsTx(ctx context.Context, tx pgx.Tx, ID uuid.UUID, points int) error

		CreateUserTx(ctx context.Context, tx pgx.Tx, user *entity.User) error
		UpdateUserTeamIDTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, teamID *uuid.UUID) error

		CreateTeamTx(ctx context.Context, tx pgx.Tx, team *entity.Team) error
		GetTeamByIDTx(ctx context.Context, tx pgx.Tx, ID uuid.UUID) (*entity.Team, error)
		GetSoloTeamByUserIDTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*entity.Team, error)

		CreateSolveTx(ctx context.Context, tx pgx.Tx, solve *entity.Solve) error
		GetSolveByTeamAndChallengeTx(ctx context.Context, tx pgx.Tx, teamID, challengeID uuid.UUID) (*entity.Solve, error)
		GetTeamScoreTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) (int, error)

		CreateHintUnlockTx(ctx context.Context, tx pgx.Tx, teamID, hintID uuid.UUID) error
		GetHintUnlockByTeamAndHintTx(ctx context.Context, tx pgx.Tx, teamID, hintID uuid.UUID) (*entity.HintUnlock, error)

		CreateAwardTx(ctx context.Context, tx pgx.Tx, award *entity.Award) error

		LockTeamTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) error
		LockUserTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error

		DeleteSolvesByTeamIDTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) error

		GetTeamByNameTx(ctx context.Context, tx pgx.Tx, name string) (*entity.Team, error)
		GetTeamByInviteTokenTx(ctx context.Context, tx pgx.Tx, inviteToken uuid.UUID) (*entity.Team, error)
		GetUsersByTeamIDTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) ([]*entity.User, error)
		DeleteTeamTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) error
		SoftDeleteTeamTx(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) error
		UpdateTeamCaptainTx(ctx context.Context, tx pgx.Tx, teamID, newCaptainID uuid.UUID) error
		CreateTeamAuditLogTx(ctx context.Context, tx pgx.Tx, log *entity.TeamAuditLog) error
		CreateAuditLogTx(ctx context.Context, tx pgx.Tx, log *entity.AuditLog) error
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
		GetScoreboardHistory(ctx context.Context, limit int) ([]*entity.ScoreboardHistoryEntry, error)
	}
)
