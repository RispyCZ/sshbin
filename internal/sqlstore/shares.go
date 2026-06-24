package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rispycz/securedrop/internal/sharing"
)

// ShareRepo implements sharing.Repository over database/sql.
type ShareRepo struct {
	db      *sql.DB
	dialect dialect
}

func (r *ShareRepo) Create(ctx context.Context, s sharing.Sharing) error {
	return r.tx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, r.dialect.Rebind(`
			INSERT INTO shares (id, file_id, file_name, created_at, configured, owner_email, expires_at, password_hash, public)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (id) DO UPDATE SET
				file_id=excluded.file_id, file_name=excluded.file_name, created_at=excluded.created_at,
				configured=excluded.configured, owner_email=excluded.owner_email, expires_at=excluded.expires_at,
				password_hash=excluded.password_hash, public=excluded.public`),
			s.ID, s.FileID, s.FileName, s.CreatedAt.Unix(), boolToInt(s.Configured),
			s.OwnerEmail, nullUnix(s.ExpiresAt), s.PasswordHash, boolToInt(s.Public))
		if err != nil {
			return err
		}
		return replaceEmails(ctx, tx, r.dialect, s.ID, s.AllowedEmails)
	})
}

func (r *ShareRepo) Update(ctx context.Context, s sharing.Sharing) error {
	return r.tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, r.dialect.Rebind(`
			UPDATE shares SET
				file_id=?, file_name=?, created_at=?, configured=?, owner_email=?,
				expires_at=?, password_hash=?, public=?
			WHERE id=?`),
			s.FileID, s.FileName, s.CreatedAt.Unix(), boolToInt(s.Configured), s.OwnerEmail,
			nullUnix(s.ExpiresAt), s.PasswordHash, boolToInt(s.Public), s.ID)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return sharing.ErrNotFound
		}
		return replaceEmails(ctx, tx, r.dialect, s.ID, s.AllowedEmails)
	})
}

func (r *ShareRepo) Get(ctx context.Context, id string) (sharing.Sharing, error) {
	var (
		s         sharing.Sharing
		createdAt int64
		expiresAt sql.NullInt64
		configured, public int
	)
	err := r.db.QueryRowContext(ctx, r.dialect.Rebind(`
		SELECT id, file_id, file_name, created_at, configured, owner_email, expires_at, password_hash, public
		FROM shares WHERE id=?`), id).
		Scan(&s.ID, &s.FileID, &s.FileName, &createdAt, &configured, &s.OwnerEmail, &expiresAt, &s.PasswordHash, &public)
	if errors.Is(err, sql.ErrNoRows) {
		return sharing.Sharing{}, sharing.ErrNotFound
	}
	if err != nil {
		return sharing.Sharing{}, err
	}
	s.CreatedAt = time.Unix(createdAt, 0).UTC()
	s.Configured = configured != 0
	s.Public = public != 0
	if expiresAt.Valid {
		t := time.Unix(expiresAt.Int64, 0).UTC()
		s.ExpiresAt = &t
	}

	emails, err := loadEmails(ctx, r.db, r.dialect, id)
	if err != nil {
		return sharing.Sharing{}, err
	}
	s.AllowedEmails = emails
	return s, nil
}

func (r *ShareRepo) tx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func replaceEmails(ctx context.Context, tx *sql.Tx, d dialect, shareID string, emails []string) error {
	if _, err := tx.ExecContext(ctx, d.Rebind(`DELETE FROM share_allowed_emails WHERE share_id=?`), shareID); err != nil {
		return err
	}
	for _, e := range emails {
		if _, err := tx.ExecContext(ctx, d.Rebind(`INSERT INTO share_allowed_emails (share_id, email) VALUES (?, ?)`), shareID, e); err != nil {
			return err
		}
	}
	return nil
}

func loadEmails(ctx context.Context, db *sql.DB, d dialect, shareID string) ([]string, error) {
	rows, err := db.QueryContext(ctx, d.Rebind(`SELECT email FROM share_allowed_emails WHERE share_id=? ORDER BY email`), shareID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullUnix(t *time.Time) sql.NullInt64 {
	if t == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: t.Unix(), Valid: true}
}
