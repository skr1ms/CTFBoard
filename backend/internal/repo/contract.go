package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
)

type (
	UserRepository interface {
		Create(ctx context.Context, user *entity.User) error
		GetByID(ctx context.Context, id string) (*entity.User, error)
		GetByEmail(ctx context.Context, email string) (*entity.User, error)
		GetByUsername(ctx context.Context, username string) (*entity.User, error)
		GetByTeamId(ctx context.Context, teamId string) ([]*entity.User, error)
		UpdateTeamId(ctx context.Context, userId string, teamId *string) error
		SetVerified(ctx context.Context, userId string) error
		UpdatePassword(ctx context.Context, userId, passwordHash string) error
	}

	ChallengeRepository interface {
		Create(ctx context.Context, challenge *entity.Challenge) error
		GetByID(ctx context.Context, id string) (*entity.Challenge, error)
		GetAll(ctx context.Context, teamId *string) ([]*ChallengeWithSolved, error)
		Update(ctx context.Context, challenge *entity.Challenge) error
		Delete(ctx context.Context, id string) error
		IncrementSolveCount(ctx context.Context, id string) (int, error)
		UpdatePoints(ctx context.Context, id string, points int) error
	}

	ChallengeWithSolved struct {
		Challenge *entity.Challenge
		Solved    bool
	}

	TeamRepository interface {
		Create(ctx context.Context, team *entity.Team) error
		GetByID(ctx context.Context, id string) (*entity.Team, error)
		GetByInviteToken(ctx context.Context, inviteToken string) (*entity.Team, error)
		GetByName(ctx context.Context, name string) (*entity.Team, error)
		Delete(ctx context.Context, id string) error
	}

	SolveRepository interface {
		Create(ctx context.Context, solve *entity.Solve) error
		GetByID(ctx context.Context, id string) (*entity.Solve, error)
		GetByTeamAndChallenge(ctx context.Context, teamId, challengeId string) (*entity.Solve, error)
		GetByUserId(ctx context.Context, userId string) ([]*entity.Solve, error)
		GetScoreboard(ctx context.Context) ([]*ScoreboardEntry, error)
		GetScoreboardFrozen(ctx context.Context, freezeTime time.Time) ([]*ScoreboardEntry, error)
		GetFirstBlood(ctx context.Context, challengeId string) (*FirstBloodEntry, error)
		GetTeamScore(ctx context.Context, teamId string) (int, error)
	}

	CompetitionRepository interface {
		Get(ctx context.Context) (*entity.Competition, error)
		Update(ctx context.Context, competition *entity.Competition) error
	}

	HintRepository interface {
		Create(ctx context.Context, hint *entity.Hint) error
		GetByID(ctx context.Context, id string) (*entity.Hint, error)
		GetByChallengeID(ctx context.Context, challengeId string) ([]*entity.Hint, error)
		Update(ctx context.Context, hint *entity.Hint) error
		Delete(ctx context.Context, id string) error
	}

	HintUnlockRepository interface {
		GetByTeamAndHint(ctx context.Context, teamId, hintId string) (*entity.HintUnlock, error)
		GetUnlockedHintIDs(ctx context.Context, teamId, challengeId string) ([]string, error)
	}

	AwardRepository interface {
		GetTeamTotalAwards(ctx context.Context, teamId string) (int, error)
	}

	TxRepository interface {
		BeginTx(ctx context.Context) (*sql.Tx, error)
		RunTransaction(ctx context.Context, fn func(context.Context, *sql.Tx) error) error

		// Challenge Tx Methods
		GetChallengeByIDTx(ctx context.Context, tx *sql.Tx, id string) (*entity.Challenge, error)
		IncrementChallengeSolveCountTx(ctx context.Context, tx *sql.Tx, id string) (int, error)
		UpdateChallengePointsTx(ctx context.Context, tx *sql.Tx, id string, points int) error

		// User Tx Methods
		CreateUserTx(ctx context.Context, tx *sql.Tx, user *entity.User) error
		UpdateUserTeamIDTx(ctx context.Context, tx *sql.Tx, userId string, teamId *string) error

		// Team Tx Methods
		CreateTeamTx(ctx context.Context, tx *sql.Tx, team *entity.Team) error

		// Solve Tx Methods
		CreateSolveTx(ctx context.Context, tx *sql.Tx, solve *entity.Solve) error
		GetSolveByTeamAndChallengeTx(ctx context.Context, tx *sql.Tx, teamId, challengeId string) (*entity.Solve, error)
		GetTeamScoreTx(ctx context.Context, tx *sql.Tx, teamId string) (int, error)

		// HintUnlock Tx Methods
		CreateHintUnlockTx(ctx context.Context, tx *sql.Tx, teamId, hintId string) error
		GetHintUnlockByTeamAndHintTx(ctx context.Context, tx *sql.Tx, teamId, hintId string) (*entity.HintUnlock, error)

		// Award Tx Methods
		CreateAwardTx(ctx context.Context, tx *sql.Tx, award *entity.Award) error

		// Utility Tx Methods
		LockTeamTx(ctx context.Context, tx *sql.Tx, teamId string) error
	}

	VerificationTokenRepository interface {
		Create(ctx context.Context, token *entity.VerificationToken) error
		GetByToken(ctx context.Context, token string) (*entity.VerificationToken, error)
		MarkUsed(ctx context.Context, id string) error
		DeleteExpired(ctx context.Context) error
		DeleteByUserAndType(ctx context.Context, userId string, tokenType entity.TokenType) error
	}

	ScoreboardEntry struct {
		TeamId   string
		TeamName string
		Points   int
		SolvedAt time.Time
	}

	FirstBloodEntry struct {
		UserId   string
		Username string
		TeamId   string
		TeamName string
		SolvedAt time.Time
	}
)
