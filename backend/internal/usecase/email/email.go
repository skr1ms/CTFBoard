package email

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/mailer"
	"github.com/skr1ms/CTFBoard/pkg/usecaseutil"
	"golang.org/x/crypto/bcrypt"
)

type EmailDeps struct {
	UserRepo    repo.UserRepository
	TokenRepo   repo.VerificationTokenRepository
	Mailer      mailer.Mailer
	VerifyTTL   time.Duration
	ResetTTL    time.Duration
	FrontendURL string
	Enabled     bool
}

type EmailUseCase struct {
	deps EmailDeps
}

func NewEmailUseCase(deps EmailDeps) *EmailUseCase {
	return &EmailUseCase{deps: deps}
}

func (uc *EmailUseCase) IsEnabled() bool {
	return uc.deps.Enabled
}

func (uc *EmailUseCase) SendVerificationEmail(ctx context.Context, user *entity.User) error {
	if !uc.deps.Enabled {
		return nil
	}

	if err := uc.deps.TokenRepo.DeleteByUserAndType(ctx, user.ID, entity.TokenTypeEmailVerification); err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - SendVerificationEmail - DeleteByUserAndType")
	}

	token, err := generateToken(32)
	if err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - SendVerificationEmail - generateToken")
	}

	hashedToken := hashToken(token)

	vt := &entity.VerificationToken{
		UserID:    user.ID,
		Token:     hashedToken,
		Type:      entity.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(uc.deps.VerifyTTL),
	}

	if err := uc.deps.TokenRepo.Create(ctx, vt); err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - SendVerificationEmail - Create")
	}

	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", uc.deps.FrontendURL, token)

	body, err := mailer.RenderVerificationEmail(mailer.VerificationData{
		Username:  user.Username,
		ActionURL: verifyURL,
		AppName:   "CTFBoard",
	}, true)
	if err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - SendVerificationEmail - RenderVerificationEmail")
	}

	msg := mailer.Message{
		To:      user.Email,
		Subject: "Verify your email - CTFBoard",
		Body:    body,
		IsHTML:  true,
	}

	if err := uc.deps.Mailer.Send(ctx, msg); err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - SendVerificationEmail - Send")
	}

	return nil
}

func (uc *EmailUseCase) VerifyEmail(ctx context.Context, tokenStr string) error {
	hashedToken := hashToken(tokenStr)
	token, err := uc.deps.TokenRepo.GetByToken(ctx, hashedToken)
	if err != nil {
		if errors.Is(err, entityError.ErrTokenNotFound) {
			return entityError.ErrTokenNotFound
		}
		return usecaseutil.Wrap(err, "EmailUseCase - VerifyEmail - GetByToken")
	}

	if token.Type != entity.TokenTypeEmailVerification {
		return entityError.ErrTokenNotFound
	}

	if token.IsExpired() {
		return entityError.ErrTokenExpired
	}

	if token.IsUsed() {
		return entityError.ErrTokenAlreadyUsed
	}

	if err := uc.deps.UserRepo.SetVerified(ctx, token.UserID); err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - VerifyEmail - SetVerified")
	}

	if err := uc.deps.TokenRepo.DeleteByUserAndType(ctx, token.UserID, entity.TokenTypeEmailVerification); err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - VerifyEmail - DeleteByUserAndType")
	}

	return nil
}

func (uc *EmailUseCase) SendPasswordResetEmail(ctx context.Context, email string) error {
	if !uc.deps.Enabled {
		return nil
	}

	user, err := uc.deps.UserRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, entityError.ErrUserNotFound) {
			return nil
		}
		return usecaseutil.Wrap(err, "EmailUseCase - SendPasswordResetEmail - GetByEmail")
	}

	if err := uc.deps.TokenRepo.DeleteByUserAndType(ctx, user.ID, entity.TokenTypePasswordReset); err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - SendPasswordResetEmail - DeleteByUserAndType")
	}

	token, err := generateToken(32)
	if err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - SendPasswordResetEmail - generateToken")
	}

	hashedToken := hashToken(token)

	vt := &entity.VerificationToken{
		UserID:    user.ID,
		Token:     hashedToken,
		Type:      entity.TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(uc.deps.ResetTTL),
	}

	if err := uc.deps.TokenRepo.Create(ctx, vt); err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - SendPasswordResetEmail - Create")
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", uc.deps.FrontendURL, token)

	body, err := mailer.RenderPasswordResetEmail(mailer.PasswordResetData{
		Username:  user.Username,
		ActionURL: resetURL,
		AppName:   "CTFBoard",
	}, true)
	if err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - SendPasswordResetEmail - RenderPasswordResetEmail")
	}

	msg := mailer.Message{
		To:      user.Email,
		Subject: "Reset your password - CTFBoard",
		Body:    body,
		IsHTML:  true,
	}

	if err := uc.deps.Mailer.Send(ctx, msg); err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - SendPasswordResetEmail - Send")
	}

	return nil
}

func (uc *EmailUseCase) ResetPassword(ctx context.Context, tokenStr, newPassword string) error {
	hashedToken := hashToken(tokenStr)
	token, err := uc.deps.TokenRepo.GetByToken(ctx, hashedToken)
	if err != nil {
		if errors.Is(err, entityError.ErrTokenNotFound) {
			return entityError.ErrTokenNotFound
		}
		return usecaseutil.Wrap(err, "EmailUseCase - ResetPassword - GetByToken")
	}

	if token.Type != entity.TokenTypePasswordReset {
		return entityError.ErrTokenNotFound
	}

	if token.IsExpired() {
		return entityError.ErrTokenExpired
	}

	if token.IsUsed() {
		return entityError.ErrTokenAlreadyUsed
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - ResetPassword - GenerateFromPassword")
	}

	if err := uc.deps.UserRepo.UpdatePassword(ctx, token.UserID, string(passwordHash)); err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - ResetPassword - UpdatePassword")
	}

	if err := uc.deps.TokenRepo.DeleteByUserAndType(ctx, token.UserID, entity.TokenTypePasswordReset); err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - ResetPassword - DeleteByUserAndType")
	}

	return nil
}

func (uc *EmailUseCase) ResendVerification(ctx context.Context, userID uuid.UUID) error {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return usecaseutil.Wrap(err, "EmailUseCase - ResendVerification - GetByID")
	}

	if user.IsVerified {
		return nil
	}

	return uc.SendVerificationEmail(ctx, user)
}

func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
