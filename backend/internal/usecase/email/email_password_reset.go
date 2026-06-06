package email

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/ctxutil"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/emailtemplate"
)

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
		subjectTmpl := uc.deps.ConfigUC.GetString(ctx, "mail_reset_subject", defaultPasswordResetSubject)
		bodyTmpl := uc.deps.ConfigUC.GetString(ctx, "mail_reset_body", defaultPasswordResetBody)
		subject := substitute(subjectTmpl, map[string]string{emailPlaceholderCTFName: ctfName, emailPlaceholderURL: resetURL})
		body := substitute(bodyTmpl, map[string]string{emailPlaceholderCTFName: ctfName, emailPlaceholderURL: resetURL})

		msg := Message{To: user.Email, Subject: subject, Body: body, IsHTML: false}

		err := uc.deps.Mailer.Send(ctx, msg)
		if err != nil {
			return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - Mailer.Send: %w", err)
		}

		return nil
	}

	body, err := emailtemplate.RenderPasswordResetEmail(emailtemplate.PasswordResetData{
		Username:  user.Username,
		ActionURL: resetURL,
		AppName:   defaultAppName,
	}, true)
	if err != nil {
		return fmt.Errorf("EmailUseCase - SendPasswordResetEmail - RenderPasswordResetEmail: %w", err)
	}

	msg := Message{
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
	// The post-commit context survives request cancellation but stays deadline-bound.
	if uc.deps.JWTRevoker != nil {
		postCtx, postCancel := ctxutil.PostCommitContext(ctx)
		defer postCancel()

		if err := uc.deps.JWTRevoker.RevokeAllForUser(postCtx, userID); err != nil {
			uc.deps.Logger.WithError(err).Error("EmailUseCase - ResetPassword - RevokeAllForUser")
		}
	}

	return nil
}

// ResetPasswordRateLimitKey returns the non-reversible key used by the transport
// rate limiter for password-reset token guesses.
func (uc *EmailUseCase) ResetPasswordRateLimitKey(tokenStr string) string {
	return hashToken(tokenStr)
}
