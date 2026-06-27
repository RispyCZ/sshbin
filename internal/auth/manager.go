package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"

	"github.com/charmbracelet/log"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidCode      = errors.New("invalid code")
	ErrChallengeExpired = errors.New("code expired or not requested")
	ErrTooManyAttempts  = errors.New("too many attempts")
)

type Session struct {
	Email     string
	ExpiresAt time.Time
}

type challenge struct {
	codeHash  [32]byte
	expiresAt time.Time
	attempts  int
}

type Options struct {
	OTPTTL      time.Duration
	SessionTTL  time.Duration
	MaxAttempts int
}

func defaults(o Options) Options {
	if o.OTPTTL == 0 {
		o.OTPTTL = 10 * time.Minute
	}
	if o.SessionTTL == 0 {
		o.SessionTTL = 24 * time.Hour
	}
	if o.MaxAttempts == 0 {
		o.MaxAttempts = 5
	}
	return o
}

// Manager issues one-time codes and manages verified sessions. Sessions live in
// a SessionStore; OTP challenges are kept in memory (short-lived).
type Manager struct {
	sender   Sender
	sessions SessionStore
	opts     Options
	now      func() time.Time

	mu         sync.Mutex
	challenges map[string]challenge // keyed by normalized email
}

func NewManager(sender Sender, sessions SessionStore, opts Options) *Manager {
	return &Manager{
		sender:     sender,
		sessions:   sessions,
		opts:       defaults(opts),
		now:        time.Now,
		challenges: make(map[string]challenge),
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// StartLogin generates a code for the email, stores the challenge, and sends it.
func (m *Manager) StartLogin(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	if email == "" {
		return fmt.Errorf("empty email")
	}
	code, err := generateCode()
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.challenges[email] = challenge{
		codeHash:  sha256.Sum256([]byte(code)),
		expiresAt: m.now().Add(m.opts.OTPTTL),
	}
	m.mu.Unlock()

	return m.sender.Send(ctx, email, code)
}

// Verify checks the code for the email and, on success, creates a session and
// returns its token.
func (m *Manager) Verify(email, code string) (string, Session, error) {
	email = normalizeEmail(email)

	m.mu.Lock()
	defer m.mu.Unlock()

	ch, ok := m.challenges[email]
	if !ok || m.now().After(ch.expiresAt) {
		delete(m.challenges, email)
		return "", Session{}, ErrChallengeExpired
	}
	if ch.attempts >= m.opts.MaxAttempts {
		delete(m.challenges, email)
		return "", Session{}, ErrTooManyAttempts
	}

	want := sha256.Sum256([]byte(code))
	if subtle.ConstantTimeCompare(ch.codeHash[:], want[:]) != 1 {
		ch.attempts++
		m.challenges[email] = ch
		return "", Session{}, ErrInvalidCode
	}

	delete(m.challenges, email)
	token, err := generateToken()
	if err != nil {
		return "", Session{}, err
	}
	sess := Session{Email: email, ExpiresAt: m.now().Add(m.opts.SessionTTL)}
	if err := m.sessions.Put(token, sess); err != nil {
		return "", Session{}, err
	}
	return token, sess, nil
}

// Session returns the session for a token if it exists and has not expired.
func (m *Manager) Session(token string) (Session, bool) {
	s, ok, err := m.sessions.Get(token)
	if err != nil {
		log.Error("auth: session lookup", "err", err)
		return Session{}, false
	}
	return s, ok
}

func (m *Manager) Logout(token string) {
	if err := m.sessions.Delete(token); err != nil {
		log.Error("auth: session delete", "err", err)
	}
}

func (m *Manager) DeleteSessionsByEmail(email string) error {
	return m.sessions.DeleteByEmail(email)
}

func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
