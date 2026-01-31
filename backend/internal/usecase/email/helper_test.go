package email

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase/email/mocks"
)

type EmailTestHelper struct {
	t    *testing.T
	deps *emailTestDeps
}

type emailTestDeps struct {
	userRepo  *mocks.MockUserRepository
	tokenRepo *mocks.MockVerificationTokenRepository
	mailer    *mocks.MockMailer
}

func NewEmailTestHelper(t *testing.T) *EmailTestHelper {
	t.Helper()
	return &EmailTestHelper{
		t: t,
		deps: &emailTestDeps{
			userRepo:  mocks.NewMockUserRepository(t),
			tokenRepo: mocks.NewMockVerificationTokenRepository(t),
			mailer:    mocks.NewMockMailer(t),
		},
	}
}

func (h *EmailTestHelper) CreateUseCase() *EmailUseCase {
	h.t.Helper()
	return NewEmailUseCase(
		h.deps.userRepo,
		h.deps.tokenRepo,
		h.deps.mailer,
		24*time.Hour,
		1*time.Hour,
		"http://localhost:3000",
		true,
	)
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
