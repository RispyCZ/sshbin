package sqlstore

// dialect isolates the SQL differences between engines. Queries are written with
// '?' placeholders and an "ON CONFLICT" upsert (SQLite/Postgres syntax); a new
// engine implements this interface to translate as needed (e.g. '?' -> $N).
type dialect interface {
	// Rebind rewrites '?' placeholders for the target engine.
	Rebind(query string) string
}

type sqliteDialect struct{}

// Rebind is a no-op for SQLite, which uses '?' placeholders natively.
func (sqliteDialect) Rebind(query string) string { return query }
