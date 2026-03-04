package mailer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/resend/resend-go/v3"
)

type Message struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

type Config struct {
	APIKey    string
	FromEmail string
	FromName  string
}

type ResendMailer struct {
	client *resend.Client
	cfg    Config
}

func New(cfg Config) *ResendMailer {
	client := resend.NewClient(cfg.APIKey)

	return &ResendMailer{
		client: client,
		cfg:    cfg,
	}
}

func (m *ResendMailer) Send(ctx context.Context, msg Message) error {
	from := m.cfg.FromEmail
	if m.cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", m.cfg.FromName, m.cfg.FromEmail)
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
				return backoff.Permanent(fmt.Errorf("failed to send email via Resend: %w", err))
			}
			return fmt.Errorf("failed to send email via Resend: %w", err)
		}
		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0
	bo.InitialInterval = 2 * time.Second
	bo.Multiplier = 2

	if err := backoff.Retry(operation, backoff.WithContext(backoff.WithMaxRetries(bo, 3), ctx)); err != nil {
		return err
	}
	return nil
}

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
