package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

type captureSender struct{ lastDest, lastCode string }

func (c *captureSender) Send(_ context.Context, dest, code string) error {
	c.lastDest, c.lastCode = dest, code
	return nil
}

func newManager(s Sender) *Manager {
	return NewManager(s, Options{MaxAttempts: 3})
}

func TestVerify_HappyPath(t *testing.T) {
	cs := &captureSender{}
	m := newManager(cs)

	if err := m.StartLogin(context.Background(), "User@Example.com "); err != nil {
		t.Fatalf("StartLogin: %v", err)
	}
	if cs.lastDest != "user@example.com" {
		t.Errorf("dest = %q, want normalized", cs.lastDest)
	}

	token, sess, err := m.Verify("user@example.com", cs.lastCode)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if sess.Email != "user@example.com" {
		t.Errorf("session email = %q", sess.Email)
	}
	got, ok := m.Session(token)
	if !ok || got.Email != "user@example.com" {
		t.Errorf("Session lookup failed: %+v ok=%v", got, ok)
	}
}

func TestVerify_WrongCode(t *testing.T) {
	cs := &captureSender{}
	m := newManager(cs)
	m.StartLogin(context.Background(), "a@b.com")

	if _, _, err := m.Verify("a@b.com", "000000"); !errors.Is(err, ErrInvalidCode) {
		// guard against accidentally guessing the real code
		if cs.lastCode != "000000" {
			t.Fatalf("err = %v, want ErrInvalidCode", err)
		}
	}
}

func TestVerify_TooManyAttempts(t *testing.T) {
	cs := &captureSender{}
	m := newManager(cs)
	m.StartLogin(context.Background(), "a@b.com")

	wrong := "111111"
	if cs.lastCode == wrong {
		wrong = "222222"
	}
	for range 3 {
		m.Verify("a@b.com", wrong)
	}
	if _, _, err := m.Verify("a@b.com", cs.lastCode); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("err = %v, want ErrTooManyAttempts", err)
	}
}

func TestVerify_Expired(t *testing.T) {
	cs := &captureSender{}
	m := NewManager(cs, Options{OTPTTL: time.Minute})
	now := time.Now()
	m.now = func() time.Time { return now }
	m.StartLogin(context.Background(), "a@b.com")

	m.now = func() time.Time { return now.Add(2 * time.Minute) }
	if _, _, err := m.Verify("a@b.com", cs.lastCode); !errors.Is(err, ErrChallengeExpired) {
		t.Fatalf("err = %v, want ErrChallengeExpired", err)
	}
}

func TestSession_ExpiryAndLogout(t *testing.T) {
	cs := &captureSender{}
	m := NewManager(cs, Options{SessionTTL: time.Hour})
	now := time.Now()
	m.now = func() time.Time { return now }
	m.StartLogin(context.Background(), "a@b.com")
	token, _, _ := m.Verify("a@b.com", cs.lastCode)

	m.Logout(token)
	if _, ok := m.Session(token); ok {
		t.Error("session should be gone after logout")
	}

	// fresh session, then expire by clock
	m.StartLogin(context.Background(), "a@b.com")
	token2, _, _ := m.Verify("a@b.com", cs.lastCode)
	m.now = func() time.Time { return now.Add(2 * time.Hour) }
	if _, ok := m.Session(token2); ok {
		t.Error("expired session should not be valid")
	}
}
