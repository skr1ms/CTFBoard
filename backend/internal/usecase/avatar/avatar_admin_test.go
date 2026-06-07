package avatar

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestAvatarUseCase_AdminUploadUserAvatar_Success(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	userID := uuid.New()
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, AvatarURL: nil}, nil)
	d.storage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int64"), "image/webp").Return(nil).Times(2)
	d.userRepo.On("UpdateAvatarURL", mock.Anything, userID, mock.AnythingOfType("string")).Return(nil)
	d.storage.On("GetPresignedURL", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return("https://cdn/full", nil).Once()
	d.storage.On("GetPresignedURL", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return("https://cdn/thumb", nil).Once()

	pngData := makePNG(512)
	fullURL, thumbURL, err := uc.AdminUploadUserAvatar(context.Background(), userID, bytes.NewReader(pngData), "avatar.png", int64(len(pngData)))

	require.NoError(t, err)
	assert.NotEmpty(t, fullURL)
	assert.NotEmpty(t, thumbURL)
}

func TestAvatarUseCase_AdminDeleteUserAvatar_Success(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	userID := uuid.New()
	oldPath := "users/" + userID.String() + "/xyz_full.webp"
	d.userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, AvatarURL: new(oldPath)}, nil)
	d.userRepo.On("ClearAvatarURL", mock.Anything, userID).Return(nil)
	d.storage.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil).Times(2)

	err := uc.AdminDeleteUserAvatar(context.Background(), userID)

	require.NoError(t, err)
	uc.Wait()
}

func TestAvatarUseCase_AdminUploadTeamAvatar_Success(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	teamID := uuid.New()
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, AvatarURL: nil}, nil)
	d.storage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int64"), "image/webp").Return(nil).Times(2)
	d.teamRepo.On("UpdateAvatarURL", mock.Anything, teamID, mock.AnythingOfType("string")).Return(nil)
	d.storage.On("GetPresignedURL", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return("https://cdn/full", nil).Once()
	d.storage.On("GetPresignedURL", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return("https://cdn/thumb", nil).Once()

	pngData := makePNG(512)
	fullURL, thumbURL, err := uc.AdminUploadTeamAvatar(context.Background(), teamID, bytes.NewReader(pngData), "team.png", int64(len(pngData)))

	require.NoError(t, err)
	assert.NotEmpty(t, fullURL)
	assert.NotEmpty(t, thumbURL)
}

func TestAvatarUseCase_AdminDeleteTeamAvatar_Success(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	teamID := uuid.New()
	oldPath := "teams/" + teamID.String() + "/xyz_full.webp"
	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(&domain.Team{ID: teamID, AvatarURL: new(oldPath)}, nil)
	d.teamRepo.On("ClearAvatarURL", mock.Anything, teamID).Return(nil)
	d.storage.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil).Times(2)

	err := uc.AdminDeleteTeamAvatar(context.Background(), teamID)

	require.NoError(t, err)
	uc.Wait()
}
