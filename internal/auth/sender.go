package auth

import (
	"context"

	"github.com/charmbracelet/log"
)

// Sender delivers a one-time code to a destination (an email address for now).
// Real SMTP/SMS implementations satisfy the same interface.
type Sender interface {
	Send(ctx context.Context, dest, code string) error
}

// LogSender writes the code to the application log. Intended for development;
// never use it in production, as codes are exposed in logs.
type LogSender struct{}

func (LogSender) Send(_ context.Context, dest, code string) error {
	log.Info("auth: OTP", "dest", dest, "code", code)
	return nil
}
