package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolveRepo_Create(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	challengeRepo := persistent.NewChallengeRepo(testDB.DB)
	solveRepo := persistent.NewSolveRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "solveuser",
		Email:        "solveuser@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "solveteam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	challenge := &entity.Challenge{
		Title:       "Solve Challenge",
		Description: "Description",
		Category:    "Web",
		Points:      100,
		FlagHash:    "hash123",
		IsHidden:    false,
	}
	err = challengeRepo.Create(ctx, challenge)
	require.NoError(t, err)

	solve := &entity.Solve{
		UserId:      user.Id,
		TeamId:      team.Id,
		ChallengeId: challenge.Id,
	}

	err = solveRepo.Create(ctx, solve)
	require.NoError(t, err)

	gotSolve, err := solveRepo.GetByTeamAndChallenge(ctx, solve.TeamId, solve.ChallengeId)
	require.NoError(t, err)
	assert.NotEmpty(t, gotSolve.Id)
	assert.False(t, gotSolve.SolvedAt.IsZero())
	solve.Id = gotSolve.Id
	solve.SolvedAt = gotSolve.SolvedAt
}

func TestSolveRepo_Create_Duplicate(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	challengeRepo := persistent.NewChallengeRepo(testDB.DB)
	solveRepo := persistent.NewSolveRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "duplicateuser",
		Email:        "duplicateuser@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "duplicateteam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	challenge := &entity.Challenge{
		Title:       "Duplicate Challenge",
		Description: "Description",
		Category:    "Web",
		Points:      100,
		FlagHash:    "hash123",
		IsHidden:    false,
	}
	err = challengeRepo.Create(ctx, challenge)
	require.NoError(t, err)

	solve1 := &entity.Solve{
		UserId:      user.Id,
		TeamId:      team.Id,
		ChallengeId: challenge.Id,
	}
	err = solveRepo.Create(ctx, solve1)
	require.NoError(t, err)

	solve2 := &entity.Solve{
		UserId:      user.Id,
		TeamId:      team.Id,
		ChallengeId: challenge.Id,
	}
	err = solveRepo.Create(ctx, solve2)
	assert.Error(t, err)
}

func TestSolveRepo_GetByID(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	challengeRepo := persistent.NewChallengeRepo(testDB.DB)
	solveRepo := persistent.NewSolveRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "getbyid",
		Email:        "getbyid@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "getbyidteam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	challenge := &entity.Challenge{
		Title:       "GetByID Challenge",
		Description: "Description",
		Category:    "Web",
		Points:      100,
		FlagHash:    "hash123",
		IsHidden:    false,
	}
	err = challengeRepo.Create(ctx, challenge)
	require.NoError(t, err)

	solve := &entity.Solve{
		UserId:      user.Id,
		TeamId:      team.Id,
		ChallengeId: challenge.Id,
	}
	err = solveRepo.Create(ctx, solve)
	require.NoError(t, err)

	gotSolveByTeam, err := solveRepo.GetByTeamAndChallenge(ctx, team.Id, challenge.Id)
	require.NoError(t, err)
	solve.Id = gotSolveByTeam.Id
	solve.SolvedAt = gotSolveByTeam.SolvedAt

	gotSolve, err := solveRepo.GetByID(ctx, solve.Id)
	require.NoError(t, err)
	assert.Equal(t, solve.Id, gotSolve.Id)
	assert.Equal(t, solve.UserId, gotSolve.UserId)
	assert.Equal(t, solve.TeamId, gotSolve.TeamId)
	assert.Equal(t, solve.ChallengeId, gotSolve.ChallengeId)
}

func TestSolveRepo_GetByID_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewSolveRepo(testDB.DB)
	ctx := context.Background()

	nonExistentID := uuid.New().String()
	_, err := repo.GetByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}

