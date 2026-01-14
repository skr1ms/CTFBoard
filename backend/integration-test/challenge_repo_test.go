package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChallengeRepo_Create(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewChallengeRepo(testDB.DB)
	ctx := context.Background()

	challenge := &entity.Challenge{
		Title:       "Test Challenge",
		Description: "Test Description",
		Category:    "Web",
		Points:      100,
		FlagHash:    "hash123",
		IsHidden:    false,
	}

	err := repo.Create(ctx, challenge)
	require.NoError(t, err)
	assert.NotEmpty(t, challenge.Id)
}

func TestChallengeRepo_GetByID(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewChallengeRepo(testDB.DB)
	ctx := context.Background()

	challenge := &entity.Challenge{
		Title:       "GetByID Challenge",
		Description: "Description",
		Category:    "Crypto",
		Points:      200,
		FlagHash:    "hash456",
		IsHidden:    false,
	}

	err := repo.Create(ctx, challenge)
	require.NoError(t, err)

	gotChallenge, err := repo.GetByID(ctx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, challenge.Id, gotChallenge.Id)
	assert.Equal(t, challenge.Title, gotChallenge.Title)
	assert.Equal(t, challenge.Points, gotChallenge.Points)
	assert.Equal(t, challenge.FlagHash, gotChallenge.FlagHash)
}

func TestChallengeRepo_GetByID_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewChallengeRepo(testDB.DB)
	ctx := context.Background()

	nonExistentID := uuid.New().String()
	_, err := repo.GetByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrChallengeNotFound))
}

func TestChallengeRepo_GetAll_NoTeam(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewChallengeRepo(testDB.DB)
	ctx := context.Background()

	challenge1 := &entity.Challenge{
		Title:       "Public Challenge 1",
		Description: "Description 1",
		Category:    "Web",
		Points:      100,
		FlagHash:    "hash1",
		IsHidden:    false,
	}
	err := repo.Create(ctx, challenge1)
	require.NoError(t, err)

	challenge2 := &entity.Challenge{
		Title:       "Public Challenge 2",
		Description: "Description 2",
		Category:    "Crypto",
		Points:      200,
		FlagHash:    "hash2",
		IsHidden:    false,
	}
	err = repo.Create(ctx, challenge2)
	require.NoError(t, err)

	hiddenChallenge := &entity.Challenge{
		Title:       "Hidden Challenge",
		Description: "Description",
		Category:    "Pwn",
		Points:      300,
		FlagHash:    "hash3",
		IsHidden:    true,
	}
	err = repo.Create(ctx, hiddenChallenge)
	require.NoError(t, err)

	challenges, err := repo.GetAll(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, challenges, 2)
	for _, ch := range challenges {
		assert.False(t, ch.Challenge.IsHidden)
		assert.False(t, ch.Solved)
	}
}

func TestChallengeRepo_GetAll_WithTeam(t *testing.T) {
	testDB := SetupTestDB(t)
	challengeRepo := persistent.NewChallengeRepo(testDB.DB)
	userRepo := persistent.NewUserRepo(testDB.DB)
	teamRepo := persistent.NewTeamRepo(testDB.DB)
	solveRepo := persistent.NewSolveRepo(testDB.DB)
	ctx := context.Background()

	user := &entity.User{
		Username:     "teamuser",
		Email:        "teamuser@example.com",
		PasswordHash: "hash123",
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.Id = gotUser.Id

	team := &entity.Team{
		Name:        "testteam",
		InviteToken: "token123",
		CaptainId:   user.Id,
	}
	err = teamRepo.Create(ctx, team)
	require.NoError(t, err)
	gotTeam, err := teamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.Id = gotTeam.Id

	err = userRepo.UpdateTeamId(ctx, user.Id, &team.Id)
	require.NoError(t, err)

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

	solve := &entity.Solve{
		UserId:      user.Id,
		TeamId:      team.Id,
		ChallengeId: challenge1.Id,
	}
	err = solveRepo.Create(ctx, solve)
	require.NoError(t, err)

	challenges, err := challengeRepo.GetAll(ctx, &team.Id)
	require.NoError(t, err)
	assert.Len(t, challenges, 2)

	solved := false
	for _, ch := range challenges {
		if ch.Challenge.Id == challenge1.Id {
			assert.True(t, ch.Solved)
			solved = true
		} else {
			assert.False(t, ch.Solved)
		}
	}
	assert.True(t, solved)
}

func TestChallengeRepo_Update(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewChallengeRepo(testDB.DB)
	ctx := context.Background()

	challenge := &entity.Challenge{
		Title:       "Original Title",
		Description: "Original Description",
		Category:    "Web",
		Points:      100,
		FlagHash:    "original_hash",
		IsHidden:    false,
	}

	err := repo.Create(ctx, challenge)
	require.NoError(t, err)

	challenge.Title = "Updated Title"
	challenge.Description = "Updated Description"
	challenge.Category = "Crypto"
	challenge.Points = 200
	challenge.FlagHash = "updated_hash"
	challenge.IsHidden = true

	err = repo.Update(ctx, challenge)
	require.NoError(t, err)

	gotChallenge, err := repo.GetByID(ctx, challenge.Id)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", gotChallenge.Title)
	assert.Equal(t, "Updated Description", gotChallenge.Description)
	assert.Equal(t, "Crypto", gotChallenge.Category)
	assert.Equal(t, 200, gotChallenge.Points)
	assert.Equal(t, "updated_hash", gotChallenge.FlagHash)
	assert.True(t, gotChallenge.IsHidden)
}

func TestChallengeRepo_Delete(t *testing.T) {
	testDB := SetupTestDB(t)
	repo := persistent.NewChallengeRepo(testDB.DB)
	ctx := context.Background()

	challenge := &entity.Challenge{
		Title:       "To Delete",
		Description: "Description",
		Category:    "Web",
		Points:      100,
		FlagHash:    "hash123",
		IsHidden:    false,
	}

	err := repo.Create(ctx, challenge)
	require.NoError(t, err)

	err = repo.Delete(ctx, challenge.Id)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, challenge.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrChallengeNotFound))
}
