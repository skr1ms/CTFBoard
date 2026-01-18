package mailer

import (
	"context"
	"crypto/tls"
	"fmt"

	"gopkg.in/gomail.v2"
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
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
	UseTLS    bool
}

type SMTPMailer struct {
	cfg    Config
	dialer *gomail.Dialer
}

func New(cfg Config) *SMTPMailer {
	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)

	if cfg.UseTLS {
		d.TLSConfig = &tls.Config{
			ServerName:         cfg.Host,
			InsecureSkipVerify: false,
		}
	} else {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &SMTPMailer{
		cfg:    cfg,
		dialer: d,
	}
}

func (m *SMTPMailer) Send(ctx context.Context, msg Message) error {
	from := m.cfg.FromEmail
	if m.cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", m.cfg.FromName, m.cfg.FromEmail)
	}

	gm := gomail.NewMessage()
	gm.SetHeader("From", from)
	gm.SetHeader("To", msg.To)
	gm.SetHeader("Subject", msg.Subject)

	contentType := "text/plain"
	if msg.IsHTML {
		contentType = "text/html"
	}
	gm.SetBody(contentType, msg.Body)

	if err := m.dialer.DialAndSend(gm); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
