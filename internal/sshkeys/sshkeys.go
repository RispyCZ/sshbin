package sshkeys

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	// ErrInvalidKey is returned when a pasted line is not a valid OpenSSH public key.
	ErrInvalidKey = errors.New("invalid public key")
	// ErrDuplicate is returned when a key with the same fingerprint is already registered.
	ErrDuplicate = errors.New("key already registered")
	// ErrNotFound is returned when a key does not exist or belongs to someone else.
	ErrNotFound = errors.New("key not found")
)

// PublicKey is a registered SSH public key owned by a web user (keyed by email).
type PublicKey struct {
	ID            string
	Email         string
	Title         string
	Fingerprint   string // SHA256 fingerprint, e.g. "SHA256:abc..."
	AuthorizedKey string // canonical "type base64" form, comment stripped
	CreatedAt     time.Time
	LastUsedAt    *time.Time
}

// Repository persists registered public keys. Fingerprints are globally unique:
// a given key belongs to exactly one owner.
type Repository interface {
	ListByEmail(ctx context.Context, email string) ([]PublicKey, error)
	// Add stores a key. It returns ErrDuplicate if the fingerprint is already registered.
	Add(ctx context.Context, k PublicKey) error
	// Delete removes a key by id, scoped to its owner. It returns ErrNotFound
	// when no matching key exists for that email.
	Delete(ctx context.Context, id, email string) error
	DeleteByEmail(ctx context.Context, email string) error
	// FindOwnerByMarshaledKey resolves the owner email for a wire-marshaled key
	// (as offered during the SSH handshake). found is false when unregistered.
	FindOwnerByMarshaledKey(ctx context.Context, marshaled []byte) (email string, found bool, err error)
	// TouchLastUsed records that a fingerprint authenticated an upload. Best-effort.
	TouchLastUsed(ctx context.Context, fingerprint string) error
}

// Parse validates a pasted authorized_keys line and returns its canonical
// "type base64" form (comment stripped) and SHA256 fingerprint.
func Parse(line string) (authorizedKey, fingerprint string, err error) {
	if strings.TrimSpace(line) == "" {
		return "", "", ErrInvalidKey
	}
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	return Canonical(pk), ssh.FingerprintSHA256(pk), nil
}

// Canonical returns the "type base64" authorized_keys form with no comment,
// so the same key pasted with different comments de-dupes on fingerprint.
func Canonical(pk ssh.PublicKey) string {
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pk)))
}

// FingerprintOfMarshaled computes the SHA256 fingerprint of a wire-marshaled key.
func FingerprintOfMarshaled(marshaled []byte) (string, error) {
	pk, err := ssh.ParsePublicKey(marshaled)
	if err != nil {
		return "", err
	}
	return ssh.FingerprintSHA256(pk), nil
}
