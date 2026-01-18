package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CreateAndGet Tests

func TestVerificationTokenRepo_CreateAndGet(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	ctx := context.Background()

	user := f.CreateUser(t, "vt_user")

	token := &entity.VerificationToken{
		UserId:    user.Id,
		Token:     "test_token_123",
		Type:      entity.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	err := f.VerificationTokenRepo.Create(ctx, token)
	require.NoError(t, err)

	fetched, err := f.VerificationTokenRepo.GetByToken(ctx, token.Token)
	require.NoError(t, err)
	assert.Equal(t, token.UserId, fetched.UserId)
	assert.Equal(t, token.Token, fetched.Token)
	assert.Equal(t, token.Type, fetched.Type)
}

// GetByToken Tests

func TestVerificationTokenRepo_GetByToken_NotFound(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	repo := f.VerificationTokenRepo
	ctx := context.Background()

	_, err := repo.GetByToken(ctx, "non_existent_token")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTokenNotFound))
}

// DeleteByUserAndType Tests

func TestVerificationTokenRepo_DeleteByUserAndType(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	repo := f.VerificationTokenRepo
	ctx := context.Background()

	user := f.CreateUser(t, "vt_del_user")

	token := &entity.VerificationToken{
		UserId:    user.Id,
		Token:     "token_to_delete",
		Type:      entity.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.Create(ctx, token))

	err := repo.DeleteByUserAndType(ctx, user.Id, entity.TokenTypeEmailVerification)
	assert.NoError(t, err)

	_, err = repo.GetByToken(ctx, token.Token)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTokenNotFound))
}

// MarkUsed Tests

func TestVerificationTokenRepo_MarkUsed(t *testing.T) {
	testDB := SetupTestDB(t)
	f := NewTestFixture(testDB.DB)
	repo := f.VerificationTokenRepo
	ctx := context.Background()

	user := f.CreateUser(t, "vt_used_user")

	token := &entity.VerificationToken{
		UserId:    user.Id,
		Token:     "token_mark_used",
		Type:      entity.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.Create(ctx, token))

	fetched, err := repo.GetByToken(ctx, token.Token)
	require.NoError(t, err)
	assert.NotEmpty(t, fetched.Id)

	err = repo.MarkUsed(ctx, fetched.Id)
	assert.NoError(t, err)

	fetchedUsed, err := repo.GetByToken(ctx, token.Token)
	require.NoError(t, err)
	assert.NotNil(t, fetchedUsed.UsedAt)
}
