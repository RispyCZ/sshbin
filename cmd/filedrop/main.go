package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/rispycz/securedrop/internal/auth"
	"github.com/rispycz/securedrop/internal/sftpd"
	"github.com/rispycz/securedrop/internal/sharing"
	"github.com/rispycz/securedrop/internal/storage"
	"github.com/rispycz/securedrop/internal/web"
)

func main() {
	sftpAddr := flag.String("sftp-listen", ":2022", "SFTP listen address")
	webAddr := flag.String("web-listen", ":8080", "web UI listen address")
	hostKeyPath := flag.String("host-key", "host_key", "path to SSH host key (generated if missing)")
	baseURL := flag.String("base-url", "http://localhost:8080", "base URL for setup and share links")
	storageDir := flag.String("storage", "uploads", "directory for uploaded files")
	flag.Parse()

	if err := os.MkdirAll(*storageDir, 0o750); err != nil {
		log.Fatalf("create storage dir: %v", err)
	}

	// One repository instance shared by both servers: SFTP creates records,
	// the web UI reads and updates them.
	repo := sharing.NewMemoryRepository()
	st := &storage.LocalStorage{BaseDir: *storageDir}

	sftpSrv := sftpd.New(sftpd.Config{
		ListenAddr:  *sftpAddr,
		HostKeyPath: *hostKeyPath,
		BaseURL:     *baseURL,
	}, st, repo)

	// LogSender prints OTP codes to the log; replace with SMTP/SMS in production.
	authMgr := auth.NewManager(auth.LogSender{}, auth.Options{})

	webSrv := web.New(web.Config{
		ListenAddr: *webAddr,
		BaseURL:    *baseURL,
	}, repo, authMgr)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return sftpSrv.ListenAndServe(ctx) })
	g.Go(func() error { return webSrv.ListenAndServe(ctx) })

	log.Printf("filedrop: sftp %s, web %s", *sftpAddr, *webAddr)
	if err := g.Wait(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
