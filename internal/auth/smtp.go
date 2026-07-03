package auth

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/wneessen/go-mail"
)

// SMTPConfig configures an SMTPSender. Port 465 selects implicit TLS; any other
// port uses STARTTLS. Username/Password are optional (omit for open relays).
type SMTPConfig struct {
	Host     string
	Username string
	Password string
	From     string
	Port     int
	// InsecureSkipVerify disables TLS certificate verification. Dev only:
	// allows self-signed certs, never use in production.
	InsecureSkipVerify bool
}

// SMTPSender delivers one-time codes over SMTP.
type SMTPSender struct {
	client *mail.Client
	from   string
}

func NewSMTPSender(cfg SMTPConfig) (*SMTPSender, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("smtp: host is required")
	}
	if cfg.From == "" {
		return nil, fmt.Errorf("smtp: from address is required")
	}

	opts := []mail.Option{mail.WithPort(cfg.Port)}
	if cfg.Username != "" {
		opts = append(opts,
			mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
			mail.WithUsername(cfg.Username),
			mail.WithPassword(cfg.Password),
		)
	}
	if cfg.Port == 465 {
		opts = append(opts, mail.WithSSL())
	}
	if cfg.InsecureSkipVerify {
		opts = append(opts, mail.WithTLSConfig(&tls.Config{InsecureSkipVerify: true, ServerName: cfg.Host})) //nolint:gosec // dev-only, gated behind explicit flag
	}

	client, err := mail.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("smtp: new client: %w", err)
	}
	return &SMTPSender{client: client, from: cfg.From}, nil
}

func (s *SMTPSender) Send(ctx context.Context, dest, code string) error {
	msg, err := s.buildMessage(dest, code)
	if err != nil {
		return err
	}
	return s.client.DialAndSendWithContext(ctx, msg)
}

func (s *SMTPSender) buildMessage(dest, code string) (*mail.Msg, error) {
	msg := mail.NewMsg()
	if err := msg.From(s.from); err != nil {
		return nil, fmt.Errorf("smtp: from %q: %w", s.from, err)
	}
	if err := msg.To(dest); err != nil {
		return nil, fmt.Errorf("smtp: to %q: %w", dest, err)
	}
	msg.Subject("Your sshbin sign-in code")
	msg.SetBodyString(mail.TypeTextPlain,
		fmt.Sprintf("Your sshbin sign-in code is %s.\n\nIt expires in 10 minutes. If you did not request it, ignore this email.", code))
	return msg, nil
}
