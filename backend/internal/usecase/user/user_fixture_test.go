package user

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wahrwelt-kit/go-jwtkit"
	jwtMock "github.com/wahrwelt-kit/go-jwtkit/mock"
	"golang.org/x/crypto/bcrypt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	userMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mock"
)

type userTestDeps struct {
	userRepo   *userMock.MockUserRepository
	teamRepo   *userMock.MockTeamRepository
	solveRepo  *userMock.MockSolveRepository
	tm         *userMock.MockTransactionManager
	jwtService *jwtMock.MockService
}

func newUserTestDeps(t *testing.T) *userTestDeps {
	t.Helper()

	return &userTestDeps{
		userRepo:   userMock.NewMockUserRepository(t),
		teamRepo:   userMock.NewMockTeamRepository(t),
		solveRepo:  userMock.NewMockSolveRepository(t),
		tm:         userMock.NewMockTransactionManager(t),
		jwtService: jwtMock.NewMockService(t),
	}
}

func (d *userTestDeps) createUseCase() *UserUseCase {
	return NewUserUseCase(UserDeps{
		UserRepo: d.userRepo, TeamRepo: d.teamRepo, SolveRepo: d.solveRepo,
		TM: d.tm, JWTService: d.jwtService, FieldValidator: nil, FieldValueRepo: nil,
	})
}

func (d *userTestDeps) setupLoginMocks(t *testing.T, email, password string) {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &domain.User{
		ID: uuid.New(), Username: "testuser", Email: email, PasswordHash: string(hashedPassword),
	}
	d.userRepo.EXPECT().GetByEmail(mock.Anything, email).Return(user, nil)

	tokenPair := &jwtkit.TokenPair{
		AccessToken: "access_token", RefreshToken: "refresh_token",
		AccessExpiresAt: time.Now().Unix(), RefreshExpiresAt: time.Now().Unix(),
	}
	d.jwtService.EXPECT().GenerateTokenPair(mock.Anything, mock.Anything, mock.Anything).Return(tokenPair, nil)
}

func userTestHashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func newTestSolve(userID, challengeID uuid.UUID) *domain.Solve {
	return &domain.Solve{
		ID: uuid.New(), UserID: userID, ChallengeID: challengeID, SolvedAt: time.Now(),
	}
}

type registerTestCase struct {
	name          string
	username      string
	email         string
	password      string
	setupMocks    func(*userMock.MockUserRepository, *userMock.MockTransactionManager)
	expectedError bool
}

func registerTestCases() []registerTestCase {
	return []registerTestCase{
		{
			name: "successful registration", username: "testuser", email: "test@example.com", password: "password123",
			setupMocks: func(userRepo *userMock.MockUserRepository, tm *userMock.MockTransactionManager) {
				tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).Once()
				userRepo.EXPECT().AcquireAdvisoryLock(mock.Anything, mock.Anything).Return(nil).Twice()
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, apperr.ErrUserNotFound).Once()
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(nil, apperr.ErrUserNotFound).Once()
				userRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, u *domain.User) {
					u.ID = uuid.New()
				}).Once()
			},
			expectedError: false,
		},
		{
			name: "username already exists", username: "existinguser", email: "test@example.com", password: "password123",
			setupMocks: func(userRepo *userMock.MockUserRepository, tm *userMock.MockTransactionManager) {
				tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).Once()
				userRepo.EXPECT().AcquireAdvisoryLock(mock.Anything, mock.Anything).Return(nil).Twice()
				userRepo.EXPECT().GetByUsername(mock.Anything, "existinguser").Return(&domain.User{}, nil).Once()
			},
			expectedError: true,
		},
		{
			name: "email already exists", username: "testuser", email: "existing@example.com", password: "password123",
			setupMocks: func(userRepo *userMock.MockUserRepository, tm *userMock.MockTransactionManager) {
				tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).Once()
				userRepo.EXPECT().AcquireAdvisoryLock(mock.Anything, mock.Anything).Return(nil).Maybe()
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, apperr.ErrUserNotFound).Once()
				userRepo.EXPECT().GetByEmail(mock.Anything, "existing@example.com").Return(&domain.User{}, nil).Once()
			},
			expectedError: true,
		},
		{
			name: "GetByUsername returns unexpected error", username: "testuser", email: "test@example.com", password: "password123",
			setupMocks: func(userRepo *userMock.MockUserRepository, tm *userMock.MockTransactionManager) {
				tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).Once()
				userRepo.EXPECT().AcquireAdvisoryLock(mock.Anything, mock.Anything).Return(nil).Twice()
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, assert.AnError).Once()
			},
			expectedError: true,
		},
		{
			name: "GetByEmail returns unexpected error", username: "testuser", email: "test@example.com", password: "password123",
			setupMocks: func(userRepo *userMock.MockUserRepository, tm *userMock.MockTransactionManager) {
				tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).Once()
				userRepo.EXPECT().AcquireAdvisoryLock(mock.Anything, mock.Anything).Return(nil).Twice()
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, apperr.ErrUserNotFound).Once()
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(nil, assert.AnError).Once()
			},
			expectedError: true,
		},
		{
			name: "Transaction returns error", username: "testuser", email: "test@example.com", password: "password123",
			setupMocks: func(_ *userMock.MockUserRepository, tm *userMock.MockTransactionManager) {
				tm.EXPECT().Run(mock.Anything, mock.Anything).Return(assert.AnError).Once()
			},
			expectedError: true,
		},
	}
}

