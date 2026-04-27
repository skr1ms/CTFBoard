package mailer

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/resend/resend-go/v3"
)

// Message holds the fields required to send a single email.
type Message struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

// Mailer is the interface satisfied by both ResendMailer and AsyncMailer.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// Config holds credentials and sender identity for the Resend email provider.
type Config struct {
	APIKey    string
	FromEmail string
	FromName  string
}

const mailerRetryInitialInterval = 2 * time.Second

// ResendMailer sends transactional email via the Resend API with exponential-backoff retry.
type ResendMailer struct {
	client *resend.Client
	cfg    Config
}

// New creates a ResendMailer using the provided configuration.
func New(cfg Config) *ResendMailer {
	client := resend.NewClient(cfg.APIKey)

	return &ResendMailer{
		client: client,
		cfg:    cfg,
	}
}

// Send delivers msg through the Resend API, retrying up to 3 times on transient errors.
func (m *ResendMailer) Send(ctx context.Context, msg Message) error {
	from := m.cfg.FromEmail
	if m.cfg.FromName != "" {
		from = (&mail.Address{Name: m.cfg.FromName, Address: m.cfg.FromEmail}).String()
	}

	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{msg.To},
		Subject: msg.Subject,
	}

	if msg.IsHTML {
		params.Html = msg.Body
	} else {
		params.Text = msg.Body
	}

	operation := func() error {
		_, err := m.client.Emails.SendWithContext(ctx, params)
		if err != nil {
			if isResendPermanentError(err) {
				return backoff.Permanent(fmt.Errorf("mailer.Send: %w", err))
			}

			return fmt.Errorf("mailer.Send: %w", err)
		}

		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0
	bo.InitialInterval = mailerRetryInitialInterval
	bo.Multiplier = 2

	return backoff.Retry(operation, backoff.WithContext(backoff.WithMaxRetries(bo, 3), ctx))
}

// isResendPermanentError returns true for HTTP 4xx status codes that should not be retried
// (400 bad request, 401 unauthorized, 403 forbidden, 404 not found, 422 unprocessable entity).
func isResendPermanentError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	for _, code := range []string{"400", "401", "403", "404", "422"} {
		if strings.Contains(msg, code) {
			return true
		}
	}

	return false
}
