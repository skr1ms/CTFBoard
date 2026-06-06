package avatar

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	avatarMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/avatar/mock"
)

func makePNG(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			img.Set(x, y, color.RGBA{R: 128, G: 64, B: 32, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

type fakeCache struct {
	mu    sync.Mutex
	store map[string][]byte
}

func (f *fakeCache) Get(_ context.Context, key string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if v, ok := f.store[key]; ok {
		return v, nil
	}

	return nil, errors.New("cache miss")
}

func (f *fakeCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.store == nil {
		f.store = make(map[string][]byte)
	}

	f.store[key] = value

	return nil
}

func (f *fakeCache) Del(_ context.Context, keys ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, k := range keys {
		delete(f.store, k)
	}

	return nil
}

type avatarTestDeps struct {
	userRepo *avatarMock.MockUserRepository
	teamRepo *avatarMock.MockTeamRepository
	storage  *avatarMock.MockAvatarStorage
	cache    *fakeCache
	logger   logkit.Logger
}

func newAvatarTestDeps(t *testing.T) *avatarTestDeps {
	t.Helper()

	l, err := logkit.New(logkit.WithLevel(logkit.ErrorLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, err)

	return &avatarTestDeps{
		userRepo: avatarMock.NewMockUserRepository(t),
		teamRepo: avatarMock.NewMockTeamRepository(t),
		storage:  avatarMock.NewMockAvatarStorage(t),
		cache:    &fakeCache{},
		logger:   l,
	}
}

func (d *avatarTestDeps) newUseCase() *AvatarUseCase {
	return NewAvatarUseCase(AvatarDeps{
		UserRepo: d.userRepo,
		TeamRepo: d.teamRepo,
		Storage:  d.storage,
		Cache:    d.cache,
		Config:   domain.GetDefaultAvatarConfig(),
		Logger:   d.logger,
	})
}

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
