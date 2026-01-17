package integration_test

import (
	"context"
	"testing"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAwardRepo_CreateTx(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	awardRepo := persistent.NewAwardRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "awarduser",
		Email:        "awarduser@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "awardteam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	tx, err := testDB.DB.BeginTx(ctx, nil)
	require.NoError(t, err)

	award := &entity.Award{
		TeamId:      team.Id,
		Value:       100,
		Description: "Test award",
	}

	err = awardRepo.CreateTx(ctx, tx, award)
	require.NoError(t, err)
	assert.NotEmpty(t, award.Id)

	err = tx.Commit()
	require.NoError(t, err)

	total, err := awardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 100, total)
}

func TestAwardRepo_CreateTx_Rollback(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	awardRepo := persistent.NewAwardRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "rollbackawarduser",
		Email:        "rollbackawarduser@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "rollbackawardteam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	tx, err := testDB.DB.BeginTx(ctx, nil)
	require.NoError(t, err)

	award := &entity.Award{
		TeamId:      team.Id,
		Value:       200,
		Description: "Rollback award",
	}

	err = awardRepo.CreateTx(ctx, tx, award)
	require.NoError(t, err)

	// Rollback instead of commit
	err = tx.Rollback()
	require.NoError(t, err)

	// Award should not exist
	total, err := awardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestAwardRepo_GetTeamTotalAwards(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	awardRepo := persistent.NewAwardRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "totalawarduser",
		Email:        "totalawarduser@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "totalawardteam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	// Create multiple awards
	awards := []int{100, 50, -25, 75}
	expectedTotal := 0
	for i, value := range awards {
		tx, err := testDB.DB.BeginTx(ctx, nil)
		require.NoError(t, err)

		award := &entity.Award{
			TeamId:      team.Id,
			Value:       value,
			Description: "Award " + string(rune('A'+i)),
		}
		err = awardRepo.CreateTx(ctx, tx, award)
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		expectedTotal += value
	}

	total, err := awardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, expectedTotal, total)
}

func TestAwardRepo_GetTeamTotalAwards_NoAwards(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	awardRepo := persistent.NewAwardRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "noawarduser",
		Email:        "noawarduser@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "noawardteam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	// No awards created
	total, err := awardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestAwardRepo_NegativeAward(t *testing.T) {
	testDB := SetupTestDB(t)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	awardRepo := persistent.NewAwardRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "negawarduser",
		Email:        "negawarduser@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "negawardteam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	// Create positive award
	tx1, err := testDB.DB.BeginTx(ctx, nil)
	require.NoError(t, err)

	award1 := &entity.Award{
		TeamId:      team.Id,
		Value:       100,
		Description: "Bonus",
	}
	err = awardRepo.CreateTx(ctx, tx1, award1)
	require.NoError(t, err)
	err = tx1.Commit()
	require.NoError(t, err)

	// Create negative award
	tx2, err := testDB.DB.BeginTx(ctx, nil)
	require.NoError(t, err)

	award2 := &entity.Award{
		TeamId:      team.Id,
		Value:       -30,
		Description: "Penalty for hint",
	}
	err = awardRepo.CreateTx(ctx, tx2, award2)
	require.NoError(t, err)
	err = tx2.Commit()
	require.NoError(t, err)

	total, err := awardRepo.GetTeamTotalAwards(ctx, team.Id)
	require.NoError(t, err)
	assert.Equal(t, 70, total)
}
