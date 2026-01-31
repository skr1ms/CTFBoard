package email

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/pkg/mailer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEmailUseCase_SendVerificationEmail_Hashing(t *testing.T) {
	h := NewEmailTestHelper(t)
	deps := h.Deps()

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	user := h.NewUser(userID, "testuser", "test@example.com")

	deps.tokenRepo.On("DeleteByUserAndType", mock.Anything, user.ID, entity.TokenTypeEmailVerification).Return(nil)

	var storedToken string
	deps.tokenRepo.On("Create", mock.Anything, mock.MatchedBy(func(vt *entity.VerificationToken) bool {
		storedToken = vt.Token
		return vt.UserID == user.ID && vt.Type == entity.TokenTypeEmailVerification
	})).Return(nil)

	var sentBody string
	deps.mailer.On("Send", mock.Anything, mock.MatchedBy(func(msg mailer.Message) bool {
		sentBody = msg.Body
		return msg.To == user.Email
	})).Return(nil)

	err := h.CreateUseCase().SendVerificationEmail(context.Background(), user)
	assert.NoError(t, err)

	assert.Contains(t, sentBody, "token=")
	start := strings.Index(sentBody, "token=") + 6
	rawToken := sentBody[start : start+64]

	expectedHash := h.HashToken(rawToken)
	assert.Equal(t, expectedHash, storedToken, "Stored token should be SHA256 hash of sent token")
	assert.NotEqual(t, rawToken, storedToken, "Stored token should not be raw token")
}

func TestEmailUseCase_VerifyEmail_Hashing(t *testing.T) {
	h := NewEmailTestHelper(t)
	deps := h.Deps()

	rawToken := "1111111111111111111111111111111111111111111111111111111111111111"
	hashedToken := h.HashToken(rawToken)
	tokenEntity := h.NewVerificationToken(uuid.New(), hashedToken, entity.TokenTypeEmailVerification)

	deps.tokenRepo.On("GetByToken", mock.Anything, hashedToken).Return(tokenEntity, nil)
	deps.userRepo.On("SetVerified", mock.Anything, tokenEntity.UserID).Return(nil)
	deps.tokenRepo.On("DeleteByUserAndType", mock.Anything, tokenEntity.UserID, entity.TokenTypeEmailVerification).Return(nil)
	err := h.CreateUseCase().VerifyEmail(context.Background(), rawToken)
	assert.NoError(t, err)
}

func TestEmailUseCase_SendPasswordResetEmail_Success(t *testing.T) {
	h := NewEmailTestHelper(t)
	deps := h.Deps()

	user := h.NewUser(uuid.New(), "testuser", "test@example.com")

	deps.userRepo.On("GetByEmail", mock.Anything, user.Email).Return(user, nil)
	deps.tokenRepo.On("DeleteByUserAndType", mock.Anything, user.ID, entity.TokenTypePasswordReset).Return(nil)

	var storedToken string
	deps.tokenRepo.On("Create", mock.Anything, mock.MatchedBy(func(vt *entity.VerificationToken) bool {
		storedToken = vt.Token
		return vt.UserID == user.ID && vt.Type == entity.TokenTypePasswordReset
	})).Return(nil)

	var sentBody string
	deps.mailer.On("Send", mock.Anything, mock.MatchedBy(func(msg mailer.Message) bool {
		sentBody = msg.Body
		return msg.To == user.Email && strings.Contains(msg.Body, "reset-password?token=")
	})).Return(nil)

	err := h.CreateUseCase().SendPasswordResetEmail(context.Background(), user.Email)
	assert.NoError(t, err)

	start := strings.Index(sentBody, "token=") + 6
	rawToken := sentBody[start : start+64]
	expectedHash := h.HashToken(rawToken)
	assert.Equal(t, expectedHash, storedToken, "Stored reset token should be SHA256 hash of emailed token")
}

func TestEmailUseCase_SendPasswordResetEmail_UserNotFound(t *testing.T) {
	h := NewEmailTestHelper(t)
	deps := h.Deps()

	deps.userRepo.On("GetByEmail", mock.Anything, "unknown@example.com").Return(nil, entityError.ErrUserNotFound)

	err := h.CreateUseCase().SendPasswordResetEmail(context.Background(), "unknown@example.com")
	assert.NoError(t, err, "Should not return error even if user is not found (security)")
}

