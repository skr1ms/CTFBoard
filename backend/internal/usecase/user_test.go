package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/usecase/mocks"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestUserUseCase_Register(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		email         string
		password      string
		setupMocks    func(*mocks.MockUserRepository, *mocks.MockTxRepository)
		expectedError bool
	}{
		{
			name:     "successful registration",
			username: "testuser",
			email:    "test@example.com",
			password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, txRepo *mocks.MockTxRepository) {
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, entityError.ErrUserNotFound).Once()
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(nil, entityError.ErrUserNotFound).Once()
				txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
					return fn(ctx, nil)
				}).Once()
				txRepo.EXPECT().CreateUserTx(mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(ctx context.Context, tx pgx.Tx, u *entity.User) {
					u.Id = uuid.New()
				}).Once()
			},
			expectedError: false,
		},
		{
			name:     "username already exists",
			username: "existinguser",
			email:    "test@example.com",
			password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, txRepo *mocks.MockTxRepository) {
				userRepo.EXPECT().GetByUsername(mock.Anything, "existinguser").Return(&entity.User{}, nil)
			},
			expectedError: true,
		},
		{
			name:     "email already exists",
			username: "testuser",
			email:    "existing@example.com",
			password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, txRepo *mocks.MockTxRepository) {
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, entityError.ErrUserNotFound)
				userRepo.EXPECT().GetByEmail(mock.Anything, "existing@example.com").Return(&entity.User{}, nil)
			},
			expectedError: true,
		},
		{
			name:     "GetByUsername returns unexpected error",
			username: "testuser",
			email:    "test@example.com",
			password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, txRepo *mocks.MockTxRepository) {
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, assert.AnError)
			},
			expectedError: true,
		},
		{
			name:     "GetByEmail returns unexpected error",
			username: "testuser",
			email:    "test@example.com",
			password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, txRepo *mocks.MockTxRepository) {
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, entityError.ErrUserNotFound)
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(nil, assert.AnError)
			},
			expectedError: true,
		},
		{
			name:     "Transaction returns error",
			username: "testuser",
			email:    "test@example.com",
			password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, txRepo *mocks.MockTxRepository) {
				userRepo.EXPECT().GetByUsername(mock.Anything, "testuser").Return(nil, entityError.ErrUserNotFound)
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(nil, entityError.ErrUserNotFound)
				txRepo.EXPECT().RunTransaction(mock.Anything, mock.Anything).Return(assert.AnError).Once()
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := mocks.NewMockUserRepository(t)
			teamRepo := mocks.NewMockTeamRepository(t)
			solveRepo := mocks.NewMockSolveRepository(t)
			txRepo := mocks.NewMockTxRepository(t)
			jwtService := mocks.NewMockJWTService(t)

			tt.setupMocks(userRepo, txRepo)

			uc := NewUserUseCase(userRepo, teamRepo, solveRepo, txRepo, jwtService)

			user, err := uc.Register(context.Background(), tt.username, tt.email, tt.password)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.username, user.Username)
				assert.Equal(t, tt.email, user.Email)
			}
		})
	}
}

func TestUserUseCase_Login(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		password      string
		setupMocks    func(*mocks.MockUserRepository, *mocks.MockJWTService)
		expectedError bool
	}{
		{
			name:     "successful login",
			email:    "test@example.com",
			password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, jwtService *mocks.MockJWTService) {
			},
			expectedError: false,
		},
		{
			name:     "user not found",
			email:    "notfound@example.com",
			password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, jwtService *mocks.MockJWTService) {
				userRepo.EXPECT().GetByEmail(mock.Anything, "notfound@example.com").Return(nil, entityError.ErrUserNotFound)
			},
			expectedError: true,
		},
		{
			name:     "invalid password",
			email:    "test@example.com",
			password: "wrongpassword",
			setupMocks: func(userRepo *mocks.MockUserRepository, jwtService *mocks.MockJWTService) {
				hashedPassword, _ := hashPassword("password123")
				user := &entity.User{
					Id:           uuid.New(),
					Username:     "testuser",
					Email:        "test@example.com",
					PasswordHash: hashedPassword,
				}
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(user, nil)
			},
			expectedError: true,
		},
		{
			name:     "GetByEmail returns unexpected error",
			email:    "test@example.com",
			password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, jwtService *mocks.MockJWTService) {
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(nil, assert.AnError)
			},
			expectedError: true,
		},
		{
			name:     "user with valid UUID",
			email:    "test@example.com",
			password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, jwtService *mocks.MockJWTService) {
				hashedPassword, _ := hashPassword("password123")
				user := &entity.User{
					Id:           uuid.New(),
					Username:     "testuser",
					Email:        "test@example.com",
					PasswordHash: hashedPassword,
				}
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(user, nil)
				jwtService.EXPECT().GenerateTokenPair(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&jwt.TokenPair{AccessToken: "token", RefreshToken: "refresh"}, nil)
			},
			expectedError: false,
		},
		{
			name:     "GenerateTokenPair returns error",
			email:    "test@example.com",
			password: "password123",
			setupMocks: func(userRepo *mocks.MockUserRepository, jwtService *mocks.MockJWTService) {
				hashedPassword, _ := hashPassword("password123")
				user := &entity.User{
					Id:           uuid.New(),
					Username:     "testuser",
					Email:        "test@example.com",
					PasswordHash: hashedPassword,
				}
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(user, nil)
				jwtService.EXPECT().GenerateTokenPair(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := mocks.NewMockUserRepository(t)
			teamRepo := mocks.NewMockTeamRepository(t)
			solveRepo := mocks.NewMockSolveRepository(t)
			txRepo := mocks.NewMockTxRepository(t)
			jwtService := mocks.NewMockJWTService(t)
			validator := mocks.NewMockValidator(t)

			tt.setupMocks(userRepo, jwtService)

			uc := NewUserUseCase(userRepo, teamRepo, solveRepo, txRepo, jwtService)
			uc.validator = validator

			if tt.name == "successful login" {
				hashedPassword, _ := hashPassword("password123")
				user := &entity.User{
					Id:           uuid.New(),
					Username:     "testuser",
					Email:        "test@example.com",
					PasswordHash: hashedPassword,
				}
				userRepo.EXPECT().GetByEmail(mock.Anything, "test@example.com").Return(user, nil)
				tokenPair := &jwt.TokenPair{
					AccessToken:      "access_token",
					RefreshToken:     "refresh_token",
					AccessExpiresAt:  time.Now().Unix(),
					RefreshExpiresAt: time.Now().Unix(),
				}
				jwtService.EXPECT().GenerateTokenPair(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tokenPair, nil)
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
		})
	}
}

