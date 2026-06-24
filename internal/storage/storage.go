package storage

import (
	"context"
	"errors"
	"io"
)

var ErrNotFound = errors.New("file not found")

type Storage interface {
	Create(ctx context.Context, id string, name string) (io.WriteCloser, error)
	// Open returns a readable, seekable handle to a stored file. It returns
	// ErrNotFound when the file does not exist.
	Open(ctx context.Context, id string, name string) (io.ReadSeekCloser, error)
}
