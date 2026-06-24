package storage_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/rispycz/securedrop/internal/storage"
)

func TestLocalStorage_CreateAndRead(t *testing.T) {
	dir := t.TempDir()
	s := &storage.LocalStorage{BaseDir: dir}

	w, err := s.Create(context.Background(), "id1", "file.txt")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := io.WriteString(w, "hello"); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "id1", "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("content = %q, want %q", got, "hello")
	}
}

func TestLocalStorage_DuplicateID(t *testing.T) {
	dir := t.TempDir()
	s := &storage.LocalStorage{BaseDir: dir}

	w, err := s.Create(context.Background(), "id1", "file.txt")
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	w.Close()

	_, err = s.Create(context.Background(), "id1", "file.txt")
	if err == nil {
		t.Fatal("expected error on duplicate, got nil")
	}
}

func TestLocalStorage_MissingBaseDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	s := &storage.LocalStorage{BaseDir: dir}

	w, err := s.Create(context.Background(), "id1", "file.txt")
	if err != nil {
		t.Fatalf("Create with missing BaseDir: %v", err)
	}
	w.Close()

	if _, err := os.Stat(filepath.Join(dir, "id1", "file.txt")); err != nil {
		t.Errorf("file not created: %v", err)
	}
}