func TestEmailUseCase_ResetPassword_Success(t *testing.T) {
	h := NewEmailTestHelper(t)
	deps := h.Deps()

	rawToken := "resetTOKEN123"
	hashedToken := h.HashToken(rawToken)
	tokenEntity := h.NewVerificationToken(uuid.New(), hashedToken, entity.TokenTypePasswordReset)

	deps.tokenRepo.On("GetByToken", mock.Anything, hashedToken).Return(tokenEntity, nil)
	deps.userRepo.On("UpdatePassword", mock.Anything, tokenEntity.UserID, mock.MatchedBy(func(pwd string) bool {
		return len(pwd) > 0
	})).Return(nil)
	deps.tokenRepo.On("DeleteByUserAndType", mock.Anything, tokenEntity.UserID, entity.TokenTypePasswordReset).Return(nil)

	err := h.CreateUseCase().ResetPassword(context.Background(), rawToken, "new-password")
	assert.NoError(t, err)
}

func TestEmailUseCase_ResetPassword_TokenInvalid(t *testing.T) {
	h := NewEmailTestHelper(t)
	deps := h.Deps()

	deps.tokenRepo.On("GetByToken", mock.Anything, mock.Anything).Return(nil, entityError.ErrTokenNotFound)

	err := h.CreateUseCase().ResetPassword(context.Background(), "invalid-token", "new-password")
	assert.ErrorIs(t, err, entityError.ErrTokenNotFound)
}

func TestEmailUseCase_ResendVerification_Success(t *testing.T) {
	h := NewEmailTestHelper(t)
	deps := h.Deps()

	user := h.NewUser(uuid.New(), "testuser", "test@example.com")
	user.IsVerified = false

	deps.userRepo.On("GetByID", mock.Anything, user.ID).Return(user, nil)
	deps.tokenRepo.On("DeleteByUserAndType", mock.Anything, user.ID, entity.TokenTypeEmailVerification).Return(nil)
	deps.tokenRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	deps.mailer.On("Send", mock.Anything, mock.Anything).Return(nil)

	err := h.CreateUseCase().ResendVerification(context.Background(), user.ID)
	assert.NoError(t, err)
}

func TestEmailUseCase_ResendVerification_AlreadyVerified(t *testing.T) {
	h := NewEmailTestHelper(t)
	deps := h.Deps()

	user := h.NewUser(uuid.New(), "testuser", "test@example.com")
	user.IsVerified = true

	deps.userRepo.On("GetByID", mock.Anything, user.ID).Return(user, nil)

	err := h.CreateUseCase().ResendVerification(context.Background(), user.ID)
	assert.NoError(t, err, "Should success (noop) if already verified")
}

func TestEmailUseCase_SendPasswordResetEmail_MailerError(t *testing.T) {
	h := NewEmailTestHelper(t)
	deps := h.Deps()

	user := h.NewUser(uuid.New(), "testuser", "test@example.com")

	deps.userRepo.On("GetByEmail", mock.Anything, user.Email).Return(user, nil)
	deps.tokenRepo.On("DeleteByUserAndType", mock.Anything, user.ID, entity.TokenTypePasswordReset).Return(nil)
	deps.tokenRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	deps.mailer.On("Send", mock.Anything, mock.Anything).Return(errors.New("mailer timeout"))

	err := h.CreateUseCase().SendPasswordResetEmail(context.Background(), user.Email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mailer timeout")
}

func TestEmailUseCase_ResendVerification_UserNotFound(t *testing.T) {
	h := NewEmailTestHelper(t)
	deps := h.Deps()

	deps.userRepo.On("GetByID", mock.Anything, uuid.Nil).Return(nil, entityError.ErrUserNotFound)

	err := h.CreateUseCase().ResendVerification(context.Background(), uuid.Nil)
	assert.ErrorIs(t, err, entityError.ErrUserNotFound)
}