func TestUserUseCase_GetByID(t *testing.T) {
	userRepo := mocks.NewMockUserRepository(t)
	teamRepo := mocks.NewMockTeamRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	jwtService := mocks.NewMockJWTService(t)

	userID := uuid.New()
	expectedUser := &entity.User{
		Id:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(expectedUser, nil)

	uc := NewUserUseCase(userRepo, teamRepo, solveRepo, txRepo, jwtService)

	user, err := uc.GetByID(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, expectedUser.Id, user.Id)
	assert.Equal(t, expectedUser.Username, user.Username)
}

func TestUserUseCase_GetByID_Error(t *testing.T) {
	userRepo := mocks.NewMockUserRepository(t)
	teamRepo := mocks.NewMockTeamRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	jwtService := mocks.NewMockJWTService(t)

	userID := uuid.New()
	expectedError := assert.AnError

	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, expectedError)

	uc := NewUserUseCase(userRepo, teamRepo, solveRepo, txRepo, jwtService)

	user, err := uc.GetByID(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestUserUseCase_GetProfile(t *testing.T) {
	userRepo := mocks.NewMockUserRepository(t)
	teamRepo := mocks.NewMockTeamRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	jwtService := mocks.NewMockJWTService(t)

	userID := uuid.New()
	user := &entity.User{
		Id:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	solves := []*entity.Solve{
		{
			Id:          uuid.New(),
			UserId:      userID,
			ChallengeId: uuid.New(),
			SolvedAt:    time.Now(),
		},
	}

	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)
	solveRepo.EXPECT().GetByUserId(mock.Anything, userID).Return(solves, nil)

	uc := NewUserUseCase(userRepo, teamRepo, solveRepo, txRepo, jwtService)

	profile, err := uc.GetProfile(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, user.Username, profile.User.Username)
	assert.Equal(t, "", profile.User.PasswordHash)
	assert.Len(t, profile.Solves, 1)
}

func TestUserUseCase_GetProfile_GetByIDError(t *testing.T) {
	userRepo := mocks.NewMockUserRepository(t)
	teamRepo := mocks.NewMockTeamRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	jwtService := mocks.NewMockJWTService(t)

	userID := uuid.New()
	expectedError := assert.AnError

	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(nil, expectedError)

	uc := NewUserUseCase(userRepo, teamRepo, solveRepo, txRepo, jwtService)

	profile, err := uc.GetProfile(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, profile)
}

func TestUserUseCase_GetProfile_GetByUserIdError(t *testing.T) {
	userRepo := mocks.NewMockUserRepository(t)
	teamRepo := mocks.NewMockTeamRepository(t)
	solveRepo := mocks.NewMockSolveRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	jwtService := mocks.NewMockJWTService(t)

	userID := uuid.New()
	user := &entity.User{
		Id:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}
	expectedError := assert.AnError

	userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)
	solveRepo.EXPECT().GetByUserId(mock.Anything, userID).Return(nil, expectedError)

	uc := NewUserUseCase(userRepo, teamRepo, solveRepo, txRepo, jwtService)

	profile, err := uc.GetProfile(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, profile)
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
