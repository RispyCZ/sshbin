package sharing

import (
	"context"
	"errors"
	"strings"
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
	// OwnerEmail is the verified email of the creator who claimed the share.
	OwnerEmail string
	// ExpiresAt is nil when the share never expires.
	ExpiresAt *time.Time
	// PasswordHash is a bcrypt hash; empty means no password is required.
	PasswordHash string
	// Public allows access without a session when true. When false (private),
	// a consumer needs a verified session whose email is in AllowedEmails.
	Public bool
	// AllowedEmails is the consumer allowlist for private shares (normalized,
	// lowercased). Ignored when Public is true.
	AllowedEmails []string
}

// AllowsEmail reports whether email may access a private share.
func (s Sharing) AllowsEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	for _, e := range s.AllowedEmails {
		if e == email {
			return true
		}
	}
	return false
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

// ParseEmails splits a free-form list (newline, comma, or space separated) into
// normalized, deduplicated lowercase addresses.
func ParseEmails(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ' ' || r == '\t' || r == ';'
	})
	seen := make(map[string]struct{}, len(fields))
	var out []string
	for _, f := range fields {
		e := strings.ToLower(strings.TrimSpace(f))
		if e == "" {
			continue
		}
		if _, dup := seen[e]; dup {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out
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
	ListByOwner(ctx context.Context, email string) ([]Sharing, error)
	Delete(ctx context.Context, id string) error
}
