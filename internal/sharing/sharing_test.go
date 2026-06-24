package sharing_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rispycz/securedrop/internal/sharing"
)

func TestMemoryRepository_CreateAndGet(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	s := sharing.Sharing{ID: "abc", FileID: "fid", FileName: "f.txt", CreatedAt: time.Now()}

	if err := repo.Create(context.Background(), s); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := repo.Get(context.Background(), "abc")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != s.ID || got.FileName != s.FileName {
		t.Errorf("got %+v, want %+v", got, s)
	}
}

func TestMemoryRepository_GetMissing(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	_, err := repo.Get(context.Background(), "nope")
	if !errors.Is(err, sharing.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestMemoryRepository_Update(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	ctx := context.Background()
	s := sharing.Sharing{ID: "x", FileName: "f.txt", CreatedAt: time.Now()}
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	s.Configured = true
	s.Public = true
	if err := repo.Update(ctx, s); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := repo.Get(ctx, "x")
	if !got.Configured || !got.Public {
		t.Errorf("update not persisted: %+v", got)
	}

	if err := repo.Update(ctx, sharing.Sharing{ID: "missing"}); !errors.Is(err, sharing.ErrNotFound) {
		t.Errorf("Update missing: err = %v, want ErrNotFound", err)
	}
}

func TestSharing_PasswordAndExpiry(t *testing.T) {
	hash, err := sharing.HashPassword("hunter2")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	s := sharing.Sharing{PasswordHash: hash}
	if !s.HasPassword() {
		t.Error("HasPassword = false, want true")
	}
	if !s.CheckPassword("hunter2") {
		t.Error("CheckPassword(correct) = false")
	}
	if s.CheckPassword("wrong") {
		t.Error("CheckPassword(wrong) = true")
	}

	past := time.Now().Add(-time.Hour)
	if !(sharing.Sharing{ExpiresAt: &past}).Expired(time.Now()) {
		t.Error("past expiry should be Expired")
	}
	if (sharing.Sharing{}).Expired(time.Now()) {
		t.Error("nil expiry should never be Expired")
	}
}

func TestMemoryRepository_ConcurrentCreates(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s := sharing.Sharing{ID: string(rune('a' + n%26)) + string(rune('0'+n/26)), FileName: "f.txt", CreatedAt: time.Now()}
			_ = repo.Create(context.Background(), s)
		}(i)
	}
	wg.Wait()
}

func TestSetupURL(t *testing.T) {
	cases := []struct {
		base, id, want string
	}{
		{"https://example.com", "abc123", "https://example.com/setup/abc123"},
		{"https://example.com/", "abc123", "https://example.com/setup/abc123"},
	}
	for _, c := range cases {
		got := sharing.SetupURL(c.base, c.id)
		if got != c.want {
			t.Errorf("SetupURL(%q, %q) = %q, want %q", c.base, c.id, got, c.want)
		}
	}
}
