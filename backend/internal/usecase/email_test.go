package usecase_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/internal/usecase/mocks"
	"github.com/skr1ms/CTFBoard/pkg/mailer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEmailUseCase_SendVerificationEmail_Hashing(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	tokenRepo := new(mocks.MockVerificationTokenRepository)
	mailerMock := new(mocks.MockMailer)

	uc := usecase.NewEmailUseCase(
		userRepo,
		tokenRepo,
		mailerMock,
		24*time.Hour,
		1*time.Hour,
		"http://localhost:3000",
		true,
	)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	user := &entity.User{
		Id:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	tokenRepo.On("DeleteByUserAndType", mock.Anything, user.Id, entity.TokenTypeEmailVerification).Return(nil)

	var storedToken string
	tokenRepo.On("Create", mock.Anything, mock.MatchedBy(func(vt *entity.VerificationToken) bool {
		storedToken = vt.Token
		return vt.UserId == user.Id && vt.Type == entity.TokenTypeEmailVerification
	})).Return(nil)

	var sentBody string
	mailerMock.On("Send", mock.Anything, mock.MatchedBy(func(msg mailer.Message) bool {
		sentBody = msg.Body
		return msg.To == user.Email
	})).Return(nil)

	err := uc.SendVerificationEmail(context.Background(), user)
	assert.NoError(t, err)

	assert.Contains(t, sentBody, "token=")
	start := strings.Index(sentBody, "token=") + 6
	rawToken := sentBody[start : start+64]

	hash := sha256.Sum256([]byte(rawToken))
	expectedHash := hex.EncodeToString(hash[:])

	assert.Equal(t, expectedHash, storedToken, "Stored token should be SHA256 hash of sent token")
	assert.NotEqual(t, rawToken, storedToken, "Stored token should not be raw token")
}

