package user

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mocks"
)

type apiTokenTestDeps struct {
	apiTokenRepo *mocks.MockAPITokenRepository
}

func newAPITokenTestDeps(t *testing.T) *apiTokenTestDeps {
	t.Helper()
	return &apiTokenTestDeps{apiTokenRepo: mocks.NewMockAPITokenRepository(t)}
}

func (d *apiTokenTestDeps) createUseCase() *APITokenUseCase {
	return NewAPITokenUseCase(APITokenDeps{Repo: d.apiTokenRepo})
}

func newTestAPIToken(userID uuid.UUID, tokenHash, description string, expiresAt *time.Time) *entity.APIToken {
	return &entity.APIToken{
		ID: uuid.New(), UserID: userID, TokenHash: tokenHash, Description: description,
		ExpiresAt: expiresAt, CreatedAt: time.Now(),
	}
}

func TestAPITokenUseCase_List_Success(t *testing.T) {
	t.Parallel()
	d := newAPITokenTestDeps(t)
	ctx := context.Background()
	userID := uuid.New()
	tokens := []*entity.APIToken{newTestAPIToken(userID, "hash", "desc", nil)}

	d.apiTokenRepo.EXPECT().GetByUserID(mock.Anything, userID).Return(tokens, nil)

	uc := d.createUseCase()
	list, err := uc.List(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, tokens[0].ID, list[0].ID)
}

func TestAPITokenUseCase_List_Error(t *testing.T) {
	t.Parallel()
	d := newAPITokenTestDeps(t)
	ctx := context.Background()
	userID := uuid.New()

	d.apiTokenRepo.EXPECT().GetByUserID(mock.Anything, userID).Return(nil, assert.AnError)

	uc := d.createUseCase()
	list, err := uc.List(ctx, userID)

	assert.Error(t, err)
	assert.Nil(t, list)
}

func TestAPITokenUseCase_Create_Success(t *testing.T) {
	t.Parallel()
	d := newAPITokenTestDeps(t)
	ctx := context.Background()
	userID := uuid.New()
	desc := "token"
	var exp *time.Time

	d.apiTokenRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, token *entity.APIToken) {
		assert.Equal(t, userID, token.UserID)
		assert.Equal(t, desc, token.Description)
		assert.Equal(t, exp, token.ExpiresAt)
		assert.NotEmpty(t, token.TokenHash)
	})

	uc := d.createUseCase()
	plaintext, token, err := uc.Create(ctx, userID, desc, exp)

	assert.NoError(t, err)
	assert.NotEmpty(t, plaintext)
	assert.NotNil(t, token)
}

func TestAPITokenUseCase_Create_Error(t *testing.T) {
	t.Parallel()
	d := newAPITokenTestDeps(t)
	ctx := context.Background()
	userID := uuid.New()

	d.apiTokenRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := d.createUseCase()
	plaintext, token, err := uc.Create(ctx, userID, "desc", nil)

	assert.Error(t, err)
	assert.Empty(t, plaintext)
	assert.Nil(t, token)
}

func TestAPITokenUseCase_Delete_Success(t *testing.T) {
	t.Parallel()
	d := newAPITokenTestDeps(t)
	ctx := context.Background()
	id, userID := uuid.New(), uuid.New()

	d.apiTokenRepo.EXPECT().Delete(mock.Anything, id, userID).Return(nil)

	uc := d.createUseCase()
	err := uc.Delete(ctx, id, userID)

	assert.NoError(t, err)
}

func TestAPITokenUseCase_Delete_Error(t *testing.T) {
	t.Parallel()
	d := newAPITokenTestDeps(t)
	ctx := context.Background()
	id, userID := uuid.New(), uuid.New()

	d.apiTokenRepo.EXPECT().Delete(mock.Anything, id, userID).Return(assert.AnError)

	uc := d.createUseCase()
	err := uc.Delete(ctx, id, userID)

	assert.Error(t, err)
}

func TestAPITokenUseCase_GetByTokenHash_Success(t *testing.T) {
	t.Parallel()
	d := newAPITokenTestDeps(t)
	ctx := context.Background()
	userID := uuid.New()
	expected := newTestAPIToken(userID, "hash1", "d", nil)

	d.apiTokenRepo.EXPECT().GetByTokenHash(mock.Anything, "hash1").Return(expected, nil)

	uc := d.createUseCase()
	token, err := uc.GetByTokenHash(ctx, "hash1")

	assert.NoError(t, err)
	assert.Equal(t, expected, token)
}

func TestAPITokenUseCase_GetByTokenHash_Error(t *testing.T) {
	t.Parallel()
	d := newAPITokenTestDeps(t)
	ctx := context.Background()

	d.apiTokenRepo.EXPECT().GetByTokenHash(mock.Anything, "hash1").Return(nil, assert.AnError)

	uc := d.createUseCase()
	token, err := uc.GetByTokenHash(ctx, "hash1")

	assert.Error(t, err)
	assert.Nil(t, token)
}

func TestAPITokenUseCase_UpdateLastUsedAt_Success(t *testing.T) {
	t.Parallel()
	d := newAPITokenTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.apiTokenRepo.EXPECT().UpdateLastUsedAt(mock.Anything, id, mock.Anything).Return(nil)

	uc := d.createUseCase()
	err := uc.UpdateLastUsedAt(ctx, id)

	assert.NoError(t, err)
}

func TestAPITokenUseCase_UpdateLastUsedAt_Error(t *testing.T) {
	t.Parallel()
	d := newAPITokenTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.apiTokenRepo.EXPECT().UpdateLastUsedAt(mock.Anything, id, mock.Anything).Return(assert.AnError)

	uc := d.createUseCase()
	err := uc.UpdateLastUsedAt(ctx, id)

	assert.Error(t, err)
}

func TestAPITokenUseCase_ValidateToken_Success(t *testing.T) {
	t.Parallel()
	d := newAPITokenTestDeps(t)
	uc := d.createUseCase()
	token := newTestAPIToken(uuid.New(), "h", "d", nil)

	ok := uc.ValidateToken(token)

	assert.True(t, ok)
}

func TestAPITokenUseCase_ValidateToken_Error(t *testing.T) {
	t.Parallel()
	d := newAPITokenTestDeps(t)
	uc := d.createUseCase()

	assert.False(t, uc.ValidateToken(nil))

	exp := time.Now().Add(-time.Hour)
	token := newTestAPIToken(uuid.New(), "h", "d", &exp)
	assert.False(t, uc.ValidateToken(token))
}