func TestSolveRepo_GetByTeamAndChallenge(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	challengeRepo := persistent.NewChallengeRepo(testDB.DB)
	solveRepo := persistent.NewSolveRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "getbyteam",
		Email:        "getbyteam@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "getbyteam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	challenge := &entity.Challenge{
		Title:       "GetByTeam Challenge",
		Description: "Description",
		Category:    "Web",
		Points:      100,
		FlagHash:    "hash123",
		IsHidden:    false,
	}
	err = challengeRepo.Create(ctx, challenge)
	require.NoError(t, err)

	solve := &entity.Solve{
		UserId:      user.Id,
		TeamId:      team.Id,
		ChallengeId: challenge.Id,
	}
	err = solveRepo.Create(ctx, solve)
	require.NoError(t, err)

	gotSolve, err := solveRepo.GetByTeamAndChallenge(ctx, team.Id, challenge.Id)
	require.NoError(t, err)
	solve.Id = gotSolve.Id
	assert.Equal(t, team.Id, gotSolve.TeamId)
	assert.Equal(t, challenge.Id, gotSolve.ChallengeId)
}

func TestSolveRepo_GetByTeamAndChallenge_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	challengeRepo := persistent.NewChallengeRepo(testDB.DB)
	solveRepo := persistent.NewSolveRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "notfound",
		Email:        "notfound@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "notfoundteam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	challenge := &entity.Challenge{
		Title:       "Not Found Challenge",
		Description: "Description",
		Category:    "Web",
		Points:      100,
		FlagHash:    "hash123",
		IsHidden:    false,
	}
	err = challengeRepo.Create(ctx, challenge)
	require.NoError(t, err)

	_, err = solveRepo.GetByTeamAndChallenge(ctx, team.Id, challenge.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}

func TestSolveRepo_GetByUserId(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	challengeRepo := persistent.NewChallengeRepo(testDB.DB)
	solveRepo := persistent.NewSolveRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "getbyuserid",
		Email:        "getbyuserid@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "getbyuseridteam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	challenge1 := &entity.Challenge{
		Title:       "Challenge 1",
		Description: "Description 1",
		Category:    "Web",
		Points:      100,
		FlagHash:    "hash1",
		IsHidden:    false,
	}
	err = challengeRepo.Create(ctx, challenge1)
	require.NoError(t, err)

	challenge2 := &entity.Challenge{
		Title:       "Challenge 2",
		Description: "Description 2",
		Category:    "Crypto",
		Points:      200,
		FlagHash:    "hash2",
		IsHidden:    false,
	}
	err = challengeRepo.Create(ctx, challenge2)
	require.NoError(t, err)

	solve1 := &entity.Solve{
		UserId:      user.Id,
		TeamId:      team.Id,
		ChallengeId: challenge1.Id,
	}
	err = solveRepo.Create(ctx, solve1)
	require.NoError(t, err)

	gotSolve1, err := solveRepo.GetByTeamAndChallenge(ctx, team.Id, challenge1.Id)
	require.NoError(t, err)
	solve1.Id = gotSolve1.Id
	solve1.SolvedAt = gotSolve1.SolvedAt

	time.Sleep(1 * time.Second)

	solve2 := &entity.Solve{
		UserId:      user.Id,
		TeamId:      team.Id,
		ChallengeId: challenge2.Id,
	}
	err = solveRepo.Create(ctx, solve2)
	require.NoError(t, err)

	gotSolve2, err := solveRepo.GetByTeamAndChallenge(ctx, team.Id, challenge2.Id)
	require.NoError(t, err)
	solve2.Id = gotSolve2.Id
	solve2.SolvedAt = gotSolve2.SolvedAt

	solves, err := solveRepo.GetByUserId(ctx, user.Id)
	require.NoError(t, err)
	assert.Len(t, solves, 2)
	assert.Equal(t, challenge2.Id, solves[0].ChallengeId)
	assert.Equal(t, challenge1.Id, solves[1].ChallengeId)
}

