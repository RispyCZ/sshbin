package sftp_test

import (
	"bytes"
	"io"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/rispycz/sshbin/internal/sftp"
)

type mockChannel struct {
	ssh.Channel
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func (m *mockChannel) Write(p []byte) (int, error)  { return m.stdout.Write(p) }
func (m *mockChannel) Stderr() io.ReadWriter        { return &m.stderr }

func TestStderrWriter_RoutesToStderr(t *testing.T) {
	ch := &mockChannel{}
	w := sftp.NewStderrWriter(ch)

	msg := []byte("setup url\n")
	if _, err := w.Write(msg); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if ch.stdout.Len() != 0 {
		t.Errorf("unexpected stdout bytes: %q", ch.stdout.Bytes())
	}
	if got := ch.stderr.Bytes(); !bytes.Equal(got, msg) {
		t.Errorf("stderr = %q, want %q", got, msg)
	}
}
