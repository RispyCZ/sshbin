package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"
	"golang.org/x/sync/errgroup"

	"github.com/rispycz/sshbin/internal/auth"
	"github.com/rispycz/sshbin/internal/sftp"
	"github.com/rispycz/sshbin/internal/sqlstore"
	"github.com/rispycz/sshbin/internal/storage"
	"github.com/rispycz/sshbin/internal/web"
)

func main() {
	sftpAddr := flag.String("sftp-listen", ":2022", "SFTP listen address")
	webAddr := flag.String("web-listen", ":8080", "web UI listen address")
	hostKeyPath := flag.String("host-key", "host_key", "path to SSH host key (generated if missing)")
	baseURL := flag.String("base-url", "http://localhost:8080", "base URL for setup and share links")
	storageDSN := flag.String("storage", "local://uploads", "storage backend DSN (local://path or s3://bucket/prefix)")
	dsn := flag.String("db", "sqlite://sshbin.db", "database DSN (e.g. sqlite://sshbin.db)")
	flag.Parse()

	db, err := sqlstore.Open(*dsn)
	if err != nil {
		log.Fatal("open database", "err", err)
	}
	defer db.Close()

	secret, err := db.EnsureSecret()
	if err != nil {
		log.Fatal("load grant secret", "err", err)
	}

	st, err := storage.Open(*storageDSN)
	if err != nil {
		log.Fatal("open storage", "err", err)
	}

	// Shares are persisted and shared by both servers: SFTP creates records,
	// the web UI reads and updates them.
	repo := db.Shares()

	sftpSrv := sftp.New(sftp.Config{
		ListenAddr:  *sftpAddr,
		HostKeyPath: *hostKeyPath,
		BaseURL:     *baseURL,
	}, st, repo)

	// LogSender prints OTP codes to the log; replace with SMTP/SMS in production.
	authMgr := auth.NewManager(auth.LogSender{}, db.Sessions(), auth.Options{})

	webSrv := web.New(web.Config{
		ListenAddr: *webAddr,
		BaseURL:    *baseURL,
		Secret:     secret,
	}, repo, st, authMgr, db.UserPrefs())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return sftpSrv.ListenAndServe(ctx) })
	g.Go(func() error { return webSrv.ListenAndServe(ctx) })

	log.Info("sshbin started", "sftp", *sftpAddr, "web", *webAddr)
	if err := g.Wait(); err != nil {
		log.Fatal("server", "err", err)
	}
}
