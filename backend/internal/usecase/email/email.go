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
	"golang.org/x/crypto/bcrypt"
)

type EmailUseCase struct {
	userRepo    repo.UserRepository
	tokenRepo   repo.VerificationTokenRepository
	mailer      mailer.Mailer
	verifyTTL   time.Duration
	resetTTL    time.Duration
	frontendURL string
	enabled     bool
}

func NewEmailUseCase(
	userRepo repo.UserRepository,
	tokenRepo repo.VerificationTokenRepository,
	mailer mailer.Mailer,
	verifyTTL time.Duration,
	resetTTL time.Duration,
	frontendURL string,
	enabled bool,
) *EmailUseCase {
	return &EmailUseCase{
		userRepo:    userRepo,
		tokenRepo:   tokenRepo,
		mailer:      mailer,
		verifyTTL:   verifyTTL,
		resetTTL:    resetTTL,
		frontendURL: frontendURL,
		enabled:     enabled,
	}
}

func (uc *EmailUseCase) IsEnabled() bool {
	return uc.enabled
}

func (uc *EmailUseCase) SendVerificationEmail(ctx context.Context, user *entity.User) error {
	if !uc.enabled {
		return nil
	}

	if err := uc.tokenRepo.DeleteByUserAndType(ctx, user.ID, entity.TokenTypeEmailVerification); err != nil {
		return fmt.Errorf("EmailUseCase - SendVerificationEmail - DeleteByUserAndType: %w", err)
	}

	token, err := generateToken(32)
	if err != nil {
		return fmt.Errorf("EmailUseCase - SendVerificationEmail - generateToken: %w", err)
	}

	hashedToken := hashToken(token)

	vt := &entity.VerificationToken{
		UserID:    user.ID,
		Token:     hashedToken,
		Type:      entity.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(uc.verifyTTL),
	}

	if err := uc.tokenRepo.Create(ctx, vt); err != nil {
		return fmt.Errorf("EmailUseCase - SendVerificationEmail - Create: %w", err)
	}

	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", uc.frontendURL, token)

	body, err := mailer.RenderVerificationEmail(mailer.VerificationData{
		Username:  user.Username,
		ActionURL: verifyURL,
		AppName:   "CTFBoard",
	}, true)
	if err != nil {
		return fmt.Errorf("EmailUseCase - SendVerificationEmail - RenderVerificationEmail: %w", err)
	}

	msg := mailer.Message{
		To:      user.Email,
		Subject: "Verify your email - CTFBoard",
		Body:    body,
		IsHTML:  true,
	}

	if err := uc.mailer.Send(ctx, msg); err != nil {
		return fmt.Errorf("EmailUseCase - SendVerificationEmail - Send: %w", err)
	}

	return nil
}

func (uc *EmailUseCase) VerifyEmail(ctx context.Context, tokenStr string) error {
	hashedToken := hashToken(tokenStr)
	token, err := uc.tokenRepo.GetByToken(ctx, hashedToken)
	if err != nil {
		if errors.Is(err, entityError.ErrTokenNotFound) {
			return entityError.ErrTokenNotFound
		}
		return fmt.Errorf("EmailUseCase - VerifyEmail - GetByToken: %w", err)
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

	if err := uc.userRepo.SetVerified(ctx, token.UserID); err != nil {
		return fmt.Errorf("EmailUseCase - VerifyEmail - SetVerified: %w", err)
	}

	if err := uc.tokenRepo.DeleteByUserAndType(ctx, token.UserID, entity.TokenTypeEmailVerification); err != nil {
		return fmt.Errorf("EmailUseCase - VerifyEmail - DeleteByUserAndType: %w", err)
	}

	return nil
}

func (uc *EmailUseCase) SendPasswordResetEmail(ctx context.Context, email string) error {
	if !uc.enabled {
		return nil
	}

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, entityError.ErrUserNotFound) {
			return nil
		}
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - GetByEmail: %w", err)
	}

	if err := uc.tokenRepo.DeleteByUserAndType(ctx, user.ID, entity.TokenTypePasswordReset); err != nil {
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - DeleteByUserAndType: %w", err)
	}

	token, err := generateToken(32)
	if err != nil {
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - generateToken: %w", err)
	}

	hashedToken := hashToken(token)

	vt := &entity.VerificationToken{
		UserID:    user.ID,
		Token:     hashedToken,
		Type:      entity.TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(uc.resetTTL),
	}

	if err := uc.tokenRepo.Create(ctx, vt); err != nil {
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - Create: %w", err)
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", uc.frontendURL, token)

	body, err := mailer.RenderPasswordResetEmail(mailer.PasswordResetData{
		Username:  user.Username,
		ActionURL: resetURL,
		AppName:   "CTFBoard",
	}, true)
	if err != nil {
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - RenderPasswordResetEmail: %w", err)
	}

	msg := mailer.Message{
		To:      user.Email,
		Subject: "Reset your password - CTFBoard",
		Body:    body,
		IsHTML:  true,
	}

	if err := uc.mailer.Send(ctx, msg); err != nil {
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - Send: %w", err)
	}

	return nil
}

func (uc *EmailUseCase) ResetPassword(ctx context.Context, tokenStr, newPassword string) error {
	hashedToken := hashToken(tokenStr)
	token, err := uc.tokenRepo.GetByToken(ctx, hashedToken)
	if err != nil {
		if errors.Is(err, entityError.ErrTokenNotFound) {
			return entityError.ErrTokenNotFound
		}
		return fmt.Errorf("EmailUseCase - ResetPassword - GetByToken: %w", err)
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
		return fmt.Errorf("EmailUseCase - ResetPassword - GenerateFromPassword: %w", err)
	}

	if err := uc.userRepo.UpdatePassword(ctx, token.UserID, string(passwordHash)); err != nil {
		return fmt.Errorf("EmailUseCase - ResetPassword - UpdatePassword: %w", err)
	}

	if err := uc.tokenRepo.DeleteByUserAndType(ctx, token.UserID, entity.TokenTypePasswordReset); err != nil {
		return fmt.Errorf("EmailUseCase - ResetPassword - DeleteByUserAndType: %w", err)
	}

	return nil
}

func (uc *EmailUseCase) ResendVerification(ctx context.Context, userID uuid.UUID) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("EmailUseCase - ResendVerification - GetByID: %w", err)
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
