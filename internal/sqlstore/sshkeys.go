package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/rispycz/sshbin/internal/sshkeys"
)

// SSHKeysRepo implements sshkeys.Repository over database/sql.
type SSHKeysRepo struct {
	db      *sql.DB
	dialect dialect
}

func (r *SSHKeysRepo) ListByEmail(ctx context.Context, email string) ([]sshkeys.PublicKey, error) {
	rows, err := r.db.QueryContext(ctx, r.dialect.Rebind(`
		SELECT id, email, title, fingerprint, authorized_key, created_at, last_used_at
		FROM ssh_public_keys WHERE email=? ORDER BY created_at DESC`), email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []sshkeys.PublicKey
	for rows.Next() {
		k, err := scanKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *SSHKeysRepo) Add(ctx context.Context, k sshkeys.PublicKey) error {
	_, err := r.db.ExecContext(ctx, r.dialect.Rebind(`
		INSERT INTO ssh_public_keys (id, email, title, fingerprint, authorized_key, created_at, last_used_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`),
		k.ID, k.Email, k.Title, k.Fingerprint, k.AuthorizedKey, k.CreatedAt.Unix(), nullUnix(k.LastUsedAt))
	if isUniqueViolation(err) {
		return sshkeys.ErrDuplicate
	}
	return err
}

func (r *SSHKeysRepo) Delete(ctx context.Context, id, email string) error {
	res, err := r.db.ExecContext(ctx, r.dialect.Rebind(`DELETE FROM ssh_public_keys WHERE id=? AND email=?`), id, email)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sshkeys.ErrNotFound
	}
	return nil
}

func (r *SSHKeysRepo) DeleteByEmail(ctx context.Context, email string) error {
	_, err := r.db.ExecContext(ctx, r.dialect.Rebind(`DELETE FROM ssh_public_keys WHERE email=?`), email)
	return err
}

func (r *SSHKeysRepo) FindOwnerByMarshaledKey(ctx context.Context, marshaled []byte) (string, bool, error) {
	fp, err := sshkeys.FingerprintOfMarshaled(marshaled)
	if err != nil {
		return "", false, err
	}
	var email string
	err = r.db.QueryRowContext(ctx, r.dialect.Rebind(`SELECT email FROM ssh_public_keys WHERE fingerprint=?`), fp).Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return email, true, nil
}

func (r *SSHKeysRepo) TouchLastUsed(ctx context.Context, fingerprint string) error {
	_, err := r.db.ExecContext(ctx, r.dialect.Rebind(`UPDATE ssh_public_keys SET last_used_at=? WHERE fingerprint=?`),
		time.Now().Unix(), fingerprint)
	return err
}

func scanKey(rows *sql.Rows) (sshkeys.PublicKey, error) {
	var (
		k          sshkeys.PublicKey
		createdAt  int64
		lastUsedAt sql.NullInt64
	)
	if err := rows.Scan(&k.ID, &k.Email, &k.Title, &k.Fingerprint, &k.AuthorizedKey, &createdAt, &lastUsedAt); err != nil {
		return sshkeys.PublicKey{}, err
	}
	k.CreatedAt = time.Unix(createdAt, 0).UTC()
	if lastUsedAt.Valid {
		t := time.Unix(lastUsedAt.Int64, 0).UTC()
		k.LastUsedAt = &t
	}
	return k, nil
}

// isUniqueViolation reports whether err is a UNIQUE-constraint failure. The
// modernc sqlite driver surfaces these as an error string; matching on it keeps
// the repo driver-agnostic without importing the driver's error types.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(strings.ToUpper(err.Error()), "UNIQUE CONSTRAINT")
}
