package sqlstore

import (
	"database/sql"
	"errors"
	"time"

	"github.com/rispycz/sshbin/internal/auth"
)

// SessionStore implements auth.SessionStore over database/sql.
type SessionStore struct {
	db      *sql.DB
	dialect dialect
}

func (s *SessionStore) Put(token string, sess auth.Session) error {
	_, err := s.db.Exec(s.dialect.Rebind(`
		INSERT INTO sessions (token, email, expires_at) VALUES (?, ?, ?)
		ON CONFLICT (token) DO UPDATE SET email=excluded.email, expires_at=excluded.expires_at`),
		token, sess.Email, sess.ExpiresAt.Unix())
	return err
}

func (s *SessionStore) Get(token string) (auth.Session, bool, error) {
	var (
		email     string
		expiresAt int64
	)
	err := s.db.QueryRow(s.dialect.Rebind(`SELECT email, expires_at FROM sessions WHERE token=?`), token).
		Scan(&email, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Session{}, false, nil
	}
	if err != nil {
		return auth.Session{}, false, err
	}
	if time.Now().After(time.Unix(expiresAt, 0)) {
		_ = s.Delete(token)
		return auth.Session{}, false, nil
	}
	return auth.Session{Email: email, ExpiresAt: time.Unix(expiresAt, 0).UTC()}, true, nil
}

func (s *SessionStore) Delete(token string) error {
	_, err := s.db.Exec(s.dialect.Rebind(`DELETE FROM sessions WHERE token=?`), token)
	return err
}
