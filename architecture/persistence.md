# Persistence

## Database

SQLite via `modernc.org/sqlite` (pure Go, no CGo). The schema is versioned with sequential migration files in `internal/sqlstore/migrations/`.

**Tables:**

| Table | Purpose |
|-------|---------|
| `shares` | Share metadata: file info, owner, visibility, expiry, password hash |
| `share_allowed_emails` | Per-share email allowlist for private shares |
| `sessions` | OTP auth sessions (token → email + expiry) |
| `app_meta` | Key/value store; holds the persistent grant-signing secret |

The `dialect` abstraction in `internal/sqlstore/db.go` keeps queries portable — placeholder rebinding is the only SQL dialect difference targeted so far, allowing future support for PostgreSQL, MySQL, or other standard SQL databases.

## File storage

Uploaded files are written to a local directory (default: `uploads/`). Each file gets a UUID subdirectory: `{storage}/{uuid}/{filename}`.

The `storage.Storage` interface (`internal/storage/storage.go`) has `Create` and `Open` methods. Only `LocalStorage` is implemented today; the interface is the extension point for S3, FTP, or other backends.
