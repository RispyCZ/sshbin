package sshkeys_test

import (
	"crypto/ed25519"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/rispycz/sshbin/internal/sshkeys"
)

// genLine returns a valid authorized_keys line (with the given comment) plus the
// canonical, comment-free form for comparison.
func genLine(t *testing.T, comment string) (line, canonical string) {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("NewPublicKey: %v", err)
	}
	canonical = sshkeys.Canonical(sshPub)
	return canonical + " " + comment, canonical
}

func TestParse_ValidStripsComment(t *testing.T) {
	line, canonical := genLine(t, "comment-here")
	got, fp, err := sshkeys.Parse(line)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got != canonical {
		t.Errorf("canonical = %q, want %q", got, canonical)
	}
	if strings.Contains(got, "comment-here") {
		t.Errorf("canonical should not contain comment: %q", got)
	}
	if !strings.HasPrefix(fp, "SHA256:") {
		t.Errorf("fingerprint = %q, want SHA256: prefix", fp)
	}
}

func TestParse_FingerprintStableAcrossComments(t *testing.T) {
	line, canonical := genLine(t, "alice@laptop")
	_, fp1, err := sshkeys.Parse(line)
	if err != nil {
		t.Fatalf("Parse 1: %v", err)
	}
	_, fp2, err := sshkeys.Parse(canonical + " bob@desktop")
	if err != nil {
		t.Fatalf("Parse 2: %v", err)
	}
	if fp1 != fp2 {
		t.Errorf("fingerprints differ by comment: %q vs %q", fp1, fp2)
	}
}

func TestParse_RejectsGarbage(t *testing.T) {
	for _, in := range []string{"", "   ", "not a key", "ssh-ed25519 not-base64"} {
		if _, _, err := sshkeys.Parse(in); !errors.Is(err, sshkeys.ErrInvalidKey) {
			t.Errorf("Parse(%q) err = %v, want ErrInvalidKey", in, err)
		}
	}
}