func TestSolveRepo_GetByUserId_Empty(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	solveRepo := persistent.NewSolveRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "emptyuser",
		Email:        "emptyuser@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	solves, err := solveRepo.GetByUserId(ctx, user.Id)
	require.NoError(t, err)
	assert.Len(t, solves, 0)
}

func TestSolveRepo_GetScoreboard(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	challengeRepo := persistent.NewChallengeRepo(testDB.DB)
	solveRepo := persistent.NewSolveRepo(testDB.DB)
	ctx := context.Background()

	user1 := &entity.User{
		Username:     "scoreuser1",
		Email:        "scoreuser1@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user1)
	require.NoError(t, err)
	gotUser1, err := userRepo.GetByEmail(ctx, user1.Email)
	require.NoError(t, err)
	user1.Id = gotUser1.Id

	user2 := &entity.User{
		Username:     "scoreuser2",
		Email:        "scoreuser2@example.com",
		PasswordHash: "hash456",
	}
	err = userRepo.Create(ctx, user2)
	require.NoError(t, err)
	gotUser2, err := userRepo.GetByEmail(ctx, user2.Email)
	require.NoError(t, err)
	user2.Id = gotUser2.Id

	team1 := &entity.Team{
		Name:        "ScoreTeam1",
		InviteToken: "token1",
		CaptainId:   user1.Id,
	}
	err = teamRepo.Create(ctx, team1)
	require.NoError(t, err)
	gotTeam1, err := teamRepo.GetByName(ctx, team1.Name)
	require.NoError(t, err)
	team1.Id = gotTeam1.Id

	team2 := &entity.Team{
		Name:        "ScoreTeam2",
		InviteToken: "token2",
		CaptainId:   user2.Id,
	}
	err = teamRepo.Create(ctx, team2)
	require.NoError(t, err)
	gotTeam2, err := teamRepo.GetByName(ctx, team2.Name)
	require.NoError(t, err)
	team2.Id = gotTeam2.Id

	challenge1 := &entity.Challenge{
		Title:       "Score Challenge 1",
		Description: "Description 1",
		Category:    "Web",
		Points:      100,
		FlagHash:    "hash1",
		IsHidden:    false,
	}
	err = challengeRepo.Create(ctx, challenge1)
	require.NoError(t, err)

	challenge2 := &entity.Challenge{
		Title:       "Score Challenge 2",
		Description: "Description 2",
		Category:    "Crypto",
		Points:      200,
		FlagHash:    "hash2",
		IsHidden:    false,
	}
	err = challengeRepo.Create(ctx, challenge2)
	require.NoError(t, err)

	solve1 := &entity.Solve{
		UserId:      user1.Id,
		TeamId:      team1.Id,
		ChallengeId: challenge1.Id,
	}
	err = solveRepo.Create(ctx, solve1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	solve2 := &entity.Solve{
		UserId:      user1.Id,
		TeamId:      team1.Id,
		ChallengeId: challenge2.Id,
	}
	err = solveRepo.Create(ctx, solve2)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	solve3 := &entity.Solve{
		UserId:      user2.Id,
		TeamId:      team2.Id,
		ChallengeId: challenge1.Id,
	}
	err = solveRepo.Create(ctx, solve3)
	require.NoError(t, err)

	scoreboard, err := solveRepo.GetScoreboard(ctx)
	require.NoError(t, err)
	assert.Len(t, scoreboard, 2)

	team1Found := false
	team2Found := false
	for _, entry := range scoreboard {
		if entry.TeamId == team1.Id {
			assert.Equal(t, "ScoreTeam1", entry.TeamName)
			assert.Equal(t, 300, entry.Points)
			team1Found = true
		}
		if entry.TeamId == team2.Id {
			assert.Equal(t, "ScoreTeam2", entry.TeamName)
			assert.Equal(t, 100, entry.Points)
			team2Found = true
		}
	}
	assert.True(t, team1Found)
	assert.True(t, team2Found)
}

func TestSolveRepo_GetScoreboard_Empty(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	solveRepo := persistent.NewSolveRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "emptyteam",
		Email:        "emptyteam@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "EmptyTeam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)

	scoreboard, err := solveRepo.GetScoreboard(ctx)
	require.NoError(t, err)
	assert.Len(t, scoreboard, 1)
	assert.Equal(t, "EmptyTeam", scoreboard[0].TeamName)
	assert.Equal(t, 0, scoreboard[0].Points)
}

func TestSolveRepo_GetFirstBlood(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	challengeRepo := persistent.NewChallengeRepo(testDB.DB)
	solveRepo := persistent.NewSolveRepo(testDB.DB)
	ctx := context.Background()

	user1 := &entity.User{
		Username:     "firstblood1",
		Email:        "firstblood1@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user1)
	require.NoError(t, err)
	gotUser1, err := userRepo.GetByEmail(ctx, user1.Email)
	require.NoError(t, err)
	user1.Id = gotUser1.Id

	user2 := &entity.User{
		Username:     "firstblood2",
		Email:        "firstblood2@example.com",
		PasswordHash: "hash456",
	}
	err = userRepo.Create(ctx, user2)
	require.NoError(t, err)
	gotUser2, err := userRepo.GetByEmail(ctx, user2.Email)
	require.NoError(t, err)
	user2.Id = gotUser2.Id

	team1 := &entity.Team{
		Name:        "FirstBloodTeam1",
		InviteToken: "token1",
		CaptainId:   user1.Id,
	}
	err = teamRepo.Create(ctx, team1)
	require.NoError(t, err)
	gotTeam1, err := teamRepo.GetByName(ctx, team1.Name)
	require.NoError(t, err)
	team1.Id = gotTeam1.Id

	team2 := &entity.Team{
		Name:        "FirstBloodTeam2",
		InviteToken: "token2",
		CaptainId:   user2.Id,
	}
	err = teamRepo.Create(ctx, team2)
	require.NoError(t, err)
	gotTeam2, err := teamRepo.GetByName(ctx, team2.Name)
	require.NoError(t, err)
	team2.Id = gotTeam2.Id

	challenge := &entity.Challenge{
		Title:       "First Blood Challenge",
		Description: "Test",
		Category:    "Web",
		Points:      100,
		FlagHash:    "hash123",
		IsHidden:    false,
	}
	err = challengeRepo.Create(ctx, challenge)
	require.NoError(t, err)

	solve1 := &entity.Solve{
		UserId:      user1.Id,
		TeamId:      team1.Id,
		ChallengeId: challenge.Id,
	}
	err = solveRepo.Create(ctx, solve1)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	solve2 := &entity.Solve{
		UserId:      user2.Id,
		TeamId:      team2.Id,
		ChallengeId: challenge.Id,
	}
	err = solveRepo.Create(ctx, solve2)
	require.NoError(t, err)

	firstBlood, err := solveRepo.GetFirstBlood(ctx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, user1.Id, firstBlood.UserId)
	assert.Equal(t, "firstblood1", firstBlood.Username)
	assert.Equal(t, team1.Id, firstBlood.TeamId)
	assert.Equal(t, "FirstBloodTeam1", firstBlood.TeamName)
}

func TestSolveRepo_GetFirstBlood_NoSolves(t *testing.T) {
	testDB := SetupTestDB(t)
	challengeRepo := persistent.NewChallengeRepo(testDB.DB)
	solveRepo := persistent.NewSolveRepo(testDB.DB)
	ctx := context.Background()

	challenge := &entity.Challenge{
		Title:       "No Solves Challenge",
		Description: "Test",
		Category:    "Web",
		Points:      100,
		FlagHash:    "hash123",
		IsHidden:    false,
	}
	err := challengeRepo.Create(ctx, challenge)
	require.NoError(t, err)

	_, err = solveRepo.GetFirstBlood(ctx, challenge.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrSolveNotFound))
}
