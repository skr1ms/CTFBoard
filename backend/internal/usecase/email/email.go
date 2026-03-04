package email

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/mailer"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const emailTokenBytes = 32

type EmailUseCase struct {
	deps EmailDeps
}

type EmailDeps struct {
	UserRepo    repo.UserRepository
	TokenRepo   repo.VerificationTokenRepository
	TM          repo.TransactionManager
	Mailer      mailer.Mailer
	VerifyTTL   time.Duration
	ResetTTL    time.Duration
	FrontendURL string
	Enabled     bool
}

var _ usecase.EmailUseCase = (*EmailUseCase)(nil)

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
		return fmt.Errorf("EmailUseCase - SendVerificationEmail - TokenRepo.DeleteByUserAndType: %w", err)
	}

	token, err := generateToken(emailTokenBytes)
	if err != nil {
		return fmt.Errorf("EmailUseCase - SendVerificationEmail - generateToken: %w", err)
	}

	hashedToken := hashToken(token)

	vt := &entity.VerificationToken{
		UserID:    user.ID,
		Token:     hashedToken,
		Type:      entity.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(uc.deps.VerifyTTL),
	}

	if err := uc.deps.TokenRepo.Create(ctx, vt); err != nil {
		return fmt.Errorf("EmailUseCase - SendVerificationEmail - TokenRepo.Create: %w", err)
	}

	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", uc.deps.FrontendURL, token)

	body, err := mailer.RenderVerificationEmail(mailer.VerificationData{
		Username:  user.Username,
		ActionURL: verifyURL,
		AppName:   "AstroCTFb",
	}, true)
	if err != nil {
		return fmt.Errorf("EmailUseCase - SendVerificationEmail - RenderVerificationEmail: %w", err)
	}

	msg := mailer.Message{
		To:      user.Email,
		Subject: "Verify your email - AstroCTFb",
		Body:    body,
		IsHTML:  true,
	}

	if err := uc.deps.Mailer.Send(ctx, msg); err != nil {
		return fmt.Errorf("EmailUseCase - SendVerificationEmail - Mailer.Send: %w", err)
	}

	return nil
}

func (uc *EmailUseCase) VerifyEmail(ctx context.Context, tokenStr string) error {
	if tokenStr == "" {
		return httperr.ErrTokenRequired
	}
	hashedToken := hashToken(tokenStr)

	return uc.deps.TM.RunSerializable(ctx, func(ctx context.Context) error {
		token, err := uc.deps.TokenRepo.GetByToken(ctx, hashedToken)
		if err != nil {
			if errors.Is(err, httperr.ErrTokenNotFound) {
				return httperr.ErrTokenNotFound
			}
			return fmt.Errorf("EmailUseCase - VerifyEmail - TokenRepo.GetByToken: %w", err)
		}

		if token.Type != entity.TokenTypeEmailVerification {
			return httperr.ErrTokenNotFound
		}

		if token.IsExpired() {
			return httperr.ErrTokenExpired
		}

		if token.IsUsed() {
			return httperr.ErrTokenAlreadyUsed
		}

		if err := uc.deps.UserRepo.SetVerified(ctx, token.UserID); err != nil {
			return fmt.Errorf("EmailUseCase - VerifyEmail - UserRepo.SetVerified: %w", err)
		}
		if err := uc.deps.TokenRepo.DeleteByUserAndType(ctx, token.UserID, entity.TokenTypeEmailVerification); err != nil {
			return fmt.Errorf("EmailUseCase - VerifyEmail - TokenRepo.DeleteByUserAndType: %w", err)
		}
		return nil
	})
}

func (uc *EmailUseCase) SendPasswordResetEmail(ctx context.Context, email string) error {
	if !uc.deps.Enabled {
		return nil
	}
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := uc.deps.UserRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, httperr.ErrUserNotFound) {
			return nil
		}
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - UserRepo.GetByEmail: %w", err)
	}

	if err := uc.deps.TokenRepo.DeleteByUserAndType(ctx, user.ID, entity.TokenTypePasswordReset); err != nil {
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - TokenRepo.DeleteByUserAndType: %w", err)
	}

	token, err := generateToken(emailTokenBytes)
	if err != nil {
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - generateToken: %w", err)
	}

	hashedToken := hashToken(token)

	vt := &entity.VerificationToken{
		UserID:    user.ID,
		Token:     hashedToken,
		Type:      entity.TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(uc.deps.ResetTTL),
	}

	if err := uc.deps.TokenRepo.Create(ctx, vt); err != nil {
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - TokenRepo.Create: %w", err)
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", uc.deps.FrontendURL, token)

	body, err := mailer.RenderPasswordResetEmail(mailer.PasswordResetData{
		Username:  user.Username,
		ActionURL: resetURL,
		AppName:   "AstroCTFb",
	}, true)
	if err != nil {
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - RenderPasswordResetEmail: %w", err)
	}

	msg := mailer.Message{
		To:      user.Email,
		Subject: "Reset your password - AstroCTFb",
		Body:    body,
		IsHTML:  true,
	}

	if err := uc.deps.Mailer.Send(ctx, msg); err != nil {
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - Mailer.Send: %w", err)
	}

	return nil
}

func (uc *EmailUseCase) ResetPassword(ctx context.Context, tokenStr, newPassword string) error {
	hashedToken := hashToken(tokenStr)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("EmailUseCase - ResetPassword - GenerateFromPassword: %w", err)
	}

	return uc.deps.TM.RunSerializable(ctx, func(ctx context.Context) error {
		token, err := uc.deps.TokenRepo.GetByToken(ctx, hashedToken)
		if err != nil {
			if errors.Is(err, httperr.ErrTokenNotFound) {
				return httperr.ErrTokenNotFound
			}
			return fmt.Errorf("EmailUseCase - ResetPassword - TokenRepo.GetByToken: %w", err)
		}

		if token.Type != entity.TokenTypePasswordReset {
			return httperr.ErrTokenNotFound
		}

		if token.IsExpired() {
			return httperr.ErrTokenExpired
		}

		if token.IsUsed() {
			return httperr.ErrTokenAlreadyUsed
		}

		if err := uc.deps.UserRepo.UpdatePassword(ctx, token.UserID, string(passwordHash)); err != nil {
			return fmt.Errorf("EmailUseCase - ResetPassword - UserRepo.UpdatePassword: %w", err)
		}
		if err := uc.deps.TokenRepo.DeleteByUserAndType(ctx, token.UserID, entity.TokenTypePasswordReset); err != nil {
			return fmt.Errorf("EmailUseCase - ResetPassword - TokenRepo.DeleteByUserAndType: %w", err)
		}
		return nil
	})
}

func (uc *EmailUseCase) ResendVerification(ctx context.Context, userID uuid.UUID) error {
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("EmailUseCase - ResendVerification - UserRepo.GetByID: %w", err)
	}

	if user.IsVerified {
		return nil
	}

	return uc.SendVerificationEmail(ctx, user)
}

func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("EmailUseCase - generateToken - rand.Read: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
