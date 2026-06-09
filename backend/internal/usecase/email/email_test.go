package email

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	emailMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/email/mock"
)

type emailTestDeps struct {
	userRepo        *emailMock.MockUserRepository
	tokenRepo       *emailMock.MockVerificationTokenRepository
	mailer          *emailMock.MockMailer
	tm              *emailMock.MockTransactionManager
	apiTokenRevoker *emailTestAPITokenRevoker
}

func newEmailTestDeps(t *testing.T) *emailTestDeps {
	t.Helper()

	return &emailTestDeps{
		userRepo:        emailMock.NewMockUserRepository(t),
		tokenRepo:       emailMock.NewMockVerificationTokenRepository(t),
		mailer:          emailMock.NewMockMailer(t),
		tm:              emailMock.NewMockTransactionManager(t),
		apiTokenRevoker: &emailTestAPITokenRevoker{},
	}
}

func (d *emailTestDeps) createUseCase() *EmailUseCase {
	return NewEmailUseCase(EmailDeps{
		UserRepo:        d.userRepo,
		TokenRepo:       d.tokenRepo,
		Mailer:          d.mailer,
		TM:              d.tm,
		VerifyTTL:       24 * time.Hour,
		ResetTTL:        1 * time.Hour,
		FrontendURL:     "http://localhost:3000",
		Enabled:         true,
		APITokenRevoker: d.apiTokenRevoker,
	})
}

type emailTestAPITokenRevoker struct {
	mock.Mock
}

func (r *emailTestAPITokenRevoker) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	args := r.Called(ctx, userID)

	return args.Error(0)
}

func (d *emailTestDeps) setupTxRun() {
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).Maybe()
	d.tm.EXPECT().RunSerializable(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).Maybe()
}

func hashTestToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))

	return hex.EncodeToString(hash[:])
}

func newTestUser(id uuid.UUID, username, email string) *domain.User {
	return &domain.User{
		ID:       id,
		Username: username,
		Email:    email,
	}
}

func newTestVerificationToken(userID uuid.UUID, token string, tokenType domain.TokenType) *domain.VerificationToken {
	return &domain.VerificationToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		Type:      tokenType,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
}

func TestEmailUseCase_SendVerificationEmail_Hashing(t *testing.T) {
	t.Parallel()
	d := newEmailTestDeps(t)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	user := newTestUser(userID, "testuser", "test@example.com")

	d.setupTxRun()
	d.tokenRepo.On("DeleteByUserAndType", mock.Anything, user.ID, domain.TokenTypeEmailVerification).Return(nil)

	var storedToken string

	d.tokenRepo.On("Create", mock.Anything, mock.MatchedBy(func(vt *domain.VerificationToken) bool {
		storedToken = vt.Token

		return vt.UserID == user.ID && vt.Type == domain.TokenTypeEmailVerification
	})).Return(nil)

	var sentBody string

	d.mailer.On("Send", mock.Anything, mock.MatchedBy(func(msg Message) bool {
		sentBody = msg.Body

		return msg.To == user.Email
	})).Return(nil)

	err := d.createUseCase().SendVerificationEmail(context.Background(), user)
	assert.NoError(t, err)

	assert.Contains(t, sentBody, "token=")
	start := strings.Index(sentBody, "token=") + 6
	rawToken := sentBody[start : start+64]

	expectedHash := hashTestToken(rawToken)
	assert.Equal(t, expectedHash, storedToken, "Stored token should be SHA256 hash of sent token")
	assert.NotEqual(t, rawToken, storedToken, "Stored token should not be raw token")
}

func TestEmailUseCase_VerifyEmail_Hashing(t *testing.T) {
	t.Parallel()
	d := newEmailTestDeps(t)

	rawToken := "1111111111111111111111111111111111111111111111111111111111111111"
	hashedToken := hashTestToken(rawToken)
	tokenentity := newTestVerificationToken(uuid.New(), hashedToken, domain.TokenTypeEmailVerification)

	d.tokenRepo.On("GetByToken", mock.Anything, hashedToken).Return(tokenentity, nil)
	d.setupTxRun()
	d.userRepo.On("SetVerified", mock.Anything, tokenentity.UserID).Return(nil)
	d.tokenRepo.On("DeleteByUserAndType", mock.Anything, tokenentity.UserID, domain.TokenTypeEmailVerification).Return(nil)
	err := d.createUseCase().VerifyEmail(context.Background(), rawToken)
	assert.NoError(t, err)
}