func runRegisterTest(t *testing.T, tt registerTestCase) {
	t.Helper()
	d := newUserTestDeps(t)
	tt.setupMocks(d.userRepo, d.tm)
	uc := d.createUseCase()

	user, err := uc.Register(context.Background(), usecase.UserRegisterParams{
		Username: tt.username,
		Email:    tt.email,
		Password: tt.password,
	})
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

type loginTestCase struct {
	name          string
	email         string
	password      string
	setupMocks    func(_ *testing.T, _ *userMock.MockUserRepository, _ *jwtMock.MockService)
	expectedError bool
}

func loginTestCases() []loginTestCase {
	return []loginTestCase{
		{
			name: "successful login", email: "test@example.com", password: "password123",
			setupMocks:    func(_ *testing.T, _ *userMock.MockUserRepository, _ *jwtMock.MockService) {},
			expectedError: false,
		},
		{
			name: "user not found", email: "notfound@example.com", password: "password123",
			setupMocks: func(_ *testing.T, userRepo *userMock.MockUserRepository, _ *jwtMock.MockService) {
				userRepo.EXPECT().GetByEmail(mock.Anything, "notfound@example.com").Return(nil, apperr.ErrUserNotFound)
			},
			expectedError: true,
		},
		{
			name: "invalid password", email: "test@example.com", password: "wrongpassword",
			setupMocks: func(t *testing.T, userRepo *userMock.MockUserRepository, _ *jwtMock.MockService) {
				t.Helper()

				hashedPassword, err := userTestHashPassword("password123")
				require.NoError(t, err)

				user := &domain.User{ID: uuid.New(), Username: "testuser", Email: "test@example.com", PasswordHash: hashedPassword}
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(user, nil)
			},
			expectedError: true,
		},
		{
			name: "GetByEmail returns unexpected error", email: "test@example.com", password: "password123",
			setupMocks: func(_ *testing.T, userRepo *userMock.MockUserRepository, _ *jwtMock.MockService) {
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(nil, assert.AnError)
			},
			expectedError: true,
		},
		{
			name: "user with valid uuid", email: "test@example.com", password: "password123",
			setupMocks: func(t *testing.T, userRepo *userMock.MockUserRepository, jwtService *jwtMock.MockService) {
				t.Helper()

				hashedPassword, err := userTestHashPassword("password123")
				require.NoError(t, err)

				user := &domain.User{ID: uuid.New(), Username: "testuser", Email: "test@example.com", PasswordHash: hashedPassword}
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(user, nil)
				jwtService.EXPECT().GenerateTokenPair(mock.Anything, mock.Anything, mock.Anything).Return(&jwtkit.TokenPair{AccessToken: "token", RefreshToken: "refresh"}, nil)
			},
			expectedError: false,
		},
		{
			name: "GenerateTokenPair returns error", email: "test@example.com", password: "password123",
			setupMocks: func(t *testing.T, userRepo *userMock.MockUserRepository, jwtService *jwtMock.MockService) {
				t.Helper()

				hashedPassword, err := userTestHashPassword("password123")
				require.NoError(t, err)

				user := &domain.User{ID: uuid.New(), Username: "testuser", Email: "test@example.com", PasswordHash: hashedPassword}
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(user, nil)
				jwtService.EXPECT().GenerateTokenPair(mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)
			},
			expectedError: true,
		},
	}
}

func runLoginTest(t *testing.T, tt loginTestCase) {
	t.Helper()
	d := newUserTestDeps(t)
	tt.setupMocks(t, d.userRepo, d.jwtService)
	uc := d.createUseCase()

	if tt.name == "successful login" {
		d.setupLoginMocks(t, tt.email, tt.password)
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
