package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	BaseDir string
}

func (s *LocalStorage) Create(ctx context.Context, id string, name string) (io.WriteCloser, error) {
	dir := filepath.Join(s.BaseDir, id)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
}

func (s *LocalStorage) Open(ctx context.Context, id string, name string) (io.ReadSeekCloser, error) {
	f, err := os.Open(filepath.Join(s.BaseDir, id, filepath.Base(name)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return f, nil
}
