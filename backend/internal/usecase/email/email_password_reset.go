package email

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/emailaddr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
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

	email = emailaddr.Normalize(email)

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
// token is checked before bcrypt so invalid public requests cannot force
// expensive password hashing. The token is checked again inside the
// serializable transaction before the password is updated and consumed.
func (uc *EmailUseCase) ResetPassword(ctx context.Context, tokenStr, newPassword string) error {
	if tokenStr == "" {
		return apperr.ErrTokenRequired
	}

	if uc.deps.ConfigUC != nil {
		minLen := uc.deps.ConfigUC.GetInt(ctx, "password_min_length", defaultPasswordMinLength)
		if minLen > 0 && len(newPassword) < minLen {
			return apperr.NewValidationErrorf("password must be at least %d characters", minLen)
		}
	}

	hashedToken := hashToken(tokenStr)

	if _, err := uc.getUsablePasswordResetToken(ctx, hashedToken); err != nil {
		return fmt.Errorf("EmailUseCase - ResetPassword - getUsablePasswordResetToken: %w", err)
	}

	passwordHash, err := uc.hashPassword(ctx, newPassword)
	if err != nil {
		return fmt.Errorf("EmailUseCase - ResetPassword - hashPassword: %w", err)
	}

	var userID uuid.UUID

	if err := uc.deps.TM.RunSerializable(ctx, func(ctx context.Context) error {
		token, err := uc.getUsablePasswordResetToken(ctx, hashedToken)
		if err != nil {
			return fmt.Errorf("EmailUseCase - ResetPassword - getUsablePasswordResetToken (tx): %w", err)
		}

		if err := uc.deps.UserRepo.UpdatePassword(ctx, token.UserID, passwordHash); err != nil {
			return fmt.Errorf("EmailUseCase - ResetPassword - UserRepo.UpdatePassword: %w", err)
		}

		if uc.deps.APITokenRevoker != nil {
			if err := uc.deps.APITokenRevoker.RevokeAllForUser(ctx, token.UserID); err != nil {
				return fmt.Errorf("EmailUseCase - ResetPassword - APITokenRevoker.RevokeAllForUser: %w", err)
			}
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
		txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
			if err := uc.deps.JWTRevoker.RevokeAllForUser(ctx, userID); err != nil {
				uc.deps.Logger.WithError(err).Error("EmailUseCase - ResetPassword - RevokeAllForUser")
			}
		})
	}

	return nil
}

func (uc *EmailUseCase) getUsablePasswordResetToken(ctx context.Context, hashedToken string) (*domain.VerificationToken, error) {
	token, err := uc.deps.TokenRepo.GetByToken(ctx, hashedToken)
	if err != nil {
		if errors.Is(err, apperr.ErrTokenNotFound) {
			return nil, apperr.ErrTokenNotFound
		}

		return nil, fmt.Errorf("EmailUseCase - getUsablePasswordResetToken - TokenRepo.GetByToken: %w", err)
	}

	if token.Type != domain.TokenTypePasswordReset {
		return nil, apperr.ErrTokenNotFound
	}

	if token.IsExpired() {
		return nil, apperr.ErrTokenExpired
	}

	if token.IsUsed() {
		return nil, apperr.ErrTokenAlreadyUsed
	}

	return token, nil
}

// ResetPasswordRateLimitKey returns the non-reversible key used by the transport
// rate limiter for password-reset token guesses.
func (uc *EmailUseCase) ResetPasswordRateLimitKey(tokenStr string) string {
	return hashToken(tokenStr)
}
