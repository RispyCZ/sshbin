package sharing_test

import (
	"context"
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
	if err == nil {
		t.Fatal("expected error, got nil")
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
