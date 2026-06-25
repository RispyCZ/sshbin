package sqlstore

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// Store is a database/sql-backed persistence layer. SQL is kept portable;
// dialect differences are isolated in dialect.go so other engines can be added
// by extending the DSN scheme handling and dialect.
type Store struct {
	db      *sql.DB
	dialect dialect
}

// Open parses a DSN of the form "<scheme>://<target>" and returns a Store with
// migrations applied. Only "sqlite" is wired today, e.g.
// "sqlite://sshbin.db" or "sqlite://:memory:".
func Open(dsn string) (*Store, error) {
	scheme, target, ok := strings.Cut(dsn, "://")
	if !ok {
		return nil, fmt.Errorf("invalid DSN %q: want scheme://target", dsn)
	}

	var driver string
	var dia dialect
	switch scheme {
	case "sqlite", "sqlite3":
		driver, dia = "sqlite", sqliteDialect{}
	default:
		return nil, fmt.Errorf("unsupported database scheme %q", scheme)
	}

	db, err := sql.Open(driver, target)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	s := &Store{db: db, dialect: dia}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Shares returns a sharing.Repository backed by this store.
func (s *Store) Shares() *ShareRepo { return &ShareRepo{db: s.db, dialect: s.dialect} }

// Sessions returns an auth.SessionStore backed by this store.
func (s *Store) Sessions() *SessionStore { return &SessionStore{db: s.db, dialect: s.dialect} }
