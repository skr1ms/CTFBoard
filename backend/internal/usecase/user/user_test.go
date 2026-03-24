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

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	userMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user/mock"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
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
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, httperr.ErrUserNotFound).Once()
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(nil, httperr.ErrUserNotFound).Once()
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
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, httperr.ErrUserNotFound).Once()
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
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, httperr.ErrUserNotFound).Once()
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
	d := newUserTestDeps(t)
	settingsRepo := userMock.NewMockSettingsRepository(t)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	settingsRepo.EXPECT().Get(mock.Anything).Return(&domain.Settings{RegistrationOpen: false}, nil).Once()
	uc := NewUserUseCase(UserDeps{
		UserRepo: d.userRepo, TeamRepo: d.teamRepo, SolveRepo: d.solveRepo,
		TM: d.tm, JWTService: d.jwtService, FieldValidator: nil, FieldValueRepo: nil,
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
				userRepo.EXPECT().GetByEmail(mock.Anything, "notfound@example.com").Return(nil, httperr.ErrUserNotFound)
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
	assert.Equal(t, "", profile.User.PasswordHash)
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

func TestUserUseCase_BanUser_Success_NoSoloTeam(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	actorID := uuid.New()
	user := &domain.User{ID: userID, Role: domain.RoleUser, TeamID: nil}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.userRepo.EXPECT().Ban(mock.Anything, userID, "reason").Return(nil).Once()
	d.solveRepo.EXPECT().GetByUserIDWithDetails(mock.Anything, userID).Return([]*domain.SolveWithDetails{}, nil).Once()
	d.jwtService.EXPECT().RevokeAllForUser(mock.Anything, userID).Return(nil).Once()

	uc := d.createUseCase()
	err := uc.BanUser(context.Background(), userID, "reason", actorID)

	assert.NoError(t, err)
}

func TestUserUseCase_BanUser_Success_HidesSoloTeamInTx(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	actorID := uuid.New()
	teamID := uuid.New()
	user := &domain.User{ID: userID, Role: domain.RoleUser, TeamID: &teamID}
	team := &domain.Team{ID: teamID, IsSolo: true, IsHidden: false}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.userRepo.EXPECT().Ban(mock.Anything, userID, "reason").Return(nil).Once()
	d.solveRepo.EXPECT().GetByUserIDWithDetails(mock.Anything, userID).Return([]*domain.SolveWithDetails{}, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().SetHidden(mock.Anything, teamID, true).Return(nil).Once()
	d.jwtService.EXPECT().RevokeAllForUser(mock.Anything, userID).Return(nil).Once()

	uc := d.createUseCase()
	err := uc.BanUser(context.Background(), userID, "reason", actorID)

	assert.NoError(t, err)
}

func TestUserUseCase_UnbanUser_Success_ShowsSoloTeamInTx(t *testing.T) {
	t.Parallel()
	d := newUserTestDeps(t)

	userID := uuid.New()
	actorID := uuid.New()
	teamID := uuid.New()
	user := &domain.User{ID: userID, Role: domain.RoleUser, TeamID: &teamID, IsBanned: true}
	team := &domain.Team{ID: teamID, IsSolo: true, IsHidden: true}

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil).Once()
	d.userRepo.EXPECT().Unban(mock.Anything, userID).Return(nil).Once()
	d.userRepo.EXPECT().SetWasInBannedTeamByIDs(mock.Anything, []uuid.UUID{userID}, false).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().SetHidden(mock.Anything, teamID, false).Return(nil).Once()

	uc := d.createUseCase()
	err := uc.UnbanUser(context.Background(), userID, actorID)

	assert.NoError(t, err)
}

func TestSanitizeCustomFieldValue(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "abc", sanitizeCustomFieldValue("abc"))
	assert.Equal(t, "a  b", sanitizeCustomFieldValue("a \x00\x1b b"))
	assert.Equal(t, "", sanitizeCustomFieldValue("\x00\x1f\x7f"))
	assert.Equal(t, "x", sanitizeCustomFieldValue("  x  "))
}

func TestSanitizeCustomFields(t *testing.T) {
	t.Parallel()
	assert.Nil(t, sanitizeCustomFields(nil))
	assert.Empty(t, sanitizeCustomFields(map[string]string{}))
	out := sanitizeCustomFields(map[string]string{"k1": " v1 \x00 ", "k2": "v2"})
	assert.Equal(t, "v1", out["k1"])
	assert.Equal(t, "v2", out["k2"])
}
