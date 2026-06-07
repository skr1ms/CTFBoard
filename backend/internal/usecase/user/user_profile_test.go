package user

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestUserUseCase_GetByID_Success(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	expectedUser := &domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(expectedUser, nil)

	uc := d.createUseCase()

	user, err := uc.GetByID(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Username, user.Username)
}

func TestUserUseCase_GetByID_Error(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	expectedError := assert.AnError

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, expectedError)

	uc := d.createUseCase()

	user, err := uc.GetByID(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestUserUseCase_GetProfile_Success(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	user := &domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	solves := []*domain.Solve{newTestSolve(userID, uuid.New())}

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)
	d.solveRepo.EXPECT().GetByUserID(mock.Anything, userID).Return(solves, nil)

	uc := d.createUseCase()

	profile, err := uc.GetProfile(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, user.Username, profile.User.Username)
	assert.Empty(t, profile.User.PasswordHash)
	assert.Len(t, profile.Solves, 1)
}

func TestUserUseCase_GetProfile_GetByIDError(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	expectedError := assert.AnError

	d.solveRepo.EXPECT().GetByUserID(mock.Anything, userID).Return([]*domain.Solve{}, nil)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Run(func(_ context.Context, _ uuid.UUID) {
		time.Sleep(1 * time.Millisecond)
	}).Return(nil, expectedError)

	uc := d.createUseCase()

	profile, err := uc.GetProfile(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, profile)
}

func TestUserUseCase_GetProfile_GetByUserIDError(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	user := &domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}
	expectedError := assert.AnError

	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)
	d.solveRepo.EXPECT().GetByUserID(mock.Anything, userID).Run(func(_ context.Context, _ uuid.UUID) {
		time.Sleep(1 * time.Millisecond)
	}).Return(nil, expectedError)

	uc := d.createUseCase()

	profile, err := uc.GetProfile(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, profile)
}
