package email

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/email/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type EmailTestHelper struct {
	t    *testing.T
	deps *emailTestDeps
}

type emailTestDeps struct {
	userRepo  *mocks.MockUserRepository
	tokenRepo *mocks.MockVerificationTokenRepository
	mailer    *mocks.MockMailer
	tm        *mocks.MockTransactionManager
}

func NewEmailTestHelper(t *testing.T) *EmailTestHelper {
	t.Helper()
	return &EmailTestHelper{
		t: t,
		deps: &emailTestDeps{
			userRepo:  mocks.NewMockUserRepository(t),
			tokenRepo: mocks.NewMockVerificationTokenRepository(t),
			mailer:    mocks.NewMockMailer(t),
			tm:        mocks.NewMockTransactionManager(t),
		},
	}
}

func (h *EmailTestHelper) CreateUseCase() *EmailUseCase {
	h.t.Helper()
	return NewEmailUseCase(EmailDeps{
		UserRepo: h.deps.userRepo, TokenRepo: h.deps.tokenRepo, Mailer: h.deps.mailer,
		TM:        h.deps.tm,
		VerifyTTL: 24 * time.Hour, ResetTTL: 1 * time.Hour, FrontendURL: "http://localhost:3000", Enabled: true,
	})
}

func (h *EmailTestHelper) SetupTxRun() {
	h.deps.tm.EXPECT().Run(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).Maybe()
	h.deps.tm.EXPECT().RunSerializable(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).Maybe()
}

func (h *EmailTestHelper) Deps() *emailTestDeps {
	h.t.Helper()
	return h.deps
}

func (h *EmailTestHelper) HashToken(rawToken string) string {
	h.t.Helper()
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}

func (h *EmailTestHelper) NewUser(id uuid.UUID, username, email string) *entity.User {
	h.t.Helper()
	return &entity.User{
		ID:       id,
		Username: username,
		Email:    email,
	}
}

func (h *EmailTestHelper) NewVerificationToken(userID uuid.UUID, token string, tokenType entity.TokenType) *entity.VerificationToken {
	h.t.Helper()
	return &entity.VerificationToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		Type:      tokenType,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
}
