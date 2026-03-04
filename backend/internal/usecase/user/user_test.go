package user

import (
	"context"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type registerTestCase struct {
	name          string
	username      string
	email         string
	password      string
	setupMocks    func(*mocks.MockUserRepository, *mocks.MockTransactionManager)
	expectedError bool
}

func registerTestCases() []registerTestCase {
	return []registerTestCase{
		{
			name: "successful registration", username: "testuser", email: "test@example.com", password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, tm *mocks.MockTransactionManager) {
				tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).Once()
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, httperr.ErrUserNotFound).Once()
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(nil, httperr.ErrUserNotFound).Once()
				userRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, u *entity.User) {
					u.ID = uuid.New()
				}).Once()
			},
			expectedError: false,
		},
		{
			name: "username already exists", username: "existinguser", email: "test@example.com", password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, tm *mocks.MockTransactionManager) {
				tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).Once()
				userRepo.EXPECT().GetByUsername(mock.Anything, "existinguser").Return(&entity.User{}, nil).Once()
			},
			expectedError: true,
		},
		{
			name: "email already exists", username: "testuser", email: "existing@example.com", password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, tm *mocks.MockTransactionManager) {
				tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).Once()
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, httperr.ErrUserNotFound).Once()
				userRepo.EXPECT().GetByEmail(mock.Anything, "existing@example.com").Return(&entity.User{}, nil).Once()
			},
			expectedError: true,
		},
		{
			name: "GetByUsername returns unexpected error", username: "testuser", email: "test@example.com", password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, tm *mocks.MockTransactionManager) {
				tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).Once()
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, assert.AnError).Once()
			},
			expectedError: true,
		},
		{
			name: "GetByEmail returns unexpected error", username: "testuser", email: "test@example.com", password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, tm *mocks.MockTransactionManager) {
				tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).Once()
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, httperr.ErrUserNotFound).Once()
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(nil, assert.AnError).Once()
			},
			expectedError: true,
		},
		{
			name: "Transaction returns error", username: "testuser", email: "test@example.com", password: "password123",
			setupMocks: func(_ *mocks.MockUserRepository, tm *mocks.MockTransactionManager) {
				tm.EXPECT().Run(mock.Anything, mock.Anything).Return(assert.AnError).Once()
			},
			expectedError: true,
		},
	}
}

func runRegisterTest(t *testing.T, tt registerTestCase) {
	t.Helper()
	h := NewUserTestHelper(t)
	tt.setupMocks(h.Deps().userRepo, h.Deps().tm)
	uc := h.CreateUseCase()
	user, err := uc.Register(context.Background(), tt.username, tt.email, tt.password, nil)
	if tt.expectedError {
		assert.Error(t, err)
		assert.Nil(t, user)
	} else {
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, tt.username, user.Username)
		assert.Equal(t, tt.email, user.Email)
	}
}

func TestUserUseCase_Register(t *testing.T) {
	t.Parallel()
	for _, tt := range registerTestCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runRegisterTest(t, tt)
		})
	}
}

func TestUserUseCase_Register_RegistrationClosed(t *testing.T) {
	t.Parallel()
	h := NewUserTestHelper(t)
	settingsRepo := mocks.NewMockSettingsRepository(t)
	settingsRepo.EXPECT().Get(mock.Anything).Return(&entity.Settings{RegistrationOpen: false}, nil).Once()
	uc := NewUserUseCase(UserDeps{
		UserRepo: h.Deps().userRepo, TeamRepo: h.Deps().teamRepo, SolveRepo: h.Deps().solveRepo,
		TM: h.Deps().tm, JWTService: h.Deps().jwtService, FieldValidator: nil, FieldValueRepo: nil,
		SettingsRepo: settingsRepo,
	})
	user, err := uc.Register(context.Background(), "testuser", "test@example.com", "password123", nil)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.ErrorIs(t, err, httperr.ErrRegistrationClosed)
}

type loginTestCase struct {
	name          string
	email         string
	password      string
	setupMocks    func(_ *testing.T, _ *mocks.MockUserRepository, _ *mocks.MockJWTService)
	expectedError bool
}

