package integration_test

import (
	"context"
	"testing"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHintRepo_CRUD(t *testing.T) {
	testDB := SetupTestDB(t)
	hintRepo := persistent.NewHintRepo(testDB.DB)
	challengeRepo := persistent.NewChallengeRepo(testDB.DB)
	ctx := context.Background()

	challenge := &entity.Challenge{
		Title:        "Hint Challenge",
		Description:  "Desc",
		Category:     "Web",
		Points:       100,
		FlagHash:     "hash",
		IsHidden:     false,
		InitialValue: 100,
		MinValue:     100,
		Decay:        0,
	}
	err := challengeRepo.Create(ctx, challenge)
	require.NoError(t, err)

	hint := &entity.Hint{
		ChallengeId: challenge.Id,
		Content:     "Secret Hint",
		Cost:        50,
		OrderIndex:  1,
	}
	err = hintRepo.Create(ctx, hint)
	require.NoError(t, err)
	assert.NotEmpty(t, hint.Id)

	gotHint, err := hintRepo.GetByID(ctx, hint.Id)
	require.NoError(t, err)
	assert.Equal(t, hint.Content, gotHint.Content)
	assert.Equal(t, hint.Cost, gotHint.Cost)

	hint.Content = "Updated Hint"
	hint.Cost = 75
	err = hintRepo.Update(ctx, hint)
	require.NoError(t, err)

	gotHintUpdated, err := hintRepo.GetByID(ctx, hint.Id)
	require.NoError(t, err)
	assert.Equal(t, "Updated Hint", gotHintUpdated.Content)
	assert.Equal(t, 75, gotHintUpdated.Cost)

	err = hintRepo.Delete(ctx, hint.Id)
	require.NoError(t, err)

	_, err = hintRepo.GetByID(ctx, hint.Id)
	assert.Error(t, err)
}

func TestHintUnlockRepo_Flow(t *testing.T) {
	testDB := SetupTestDB(t)
	hintRepo := persistent.NewHintRepo(testDB.DB)
	hintUnlockRepo := persistent.NewHintUnlockRepo(testDB.DB)
	challengeRepo := persistent.NewChallengeRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	userRepo := persistent.NewUserRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{Username: "u1", Email: "e1", PasswordHash: "p1"}
	require.NoError(t, userRepo.Create(ctx, user))
	gotUser, _ := userRepo.GetByEmail(ctx, user.Email)
	user.Id = gotUser.Id

	team := &entity.Team{Name: "t1", InviteToken: "tok", CaptainId: user.Id}
	require.NoError(t, teamRepo.Create(ctx, team))
	gotTeam, _ := teamRepo.GetByName(ctx, team.Name)
	team.Id = gotTeam.Id

	challenge := &entity.Challenge{Title: "C1", Description: "D1", Category: "Web", Points: 100, FlagHash: "h", IsHidden: false, InitialValue: 100, MinValue: 100, Decay: 0}
	require.NoError(t, challengeRepo.Create(ctx, challenge))

	hint := &entity.Hint{ChallengeId: challenge.Id, Content: "H1", Cost: 10, OrderIndex: 1}
	require.NoError(t, hintRepo.Create(ctx, hint))

	tx, err := testDB.DB.BeginTx(ctx, nil)
	require.NoError(t, err)

	err = hintUnlockRepo.CreateTx(ctx, tx, team.Id, hint.Id)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	unlock, err := hintUnlockRepo.GetByTeamAndHint(ctx, team.Id, hint.Id)
	require.NoError(t, err)
	assert.Equal(t, team.Id, unlock.TeamId)
	assert.Equal(t, hint.Id, unlock.HintId)

	ids, err := hintUnlockRepo.GetUnlockedHintIDs(ctx, team.Id, challenge.Id)
	require.NoError(t, err)
	assert.Contains(t, ids, hint.Id)
}

func TestAwardRepo_CreateTx_And_Total(t *testing.T) {
	testDB := SetupTestDB(t)
	awardRepo := persistent.NewAwardRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	userRepo := persistent.NewUserRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{Username: "u2", Email: "e2", PasswordHash: "p2"}
	require.NoError(t, userRepo.Create(ctx, user))
	gotUser, _ := userRepo.GetByEmail(ctx, user.Email)
	user.Id = gotUser.Id

	team := &entity.Team{Name: "t2", InviteToken: "tok2", CaptainId: user.Id}
	require.NoError(t, teamRepo.Create(ctx, team))
	gotTeam, _ := teamRepo.GetByName(ctx, team.Name)
	team.Id = gotTeam.Id

	tx, err := testDB.DB.BeginTx(ctx, nil)
	require.NoError(t, err)

	award := &entity.Award{
		TeamId:      team.Id,
		Value:       -50,
		Description: "Hint penalty",
	}
	err = awardRepo.CreateTx(ctx, tx, award)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	total, err := awardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, -50, total)

	tx2, _ := testDB.DB.BeginTx(ctx, nil)
	award2 := &entity.Award{TeamId: team.Id, Value: 100, Description: "Bonus"}
	require.NoError(t, awardRepo.CreateTx(ctx, tx2, award2))
	require.NoError(t, tx2.Commit())

	total, err = awardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 50, total) // -50 + 100 = 50
}

func TestScoreboardWithAwards(t *testing.T) {
	testDB := SetupTestDB(t)
	solveRepo := persistent.NewSolveRepo(testDB.DB)
	awardRepo := persistent.NewAwardRepo(testDB.DB)
	challengeRepo := persistent.NewChallengeRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	userRepo := persistent.NewUserRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{Username: "u3", Email: "e3", PasswordHash: "p3"}
	require.NoError(t, userRepo.Create(ctx, user))
	gotUser, _ := userRepo.GetByEmail(ctx, user.Email)
	user.Id = gotUser.Id

	team := &entity.Team{Name: "t3", InviteToken: "tok3", CaptainId: user.Id}
	require.NoError(t, teamRepo.Create(ctx, team))
	gotTeam, _ := teamRepo.GetByName(ctx, team.Name)
	team.Id = gotTeam.Id

	require.NoError(t, userRepo.UpdateTeamId(ctx, user.Id, &team.Id))

	challenge := &entity.Challenge{Title: "C3", Description: "D3", Category: "Web", Points: 100, FlagHash: "h", IsHidden: false, InitialValue: 100, MinValue: 100, Decay: 0}
	require.NoError(t, challengeRepo.Create(ctx, challenge))

	solve := &entity.Solve{UserId: user.Id, TeamId: team.Id, ChallengeId: challenge.Id}
	require.NoError(t, solveRepo.Create(ctx, solve))

	score, err := solveRepo.GetTeamScore(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 100, score)

	tx, _ := testDB.DB.BeginTx(ctx, nil)
	award := &entity.Award{TeamId: team.Id, Value: -20, Description: "Penalty"}
	require.NoError(t, awardRepo.CreateTx(ctx, tx, award))
	require.NoError(t, tx.Commit())

	score, err = solveRepo.GetTeamScore(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 80, score)

	scoreboard, err := solveRepo.GetScoreboard(ctx)
	require.NoError(t, err)
	found := false
	for _, entry := range scoreboard {
		if entry.TeamId == team.Id {
			assert.Equal(t, 80, entry.Points)
			found = true
			break
		}
	}
	assert.True(t, found)
}
