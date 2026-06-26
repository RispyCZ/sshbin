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

The `storage.Storage` interface (`internal/storage/storage.go`) has two methods:

```go
Create(ctx, id, name string) (io.WriteCloser, error)
Open(ctx, id, name string)   (io.ReadSeekCloser, error)
```

The backend is selected at startup via the `-storage` flag, which accepts a DSN URL:

| DSN | Backend |
|-----|---------|
| `local://./uploads` | Local filesystem (default) |
| `s3://my-bucket/prefix` | S3-compatible object storage |

A bare path (e.g. `uploads`) is treated as `local://uploads` for backward compatibility.

### Local backend

Files land at `{base}/{id}/{name}`. Reads return an `*os.File`, which is natively seekable — `http.ServeContent` uses this for byte-range responses.

### S3 backend

Implemented in `internal/storage/s3.go`. Object key layout: `{prefix}/{id}/{name}` (prefix is optional).

**Cloudflare R2 and other S3-compatible stores** work via the standard `AWS_ENDPOINT_URL` environment variable, which `aws-sdk-go-v2` picks up automatically.

**Seekable reads:** `Open` issues a `HeadObject` to learn the file size, then returns an `s3ReadSeekCloser`. This wrapper issues `GetObject` with a `Range: bytes=N-` header lazily on `Read`, and closes the stream on `Seek` so subsequent reads re-open from the new offset. This allows `http.ServeContent` to serve byte-range requests without buffering the whole object in memory.

**Uploads** use `io.Pipe` + a goroutine running `PutObject`: the caller writes to the pipe writer while the SDK streams bytes to S3. `Close()` on the writer blocks until `PutObject` completes and surfaces any upload error.

### Adding a new backend

Implement `storage.Storage` in a new file under `internal/storage/`, add a scheme case to the `Open` factory in `storage.go`, and add credentials/config loading as needed.
