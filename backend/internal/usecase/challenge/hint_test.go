package challenge

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/usecase/challenge/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHintUseCase_Create(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	deps.hintRepo.On("Create", mock.Anything, mock.MatchedBy(func(h *entity.Hint) bool {
		return h.Content == "test hint" && h.Cost == 50
	})).Return(nil).Run(func(args mock.Arguments) {
		h, ok := args.Get(1).(*entity.Hint)
		if !ok {
			return
		}
		h.ID = uuid.New()
	})

	hint, err := uc.Create(context.Background(), uuid.New(), "test hint", 50, 0)

	assert.NoError(t, err)
	assert.NotNil(t, hint)
	assert.Equal(t, "test hint", hint.Content)
	assert.Equal(t, 50, hint.Cost)
}

func TestHintUseCase_Create_Error(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	deps.hintRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))

	hint, err := uc.Create(context.Background(), uuid.New(), "test hint", 50, 0)

	assert.Error(t, err)
	assert.Nil(t, hint)
}

// GetByID Tests

func TestHintUseCase_GetByID(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	hintID := uuid.New()
	hint := &entity.Hint{ID: hintID, Content: "Secret hint", Cost: 50}
	deps.hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)

	result, err := uc.GetByID(context.Background(), hintID)

	assert.NoError(t, err)
	assert.Equal(t, hintID, result.ID)
	assert.Equal(t, "Secret hint", result.Content)
}

func TestHintUseCase_GetByID_NotFound(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	hintID := uuid.New()
	deps.hintRepo.On("GetByID", mock.Anything, hintID).Return(nil, entityError.ErrHintNotFound)

	result, err := uc.GetByID(context.Background(), hintID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, entityError.ErrHintNotFound))
}

// GetByChallengeID Tests

func TestHintUseCase_GetByChallengeID(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	hint1ID := uuid.New()
	hint2ID := uuid.New()

	hints := []*entity.Hint{{ID: hint1ID, ChallengeID: challengeID, Content: "Hint 1", Cost: 10, OrderIndex: 0}, {ID: hint2ID, ChallengeID: challengeID, Content: "Hint 2", Cost: 20, OrderIndex: 1}}

	deps.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return(hints, nil)
	deps.hintUnlockRepo.On("GetUnlockedHintIDs", mock.Anything, teamID, challengeID).Return([]uuid.UUID{hint1ID}, nil)

	result, err := uc.GetByChallengeID(context.Background(), challengeID, &teamID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, hint1ID, result[0].Hint.ID)
	assert.True(t, result[0].Unlocked)
	assert.Equal(t, "Hint 1", result[0].Hint.Content)
	assert.Equal(t, hint2ID, result[1].Hint.ID)
	assert.False(t, result[1].Unlocked)
	assert.Empty(t, result[1].Hint.Content)
}

