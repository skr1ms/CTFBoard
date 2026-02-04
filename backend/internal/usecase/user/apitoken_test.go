package user

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAPITokenUseCase_List_Success(t *testing.T) {
	h := NewUserTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID := uuid.New()
	tokens := []*entity.APIToken{h.NewAPIToken(userID, "hash", "desc", nil)}

	deps.apiTokenRepo.EXPECT().GetByUserID(mock.Anything, userID).Return(tokens, nil)

	uc := h.CreateAPITokenUseCase()
	list, err := uc.List(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, tokens[0].ID, list[0].ID)
}

func TestAPITokenUseCase_List_Error(t *testing.T) {
	h := NewUserTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID := uuid.New()

	deps.apiTokenRepo.EXPECT().GetByUserID(mock.Anything, userID).Return(nil, assert.AnError)

	uc := h.CreateAPITokenUseCase()
	list, err := uc.List(ctx, userID)

	assert.Error(t, err)
	assert.Nil(t, list)
}

func TestAPITokenUseCase_Create_Success(t *testing.T) {
	h := NewUserTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID := uuid.New()
	desc := "token"
	var exp *time.Time

	deps.apiTokenRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, token *entity.APIToken) {
		assert.Equal(t, userID, token.UserID)
		assert.Equal(t, desc, token.Description)
		assert.Equal(t, exp, token.ExpiresAt)
		assert.NotEmpty(t, token.TokenHash)
	})

	uc := h.CreateAPITokenUseCase()
	plaintext, token, err := uc.Create(ctx, userID, desc, exp)

	assert.NoError(t, err)
	assert.NotEmpty(t, plaintext)
	assert.NotNil(t, token)
}

func TestAPITokenUseCase_Create_Error(t *testing.T) {
	h := NewUserTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID := uuid.New()

	deps.apiTokenRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := h.CreateAPITokenUseCase()
	plaintext, token, err := uc.Create(ctx, userID, "desc", nil)

	assert.Error(t, err)
	assert.Empty(t, plaintext)
	assert.Nil(t, token)
}

func TestAPITokenUseCase_Delete_Success(t *testing.T) {
	h := NewUserTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id, userID := uuid.New(), uuid.New()

	deps.apiTokenRepo.EXPECT().Delete(mock.Anything, id, userID).Return(nil)

	uc := h.CreateAPITokenUseCase()
	err := uc.Delete(ctx, id, userID)

	assert.NoError(t, err)
}

func TestAPITokenUseCase_Delete_Error(t *testing.T) {
	h := NewUserTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id, userID := uuid.New(), uuid.New()

	deps.apiTokenRepo.EXPECT().Delete(mock.Anything, id, userID).Return(assert.AnError)

	uc := h.CreateAPITokenUseCase()
	err := uc.Delete(ctx, id, userID)

	assert.Error(t, err)
}

func TestAPITokenUseCase_GetByTokenHash_Success(t *testing.T) {
	h := NewUserTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID := uuid.New()
	expected := h.NewAPIToken(userID, "hash1", "d", nil)

	deps.apiTokenRepo.EXPECT().GetByTokenHash(mock.Anything, "hash1").Return(expected, nil)

	uc := h.CreateAPITokenUseCase()
	token, err := uc.GetByTokenHash(ctx, "hash1")

	assert.NoError(t, err)
	assert.Equal(t, expected, token)
}

func TestAPITokenUseCase_GetByTokenHash_Error(t *testing.T) {
	h := NewUserTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()

	deps.apiTokenRepo.EXPECT().GetByTokenHash(mock.Anything, "hash1").Return(nil, assert.AnError)

	uc := h.CreateAPITokenUseCase()
	token, err := uc.GetByTokenHash(ctx, "hash1")

	assert.Error(t, err)
	assert.Nil(t, token)
}

func TestAPITokenUseCase_UpdateLastUsedAt_Success(t *testing.T) {
	h := NewUserTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.apiTokenRepo.EXPECT().UpdateLastUsedAt(mock.Anything, id, mock.Anything).Return(nil)

	uc := h.CreateAPITokenUseCase()
	err := uc.UpdateLastUsedAt(ctx, id)

	assert.NoError(t, err)
}

func TestAPITokenUseCase_UpdateLastUsedAt_Error(t *testing.T) {
	h := NewUserTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.apiTokenRepo.EXPECT().UpdateLastUsedAt(mock.Anything, id, mock.Anything).Return(assert.AnError)

	uc := h.CreateAPITokenUseCase()
	err := uc.UpdateLastUsedAt(ctx, id)

	assert.Error(t, err)
}

func TestAPITokenUseCase_ValidateToken_Success(t *testing.T) {
	h := NewUserTestHelper(t)
	uc := h.CreateAPITokenUseCase()
	token := h.NewAPIToken(uuid.New(), "h", "d", nil)

	ok := uc.ValidateToken(token)

	assert.True(t, ok)
}

func TestAPITokenUseCase_ValidateToken_Error(t *testing.T) {
	h := NewUserTestHelper(t)
	uc := h.CreateAPITokenUseCase()

	assert.False(t, uc.ValidateToken(nil))

	exp := time.Now().Add(-time.Hour)
	token := h.NewAPIToken(uuid.New(), "h", "d", &exp)
	assert.False(t, uc.ValidateToken(token))
}
