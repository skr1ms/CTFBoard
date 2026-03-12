package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func TestOAuthRepo_Create_Success(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "oauth_create")
	oauthRepo := persistent.NewOAuthRepo(testPool.Pool)
	ctx := context.Background()

	acc := &entity.OAuthAccount{
		UserID:         user.ID,
		Provider:       "github",
		ProviderUserID: "gh-create-" + uuid.New().String(),
		AccessToken:    "access-token",
	}
	err := oauthRepo.Create(ctx, acc)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, acc.ID)
}

func TestOAuthRepo_Create_Error_Duplicate(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "oauth_dup")
	oauthRepo := persistent.NewOAuthRepo(testPool.Pool)
	ctx := context.Background()

	providerUserID := "gh-dup-" + uuid.New().String()
	acc := &entity.OAuthAccount{
		UserID:         user.ID,
		Provider:       "github",
		ProviderUserID: providerUserID,
		AccessToken:    "token",
	}
	require.NoError(t, oauthRepo.Create(ctx, acc))

	acc2 := &entity.OAuthAccount{
		UserID:         user.ID,
		Provider:       "github",
		ProviderUserID: providerUserID,
		AccessToken:    "token2",
	}
	err := oauthRepo.Create(ctx, acc2)
	assert.Error(t, err)
}

func TestOAuthRepo_Upsert_Success(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "oauth_upsert")
	oauthRepo := persistent.NewOAuthRepo(testPool.Pool)
	ctx := context.Background()

	providerUserID := "gh-upsert-" + uuid.New().String()
	acc := &entity.OAuthAccount{
		UserID:         user.ID,
		Provider:       "github",
		ProviderUserID: providerUserID,
		AccessToken:    "original-token",
	}
	require.NoError(t, oauthRepo.Upsert(ctx, acc))
	assert.NotEqual(t, uuid.Nil, acc.ID)

	acc.AccessToken = "updated-token"
	err := oauthRepo.Upsert(ctx, acc)
	require.NoError(t, err)
}

func TestOAuthRepo_GetByProvider_Success(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "oauth_getprov")
	oauthRepo := persistent.NewOAuthRepo(testPool.Pool)
	ctx := context.Background()

	providerUserID := "gh-get-" + uuid.New().String()
	acc := &entity.OAuthAccount{
		UserID:         user.ID,
		Provider:       "github",
		ProviderUserID: providerUserID,
		AccessToken:    "token",
	}
	require.NoError(t, oauthRepo.Create(ctx, acc))

	got, err := oauthRepo.GetByProvider(ctx, "github", providerUserID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.UserID)
	assert.Equal(t, "github", got.Provider)
	assert.Equal(t, providerUserID, got.ProviderUserID)
}

func TestOAuthRepo_GetByProvider_NotFound(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	oauthRepo := persistent.NewOAuthRepo(testPool.Pool)
	ctx := context.Background()

	_, err := oauthRepo.GetByProvider(ctx, "github", "nonexistent-id")
	assert.ErrorIs(t, err, httperr.ErrOAuthAccountNotFound)
}

func TestOAuthRepo_GetByUserID_Success(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "oauth_getuserid")
	oauthRepo := persistent.NewOAuthRepo(testPool.Pool)
	ctx := context.Background()

	acc := &entity.OAuthAccount{
		UserID:         user.ID,
		Provider:       "github",
		ProviderUserID: "gh-uid-" + uuid.New().String(),
		AccessToken:    "token",
	}
	require.NoError(t, oauthRepo.Create(ctx, acc))

	accs, err := oauthRepo.GetByUserID(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, accs, 1)
	assert.Equal(t, user.ID, accs[0].UserID)
}

func TestOAuthRepo_GetByUserID_Empty(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	oauthRepo := persistent.NewOAuthRepo(testPool.Pool)
	ctx := context.Background()

	accs, err := oauthRepo.GetByUserID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, accs)
}
