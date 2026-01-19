package mailer

import (
	"context"
	"fmt"

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

	_, err := m.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to send email via Resend: %w", err)
	}

	return nil
}
