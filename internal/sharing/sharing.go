package sharing

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrNotFound = errors.New("sharing not found")

type Sharing struct {
	ID        string
	FileID    string
	FileName  string
	CreatedAt time.Time

	// Configured is set once the uploader completes setup via the web UI.
	Configured bool
	// ExpiresAt is nil when the share never expires.
	ExpiresAt *time.Time
	// PasswordHash is a bcrypt hash; empty means no password is required.
	PasswordHash string
	// Public allows access without authentication when true.
	Public bool
}

func (s Sharing) Expired(now time.Time) bool {
	return s.ExpiresAt != nil && now.After(*s.ExpiresAt)
}

func (s Sharing) HasPassword() bool {
	return s.PasswordHash != ""
}

func (s Sharing) CheckPassword(pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(s.PasswordHash), []byte(pw)) == nil
}

func HashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

type Repository interface {
	Create(ctx context.Context, s Sharing) error
	Get(ctx context.Context, id string) (Sharing, error)
	Update(ctx context.Context, s Sharing) error
}
