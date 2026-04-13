package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestVerificationTokenRepo_CreateAndGet(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "vt_user")

	token := &domain.VerificationToken{
		UserID:    user.ID,
		Token:     "test_token_123",
		Type:      domain.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	err := f.VerificationTokenRepo.Create(ctx, token)
	require.NoError(t, err)

	fetched, err := f.VerificationTokenRepo.GetByToken(ctx, token.Token)
	require.NoError(t, err)
	assert.Equal(t, token.UserID, fetched.UserID)
	assert.Equal(t, token.Token, fetched.Token)
	assert.Equal(t, token.Type, fetched.Type)
}

func TestVerificationTokenRepo_GetByToken_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	repo := f.VerificationTokenRepo
	ctx := context.Background()

	_, err := repo.GetByToken(ctx, "non_existent_token")
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTokenNotFound)
}

func TestVerificationTokenRepo_DeleteByUserAndType(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	repo := f.VerificationTokenRepo
	ctx := context.Background()

	user := f.CreateUser(t, "vt_del_user")

	token := &domain.VerificationToken{
		UserID:    user.ID,
		Token:     "token_to_delete",
		Type:      domain.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.Create(ctx, token))

	err := repo.DeleteByUserAndType(ctx, user.ID, domain.TokenTypeEmailVerification)
	assert.NoError(t, err)

	_, err = repo.GetByToken(ctx, token.Token)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTokenNotFound)
}

func TestVerificationTokenRepo_MarkUsed(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	repo := f.VerificationTokenRepo
	ctx := context.Background()

	user := f.CreateUser(t, "vt_used_user")

	token := &domain.VerificationToken{
		UserID:    user.ID,
		Token:     "token_mark_used",
		Type:      domain.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.Create(ctx, token))

	fetched, err := repo.GetByToken(ctx, token.Token)
	require.NoError(t, err)
	assert.NotEmpty(t, fetched.ID)

	err = repo.MarkUsed(ctx, fetched.ID)
	assert.NoError(t, err)

	fetchedUsed, err := repo.GetByToken(ctx, token.Token)
	require.NoError(t, err)
	assert.NotNil(t, fetchedUsed.UsedAt)
}
