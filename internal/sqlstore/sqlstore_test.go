package sqlstore_test

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/rispycz/securedrop/internal/auth"
	"github.com/rispycz/securedrop/internal/sharing"
	"github.com/rispycz/securedrop/internal/sqlstore"
)

func openTemp(t *testing.T) *sqlstore.Store {
	t.Helper()
	dsn := "sqlite://" + filepath.Join(t.TempDir(), "test.db")
	st, err := sqlstore.Open(dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestShares_RoundTrip(t *testing.T) {
	st := openTemp(t)
	repo := st.Shares()
	ctx := context.Background()

	created := time.Now().Truncate(time.Second)
	exp := created.Add(24 * time.Hour)
	in := sharing.Sharing{
		ID: "abc", FileID: "fid", FileName: "f.txt", CreatedAt: created,
		Configured: true, OwnerEmail: "owner@example.com", ExpiresAt: &exp,
		PasswordHash: "hash", Public: false,
		AllowedEmails: []string{"alice@example.com", "bob@example.com"},
	}
	if err := repo.Create(ctx, in); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, "abc")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.FileName != "f.txt" || !got.Configured || got.Public || got.OwnerEmail != "owner@example.com" {
		t.Errorf("scalar mismatch: %+v", got)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(exp) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, exp)
	}
	if !got.AllowsEmail("alice@example.com") || !got.AllowsEmail("bob@example.com") {
		t.Errorf("emails not persisted: %v", got.AllowedEmails)
	}
}

func TestShares_UpdateReplacesEmails(t *testing.T) {
	st := openTemp(t)
	repo := st.Shares()
	ctx := context.Background()

	repo.Create(ctx, sharing.Sharing{ID: "abc", FileName: "f.txt", CreatedAt: time.Now(), AllowedEmails: []string{"old@example.com"}})

	s, _ := repo.Get(ctx, "abc")
	s.Public = true
	s.AllowedEmails = []string{"new@example.com"}
	if err := repo.Update(ctx, s); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.Get(ctx, "abc")
	if !got.Public {
		t.Error("Public not updated")
	}
	if got.AllowsEmail("old@example.com") {
		t.Error("old email should be removed")
	}
	if !got.AllowsEmail("new@example.com") {
		t.Error("new email missing")
	}
}

func TestShares_GetMissing_And_UpdateMissing(t *testing.T) {
	st := openTemp(t)
	repo := st.Shares()
	ctx := context.Background()

	if _, err := repo.Get(ctx, "nope"); !errors.Is(err, sharing.ErrNotFound) {
		t.Errorf("Get missing: %v, want ErrNotFound", err)
	}
	if err := repo.Update(ctx, sharing.Sharing{ID: "nope"}); !errors.Is(err, sharing.ErrNotFound) {
		t.Errorf("Update missing: %v, want ErrNotFound", err)
	}
}

func TestShares_NilExpiry(t *testing.T) {
	st := openTemp(t)
	repo := st.Shares()
	ctx := context.Background()

	repo.Create(ctx, sharing.Sharing{ID: "abc", FileName: "f.txt", CreatedAt: time.Now()})
	got, _ := repo.Get(ctx, "abc")
	if got.ExpiresAt != nil {
		t.Errorf("ExpiresAt = %v, want nil", got.ExpiresAt)
	}
}

func TestSessions_Lifecycle(t *testing.T) {
	st := openTemp(t)
	store := st.Sessions()

	sess := auth.Session{Email: "u@e.com", ExpiresAt: time.Now().Add(time.Hour)}
	if err := store.Put("tok", sess); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, ok, err := store.Get("tok")
	if err != nil || !ok || got.Email != "u@e.com" {
		t.Fatalf("Get = %+v ok=%v err=%v", got, ok, err)
	}
	if err := store.Delete("tok"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok, _ := store.Get("tok"); ok {
		t.Error("session should be gone after delete")
	}
}

func TestSessions_Expired(t *testing.T) {
	st := openTemp(t)
	store := st.Sessions()

	store.Put("tok", auth.Session{Email: "u@e.com", ExpiresAt: time.Now().Add(-time.Minute)})
	if _, ok, _ := store.Get("tok"); ok {
		t.Error("expired session should not be returned")
	}
}

func TestSecret_StableAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	dsn := "sqlite://" + filepath.Join(dir, "test.db")

	st1, err := sqlstore.Open(dsn)
	if err != nil {
		t.Fatalf("Open 1: %v", err)
	}
	sec1, err := st1.EnsureSecret()
	if err != nil {
		t.Fatalf("EnsureSecret 1: %v", err)
	}
	st1.Close()

	st2, err := sqlstore.Open(dsn)
	if err != nil {
		t.Fatalf("Open 2: %v", err)
	}
	defer st2.Close()
	sec2, err := st2.EnsureSecret()
	if err != nil {
		t.Fatalf("EnsureSecret 2: %v", err)
	}

	if len(sec1) != 32 || !bytes.Equal(sec1, sec2) {
		t.Errorf("secret not stable across reopen")
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	dir := t.TempDir()
	dsn := "sqlite://" + filepath.Join(dir, "test.db")

	st1, err := sqlstore.Open(dsn)
	if err != nil {
		t.Fatalf("Open 1: %v", err)
	}
	st1.Shares().Create(context.Background(), sharing.Sharing{ID: "abc", FileName: "f.txt", CreatedAt: time.Now()})
	st1.Close()

	// Reopen runs migrate again; must not error or wipe data.
	st2, err := sqlstore.Open(dsn)
	if err != nil {
		t.Fatalf("Open 2 (re-migrate): %v", err)
	}
	defer st2.Close()
	if _, err := st2.Shares().Get(context.Background(), "abc"); err != nil {
		t.Errorf("data lost after re-migrate: %v", err)
	}
}
