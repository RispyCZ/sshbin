package storage

import (
	"context"
	"io"
)

type Storage interface {
	Create(ctx context.Context, id string, name string) (io.WriteCloser, error)
}
