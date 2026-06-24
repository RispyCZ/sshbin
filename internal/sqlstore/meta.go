package sqlstore

import (
	"crypto/rand"
	"database/sql"
	"errors"
)

const grantSecretKey = "grant_secret"

// EnsureSecret returns the persisted grant-signing secret, generating and
// storing a 32-byte one on first use so password-grant cookies survive restarts.
func (s *Store) EnsureSecret() ([]byte, error) {
	var value []byte
	err := s.db.QueryRow(s.dialect.Rebind(`SELECT value FROM app_meta WHERE key=?`), grantSecretKey).Scan(&value)
	if err == nil {
		return value, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(s.dialect.Rebind(`INSERT INTO app_meta (key, value) VALUES (?, ?)`), grantSecretKey, secret); err != nil {
		return nil, err
	}
	return secret, nil
}
