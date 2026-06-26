package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
)

var ErrNotFound = errors.New("file not found")

// Storage persists uploaded files.
type Storage interface {
	Create(ctx context.Context, id string, name string) (io.WriteCloser, error)
	// Open returns a readable, seekable handle to a stored file. It returns
	// ErrNotFound when the file does not exist.
	Open(ctx context.Context, id string, name string) (io.ReadSeekCloser, error)
}

// Open parses a storage DSN and returns the appropriate Storage implementation.
// Supported schemes: local://<path>, s3://<bucket>/<prefix>.
// A bare path without a scheme is treated as local://<path>.
func Open(dsn string) (Storage, error) {
	if !strings.Contains(dsn, "://") {
		return NewLocal(dsn), nil
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "local":
		dir := u.Host + u.Path
		if dir == "" {
			dir = "."
		}
		return NewLocal(dir), nil
	case "s3":
		return newS3Storage(context.Background(), u)
	default:
		return nil, fmt.Errorf("unsupported storage scheme: %q", u.Scheme)
	}
}
