package sftpd_test

import (
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/rispycz/securedrop/internal/sharing"
	"github.com/rispycz/securedrop/internal/sftpd"
	"github.com/rispycz/securedrop/internal/storage"
)

func startServer(t *testing.T, storageDir string) (addr string, repo *sharing.MemoryRepository) {
	t.Helper()
	repo = sharing.NewMemoryRepository()
	st := &storage.LocalStorage{BaseDir: storageDir}

	keyPath := filepath.Join(t.TempDir(), "host_key")
	cfg := sftpd.Config{
		ListenAddr:  "127.0.0.1:0",
		HostKeyPath: keyPath,
		BaseURL:     "http://localhost:8080",
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr = ln.Addr().String()
	ln.Close()

	cfg.ListenAddr = addr
	srv := sftpd.New(cfg, st, repo)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	ready := make(chan struct{})
	go func() {
		close(ready)
		srv.ListenAndServe(ctx)
	}()
	<-ready
	time.Sleep(20 * time.Millisecond) // let listener bind
	return addr, repo
}

func sftpClient(t *testing.T, addr string) *sftp.Client {
	t.Helper()
	clientCfg := &ssh.ClientConfig{
		User:            "anon",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", addr, clientCfg)
	if err != nil {
		t.Fatalf("ssh dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	client, err := sftp.NewClient(conn)
	if err != nil {
		t.Fatalf("sftp client: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestStat_TargetDirIsDirectory(t *testing.T) {
	// scp stats the destination dir before uploading; it must come back as a
	// directory (S_IFDIR over the wire) or scp aborts with `dest open`.
	dir := t.TempDir()
	addr, _ := startServer(t, dir)
	client := sftpClient(t, addr)

	for _, target := range []string{".", "/"} {
		fi, err := client.Stat(target)
		if err != nil {
			t.Fatalf("Stat(%q): %v", target, err)
		}
		if !fi.IsDir() {
			t.Errorf("Stat(%q).IsDir() = false, want true (mode=%v)", target, fi.Mode())
		}
	}
}

func TestUpload_HappyPath(t *testing.T) {
	dir := t.TempDir()
	addr, repo := startServer(t, dir)
	client := sftpClient(t, addr)

	f, err := client.Create("hello.txt")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := io.WriteString(f, "hello world"); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// sharing record created
	ctx := context.Background()
	found := false
	// walk storage dir to find the file
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "hello.txt" {
			found = true
			data, _ := os.ReadFile(path)
			if string(data) != "hello world" {
				t.Errorf("file content = %q, want %q", data, "hello world")
			}
			// sharing ID is the parent dir name
			id := filepath.Base(filepath.Dir(path))
			s, err := repo.Get(ctx, id)
			if err != nil {
				t.Errorf("sharing not found: %v", err)
			}
			if s.FileName != "hello.txt" {
				t.Errorf("sharing.FileName = %q, want %q", s.FileName, "hello.txt")
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if !found {
		t.Error("uploaded file not found in storage")
	}
}

func TestUpload_DownloadDenied(t *testing.T) {
	dir := t.TempDir()
	addr, _ := startServer(t, dir)
	client := sftpClient(t, addr)

	// Upload first
	f, _ := client.Create("deny.txt")
	io.WriteString(f, "x")
	f.Close()

	_, err := client.Open("deny.txt")
	if err == nil {
		t.Fatal("expected error downloading, got nil")
	}
}

func TestUpload_SetstatNoOp(t *testing.T) {
	// scp runs fsetstat after upload to copy mtime/perms; it must not error.
	dir := t.TempDir()
	addr, _ := startServer(t, dir)
	client := sftpClient(t, addr)

	f, err := client.Create("stat.txt")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	io.WriteString(f, "x")
	f.Close()

	if err := client.Chmod("stat.txt", 0o600); err != nil {
		t.Fatalf("Chmod (setstat) should be no-op success, got: %v", err)
	}
}

func TestUpload_ReadDirDenied(t *testing.T) {
	dir := t.TempDir()
	addr, _ := startServer(t, dir)
	client := sftpClient(t, addr)

	_, err := client.ReadDir(".")
	if err == nil {
		t.Fatal("expected error on ReadDir, got nil")
	}
}

func TestUpload_MkdirDenied(t *testing.T) {
	dir := t.TempDir()
	addr, _ := startServer(t, dir)
	client := sftpClient(t, addr)

	err := client.Mkdir("somedir")
	if err == nil {
		t.Fatal("expected error on Mkdir, got nil")
	}
}

func TestUpload_ConcurrentUploads(t *testing.T) {
	dir := t.TempDir()
	addr, _ := startServer(t, dir)

	var wg sync.WaitGroup
	for i := range 5 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c := sftpClient(t, addr)
			f, err := c.Create("file.txt")
			if err != nil {
				t.Errorf("goroutine %d Create: %v", n, err)
				return
			}
			io.WriteString(f, "data")
			f.Close()
		}(i)
	}
	wg.Wait()

	// All 5 files should be stored (each under distinct UUID dir)
	entries, _ := filepath.Glob(filepath.Join(dir, "*/file.txt"))
	if len(entries) != 5 {
		t.Errorf("expected 5 files, found %d", len(entries))
	}
}

func TestUpload_StorageError(t *testing.T) {
	repo := sharing.NewMemoryRepository()
	// Use a read-only dir to force storage errors
	roDir := t.TempDir()
	os.Chmod(roDir, 0o444)
	t.Cleanup(func() { os.Chmod(roDir, 0o755) })

	st := &storage.LocalStorage{BaseDir: roDir}
	keyPath := filepath.Join(t.TempDir(), "host_key")
	cfg := sftpd.Config{
		ListenAddr:  "127.0.0.1:0",
		HostKeyPath: keyPath,
		BaseURL:     "http://localhost:8080",
	}

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()
	cfg.ListenAddr = addr

	srv := sftpd.New(cfg, st, repo)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go srv.ListenAndServe(ctx)
	time.Sleep(20 * time.Millisecond)

	clientCfg := &ssh.ClientConfig{
		User:            "anon",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", addr, clientCfg)
	if err != nil {
		t.Fatalf("ssh dial: %v", err)
	}
	defer conn.Close()
	client, err := sftp.NewClient(conn)
	if err != nil {
		t.Fatalf("sftp client: %v", err)
	}
	defer client.Close()

	_, err = client.Create("fail.txt")
	if err == nil {
		t.Fatal("expected error due to storage failure, got nil")
	}
	_ = strings.Contains(err.Error(), "") // suppress unused warning
}
