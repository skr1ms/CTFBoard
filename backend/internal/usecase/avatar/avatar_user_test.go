package avatar

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestAvatarUseCase_UploadUserAvatar_Success(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	userID := uuid.New()
	user := &domain.User{ID: userID, IsBanned: false, AvatarURL: nil}

	d.userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	d.storage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int64"), "image/webp").Return(nil).Times(2)
	d.userRepo.On("UpdateAvatarURL", mock.Anything, userID, mock.AnythingOfType("string")).Return(nil)
	d.storage.On("GetPresignedURL", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return("https://cdn/full", nil).Once()
	d.storage.On("GetPresignedURL", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return("https://cdn/thumb", nil).Once()

	pngData := makePNG(512)
	fullURL, thumbURL, err := uc.UploadUserAvatar(context.Background(), userID, bytes.NewReader(pngData), "avatar.png", int64(len(pngData)))

	require.NoError(t, err)
	assert.Equal(t, "https://cdn/full", fullURL)
	assert.Equal(t, "https://cdn/thumb", thumbURL)
}

func TestAvatarUseCase_UploadUserAvatar_UserBanned(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	userID := uuid.New()
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, IsBanned: true}, nil)

	pngData := makePNG(512)
	_, _, err := uc.UploadUserAvatar(context.Background(), userID, bytes.NewReader(pngData), "avatar.png", int64(len(pngData)))

	require.ErrorIs(t, err, apperr.ErrUserBanned)
}

func TestAvatarUseCase_UploadUserAvatar_UserNotFound(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	userID := uuid.New()
	d.userRepo.On("GetByID", mock.Anything, userID).Return(nil, apperr.ErrUserNotFound)

	pngData := makePNG(512)
	_, _, err := uc.UploadUserAvatar(context.Background(), userID, bytes.NewReader(pngData), "avatar.png", int64(len(pngData)))

	require.ErrorIs(t, err, apperr.ErrUserNotFound)
}

// When thumb upload fails, the already-uploaded full image must be deleted (rollback).
func TestAvatarUseCase_UploadUserAvatar_ThumbUploadError_Rollback(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	userID := uuid.New()
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, IsBanned: false}, nil)

	storageErr := errors.New("storage error")

	d.storage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int64"), "image/webp").Return(nil).Once()
	d.storage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int64"), "image/webp").Return(storageErr).Once()
	// rollback: full image must be deleted
	d.storage.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil).Once()

	pngData := makePNG(512)
	_, _, err := uc.UploadUserAvatar(context.Background(), userID, bytes.NewReader(pngData), "avatar.png", int64(len(pngData)))

	require.Error(t, err)
}

// When UpdateAvatarURL fails, both uploaded files must be cleaned up.
func TestAvatarUseCase_UploadUserAvatar_UpdateURLError_Rollback(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	userID := uuid.New()
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, IsBanned: false}, nil)
	d.storage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int64"), "image/webp").Return(nil).Times(2)
	d.userRepo.On("UpdateAvatarURL", mock.Anything, userID, mock.AnythingOfType("string")).Return(errors.New("db error"))
	// rollback: both files deleted
	d.storage.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil).Times(2)

	pngData := makePNG(512)
	_, _, err := uc.UploadUserAvatar(context.Background(), userID, bytes.NewReader(pngData), "avatar.png", int64(len(pngData)))

	require.Error(t, err)
}

