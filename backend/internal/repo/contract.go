package repo

import (
	"context"
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
	}

	ChallengeRepository interface {
		Create(ctx context.Context, challenge *entity.Challenge) error
		GetByID(ctx context.Context, id string) (*entity.Challenge, error)
		GetAll(ctx context.Context, teamId *string) ([]*ChallengeWithSolved, error)
		Update(ctx context.Context, challenge *entity.Challenge) error
		Delete(ctx context.Context, id string) error
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
	}

	SolveRepository interface {
		Create(ctx context.Context, solve *entity.Solve) error
		GetByID(ctx context.Context, id string) (*entity.Solve, error)
		GetByTeamAndChallenge(ctx context.Context, teamId, challengeId string) (*entity.Solve, error)
		GetByUserId(ctx context.Context, userId string) ([]*entity.Solve, error)
		GetScoreboard(ctx context.Context) ([]*ScoreboardEntry, error)
		GetScoreboardFrozen(ctx context.Context, freezeTime time.Time) ([]*ScoreboardEntry, error)
		GetFirstBlood(ctx context.Context, challengeId string) (*FirstBloodEntry, error)
	}

	CompetitionRepository interface {
		Get(ctx context.Context) (*entity.Competition, error)
		Update(ctx context.Context, competition *entity.Competition) error
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
