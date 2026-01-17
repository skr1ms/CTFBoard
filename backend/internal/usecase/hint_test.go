package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/usecase/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Create Tests

func TestHintUseCase_Create(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	hintRepo.On("Create", mock.Anything, mock.MatchedBy(func(h *entity.Hint) bool {
		return h.Content == "test hint" && h.Cost == 50
	})).Return(nil).Run(func(args mock.Arguments) {
		h := args.Get(1).(*entity.Hint)
		h.Id = uuid.New().String()
	})

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	hint, err := uc.Create(context.Background(), uuid.New().String(), "test hint", 50, 0)

	assert.NoError(t, err)
	assert.NotNil(t, hint)
	assert.Equal(t, "test hint", hint.Content)
	assert.Equal(t, 50, hint.Cost)
}

func TestHintUseCase_Create_Error(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	hintRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	hint, err := uc.Create(context.Background(), uuid.New().String(), "test hint", 50, 0)

	assert.Error(t, err)
	assert.Nil(t, hint)
}

// GetByID Tests

func TestHintUseCase_GetByID(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	hintID := uuid.New().String()
	hint := &entity.Hint{Id: hintID, Content: "Secret hint", Cost: 50}

	hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	result, err := uc.GetByID(context.Background(), hintID)

	assert.NoError(t, err)
	assert.Equal(t, hintID, result.Id)
	assert.Equal(t, "Secret hint", result.Content)
}

func TestHintUseCase_GetByID_NotFound(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	hintID := uuid.New().String()

	hintRepo.On("GetByID", mock.Anything, hintID).Return(nil, entityError.ErrHintNotFound)

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	result, err := uc.GetByID(context.Background(), hintID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, entityError.ErrHintNotFound))
}

// GetByChallengeID Tests

func TestHintUseCase_GetByChallengeID(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New().String()
	teamID := uuid.New().String()
	hint1ID := uuid.New().String()
	hint2ID := uuid.New().String()

	hints := []*entity.Hint{{Id: hint1ID, ChallengeId: challengeID, Content: "Hint 1", Cost: 10, OrderIndex: 0}, {Id: hint2ID, ChallengeId: challengeID, Content: "Hint 2", Cost: 20, OrderIndex: 1}}

	hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return(hints, nil)
	hintUnlockRepo.On("GetUnlockedHintIDs", mock.Anything, teamID, challengeID).Return([]string{hint1ID}, nil)

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	result, err := uc.GetByChallengeID(context.Background(), challengeID, &teamID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, hint1ID, result[0].Hint.Id)
	assert.True(t, result[0].Unlocked)
	assert.Equal(t, "Hint 1", result[0].Hint.Content)
	assert.Equal(t, hint2ID, result[1].Hint.Id)
	assert.False(t, result[1].Unlocked)
	assert.Empty(t, result[1].Hint.Content)
}