func TestEmailUseCase_SendPasswordResetEmail_Success(t *testing.T) {
	t.Parallel()
	d := newEmailTestDeps(t)

	user := newTestUser(uuid.New(), "testuser", "test@example.com")

	d.setupTxRun()
	d.userRepo.On("GetByEmail", mock.Anything, user.Email).Return(user, nil)
	d.tokenRepo.On("DeleteByUserAndType", mock.Anything, user.ID, domain.TokenTypePasswordReset).Return(nil)

	var storedToken string

	d.tokenRepo.On("Create", mock.Anything, mock.MatchedBy(func(vt *domain.VerificationToken) bool {
		storedToken = vt.Token

		return vt.UserID == user.ID && vt.Type == domain.TokenTypePasswordReset
	})).Return(nil)

	var sentBody string

	d.mailer.On("Send", mock.Anything, mock.MatchedBy(func(msg Message) bool {
		sentBody = msg.Body

		return msg.To == user.Email && strings.Contains(msg.Body, "reset-password?token=")
	})).Return(nil)

	err := d.createUseCase().SendPasswordResetEmail(context.Background(), user.Email)
	assert.NoError(t, err)

	start := strings.Index(sentBody, "token=") + 6
	rawToken := sentBody[start : start+64]
	expectedHash := hashTestToken(rawToken)
	assert.Equal(t, expectedHash, storedToken, "Stored reset token should be SHA256 hash of emailed token")
}

func TestEmailUseCase_SendPasswordResetEmail_UserNotFound(t *testing.T) {
	t.Parallel()
	d := newEmailTestDeps(t)

	d.userRepo.On("GetByEmail", mock.Anything, "unknown@example.com").Return(nil, apperr.ErrUserNotFound)

	err := d.createUseCase().SendPasswordResetEmail(context.Background(), "unknown@example.com")
	assert.NoError(t, err, "Should not return error even if user is not found (security)")
}

func TestEmailUseCase_ResetPassword_Success(t *testing.T) {
	t.Parallel()
	d := newEmailTestDeps(t)

	rawToken := "resetTOKEN123"
	hashedToken := hashTestToken(rawToken)
	tokenentity := newTestVerificationToken(uuid.New(), hashedToken, domain.TokenTypePasswordReset)

	d.tokenRepo.On("GetByToken", mock.Anything, hashedToken).Return(tokenentity, nil)
	d.setupTxRun()
	d.userRepo.On("UpdatePassword", mock.Anything, tokenentity.UserID, mock.MatchedBy(func(pwd string) bool {
		return len(pwd) > 0
	})).Return(nil)
	d.apiTokenRevoker.On("RevokeAllForUser", mock.Anything, tokenentity.UserID).Return(nil)
	d.tokenRepo.On("DeleteByUserAndType", mock.Anything, tokenentity.UserID, domain.TokenTypePasswordReset).Return(nil)

	err := d.createUseCase().ResetPassword(context.Background(), rawToken, "new-password")
	assert.NoError(t, err)
}

func TestEmailUseCase_ResetPassword_APITokenRevokeError(t *testing.T) {
	t.Parallel()
	d := newEmailTestDeps(t)

	rawToken := "resetTOKEN123"
	hashedToken := hashTestToken(rawToken)
	tokenentity := newTestVerificationToken(uuid.New(), hashedToken, domain.TokenTypePasswordReset)

	d.tokenRepo.On("GetByToken", mock.Anything, hashedToken).Return(tokenentity, nil)
	d.setupTxRun()
	d.userRepo.On("UpdatePassword", mock.Anything, tokenentity.UserID, mock.Anything).Return(nil)
	d.apiTokenRevoker.On("RevokeAllForUser", mock.Anything, tokenentity.UserID).Return(assert.AnError)

	err := d.createUseCase().ResetPassword(context.Background(), rawToken, "new-password")

	assert.ErrorIs(t, err, assert.AnError)
	d.tokenRepo.AssertNotCalled(t, "DeleteByUserAndType", mock.Anything, tokenentity.UserID, domain.TokenTypePasswordReset)
}

func TestEmailUseCase_ResetPasswordRateLimitKey(t *testing.T) {
	t.Parallel()
	d := newEmailTestDeps(t)

	rawToken := "resetTOKEN123"

	got := d.createUseCase().ResetPasswordRateLimitKey(rawToken)

	assert.Equal(t, hashTestToken(rawToken), got)
	assert.NotEqual(t, rawToken, got)
}

