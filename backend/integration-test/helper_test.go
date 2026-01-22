package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/stretchr/testify/require"
)

type TestFixture struct {
	Pool                  *pgxpool.Pool
	UserRepo              *persistent.UserRepo
	TeamRepo              *persistent.TeamRepo
	ChallengeRepo         *persistent.ChallengeRepo
	SolveRepo             *persistent.SolveRepo
	HintRepo              *persistent.HintRepo
	HintUnlockRepo        *persistent.HintUnlockRepo
	AwardRepo             *persistent.AwardRepo
	TxRepo                *persistent.TxRepo
	CompetitionRepo       *persistent.CompetitionRepo
	VerificationTokenRepo *persistent.VerificationTokenRepo
	FileRepo              *persistent.FileRepository
}

func NewTestFixture(Pool *pgxpool.Pool) *TestFixture {
	return &TestFixture{
		Pool:                  Pool,
		UserRepo:              persistent.NewUserRepo(Pool),
		TeamRepo:              persistent.NewTeamRepo(Pool),
		ChallengeRepo:         persistent.NewChallengeRepo(Pool),
		SolveRepo:             persistent.NewSolveRepo(Pool),
		HintRepo:              persistent.NewHintRepo(Pool),
		HintUnlockRepo:        persistent.NewHintUnlockRepo(Pool),
		AwardRepo:             persistent.NewAwardRepo(Pool),
		TxRepo:                persistent.NewTxRepo(Pool),
		CompetitionRepo:       persistent.NewCompetitionRepo(Pool),
		VerificationTokenRepo: persistent.NewVerificationTokenRepo(Pool),
		FileRepo:              persistent.NewFileRepository(Pool),
	}
}

func (f *TestFixture) CreateUser(t *testing.T, suffix string) *entity.User {
	ctx := context.Background()
	user := &entity.User{
		Username:     "user_" + suffix,
		Email:        "user_" + suffix + "@example.com",
		PasswordHash: "hash123",
	}
	err := f.UserRepo.Create(ctx, user)
	require.NoError(t, err)

	gotUser, err := f.UserRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id
	return user
}

func (f *TestFixture) CreateTeam(t *testing.T, suffix string, captainId uuid.UUID) *entity.Team {
	ctx := context.Background()
	team := &entity.Team{
		Name:        "team_" + suffix,
		InviteToken: uuid.New(),
		CaptainId:   captainId,
	}
	err := f.TeamRepo.Create(ctx, team)
	require.NoError(t, err)

	gotTeam, err := f.TeamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id
	return team
}

func (f *TestFixture) CreateUserWithTeam(t *testing.T, suffix string) (*entity.User, *entity.Team) {
	user := f.CreateUser(t, suffix)
	team := f.CreateTeam(t, suffix, user.Id)
	return user, team
}

func (f *TestFixture) CreateChallenge(t *testing.T, suffix string, points int) *entity.Challenge {
	ctx := context.Background()
	challenge := &entity.Challenge{
		Title:        "Challenge " + suffix,
		Description:  "Description " + suffix,
		Category:     "Web",
		Points:       points,
		FlagHash:     "hash_" + suffix,
		IsHidden:     false,
		InitialValue: points,
		MinValue:     points,
		Decay:        0,
	}
	err := f.ChallengeRepo.Create(ctx, challenge)
	require.NoError(t, err)
	return challenge
}

func (f *TestFixture) CreateDynamicChallenge(t *testing.T, suffix string, initial, min, decay int) *entity.Challenge {
	ctx := context.Background()
	challenge := &entity.Challenge{
		Title:        "Dynamic " + suffix,
		Description:  "Description " + suffix,
		Category:     "Pwn",
		Points:       initial,
		FlagHash:     "hash_" + suffix,
		IsHidden:     false,
		InitialValue: initial,
		MinValue:     min,
		Decay:        decay,
	}
	err := f.ChallengeRepo.Create(ctx, challenge)
	require.NoError(t, err)
	return challenge
}

func (f *TestFixture) CreateHint(t *testing.T, challengeId uuid.UUID, cost int, order int) *entity.Hint {
	ctx := context.Background()
	hint := &entity.Hint{
		ChallengeId: challengeId,
		Content:     "Hint content",
		Cost:        cost,
		OrderIndex:  order,
	}
	err := f.HintRepo.Create(ctx, hint)
	require.NoError(t, err)
	return hint
}

func (f *TestFixture) CreateSolve(t *testing.T, userId, teamId, challengeId uuid.UUID) *entity.Solve {
	ctx := context.Background()
	solve := &entity.Solve{
		UserId:      userId,
		TeamId:      teamId,
		ChallengeId: challengeId,
	}
	err := f.SolveRepo.Create(ctx, solve)
	require.NoError(t, err)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, teamId, challengeId)
	require.NoError(t, err)
	solve.Id = gotSolve.Id
	solve.SolvedAt = gotSolve.SolvedAt
	return solve
}

func (f *TestFixture) CreateAwardTx(t *testing.T, tx pgx.Tx, teamId uuid.UUID, value int, desc string) *entity.Award {
	ctx := context.Background()
	award := &entity.Award{
		TeamId:      teamId,
		Value:       value,
		Description: desc,
	}
	err := f.TxRepo.CreateAwardTx(ctx, tx, award)
	require.NoError(t, err)
	return award
}

func (f *TestFixture) AddUserToTeam(t *testing.T, userID, teamID uuid.UUID) {
	ctx := context.Background()
	_, err := f.Pool.Exec(ctx, "UPDATE users SET team_id = $1 WHERE id = $2", teamID, userID)
	require.NoError(t, err)
}
