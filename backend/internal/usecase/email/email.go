package email

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/crypto/bcrypt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/mailer"
)

// JWTRevoker is the subset of jwtkit.Service used to invalidate all active
// tokens for a user after a password reset.
type JWTRevoker interface {
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

const (
	emailTokenBytes = 32
	defaultAppName  = "AstroCTFb"
)

func substitute(s string, m map[string]string) string {
	for k, v := range m {
		s = strings.ReplaceAll(s, k, v)
	}

	return s
}

type EmailUseCase struct {
	deps EmailDeps
}

type ConfigGetter interface {
	GetString(ctx context.Context, key, defaultVal string) string
}

type EmailDeps struct {
	UserRepo    repo.UserRepository
	TokenRepo   repo.VerificationTokenRepository
	TM          repo.TransactionManager
	Mailer      mailer.Mailer
	ConfigUC    ConfigGetter
	JWTRevoker  JWTRevoker
	VerifyTTL   time.Duration
	ResetTTL    time.Duration
	FrontendURL string
	Enabled     bool
	Logger      logkit.Logger
	BcryptCost  int // 0 = bcrypt.DefaultCost; set to bcrypt.MinCost in tests
}

var _ usecase.EmailUseCase = (*EmailUseCase)(nil)

func NewEmailUseCase(deps EmailDeps) *EmailUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	return &EmailUseCase{deps: deps}
}

func (uc *EmailUseCase) IsEnabled() bool {
	return uc.deps.Enabled
}

