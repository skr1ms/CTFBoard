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
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

type UserTestHelper struct {
	t    *testing.T
	deps *testDependencies
}

type testDependencies struct {
	userRepo     *mocks.MockUserRepository
	teamRepo     *mocks.MockTeamRepository
	solveRepo    *mocks.MockSolveRepository
	tm           *mocks.MockTransactionManager
	jwtService   *mocks.MockJWTService
	apiTokenRepo *mocks.MockAPITokenRepository
}

func NewUserTestHelper(t *testing.T) *UserTestHelper {
	t.Helper()
	return &UserTestHelper{
		t: t,
		deps: &testDependencies{
			userRepo:     mocks.NewMockUserRepository(t),
			teamRepo:     mocks.NewMockTeamRepository(t),
			solveRepo:    mocks.NewMockSolveRepository(t),
			tm:           mocks.NewMockTransactionManager(t),
			jwtService:   mocks.NewMockJWTService(t),
			apiTokenRepo: mocks.NewMockAPITokenRepository(t),
		},
	}
}

func (h *UserTestHelper) CreateUseCase() *UserUseCase {
	h.t.Helper()
	return NewUserUseCase(UserDeps{
		UserRepo: h.deps.userRepo, TeamRepo: h.deps.teamRepo, SolveRepo: h.deps.solveRepo,
		TM: h.deps.tm, JWTService: h.deps.jwtService, FieldValidator: nil, FieldValueRepo: nil,
	})
}

func (h *UserTestHelper) Deps() *testDependencies {
	h.t.Helper()
	return h.deps
}

func (h *UserTestHelper) HashPassword(password string) string {
	h.t.Helper()
	s, err := hashPassword(password)
	if err != nil {
		h.t.Fatalf("hashPassword: %v", err)
	}
	return s
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (h *UserTestHelper) NewUser(username, email, passwordHash string) *entity.User {
	h.t.Helper()
	return &entity.User{
		ID:           uuid.New(),
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
	}
}

func (h *UserTestHelper) NewUserWithID(id uuid.UUID, username, email, passwordHash string) *entity.User {
	h.t.Helper()
	u := h.NewUser(username, email, passwordHash)
	u.ID = id
	return u
}

func (h *UserTestHelper) NewSolve(userID, challengeID uuid.UUID) *entity.Solve {
	h.t.Helper()
	return &entity.Solve{
		ID:          uuid.New(),
		UserID:      userID,
		ChallengeID: challengeID,
		SolvedAt:    time.Now(),
	}
}

func (h *UserTestHelper) SetupLoginMocks(email, password string) {
	h.t.Helper()
	hashedPassword := h.HashPassword(password)
	user := &entity.User{
		ID:           uuid.New(),
		Username:     "testuser",
		Email:        email,
		PasswordHash: hashedPassword,
	}
	h.deps.userRepo.EXPECT().GetByEmail(mock.Anything, email).Return(user, nil)
	tokenPair := &jwt.TokenPair{
		AccessToken:      "access_token",
		RefreshToken:     "refresh_token",
		AccessExpiresAt:  time.Now().Unix(),
		RefreshExpiresAt: time.Now().Unix(),
	}
	h.deps.jwtService.EXPECT().GenerateTokenPair(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tokenPair, nil)
}

func (h *UserTestHelper) SetupRegisterSuccessMocks(username, email string) {
	h.t.Helper()
	h.deps.userRepo.EXPECT().GetByUsername(mock.Anything, username).Return(nil, httperr.ErrUserNotFound)
	h.deps.userRepo.EXPECT().GetByEmail(mock.Anything, email).Return(nil, httperr.ErrUserNotFound)
	h.deps.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	h.deps.userRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, u *entity.User) {
		u.ID = uuid.New()
	}).Once()
}

func (h *UserTestHelper) CreateAPITokenUseCase() *APITokenUseCase {
	h.t.Helper()
	return NewAPITokenUseCase(APITokenDeps{Repo: h.deps.apiTokenRepo})
}

func (h *UserTestHelper) NewAPIToken(userID uuid.UUID, tokenHash, description string, expiresAt *time.Time) *entity.APIToken {
	h.t.Helper()
	return &entity.APIToken{
		ID:          uuid.New(),
		UserID:      userID,
		TokenHash:   tokenHash,
		Description: description,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}
}
