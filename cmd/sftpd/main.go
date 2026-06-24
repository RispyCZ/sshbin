package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rispycz/securedrop/internal/sharing"
	"github.com/rispycz/securedrop/internal/sftpd"
	"github.com/rispycz/securedrop/internal/storage"
)

func main() {
	listenAddr := flag.String("listen", ":2022", "SSH listen address")
	hostKeyPath := flag.String("host-key", "host_key", "path to host key (generated if missing)")
	baseURL := flag.String("base-url", "http://localhost:8080", "base URL for setup links")
	storageDir := flag.String("storage", "uploads", "directory for uploaded files")
	flag.Parse()

	if err := os.MkdirAll(*storageDir, 0o750); err != nil {
		log.Fatalf("create storage dir: %v", err)
	}

	st := &storage.LocalStorage{BaseDir: *storageDir}
	repo := sharing.NewMemoryRepository()
	srv := sftpd.New(sftpd.Config{
		ListenAddr:  *listenAddr,
		HostKeyPath: *hostKeyPath,
		BaseURL:     *baseURL,
	}, st, repo)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("filedrop listening on %s", *listenAddr)
	if err := srv.ListenAndServe(ctx); err != nil {
		log.Fatalf("server: %v", err)
	}
}
