package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/usecase/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func sha256Hash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// GetAll Tests

func TestChallengeUseCase_GetAll(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	teamID := uuid.New()
	challenges := []*repo.ChallengeWithSolved{
		{
			Challenge: &entity.Challenge{
				Id:          uuid.New(),
				Title:       "Test Challenge",
				Description: "Test Description",
				Category:    "Web",
				Points:      100,
			},
			Solved: true,
		},
	}

	challengeRepo.On("GetAll", mock.Anything, &teamID).Return(challenges, nil)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	result, err := uc.GetAll(context.Background(), &teamID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, challenges[0].Challenge.Title, result[0].Challenge.Title)
}

func TestChallengeUseCase_GetAll_NoTeamId(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challenges := []*repo.ChallengeWithSolved{
		{
			Challenge: &entity.Challenge{
				Id:          uuid.New(),
				Title:       "Test Challenge",
				Description: "Test Description",
				Category:    "Web",
				Points:      100,
			},
			Solved: false,
		},
	}

	challengeRepo.On("GetAll", mock.Anything, (*uuid.UUID)(nil)).Return(challenges, nil)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	result, err := uc.GetAll(context.Background(), nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
}

func TestChallengeUseCase_GetAll_Error(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	teamID := uuid.New()
	expectedError := assert.AnError

	challengeRepo.On("GetAll", mock.Anything, &teamID).Return(nil, expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	result, err := uc.GetAll(context.Background(), &teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// Create Tests

func TestChallengeUseCase_Create(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.Title == "New Challenge" && c.Points == 200
	})).Return(nil).Run(func(args mock.Arguments) {
		c := args.Get(1).(*entity.Challenge)
		c.Id = uuid.New()
	})

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	challenge, err := uc.Create(context.Background(), "New Challenge", "Description", "Crypto", 200, 500, 100, 20, "flag{test}", false)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, "New Challenge", challenge.Title)
	assert.Equal(t, 200, challenge.Points)
	assert.NotEmpty(t, challenge.FlagHash)
}

func TestChallengeUseCase_Create_Error(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	expectedError := assert.AnError
	challengeRepo.On("Create", mock.Anything, mock.Anything).Return(expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	challenge, err := uc.Create(context.Background(), "New Challenge", "Description", "Crypto", 200, 500, 100, 20, "flag{test}", false)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

// Update Tests

func TestChallengeUseCase_Update(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New()
	existingChallenge := &entity.Challenge{
		Id:          challengeID,
		Title:       "Old Title",
		Description: "Old Description",
		Category:    "Web",
		Points:      100,
		FlagHash:    "old_hash",
		IsHidden:    false,
	}

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.Id == challengeID && c.Title == "Updated Title" && c.Points == 150
	})).Return(nil)
	redisClient.On("Del", mock.Anything, []string{"scoreboard"}).Return(redis.NewIntCmd(context.Background()))

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "", false)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, "Updated Title", challenge.Title)
	assert.Equal(t, 150, challenge.Points)
}

func TestChallengeUseCase_Update_WithNewFlag(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New()
	existingChallenge := &entity.Challenge{
		Id:          challengeID,
		Title:       "Old Title",
		Description: "Old Description",
		Category:    "Web",
		Points:      100,
		FlagHash:    "old_hash",
		IsHidden:    false,
	}

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Challenge) bool {
		return c.Id == challengeID && c.FlagHash != "old_hash"
	})).Return(nil)
	redisClient.On("Del", mock.Anything, []string{"scoreboard"}).Return(redis.NewIntCmd(context.Background()))

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "new_flag", false)

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.NotEqual(t, "old_hash", challenge.FlagHash)
}

