package sqlstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/rispycz/sshbin/internal/userprefs"
)

// UserPrefsRepo implements userprefs.Repository over database/sql.
type UserPrefsRepo struct {
	db      *sql.DB
	dialect dialect
}

func (r *UserPrefsRepo) Get(ctx context.Context, email string) (userprefs.UserPrefs, error) {
	var defaultPublic int
	err := r.db.QueryRowContext(ctx, r.dialect.Rebind(`SELECT default_public FROM user_preferences WHERE email=?`), email).
		Scan(&defaultPublic)
	if errors.Is(err, sql.ErrNoRows) {
		return userprefs.UserPrefs{Email: email}, nil
	}
	if err != nil {
		return userprefs.UserPrefs{}, err
	}
	return userprefs.UserPrefs{Email: email, DefaultPublic: defaultPublic != 0}, nil
}

func (r *UserPrefsRepo) Upsert(ctx context.Context, prefs userprefs.UserPrefs) error {
	_, err := r.db.ExecContext(ctx, r.dialect.Rebind(`
		INSERT INTO user_preferences (email, default_public) VALUES (?, ?)
		ON CONFLICT (email) DO UPDATE SET default_public=excluded.default_public`),
		prefs.Email, boolToInt(prefs.DefaultPublic))
	return err
}

func (r *UserPrefsRepo) Delete(ctx context.Context, email string) error {
	_, err := r.db.ExecContext(ctx, r.dialect.Rebind(`DELETE FROM user_preferences WHERE email=?`), email)
	return err
}
