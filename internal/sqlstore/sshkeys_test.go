package sqlstore_test

import (
	"context"
	"crypto/ed25519"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/rispycz/sshbin/internal/sshkeys"
)

// testKey generates an ed25519 SSH public key and returns its canonical form,
// fingerprint, and wire-marshaled bytes.
func testKey(t *testing.T) (canonical, fingerprint string, marshaled []byte) {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("NewPublicKey: %v", err)
	}
	return sshkeys.Canonical(sshPub), ssh.FingerprintSHA256(sshPub), sshPub.Marshal()
}

func TestSSHKeys_RoundTrip(t *testing.T) {
	st := openTemp(t)
	repo := st.SSHKeys()
	ctx := context.Background()

	canonical, fp, marshaled := testKey(t)
	k := sshkeys.PublicKey{
		ID: "id1", Email: "owner@example.com", Title: "laptop",
		Fingerprint: fp, AuthorizedKey: canonical, CreatedAt: time.Now().Truncate(time.Second),
	}
	if err := repo.Add(ctx, k); err != nil {
		t.Fatalf("Add: %v", err)
	}

	list, err := repo.ListByEmail(ctx, "owner@example.com")
	if err != nil {
		t.Fatalf("ListByEmail: %v", err)
	}
	if len(list) != 1 || list[0].Title != "laptop" || list[0].Fingerprint != fp {
		t.Fatalf("ListByEmail mismatch: %+v", list)
	}

	email, found, err := repo.FindOwnerByMarshaledKey(ctx, marshaled)
	if err != nil {
		t.Fatalf("FindOwnerByMarshaledKey: %v", err)
	}
	if !found || email != "owner@example.com" {
		t.Errorf("owner lookup = %q,%v, want owner@example.com,true", email, found)
	}
}

func TestSSHKeys_DuplicateFingerprintRejected(t *testing.T) {
	st := openTemp(t)
	repo := st.SSHKeys()
	ctx := context.Background()

	canonical, fp, _ := testKey(t)
	base := sshkeys.PublicKey{Fingerprint: fp, AuthorizedKey: canonical, CreatedAt: time.Now()}

	first := base
	first.ID, first.Email = "a", "alice@example.com"
	if err := repo.Add(ctx, first); err != nil {
		t.Fatalf("Add first: %v", err)
	}

	second := base
	second.ID, second.Email = "b", "bob@example.com"
	if err := repo.Add(ctx, second); !errors.Is(err, sshkeys.ErrDuplicate) {
		t.Fatalf("Add duplicate err = %v, want ErrDuplicate", err)
	}
}

func TestSSHKeys_DeleteScopedToOwner(t *testing.T) {
	st := openTemp(t)
	repo := st.SSHKeys()
	ctx := context.Background()

	canonical, fp, _ := testKey(t)
	if err := repo.Add(ctx, sshkeys.PublicKey{ID: "id1", Email: "owner@example.com", Fingerprint: fp, AuthorizedKey: canonical, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Wrong owner cannot delete.
	if err := repo.Delete(ctx, "id1", "someone@else.com"); !errors.Is(err, sshkeys.ErrNotFound) {
		t.Fatalf("Delete wrong owner err = %v, want ErrNotFound", err)
	}
	// Correct owner deletes.
	if err := repo.Delete(ctx, "id1", "owner@example.com"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	list, _ := repo.ListByEmail(ctx, "owner@example.com")
	if len(list) != 0 {
		t.Errorf("after delete: %d keys, want 0", len(list))
	}
}

func TestSSHKeys_DeleteByEmailAndMiss(t *testing.T) {
	st := openTemp(t)
	repo := st.SSHKeys()
	ctx := context.Background()

	canonical, fp, marshaled := testKey(t)
	if err := repo.Add(ctx, sshkeys.PublicKey{ID: "id1", Email: "owner@example.com", Fingerprint: fp, AuthorizedKey: canonical, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := repo.DeleteByEmail(ctx, "owner@example.com"); err != nil {
		t.Fatalf("DeleteByEmail: %v", err)
	}
	_, found, err := repo.FindOwnerByMarshaledKey(ctx, marshaled)
	if err != nil {
		t.Fatalf("FindOwnerByMarshaledKey: %v", err)
	}
	if found {
		t.Error("key still found after DeleteByEmail")
	}
}
