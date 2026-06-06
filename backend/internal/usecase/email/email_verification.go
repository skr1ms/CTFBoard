package email

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/emailtemplate"
)

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
		subjectTmpl := uc.deps.ConfigUC.GetString(ctx, "mail_verification_subject", defaultVerificationSubject)
		bodyTmpl := uc.deps.ConfigUC.GetString(ctx, "mail_verification_body", defaultVerificationBody)
		subject := substitute(subjectTmpl, map[string]string{emailPlaceholderCTFName: ctfName, emailPlaceholderURL: verifyURL})
		body := substitute(bodyTmpl, map[string]string{emailPlaceholderCTFName: ctfName, emailPlaceholderURL: verifyURL})

		msg := Message{To: user.Email, Subject: subject, Body: body, IsHTML: false}

		err := uc.deps.Mailer.Send(ctx, msg)
		if err != nil {
			return fmt.Errorf("EmailUseCase - SendVerificationEmail - Mailer.Send: %w", err)
		}

		return nil
	}

	body, err := emailtemplate.RenderVerificationEmail(emailtemplate.VerificationData{
		Username:  user.Username,
		ActionURL: verifyURL,
		AppName:   defaultAppName,
	}, true)
	if err != nil {
		return fmt.Errorf("EmailUseCase - SendVerificationEmail - RenderVerificationEmail: %w", err)
	}

	msg := Message{
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