func TestHintUseCase_GetByChallengeID_RepoError(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New().String()
	teamID := uuid.New().String()

	hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return(nil, errors.New("db error"))

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	result, err := uc.GetByChallengeID(context.Background(), challengeID, &teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestHintUseCase_GetByChallengeID_UnlockRepoError(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New().String()
	teamID := uuid.New().String()

	hints := []*entity.Hint{{Id: uuid.New().String(), ChallengeId: challengeID}}

	hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return(hints, nil)
	hintUnlockRepo.On("GetUnlockedHintIDs", mock.Anything, teamID, challengeID).Return(nil, errors.New("db error"))

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	result, err := uc.GetByChallengeID(context.Background(), challengeID, &teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// Update Tests

func TestHintUseCase_Update(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	hintID := uuid.New().String()
	hint := &entity.Hint{Id: hintID, Content: "Old content", Cost: 50, OrderIndex: 0}

	hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	hintRepo.On("Update", mock.Anything, mock.MatchedBy(func(h *entity.Hint) bool {
		return h.Content == "New content" && h.Cost == 100 && h.OrderIndex == 1
	})).Return(nil)

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	result, err := uc.Update(context.Background(), hintID, "New content", 100, 1)

	assert.NoError(t, err)
	assert.Equal(t, "New content", result.Content)
	assert.Equal(t, 100, result.Cost)
	assert.Equal(t, 1, result.OrderIndex)
}

func TestHintUseCase_Update_NotFound(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	hintID := uuid.New().String()

	hintRepo.On("GetByID", mock.Anything, hintID).Return(nil, entityError.ErrHintNotFound)

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	result, err := uc.Update(context.Background(), hintID, "New content", 100, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, entityError.ErrHintNotFound))
}

func TestHintUseCase_Update_RepoError(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	hintID := uuid.New().String()
	hint := &entity.Hint{Id: hintID, Content: "Old content", Cost: 50}

	hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	hintRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("db error"))

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	result, err := uc.Update(context.Background(), hintID, "New content", 100, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// Delete Tests

func TestHintUseCase_Delete(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	hintID := uuid.New().String()

	hintRepo.On("Delete", mock.Anything, hintID).Return(nil)

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	err := uc.Delete(context.Background(), hintID)

	assert.NoError(t, err)
}

func TestHintUseCase_Delete_Error(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	hintID := uuid.New().String()

	hintRepo.On("Delete", mock.Anything, hintID).Return(errors.New("db error"))

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	err := uc.Delete(context.Background(), hintID)

	assert.Error(t, err)
}

// UnlockHint Tests

func TestHintUseCase_UnlockHint_Success(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	teamID := uuid.New().String()
	hintID := uuid.New().String()

	hint := &entity.Hint{Id: hintID, Content: "Secret hint", Cost: 50}

	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	sqlMock.ExpectBegin()
	sqlMock.ExpectCommit()

	mockTx, err := db.Begin()
	assert.NoError(t, err)

	hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	hintUnlockRepo.On("GetByTeamAndHintTx", mock.Anything, mockTx, teamID, hintID).Return(nil, entityError.ErrHintNotFound)
	solveRepo.On("GetTeamScoreTx", mock.Anything, mockTx, teamID).Return(100, nil)
	awardRepo.On("CreateTx", mock.Anything, mockTx, mock.MatchedBy(func(a *entity.Award) bool {
		return a.Value == -50 && a.TeamId == teamID
	})).Return(nil)
	hintUnlockRepo.On("CreateTx", mock.Anything, mockTx, teamID, hintID).Return(nil)
	redisClient.On("Del", mock.Anything, mock.Anything).Return(redis.NewIntCmd(context.Background())).Maybe()

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	unlocked, err := uc.UnlockHint(context.Background(), teamID, hintID)

	assert.NoError(t, err)
	assert.NotNil(t, unlocked)
	assert.Equal(t, hintID, unlocked.Id)
	assert.Equal(t, "Secret hint", unlocked.Content)
}

func TestHintUseCase_UnlockHint_FreeHint(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	teamID := uuid.New().String()
	hintID := uuid.New().String()

	hint := &entity.Hint{Id: hintID, Content: "Free hint", Cost: 0}

	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	sqlMock.ExpectBegin()
	sqlMock.ExpectCommit()

	mockTx, err := db.Begin()
	assert.NoError(t, err)

	hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	hintUnlockRepo.On("GetByTeamAndHintTx", mock.Anything, mockTx, teamID, hintID).Return(nil, entityError.ErrHintNotFound)
	hintUnlockRepo.On("CreateTx", mock.Anything, mockTx, teamID, hintID).Return(nil)
	redisClient.On("Del", mock.Anything, mock.Anything).Return(redis.NewIntCmd(context.Background())).Maybe()

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	unlocked, err := uc.UnlockHint(context.Background(), teamID, hintID)

	assert.NoError(t, err)
	assert.NotNil(t, unlocked)
	assert.Equal(t, "Free hint", unlocked.Content)
	awardRepo.AssertNotCalled(t, "CreateTx", mock.Anything, mock.Anything, mock.Anything)
	solveRepo.AssertNotCalled(t, "GetTeamScoreTx", mock.Anything, mock.Anything, mock.Anything)
}

func TestHintUseCase_UnlockHint_NotFound(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	hintID := uuid.New().String()

	hintRepo.On("GetByID", mock.Anything, hintID).Return(nil, entityError.ErrHintNotFound)

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	unlocked, err := uc.UnlockHint(context.Background(), uuid.New().String(), hintID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrHintNotFound))
	assert.Nil(t, unlocked)
}

func TestHintUseCase_UnlockHint_BeginTxError(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	teamID := uuid.New().String()
	hintID := uuid.New().String()

	hint := &entity.Hint{Id: hintID, Content: "Secret", Cost: 50}

	hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	txRepo.On("BeginTx", mock.Anything).Return(nil, errors.New("tx error"))

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	unlocked, err := uc.UnlockHint(context.Background(), teamID, hintID)

	assert.Error(t, err)
	assert.Nil(t, unlocked)
	assert.Contains(t, err.Error(), "tx error")
}

func TestHintUseCase_UnlockHint_AlreadyUnlocked(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	teamID := uuid.New().String()
	hintID := uuid.New().String()

	hint := &entity.Hint{Id: hintID, Cost: 50}

	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()

	mockTx, err := db.Begin()
	assert.NoError(t, err)

	hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	hintUnlockRepo.On("GetByTeamAndHintTx", mock.Anything, mockTx, teamID, hintID).Return(&entity.HintUnlock{}, nil)

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	unlocked, err := uc.UnlockHint(context.Background(), teamID, hintID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrHintAlreadyUnlocked))
	assert.Nil(t, unlocked)
}

func TestHintUseCase_UnlockHint_InsufficientPoints(t *testing.T) {
	hintRepo := mocks.NewMockHintRepository(t)
	hintUnlockRepo := mocks.NewMockHintUnlockRepository(t)
	awardRepo := mocks.NewMockAwardRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	teamID := uuid.New().String()
	hintID := uuid.New().String()

	hint := &entity.Hint{Id: hintID, Cost: 100}

	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()

	mockTx, err := db.Begin()
	assert.NoError(t, err)

	hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	hintUnlockRepo.On("GetByTeamAndHintTx", mock.Anything, mockTx, teamID, hintID).Return(nil, entityError.ErrHintNotFound)
	solveRepo.On("GetTeamScoreTx", mock.Anything, mockTx, teamID).Return(50, nil)

	uc := NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)

	unlocked, err := uc.UnlockHint(context.Background(), teamID, hintID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrInsufficientPoints))
	assert.Nil(t, unlocked)
}
