package storage_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/rispycz/sshbin/internal/storage"
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

func TestLocalStorage_OpenRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := &storage.LocalStorage{BaseDir: dir}

	w, _ := s.Create(context.Background(), "id1", "file.txt")
	io.WriteString(w, "payload")
	w.Close()

	rc, err := s.Open(context.Background(), "id1", "file.txt")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()
	got, _ := io.ReadAll(rc)
	if string(got) != "payload" {
		t.Errorf("read = %q, want %q", got, "payload")
	}
}

func TestLocalStorage_OpenMissing(t *testing.T) {
	s := &storage.LocalStorage{BaseDir: t.TempDir()}
	_, err := s.Open(context.Background(), "nope", "x.txt")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestLocalStorage_Delete(t *testing.T) {
	dir := t.TempDir()
	s := &storage.LocalStorage{BaseDir: dir}
	ctx := context.Background()

	w, _ := s.Create(ctx, "id1", "file.txt")
	w.Close()

	if err := s.Delete(ctx, "id1", "file.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Open(ctx, "id1", "file.txt"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("Open after Delete: %v, want ErrNotFound", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "id1")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("id dir still present: %v", err)
	}
}

func TestLocalStorage_DeleteMissing_NoError(t *testing.T) {
	s := &storage.LocalStorage{BaseDir: t.TempDir()}
	if err := s.Delete(context.Background(), "nope", "x.txt"); err != nil {
		t.Fatalf("Delete missing: %v, want nil", err)
	}
}

func TestLocalStorage_List(t *testing.T) {
	dir := t.TempDir()
	s := &storage.LocalStorage{BaseDir: dir}
	ctx := context.Background()

	for _, b := range []struct{ id, name string }{{"id1", "a.txt"}, {"id2", "b.bin"}} {
		w, _ := s.Create(ctx, b.id, b.name)
		w.Close()
	}

	blobs, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	got := map[string]string{}
	for _, b := range blobs {
		got[b.ID] = b.Name
		if b.ModTime.IsZero() {
			t.Errorf("blob %s has zero ModTime", b.ID)
		}
	}
	if got["id1"] != "a.txt" || got["id2"] != "b.bin" {
		t.Errorf("List = %v, want id1/a.txt id2/b.bin", got)
	}
}

func TestLocalStorage_List_MissingBaseDir(t *testing.T) {
	s := &storage.LocalStorage{BaseDir: filepath.Join(t.TempDir(), "nonexistent")}
	blobs, err := s.List(context.Background())
	if err != nil {
		t.Fatalf("List missing BaseDir: %v", err)
	}
	if len(blobs) != 0 {
		t.Errorf("List = %v, want empty", blobs)
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