// SendVerificationEmail generates a cryptographically random token, stores its
// SHA-256 hash in the verification token repository with the configured TTL
// (replacing any previously issued token for the same user and type), and sends
// an email containing the raw token embedded in a verification URL. The email
// body is rendered either from the configurable template strings fetched via
// ConfigUC or from the built-in HTML template. The function is a no-op when
// email sending is disabled.
func (uc *EmailUseCase) SendVerificationEmail(ctx context.Context, user *domain.User) error {
	if !uc.deps.Enabled {
		return nil
	}

	token, err := generateToken(emailTokenBytes)
	if err != nil {
		return fmt.Errorf("EmailUseCase - SendVerificationEmail - generateToken: %w", err)
	}

	hashedToken := hashToken(token)

	vt := &domain.VerificationToken{
		UserID:    user.ID,
		Token:     hashedToken,
		Type:      domain.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(uc.deps.VerifyTTL),
	}

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.TokenRepo.DeleteByUserAndType(ctx, user.ID, domain.TokenTypeEmailVerification); err != nil {
			return fmt.Errorf("EmailUseCase - SendVerificationEmail - TokenRepo.DeleteByUserAndType: %w", err)
		}

		if err := uc.deps.TokenRepo.Create(ctx, vt); err != nil {
			return fmt.Errorf("EmailUseCase - SendVerificationEmail - TokenRepo.Create: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("EmailUseCase - SendVerificationEmail: %w", err)
	}

	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", uc.deps.FrontendURL, token)
	ctfName := defaultAppName

	if uc.deps.ConfigUC != nil {
		ctfName = uc.deps.ConfigUC.GetString(ctx, "ctf_name", ctfName)
		subjectTmpl := uc.deps.ConfigUC.GetString(ctx, "mail_verification_subject", "Verify your email - {ctf_name}")
		bodyTmpl := uc.deps.ConfigUC.GetString(ctx, "mail_verification_body", "Follow the link to verify: {url}")
		subject := substitute(subjectTmpl, map[string]string{"{ctf_name}": ctfName, "{url}": verifyURL})
		body := substitute(bodyTmpl, map[string]string{"{ctf_name}": ctfName, "{url}": verifyURL})

		msg := mailer.Message{To: user.Email, Subject: subject, Body: body, IsHTML: false}

		err := uc.deps.Mailer.Send(ctx, msg)
		if err != nil {
			return fmt.Errorf("EmailUseCase - SendVerificationEmail - Mailer.Send: %w", err)
		}

		return nil
	}

	body, err := mailer.RenderVerificationEmail(mailer.VerificationData{
		Username:  user.Username,
		ActionURL: verifyURL,
		AppName:   defaultAppName,
	}, true)
	if err != nil {
		return fmt.Errorf("EmailUseCase - SendVerificationEmail - RenderVerificationEmail: %w", err)
	}

	msg := mailer.Message{
		To:      user.Email,
		Subject: "Verify your email - " + defaultAppName,
		Body:    body,
		IsHTML:  true,
	}

	if err := uc.deps.Mailer.Send(ctx, msg); err != nil {
		return fmt.Errorf("EmailUseCase - SendVerificationEmail - Mailer.Send: %w", err)
	}

	return nil
}

// VerifyEmail validates a raw verification token and marks the user's email as verified.
// The raw token is SHA-256 hashed before the repository lookup. The entire operation runs
// in a SERIALIZABLE transaction to prevent concurrent double-verification (two requests
// with the same token racing past the IsUsed check). The token is deleted on success
// so it cannot be replayed.
func (uc *EmailUseCase) VerifyEmail(ctx context.Context, tokenStr string) error {
	if tokenStr == "" {
		return apperr.ErrTokenRequired
	}

	hashedToken := hashToken(tokenStr)

	return uc.deps.TM.RunSerializable(ctx, func(ctx context.Context) error {
		token, err := uc.deps.TokenRepo.GetByToken(ctx, hashedToken)
		if err != nil {
			if errors.Is(err, apperr.ErrTokenNotFound) {
				return apperr.ErrTokenNotFound
			}

			return fmt.Errorf("EmailUseCase - VerifyEmail - TokenRepo.GetByToken: %w", err)
		}

		if token.Type != domain.TokenTypeEmailVerification {
			return apperr.ErrTokenNotFound
		}

		if token.IsExpired() {
			return apperr.ErrTokenExpired
		}

		if token.IsUsed() {
			return apperr.ErrTokenAlreadyUsed
		}

		if err := uc.deps.UserRepo.SetVerified(ctx, token.UserID); err != nil {
			return fmt.Errorf("EmailUseCase - VerifyEmail - UserRepo.SetVerified: %w", err)
		}

		if err := uc.deps.TokenRepo.DeleteByUserAndType(ctx, token.UserID, domain.TokenTypeEmailVerification); err != nil {
			return fmt.Errorf("EmailUseCase - VerifyEmail - TokenRepo.DeleteByUserAndType: %w", err)
		}

		return nil
	})
}

// SendPasswordResetEmail generates a password-reset token and sends the reset
// email to the given address. To prevent user enumeration the function returns
// nil without sending anything when the email address is not registered. It
// stores the SHA-256 hash of the token atomically in a transaction, replacing
// any existing reset token for the user, then dispatches the email with the raw
// token embedded in the reset URL. The function is a no-op when email sending is
// disabled.
func (uc *EmailUseCase) SendPasswordResetEmail(ctx context.Context, email string) error {
	if !uc.deps.Enabled {
		return nil
	}

	email = strings.ToLower(strings.TrimSpace(email))

	user, err := uc.deps.UserRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, apperr.ErrUserNotFound) {
			return nil
		}

		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - UserRepo.GetByEmail: %w", err)
	}

	token, err := generateToken(emailTokenBytes)
	if err != nil {
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - generateToken: %w", err)
	}

	hashedToken := hashToken(token)

	vt := &domain.VerificationToken{
		UserID:    user.ID,
		Token:     hashedToken,
		Type:      domain.TokenTypePasswordReset,
		ExpiresAt: time.Now().Add(uc.deps.ResetTTL),
	}
	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		err := uc.deps.TokenRepo.DeleteByUserAndType(ctx, user.ID, domain.TokenTypePasswordReset)
		if err != nil {
			return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - TokenRepo.DeleteByUserAndType: %w", err)
		}

		err = uc.deps.TokenRepo.Create(ctx, vt)
		if err != nil {
			return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - TokenRepo.Create: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail: %w", err)
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", uc.deps.FrontendURL, token)
	if uc.deps.ConfigUC != nil {
		ctfName := uc.deps.ConfigUC.GetString(ctx, "ctf_name", defaultAppName)
		subjectTmpl := uc.deps.ConfigUC.GetString(ctx, "mail_reset_subject", "Password reset - {ctf_name}")
		bodyTmpl := uc.deps.ConfigUC.GetString(ctx, "mail_reset_body", "Follow the link to reset password: {url}")
		subject := substitute(subjectTmpl, map[string]string{"{ctf_name}": ctfName, "{url}": resetURL})
		body := substitute(bodyTmpl, map[string]string{"{ctf_name}": ctfName, "{url}": resetURL})

		msg := mailer.Message{To: user.Email, Subject: subject, Body: body, IsHTML: false}

		err := uc.deps.Mailer.Send(ctx, msg)
		if err != nil {
			return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - Mailer.Send: %w", err)
		}

		return nil
	}

	body, err := mailer.RenderPasswordResetEmail(mailer.PasswordResetData{
		Username:  user.Username,
		ActionURL: resetURL,
		AppName:   defaultAppName,
	}, true)
	if err != nil {
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - RenderPasswordResetEmail: %w", err)
	}

	msg := mailer.Message{
		To:      user.Email,
		Subject: "Reset your password - " + defaultAppName,
		Body:    body,
		IsHTML:  true,
	}
	if err := uc.deps.Mailer.Send(ctx, msg); err != nil {
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - Mailer.Send: %w", err)
	}

	return nil
}

