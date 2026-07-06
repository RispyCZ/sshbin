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

func NewLocal(dir string) *LocalStorage {
	return &LocalStorage{BaseDir: dir}
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

// Delete removes the per-upload directory. A missing directory is not an error.
func (s *LocalStorage) Delete(ctx context.Context, id string, name string) error {
	return os.RemoveAll(filepath.Join(s.BaseDir, id))
}

// List returns one BlobInfo per stored file. Each upload lives in its own
// <BaseDir>/<id>/ directory holding a single file. A missing BaseDir yields an
// empty list.
func (s *LocalStorage) List(ctx context.Context) ([]BlobInfo, error) {
	dirs, err := os.ReadDir(s.BaseDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var blobs []BlobInfo
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		id := d.Name()
		entries, err := os.ReadDir(filepath.Join(s.BaseDir, id))
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, err := e.Info()
			if err != nil {
				return nil, err
			}
			blobs = append(blobs, BlobInfo{ID: id, Name: e.Name(), ModTime: info.ModTime()})
		}
	}
	return blobs, nil
}