func TestHintUseCase_GetByChallengeID_RepoError(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()

	deps.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return(nil, errors.New("db error"))

	result, err := uc.GetByChallengeID(context.Background(), challengeID, &teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestHintUseCase_GetByChallengeID_UnlockRepoError(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()

	hints := []*entity.Hint{{ID: uuid.New(), ChallengeID: challengeID}}

	deps.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return(hints, nil)
	deps.hintUnlockRepo.On("GetUnlockedHintIDs", mock.Anything, teamID, challengeID).Return(nil, errors.New("db error"))

	result, err := uc.GetByChallengeID(context.Background(), challengeID, &teamID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// Update Tests

func TestHintUseCase_Update(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	hintID := uuid.New()
	hint := &entity.Hint{ID: hintID, Content: "Old content", Cost: 50, OrderIndex: 0}

	deps.hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	deps.hintRepo.On("Update", mock.Anything, mock.MatchedBy(func(h *entity.Hint) bool {
		return h.Content == "New content" && h.Cost == 100 && h.OrderIndex == 1
	})).Return(nil)

	result, err := uc.Update(context.Background(), hintID, "New content", 100, 1)

	assert.NoError(t, err)
	assert.Equal(t, "New content", result.Content)
	assert.Equal(t, 100, result.Cost)
	assert.Equal(t, 1, result.OrderIndex)
}

func TestHintUseCase_Update_NotFound(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	hintID := uuid.New()

	deps.hintRepo.On("GetByID", mock.Anything, hintID).Return(nil, entityError.ErrHintNotFound)

	result, err := uc.Update(context.Background(), hintID, "New content", 100, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, entityError.ErrHintNotFound))
}

func TestHintUseCase_Update_RepoError(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	hintID := uuid.New()
	hint := &entity.Hint{ID: hintID, Content: "Old content", Cost: 50}

	deps.hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	deps.hintRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("db error"))

	result, err := uc.Update(context.Background(), hintID, "New content", 100, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// Delete Tests

func TestHintUseCase_Delete(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	hintID := uuid.New()

	deps.hintRepo.On("Delete", mock.Anything, hintID).Return(nil)

	err := uc.Delete(context.Background(), hintID)

	assert.NoError(t, err)
}

func TestHintUseCase_Delete_Error(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	hintID := uuid.New()

	deps.hintRepo.On("Delete", mock.Anything, hintID).Return(errors.New("db error"))

	err := uc.Delete(context.Background(), hintID)

	assert.Error(t, err)
}

// UnlockHint Tests

func TestHintUseCase_UnlockHint_Success(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateHintUseCase()

	teamID := uuid.New()
	hintID := uuid.New()

	hint := &entity.Hint{ID: hintID, Content: "Secret hint", Cost: 50}

	mockTx := mocks.NewMockPgxTx(t)
	mockTx.On("Commit", mock.Anything).Return(nil)

	deps.hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	deps.txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	deps.txRepo.On("LockTeamTx", mock.Anything, mock.Anything, teamID).Return(nil)
	deps.txRepo.On("GetHintUnlockByTeamAndHintTx", mock.Anything, mock.Anything, teamID, hintID).Return(nil, entityError.ErrHintNotFound)
	deps.txRepo.On("GetTeamScoreTx", mock.Anything, mock.Anything, teamID).Return(100, nil)
	deps.txRepo.On("CreateAwardTx", mock.Anything, mock.Anything, mock.MatchedBy(func(a *entity.Award) bool {
		return a.Value == -50 && a.TeamID == teamID
	})).Return(nil)
	deps.txRepo.On("CreateHintUnlockTx", mock.Anything, mock.Anything, teamID, hintID).Return(nil)
	redisClient.ExpectDel("scoreboard").SetVal(0)
	redisClient.ExpectDel("scoreboard:frozen").SetVal(0)

	unlocked, err := uc.UnlockHint(context.Background(), teamID, hintID)

	assert.NoError(t, err)
	assert.NotNil(t, unlocked)
	assert.Equal(t, hintID, unlocked.ID)
	assert.Equal(t, "Secret hint", unlocked.Content)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestHintUseCase_UnlockHint_FreeHint(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateHintUseCase()

	teamID := uuid.New()
	hintID := uuid.New()

	hint := &entity.Hint{ID: hintID, Content: "Free hint", Cost: 0}

	mockTx := mocks.NewMockPgxTx(t)
	mockTx.On("Commit", mock.Anything).Return(nil)

	deps.hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	deps.txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	deps.txRepo.On("LockTeamTx", mock.Anything, mock.Anything, teamID).Return(nil)
	deps.txRepo.On("GetHintUnlockByTeamAndHintTx", mock.Anything, mock.Anything, teamID, hintID).Return(nil, entityError.ErrHintNotFound)
	deps.txRepo.On("CreateHintUnlockTx", mock.Anything, mock.Anything, teamID, hintID).Return(nil)
	redisClient.ExpectDel("scoreboard").SetVal(0)
	redisClient.ExpectDel("scoreboard:frozen").SetVal(0)

	unlocked, err := uc.UnlockHint(context.Background(), teamID, hintID)

	assert.NoError(t, err)
	assert.NotNil(t, unlocked)
	assert.Equal(t, "Free hint", unlocked.Content)
	deps.txRepo.AssertNotCalled(t, "CreateAwardTx", mock.Anything, mock.Anything, mock.Anything)
	deps.txRepo.AssertNotCalled(t, "GetTeamScoreTx", mock.Anything, mock.Anything, mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestHintUseCase_UnlockHint_NotFound(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	hintID := uuid.New()

	deps.hintRepo.On("GetByID", mock.Anything, hintID).Return(nil, entityError.ErrHintNotFound)

	unlocked, err := uc.UnlockHint(context.Background(), uuid.New(), hintID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrHintNotFound))
	assert.Nil(t, unlocked)
}

func TestHintUseCase_UnlockHint_BeginTxError(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	teamID := uuid.New()
	hintID := uuid.New()

	hint := &entity.Hint{ID: hintID, Content: "Secret", Cost: 50}

	deps.hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	deps.txRepo.On("BeginTx", mock.Anything).Return(nil, errors.New("tx error"))

	unlocked, err := uc.UnlockHint(context.Background(), teamID, hintID)

	assert.Error(t, err)
	assert.Nil(t, unlocked)
	assert.Contains(t, err.Error(), "tx error")
}

func TestHintUseCase_UnlockHint_AlreadyUnlocked(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	teamID := uuid.New()
	hintID := uuid.New()

	hint := &entity.Hint{ID: hintID, Cost: 50}

	mockTx := mocks.NewMockPgxTx(t)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	deps.hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	deps.txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	deps.txRepo.On("LockTeamTx", mock.Anything, mock.Anything, teamID).Return(nil)
	deps.txRepo.On("GetHintUnlockByTeamAndHintTx", mock.Anything, mock.Anything, teamID, hintID).Return(&entity.HintUnlock{}, nil)

	unlocked, err := uc.UnlockHint(context.Background(), teamID, hintID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrHintAlreadyUnlocked))
	assert.Nil(t, unlocked)
}

func TestHintUseCase_UnlockHint_InsufficientPoints(t *testing.T) {
	h := NewChallengeTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateHintUseCase()

	teamID := uuid.New()
	hintID := uuid.New()

	hint := &entity.Hint{ID: hintID, Cost: 100}

	mockTx := mocks.NewMockPgxTx(t)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	deps.hintRepo.On("GetByID", mock.Anything, hintID).Return(hint, nil)
	deps.txRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	deps.txRepo.On("LockTeamTx", mock.Anything, mock.Anything, teamID).Return(nil)
	deps.txRepo.On("GetHintUnlockByTeamAndHintTx", mock.Anything, mock.Anything, teamID, hintID).Return(nil, entityError.ErrHintNotFound)
	deps.txRepo.On("GetTeamScoreTx", mock.Anything, mock.Anything, teamID).Return(50, nil)

	unlocked, err := uc.UnlockHint(context.Background(), teamID, hintID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrInsufficientPoints))
	assert.Nil(t, unlocked)
}
