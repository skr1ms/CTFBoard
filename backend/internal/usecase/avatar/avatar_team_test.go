package avatar

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestAvatarUseCase_UploadTeamAvatar_Success(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	teamID := uuid.New()
	captainID := uuid.New()
	team := &domain.Team{ID: teamID, CaptainID: captainID, IsBanned: false, AvatarURL: nil}

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.storage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int64"), "image/webp").Return(nil).Times(2)
	d.teamRepo.On("UpdateAvatarURL", mock.Anything, teamID, mock.AnythingOfType("string")).Return(nil)
	d.storage.On("GetPresignedURL", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return("https://cdn/full", nil).Once()
	d.storage.On("GetPresignedURL", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return("https://cdn/thumb", nil).Once()

	pngData := makePNG(512)
	fullURL, thumbURL, err := uc.UploadTeamAvatar(context.Background(), teamID, captainID, bytes.NewReader(pngData), "team.png", int64(len(pngData)))

	require.NoError(t, err)
	assert.Equal(t, "https://cdn/full", fullURL)
	assert.Equal(t, "https://cdn/thumb", thumbURL)
}

func TestAvatarUseCase_UploadTeamAvatar_NotCaptain(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	teamID := uuid.New()
	captainID := uuid.New()
	callerID := uuid.New()
	team := &domain.Team{ID: teamID, CaptainID: captainID, IsBanned: false}

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)

	pngData := makePNG(512)
	_, _, err := uc.UploadTeamAvatar(context.Background(), teamID, callerID, bytes.NewReader(pngData), "team.png", int64(len(pngData)))

	require.ErrorIs(t, err, apperr.ErrNotTeamCaptain)
}

func TestAvatarUseCase_UploadTeamAvatar_TeamBanned(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	teamID := uuid.New()
	captainID := uuid.New()
	team := &domain.Team{ID: teamID, CaptainID: captainID, IsBanned: true}

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)

	pngData := makePNG(512)
	_, _, err := uc.UploadTeamAvatar(context.Background(), teamID, captainID, bytes.NewReader(pngData), "team.png", int64(len(pngData)))

	require.ErrorIs(t, err, apperr.ErrTeamBanned)
}

func TestAvatarUseCase_DeleteTeamAvatar_Success(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	teamID := uuid.New()
	captainID := uuid.New()
	oldPath := "teams/" + teamID.String() + "/abc_full.webp"
	team := &domain.Team{ID: teamID, CaptainID: captainID, AvatarURL: new(oldPath)}

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.teamRepo.On("ClearAvatarURL", mock.Anything, teamID).Return(nil)
	d.storage.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil).Times(2)

	err := uc.DeleteTeamAvatar(context.Background(), teamID, captainID)

	require.NoError(t, err)
	uc.Wait()
}

func TestAvatarUseCase_DeleteTeamAvatar_NotCaptain(t *testing.T) {
	t.Parallel()
	d := newAvatarTestDeps(t)
	uc := d.newUseCase()

	teamID := uuid.New()
	captainID := uuid.New()
	callerID := uuid.New()
	team := &domain.Team{ID: teamID, CaptainID: captainID}

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)

	err := uc.DeleteTeamAvatar(context.Background(), teamID, callerID)

	require.ErrorIs(t, err, apperr.ErrNotTeamCaptain)
}
