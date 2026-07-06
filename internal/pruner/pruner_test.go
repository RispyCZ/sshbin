package pruner

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rispycz/sshbin/internal/sharing"
	"github.com/rispycz/sshbin/internal/storage"
)

func writeBlob(t *testing.T, s storage.Storage, id, name, body string) {
	t.Helper()
	w, err := s.Create(context.Background(), id, name)
	if err != nil {
		t.Fatalf("Create blob %s: %v", id, err)
	}
	io.WriteString(w, body)
	if err := w.Close(); err != nil {
		t.Fatalf("Close blob %s: %v", id, err)
	}
}

func exists(t *testing.T, s storage.Storage, id, name string) bool {
	t.Helper()
	rc, err := s.Open(context.Background(), id, name)
	if errors.Is(err, storage.ErrNotFound) {
		return false
	}
	if err != nil {
		t.Fatalf("Open %s: %v", id, err)
	}
	rc.Close()
	return true
}

func TestPrune_DeletesExpiredStaleAndOrphans(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewLocal(dir)
	repo := sharing.NewMemoryRepository()
	ctx := context.Background()
	now := time.Now()

	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	repo.Create(ctx, sharing.Sharing{ID: "expired", FileID: "expired", FileName: "e.txt", CreatedAt: now.Add(-48 * time.Hour), Configured: true, ExpiresAt: &past})
	repo.Create(ctx, sharing.Sharing{ID: "stale", FileID: "stale", FileName: "s.txt", CreatedAt: now.Add(-48 * time.Hour)})
	repo.Create(ctx, sharing.Sharing{ID: "live", FileID: "live", FileName: "l.txt", CreatedAt: now, Configured: true, ExpiresAt: &future})
	for _, b := range []string{"expired", "stale", "live"} {
		writeBlob(t, store, b, b[:1]+".txt", "x")
	}
	writeBlob(t, store, "orphan", "o.txt", "x")       // no share, old -> swept
	writeBlob(t, store, "orphanFresh", "of.txt", "x") // no share, fresh -> kept

	// Backdate the old orphan past the grace window.
	old := now.Add(-48 * time.Hour)
	os.Chtimes(filepath.Join(dir, "orphan", "o.txt"), old, old)

	p := New(repo, store, Config{Interval: time.Hour, UnconfiguredAfter: 24 * time.Hour})
	p.now = func() time.Time { return now }

	if err := p.prune(ctx); err != nil {
		t.Fatalf("prune: %v", err)
	}

	// expired + stale rows gone, live kept
	if _, err := repo.Get(ctx, "expired"); !errors.Is(err, sharing.ErrNotFound) {
		t.Errorf("expired row still present: %v", err)
	}
	if _, err := repo.Get(ctx, "stale"); !errors.Is(err, sharing.ErrNotFound) {
		t.Errorf("stale row still present: %v", err)
	}
	if _, err := repo.Get(ctx, "live"); err != nil {
		t.Errorf("live row removed: %v", err)
	}

	// blobs: expired/stale deleted, live kept
	if exists(t, store, "expired", "e.txt") {
		t.Error("expired blob not deleted")
	}
	if exists(t, store, "stale", "s.txt") {
		t.Error("stale blob not deleted")
	}
	if !exists(t, store, "live", "l.txt") {
		t.Error("live blob deleted")
	}
	// orphan sweep: old removed, fresh kept
	if exists(t, store, "orphan", "o.txt") {
		t.Error("old orphan blob not swept")
	}
	if !exists(t, store, "orphanFresh", "of.txt") {
		t.Error("fresh orphan blob wrongly swept")
	}
}

// errOnDeleteStore fails Delete for one id to prove a per-item error doesn't
// abort the whole pass.
type errOnDeleteStore struct {
	storage.Storage
	failID string
}

func (s errOnDeleteStore) Delete(ctx context.Context, id, name string) error {
	if id == s.failID {
		return errors.New("boom")
	}
	return s.Storage.Delete(ctx, id, name)
}

func TestPrune_PerItemErrorDoesNotAbort(t *testing.T) {
	dir := t.TempDir()
	base := storage.NewLocal(dir)
	store := errOnDeleteStore{Storage: base, failID: "bad"}
	repo := sharing.NewMemoryRepository()
	ctx := context.Background()
	now := time.Now()

	repo.Create(ctx, sharing.Sharing{ID: "bad", FileID: "bad", FileName: "b.txt", CreatedAt: now.Add(-48 * time.Hour)})
	repo.Create(ctx, sharing.Sharing{ID: "good", FileID: "good", FileName: "g.txt", CreatedAt: now.Add(-48 * time.Hour)})
	writeBlob(t, base, "bad", "b.txt", "x")
	writeBlob(t, base, "good", "g.txt", "x")

	p := New(repo, store, Config{Interval: time.Hour, UnconfiguredAfter: 24 * time.Hour})
	p.now = func() time.Time { return now }

	if err := p.prune(ctx); err != nil {
		t.Fatalf("prune: %v", err)
	}

	// bad blob failed to delete, so its row is left intact (not orphaned)
	if _, err := repo.Get(ctx, "bad"); err != nil {
		t.Errorf("bad row removed despite blob-delete failure: %v", err)
	}
	// good one still pruned
	if _, err := repo.Get(ctx, "good"); !errors.Is(err, sharing.ErrNotFound) {
		t.Errorf("good row not pruned: %v", err)
	}
	if exists(t, base, "good", "g.txt") {
		t.Error("good blob not deleted")
	}
}

func TestRun_StopsOnContextCancel(t *testing.T) {
	store := storage.NewLocal(t.TempDir())
	repo := sharing.NewMemoryRepository()
	p := New(repo, store, Config{Interval: time.Hour, UnconfiguredAfter: 24 * time.Hour})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run returned %v, want nil on cancel", err)
	}
}