func TestChallengeUseCase_Update_GetByIDError(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New()
	expectedError := assert.AnError

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "", false)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_Update_UpdateError(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New()
	existingChallenge := &entity.Challenge{
		Id:          challengeID,
		Title:       "Old Title",
		Description: "Old Description",
		Category:    "Web",
		Points:      100,
		FlagHash:    "old_hash",
		IsHidden:    false,
	}
	expectedError := assert.AnError

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	challengeRepo.On("Update", mock.Anything, mock.Anything).Return(expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	challenge, err := uc.Update(context.Background(), challengeID, "Updated Title", "Updated Description", "Crypto", 150, 500, 100, 20, "", false)

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

// Delete Tests

func TestChallengeUseCase_Delete(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New()
	challengeRepo.On("Delete", mock.Anything, challengeID).Return(nil)
	redisClient.On("Del", mock.Anything, []string{"scoreboard"}).Return(redis.NewIntCmd(context.Background()))

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	err := uc.Delete(context.Background(), challengeID)

	assert.NoError(t, err)
}

func TestChallengeUseCase_Delete_Error(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New()
	expectedError := assert.AnError
	challengeRepo.On("Delete", mock.Anything, challengeID).Return(expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	err := uc.Delete(context.Background(), challengeID)

	assert.Error(t, err)
}

// SubmitFlag Tests

func TestChallengeUseCase_SubmitFlag_Success(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := sha256Hash(flag)
	challenge := &entity.Challenge{
		Id:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}

	mockTx := mocks.NewMockPgxTx(t)
	mockTx.On("Commit", mock.Anything).Return(nil)

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, entityError.ErrSolveNotFound)
	txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("CreateSolveTx", mock.Anything, mock.Anything, mock.MatchedBy(func(s *entity.Solve) bool {
		return s.ChallengeId == challengeID && s.TeamId == teamID && s.UserId == userID
	})).Return(nil)
	txRepo.On("IncrementChallengeSolveCountTx", mock.Anything, mock.Anything, challengeID).Return(1, nil)
	redisClient.On("Del", mock.Anything, []string{"scoreboard"}).Return(redis.NewIntCmd(context.Background()))

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestChallengeUseCase_SubmitFlag_InvalidFlag(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()

	challenge := &entity.Challenge{
		Id:       challengeID,
		Title:    "Test Challenge",
		FlagHash: sha256Hash("flag{correct}"),
		Points:   100,
	}

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{wrong}", userID, &teamID)

	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_NoTeam(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New()
	userID := uuid.New()

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, nil)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrUserMustBeInTeam))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_ChallengeNotFound(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, entityError.ErrChallengeNotFound)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrChallengeNotFound))
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_GetByIDUnexpectedError(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	expectedError := assert.AnError

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(nil, expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, "flag{test}", userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_AlreadySolved(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := sha256Hash(flag)
	challenge := &entity.Challenge{
		Id:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}

	existingSolve := &entity.Solve{
		Id:          uuid.New(),
		TeamId:      teamID,
		ChallengeId: challengeID,
	}

	mockTx := mocks.NewMockPgxTx(t)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(existingSolve, nil)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrAlreadySolved))
	assert.True(t, valid)
}

func TestChallengeUseCase_SubmitFlag_BeginTxError(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := sha256Hash(flag)
	challenge := &entity.Challenge{
		Id:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	expectedError := assert.AnError

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("BeginTx", mock.Anything).Return(nil, expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_GetByTeamAndChallengeTxUnexpectedError(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := sha256Hash(flag)
	challenge := &entity.Challenge{
		Id:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	expectedError := assert.AnError

	mockTx := mocks.NewMockPgxTx(t)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_CreateTxError(t *testing.T) {
	challengeRepo := mocks.NewMockChallengeRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"

	hash := sha256Hash(flag)
	challenge := &entity.Challenge{
		Id:       challengeID,
		Title:    "Test Challenge",
		FlagHash: hash,
		Points:   100,
	}
	expectedError := assert.AnError

	mockTx := mocks.NewMockPgxTx(t)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, entityError.ErrSolveNotFound)
	txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("CreateSolveTx", mock.Anything, mock.Anything, mock.Anything).Return(expectedError)

	uc := NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, nil)

	valid, err := uc.SubmitFlag(context.Background(), challengeID, flag, userID, &teamID)

	assert.Error(t, err)
	assert.False(t, valid)
}