// ResetPassword validates the password-reset token and updates the user's
// password. The raw token is SHA-256 hashed before the repository lookup. The
// password hash is computed with bcrypt before the serializable transaction
// begins so that the slow hash does not hold the database transaction open.
// Inside the transaction it verifies the token type, expiry, and used status,
// updates the password, and deletes the consumed token. After the transaction
// all existing JWTs for the user are revoked so that any active sessions are
// invalidated.
func (uc *EmailUseCase) ResetPassword(ctx context.Context, tokenStr, newPassword string) error {
	if tokenStr == "" {
		return apperr.ErrTokenRequired
	}

	hashedToken := hashToken(tokenStr)

	bcryptCost := uc.deps.BcryptCost
	if bcryptCost == 0 {
		bcryptCost = bcrypt.DefaultCost
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("EmailUseCase - ResetPassword - GenerateFromPassword: %w", err)
	}

	var userID uuid.UUID

	if err := uc.deps.TM.RunSerializable(ctx, func(ctx context.Context) error {
		token, err := uc.deps.TokenRepo.GetByToken(ctx, hashedToken)
		if err != nil {
			if errors.Is(err, apperr.ErrTokenNotFound) {
				return apperr.ErrTokenNotFound
			}

			return fmt.Errorf("EmailUseCase - ResetPassword - TokenRepo.GetByToken: %w", err)
		}

		if token.Type != domain.TokenTypePasswordReset {
			return apperr.ErrTokenNotFound
		}

		if token.IsExpired() {
			return apperr.ErrTokenExpired
		}

		if token.IsUsed() {
			return apperr.ErrTokenAlreadyUsed
		}

		if err := uc.deps.UserRepo.UpdatePassword(ctx, token.UserID, string(passwordHash)); err != nil {
			return fmt.Errorf("EmailUseCase - ResetPassword - UserRepo.UpdatePassword: %w", err)
		}

		if err := uc.deps.TokenRepo.DeleteByUserAndType(ctx, token.UserID, domain.TokenTypePasswordReset); err != nil {
			return fmt.Errorf("EmailUseCase - ResetPassword - TokenRepo.DeleteByUserAndType: %w", err)
		}

		userID = token.UserID

		return nil
	}); err != nil {
		return fmt.Errorf("EmailUseCase - ResetPassword - TM.RunSerializable: %w", err)
	}

	// Revoke all active JWTs so stolen tokens cannot be used after a password reset.
	// Uses WithoutCancel so the revocation completes even if the request context is
	// cancelled immediately after the transaction commits.
	if uc.deps.JWTRevoker != nil {
		if err := uc.deps.JWTRevoker.RevokeAllForUser(context.WithoutCancel(ctx), userID); err != nil {
			uc.deps.Logger.WithError(err).Error("EmailUseCase - ResetPassword - RevokeAllForUser")
		}
	}

	return nil
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

// generateToken returns a cryptographically random hex string of the given byte
// length via crypto/rand. Used for verification and password-reset tokens.
func generateToken(length int) (string, error) {
	s, err := crypto.SecureRandomHex(length)
	if err != nil {
		return "", fmt.Errorf("EmailUseCase - generateToken: %w", err)
	}

	return s, nil
}

func hashToken(token string) string {
	return crypto.SHA256Hex(token)
}