// When user already has an avatar, old avatar must be async-deleted after successful upload.
func TestAvatarUseCase_UploadUserAvatar_DeletesOldAvatar(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	userID := uuid.New()
	oldPath := "users/" + userID.String() + "/abc123_full.webp"
	user := &domain.User{ID: userID, IsBanned: false, AvatarURL: new(oldPath)}

	d.userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	d.storage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int64"), "image/webp").Return(nil).Times(2)
	d.userRepo.On("UpdateAvatarURL", mock.Anything, userID, mock.AnythingOfType("string")).Return(nil)
	d.storage.On("GetPresignedURL", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return("https://cdn/full", nil).Once()
	d.storage.On("GetPresignedURL", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return("https://cdn/thumb", nil).Once()
	// old avatar (full + thumb) deleted asynchronously
	d.storage.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil).Times(2)

	pngData := makePNG(512)
	fullURL, thumbURL, err := uc.UploadUserAvatar(context.Background(), userID, bytes.NewReader(pngData), "avatar.png", int64(len(pngData)))

	require.NoError(t, err)
	assert.NotEmpty(t, fullURL)
	assert.NotEmpty(t, thumbURL)

	uc.Wait()
}

func TestAvatarUseCase_DeleteUserAvatar_Success(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	userID := uuid.New()
	oldPath := "users/" + userID.String() + "/abc_full.webp"
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, AvatarURL: new(oldPath)}, nil)
	d.userRepo.On("ClearAvatarURL", mock.Anything, userID).Return(nil)
	d.storage.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil).Times(2)

	err := uc.DeleteUserAvatar(context.Background(), userID)

	require.NoError(t, err)
	uc.Wait()
}

func TestAvatarUseCase_DeleteUserAvatar_NoAvatar(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	userID := uuid.New()
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, AvatarURL: nil}, nil)
	d.userRepo.On("ClearAvatarURL", mock.Anything, userID).Return(nil)

	err := uc.DeleteUserAvatar(context.Background(), userID)

	require.NoError(t, err)
}

func TestAvatarUseCase_GetUserAvatarURL_CacheMiss_FetchesAndCaches(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	userID := uuid.New()
	storagePath := "users/" + userID.String() + "/abc_full.webp"
	thumbPath := storagePath[:len(storagePath)-10] + "_thumb.webp"

	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, AvatarURL: new(storagePath)}, nil)
	d.storage.On("GetPresignedURL", mock.Anything, storagePath, mock.Anything).Return("https://cdn/full", nil).Once()
	d.storage.On("GetPresignedURL", mock.Anything, thumbPath, mock.Anything).Return("https://cdn/thumb", nil).Once()

	fullURL, thumbURL, err := uc.GetUserAvatarURL(context.Background(), userID)

	require.NoError(t, err)
	require.NotNil(t, fullURL)
	require.NotNil(t, thumbURL)
	assert.Equal(t, "https://cdn/full", *fullURL)
	assert.Equal(t, "https://cdn/thumb", *thumbURL)
}

func TestAvatarUseCase_GetUserAvatarURL_NoAvatar_ReturnsNil(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	userID := uuid.New()
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, AvatarURL: nil}, nil)

	fullURL, thumbURL, err := uc.GetUserAvatarURL(context.Background(), userID)

	require.NoError(t, err)
	assert.Nil(t, fullURL)
	assert.Nil(t, thumbURL)
}

func TestAvatarUseCase_GetUserAvatarURL_CacheHit(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	userID := uuid.New()
	storagePath := "users/" + userID.String() + "/abc_full.webp"
	thumbPath := storagePath[:len(storagePath)-10] + "_thumb.webp"

	// first call: cache miss -> fetches from storage and populates cache
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, AvatarURL: new(storagePath)}, nil).Times(2)
	d.storage.On("GetPresignedURL", mock.Anything, storagePath, mock.Anything).Return("https://cdn/full", nil).Once()
	d.storage.On("GetPresignedURL", mock.Anything, thumbPath, mock.Anything).Return("https://cdn/thumb", nil).Once()

	_, _, err := uc.GetUserAvatarURL(context.Background(), userID)
	require.NoError(t, err)

	// second call: cache hit - storage.GetPresignedURL must NOT be called again
	fullURL, thumbURL, err := uc.GetUserAvatarURL(context.Background(), userID)

	require.NoError(t, err)
	require.NotNil(t, fullURL)
	require.NotNil(t, thumbURL)
}