func TestEmailUseCase_VerifyEmail_Hashing(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	tokenRepo := new(mocks.MockVerificationTokenRepository)
	mailerMock := new(mocks.MockMailer)

	uc := usecase.NewEmailUseCase(
		userRepo,
		tokenRepo,
		mailerMock,
		24*time.Hour,
		1*time.Hour,
		"http://localhost:3000",
		true,
	)

	rawToken := "1111111111111111111111111111111111111111111111111111111111111111"
	hash := sha256.Sum256([]byte(rawToken))
	hashedToken := hex.EncodeToString(hash[:])

	tokenEntity := &entity.VerificationToken{
		Id:        uuid.New(),
		UserId:    uuid.New(),
		Token:     hashedToken,
		Type:      entity.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	tokenRepo.On("GetByToken", mock.Anything, hashedToken).Return(tokenEntity, nil)
	userRepo.On("SetVerified", mock.Anything, tokenEntity.UserId).Return(nil)
	tokenRepo.On("DeleteByUserAndType", mock.Anything, tokenEntity.UserId, entity.TokenTypeEmailVerification).Return(nil)
	err := uc.VerifyEmail(context.Background(), rawToken)
	assert.NoError(t, err)
}

func TestEmailUseCase_SendPasswordResetEmail_Success(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	tokenRepo := new(mocks.MockVerificationTokenRepository)
	mailerMock := new(mocks.MockMailer)

	uc := usecase.NewEmailUseCase(
		userRepo, tokenRepo, mailerMock,
		24*time.Hour, 1*time.Hour,
		"http://localhost:3000", true,
	)

	user := &entity.User{
		Id:       uuid.New(),
		Username: "testuser",
		Email:    "test@example.com",
	}

	userRepo.On("GetByEmail", mock.Anything, user.Email).Return(user, nil)
	tokenRepo.On("DeleteByUserAndType", mock.Anything, user.Id, entity.TokenTypePasswordReset).Return(nil)

	var storedToken string
	tokenRepo.On("Create", mock.Anything, mock.MatchedBy(func(vt *entity.VerificationToken) bool {
		storedToken = vt.Token
		return vt.UserId == user.Id && vt.Type == entity.TokenTypePasswordReset
	})).Return(nil)

	var sentBody string
	mailerMock.On("Send", mock.Anything, mock.MatchedBy(func(msg mailer.Message) bool {
		sentBody = msg.Body
		return msg.To == user.Email && strings.Contains(msg.Body, "reset-password?token=")
	})).Return(nil)

	err := uc.SendPasswordResetEmail(context.Background(), user.Email)
	assert.NoError(t, err)

	start := strings.Index(sentBody, "token=") + 6
	rawToken := sentBody[start : start+64]

	hash := sha256.Sum256([]byte(rawToken))
	expectedHash := hex.EncodeToString(hash[:])

	assert.Equal(t, expectedHash, storedToken, "Stored reset token should be SHA256 hash of emailed token")
}

func TestEmailUseCase_SendPasswordResetEmail_UserNotFound(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	tokenRepo := new(mocks.MockVerificationTokenRepository)
	mailerMock := new(mocks.MockMailer)

	uc := usecase.NewEmailUseCase(
		userRepo, tokenRepo, mailerMock,
		24*time.Hour, 1*time.Hour,
		"http://localhost:3000", true,
	)

	userRepo.On("GetByEmail", mock.Anything, "unknown@example.com").Return(nil, entityError.ErrUserNotFound)

	err := uc.SendPasswordResetEmail(context.Background(), "unknown@example.com")
	assert.NoError(t, err, "Should not return error even if user is not found (security)")
}

func TestEmailUseCase_ResetPassword_Success(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	tokenRepo := new(mocks.MockVerificationTokenRepository)
	mailerMock := new(mocks.MockMailer)

	uc := usecase.NewEmailUseCase(
		userRepo, tokenRepo, mailerMock,
		24*time.Hour, 1*time.Hour,
		"http://localhost:3000", true,
	)

	rawToken := "resetTOKEN123"
	hash := sha256.Sum256([]byte(rawToken))
	hashedToken := hex.EncodeToString(hash[:])

	tokenEntity := &entity.VerificationToken{
		Id:        uuid.New(),
		UserId:    uuid.New(),
		Token:     hashedToken,
		Type:      entity.TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	tokenRepo.On("GetByToken", mock.Anything, hashedToken).Return(tokenEntity, nil)
	userRepo.On("UpdatePassword", mock.Anything, tokenEntity.UserId, mock.MatchedBy(func(pwd string) bool {
		return len(pwd) > 0
	})).Return(nil)
	tokenRepo.On("DeleteByUserAndType", mock.Anything, tokenEntity.UserId, entity.TokenTypePasswordReset).Return(nil)

	err := uc.ResetPassword(context.Background(), rawToken, "new-password")
	assert.NoError(t, err)
}

func TestEmailUseCase_ResetPassword_TokenInvalid(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	tokenRepo := new(mocks.MockVerificationTokenRepository)
	mailerMock := new(mocks.MockMailer)

	uc := usecase.NewEmailUseCase(
		userRepo, tokenRepo, mailerMock,
		24*time.Hour, 1*time.Hour,
		"http://localhost:3000", true,
	)

	tokenRepo.On("GetByToken", mock.Anything, mock.Anything).Return(nil, entityError.ErrTokenNotFound)

	err := uc.ResetPassword(context.Background(), "invalid-token", "new-password")
	assert.ErrorIs(t, err, entityError.ErrTokenNotFound)
}

func TestEmailUseCase_ResendVerification_Success(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	tokenRepo := new(mocks.MockVerificationTokenRepository)
	mailerMock := new(mocks.MockMailer)

	uc := usecase.NewEmailUseCase(
		userRepo, tokenRepo, mailerMock,
		24*time.Hour, 1*time.Hour,
		"http://localhost:3000", true,
	)

	user := &entity.User{
		Id:         uuid.New(),
		IsVerified: false,
		Email:      "test@example.com",
	}

	userRepo.On("GetByID", mock.Anything, user.Id).Return(user, nil)
	tokenRepo.On("DeleteByUserAndType", mock.Anything, user.Id, entity.TokenTypeEmailVerification).Return(nil)
	tokenRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mailerMock.On("Send", mock.Anything, mock.Anything).Return(nil)

	err := uc.ResendVerification(context.Background(), user.Id)
	assert.NoError(t, err)
}

func TestEmailUseCase_ResendVerification_AlreadyVerified(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	tokenRepo := new(mocks.MockVerificationTokenRepository)
	mailerMock := new(mocks.MockMailer)

	uc := usecase.NewEmailUseCase(
		userRepo, tokenRepo, mailerMock,
		24*time.Hour, 1*time.Hour,
		"http://localhost:3000", true,
	)

	user := &entity.User{
		Id:         uuid.New(),
		IsVerified: true,
	}

	userRepo.On("GetByID", mock.Anything, user.Id).Return(user, nil)

	err := uc.ResendVerification(context.Background(), user.Id)
	assert.NoError(t, err, "Should success (noop) if already verified")
}

func TestEmailUseCase_SendPasswordResetEmail_MailerError(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	tokenRepo := new(mocks.MockVerificationTokenRepository)
	mailerMock := new(mocks.MockMailer)

	uc := usecase.NewEmailUseCase(
		userRepo, tokenRepo, mailerMock,
		24*time.Hour, 1*time.Hour,
		"http://localhost:3000", true,
	)

	user := &entity.User{
		Id:       uuid.New(),
		Username: "testuser",
		Email:    "test@example.com",
	}

	userRepo.On("GetByEmail", mock.Anything, user.Email).Return(user, nil)
	tokenRepo.On("DeleteByUserAndType", mock.Anything, user.Id, entity.TokenTypePasswordReset).Return(nil)
	tokenRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mailerMock.On("Send", mock.Anything, mock.Anything).Return(errors.New("mailer timeout"))

	err := uc.SendPasswordResetEmail(context.Background(), user.Email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mailer timeout")
}

func TestEmailUseCase_ResendVerification_UserNotFound(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	tokenRepo := new(mocks.MockVerificationTokenRepository)
	mailerMock := new(mocks.MockMailer)

	uc := usecase.NewEmailUseCase(
		userRepo, tokenRepo, mailerMock,
		24*time.Hour, 1*time.Hour,
		"http://localhost:3000", true,
	)

	userRepo.On("GetByID", mock.Anything, uuid.Nil).Return(nil, entityError.ErrUserNotFound)

	err := uc.ResendVerification(context.Background(), uuid.Nil)
	assert.ErrorIs(t, err, entityError.ErrUserNotFound)
}