func TestEmailUseCase_ResetPassword_TokenInvalid(t *testing.T) {
	t.Parallel()
	d := newEmailTestDeps(t)

	d.tokenRepo.On("GetByToken", mock.Anything, mock.Anything).Return(nil, apperr.ErrTokenNotFound)

	err := d.createUseCase().ResetPassword(context.Background(), "invalid-token", "new-password")
	assert.ErrorIs(t, err, apperr.ErrTokenNotFound)
	d.tm.AssertNotCalled(t, "RunSerializable", mock.Anything, mock.Anything)
	d.userRepo.AssertNotCalled(t, "UpdatePassword", mock.Anything, mock.Anything, mock.Anything)
}

func TestEmailUseCase_ResendVerification_Success(t *testing.T) {
	t.Parallel()
	d := newEmailTestDeps(t)

	user := newTestUser(uuid.New(), "testuser", "test@example.com")
	user.IsVerified = false

	d.setupTxRun()
	d.userRepo.On("GetByID", mock.Anything, user.ID).Return(user, nil)
	d.tokenRepo.On("DeleteByUserAndType", mock.Anything, user.ID, domain.TokenTypeEmailVerification).Return(nil)
	d.tokenRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	d.mailer.On("Send", mock.Anything, mock.Anything).Return(nil)

	err := d.createUseCase().ResendVerification(context.Background(), user.ID)
	assert.NoError(t, err)
}

func TestEmailUseCase_ResendVerification_AlreadyVerified(t *testing.T) {
	t.Parallel()
	d := newEmailTestDeps(t)

	user := newTestUser(uuid.New(), "testuser", "test@example.com")
	user.IsVerified = true

	d.userRepo.On("GetByID", mock.Anything, user.ID).Return(user, nil)

	err := d.createUseCase().ResendVerification(context.Background(), user.ID)
	assert.NoError(t, err, "Should success (noop) if already verified")
}

func TestEmailUseCase_SendPasswordResetEmail_MailerError(t *testing.T) {
	t.Parallel()
	d := newEmailTestDeps(t)

	user := newTestUser(uuid.New(), "testuser", "test@example.com")

	d.setupTxRun()
	d.userRepo.On("GetByEmail", mock.Anything, user.Email).Return(user, nil)
	d.tokenRepo.On("DeleteByUserAndType", mock.Anything, user.ID, domain.TokenTypePasswordReset).Return(nil)
	d.tokenRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	d.mailer.On("Send", mock.Anything, mock.Anything).Return(errors.New("mailer timeout"))

	err := d.createUseCase().SendPasswordResetEmail(context.Background(), user.Email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mailer timeout")
}

func TestEmailUseCase_ResendVerification_UserNotFound(t *testing.T) {
	t.Parallel()
	d := newEmailTestDeps(t)

	d.userRepo.On("GetByID", mock.Anything, uuid.Nil).Return(nil, apperr.ErrUserNotFound)

	err := d.createUseCase().ResendVerification(context.Background(), uuid.Nil)
	assert.ErrorIs(t, err, apperr.ErrUserNotFound)
}

func TestEmailUseCase_ResendVerificationByEmail_Success(t *testing.T) {
	t.Parallel()
	d := newEmailTestDeps(t)

	user := newTestUser(uuid.New(), "testuser", "test@example.com")
	user.IsVerified = false

	d.setupTxRun()
	d.userRepo.On("GetByEmail", mock.Anything, user.Email).Return(user, nil)
	d.tokenRepo.On("DeleteByUserAndType", mock.Anything, user.ID, domain.TokenTypeEmailVerification).Return(nil)
	d.tokenRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	d.mailer.On("Send", mock.Anything, mock.Anything).Return(nil)

	err := d.createUseCase().ResendVerificationByEmail(context.Background(), user.Email)
	assert.NoError(t, err)
}

func TestEmailUseCase_ResendVerificationByEmail_UserNotFound(t *testing.T) {
	t.Parallel()
	d := newEmailTestDeps(t)

	d.userRepo.On("GetByEmail", mock.Anything, "unknown@example.com").Return(nil, apperr.ErrUserNotFound)

	err := d.createUseCase().ResendVerificationByEmail(context.Background(), "unknown@example.com")
	assert.NoError(t, err, "must return nil to prevent user enumeration")
}

func TestEmailUseCase_ResendVerificationByEmail_AlreadyVerified(t *testing.T) {
	t.Parallel()
	d := newEmailTestDeps(t)

	user := newTestUser(uuid.New(), "testuser", "verified@example.com")
	user.IsVerified = true

	d.userRepo.On("GetByEmail", mock.Anything, user.Email).Return(user, nil)

	err := d.createUseCase().ResendVerificationByEmail(context.Background(), user.Email)
	assert.NoError(t, err, "must return nil without sending when already verified")
}