func loginTestCases() []loginTestCase {
	return []loginTestCase{
		{
			name: "successful login", email: "test@example.com", password: "password123",
			setupMocks:    func(_ *testing.T, _ *mocks.MockUserRepository, _ *mocks.MockJWTService) {},
			expectedError: false,
		},
		{
			name: "user not found", email: "notfound@example.com", password: "password123",
			setupMocks: func(_ *testing.T, userRepo *mocks.MockUserRepository, _ *mocks.MockJWTService) {
				userRepo.EXPECT().GetByEmail(mock.Anything, "notfound@example.com").Return(nil, httperr.ErrUserNotFound)
			},
			expectedError: true,
		},
		{
			name: "invalid password", email: "test@example.com", password: "wrongpassword",
			setupMocks: func(t *testing.T, userRepo *mocks.MockUserRepository, _ *mocks.MockJWTService) {
				t.Helper()
				hashedPassword, err := hashPassword("password123")
				require.NoError(t, err)
				user := &entity.User{ID: uuid.New(), Username: "testuser", Email: "test@example.com", PasswordHash: hashedPassword}
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(user, nil)
			},
			expectedError: true,
		},
		{
			name: "GetByEmail returns unexpected error", email: "test@example.com", password: "password123",
			setupMocks: func(_ *testing.T, userRepo *mocks.MockUserRepository, _ *mocks.MockJWTService) {
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(nil, assert.AnError)
			},
			expectedError: true,
		},
		{
			name: "user with valid uuid", email: "test@example.com", password: "password123",
			setupMocks: func(t *testing.T, userRepo *mocks.MockUserRepository, jwtService *mocks.MockJWTService) {
				t.Helper()
				hashedPassword, err := hashPassword("password123")
				require.NoError(t, err)
				user := &entity.User{ID: uuid.New(), Username: "testuser", Email: "test@example.com", PasswordHash: hashedPassword}
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(user, nil)
				jwtService.EXPECT().GenerateTokenPair(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&jwt.TokenPair{AccessToken: "token", RefreshToken: "refresh"}, nil)
			},
			expectedError: false,
		},
		{
			name: "GenerateTokenPair returns error", email: "test@example.com", password: "password123",
			setupMocks: func(t *testing.T, userRepo *mocks.MockUserRepository, jwtService *mocks.MockJWTService) {
				t.Helper()
				hashedPassword, err := hashPassword("password123")
				require.NoError(t, err)
				user := &entity.User{ID: uuid.New(), Username: "testuser", Email: "test@example.com", PasswordHash: hashedPassword}
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(user, nil)
				jwtService.EXPECT().GenerateTokenPair(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)
			},
			expectedError: true,
		},
	}
}

func runLoginTest(t *testing.T, tt loginTestCase) {
	t.Helper()
	h := NewUserTestHelper(t)
	tt.setupMocks(t, h.Deps().userRepo, h.Deps().jwtService)
	uc := h.CreateUseCase()
	if tt.name == "successful login" {
		h.SetupLoginMocks(tt.email, tt.password)
	}
	tokenPair, err := uc.Login(context.Background(), tt.email, tt.password)
	if tt.expectedError {
		assert.Error(t, err)
		assert.Nil(t, tokenPair)
	} else {
		assert.NoError(t, err)
		assert.NotNil(t, tokenPair)
		assert.NotEmpty(t, tokenPair.AccessToken)
	}
}

func TestUserUseCase_Login(t *testing.T) {
	t.Parallel()
	for _, tt := range loginTestCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runLoginTest(t, tt)
		})
	}
}

func TestUserUseCase_GetByID_Success(t *testing.T) {
	t.Parallel()
	h := NewUserTestHelper(t)
	deps := h.Deps()

	userID := uuid.New()
	expectedUser := &entity.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(expectedUser, nil)

	uc := h.CreateUseCase()

	user, err := uc.GetByID(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Username, user.Username)
}

func TestUserUseCase_GetByID_Error(t *testing.T) {
	t.Parallel()
	h := NewUserTestHelper(t)
	deps := h.Deps()

	userID := uuid.New()
	expectedError := assert.AnError

	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, expectedError)

	uc := h.CreateUseCase()

	user, err := uc.GetByID(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestUserUseCase_GetProfile_Success(t *testing.T) {
	t.Parallel()
	h := NewUserTestHelper(t)
	deps := h.Deps()

	userID := uuid.New()
	user := &entity.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	solves := []*entity.Solve{h.NewSolve(userID, uuid.New())}

	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)
	deps.solveRepo.EXPECT().GetByUserID(mock.Anything, userID).Return(solves, nil)

	uc := h.CreateUseCase()

	profile, err := uc.GetProfile(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, user.Username, profile.User.Username)
	assert.Equal(t, "", profile.User.PasswordHash)
	assert.Len(t, profile.Solves, 1)
}

func TestUserUseCase_GetProfile_GetByIDError(t *testing.T) {
	t.Parallel()
	h := NewUserTestHelper(t)
	deps := h.Deps()

	userID := uuid.New()
	expectedError := assert.AnError

	deps.solveRepo.EXPECT().GetByUserID(mock.Anything, userID).Return([]*entity.Solve{}, nil)
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Run(func(_ context.Context, _ uuid.UUID) {
		time.Sleep(50 * time.Millisecond)
	}).Return(nil, expectedError)

	uc := h.CreateUseCase()

	profile, err := uc.GetProfile(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, profile)
}

func TestUserUseCase_GetProfile_GetByUserIDError(t *testing.T) {
	t.Parallel()
	h := NewUserTestHelper(t)
	deps := h.Deps()

	userID := uuid.New()
	user := &entity.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}
	expectedError := assert.AnError

	// Delay GetByUserID so GetByID completes first; avoids testify mock/context cancellation race.
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)
	deps.solveRepo.EXPECT().GetByUserID(mock.Anything, userID).Run(func(_ context.Context, _ uuid.UUID) {
		time.Sleep(50 * time.Millisecond)
	}).Return(nil, expectedError)

	uc := h.CreateUseCase()

	profile, err := uc.GetProfile(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, profile)
}
