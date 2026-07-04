package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"
	"golang.org/x/sync/errgroup"

	"github.com/rispycz/sshbin/internal/auth"
	"github.com/rispycz/sshbin/internal/httpserver"
	"github.com/rispycz/sshbin/internal/sftp"
	"github.com/rispycz/sshbin/internal/sqlstore"
	"github.com/rispycz/sshbin/internal/storage"
)

func main() {
	// Every flag also reads SSHBIN_<NAME> (e.g. -sftp-listen -> SSHBIN_SFTP_LISTEN).
	// Precedence: explicit flag > env var > default. See env.go.
	sftpAddr := flagString("sftp-listen", ":2022", "SFTP listen address")
	webAddr := flagString("web-listen", ":8080", "web UI listen address")
	hostKeyPath := flagString("host-key", "host_key", "path to SSH host key (generated if missing)")
	baseURL := flagString("base-url", "http://localhost:8080", "base URL for setup and share links")
	storageDSN := flagString("storage", "local://uploads", "storage backend DSN (local://path or s3://bucket/prefix)")
	dsn := flagString("db", "sqlite://sshbin.db", "database DSN (e.g. sqlite://sshbin.db)")
	dev := flagBool("dev", false, "serve the SPA from the Vite dev server for HMR (run `vp dev` alongside)")
	viteOrigin := flagString("vite-origin", "http://localhost:5173", "Vite dev server URL used with -dev")
	smtpHost := flagString("smtp-host", "", "SMTP host for delivering login codes (empty logs codes instead)")
	smtpPort := flagInt("smtp-port", 587, "SMTP port (465 uses implicit TLS, otherwise STARTTLS)")
	smtpUser := flagString("smtp-user", "", "SMTP username (password read from SSHBIN_SMTP_PASSWORD)")
	smtpFrom := flagString("smtp-from", "", "From address for login-code emails")
	smtpInsecure := flagBool("smtp-insecure", false, "skip SMTP TLS certificate verification (dev only, allows self-signed)")
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

	var sender auth.Sender
	if *smtpHost != "" {
		s, err := auth.NewSMTPSender(auth.SMTPConfig{
			Host:               *smtpHost,
			Port:               *smtpPort,
			Username:           *smtpUser,
			Password:           smtpPassword(),
			From:               *smtpFrom,
			InsecureSkipVerify: *smtpInsecure,
		})
		if err != nil {
			log.Fatal("configure SMTP sender", "err", err)
		}
		sender = s
		log.Info("auth: using SMTP sender", "host", *smtpHost, "port", *smtpPort)
	} else {
		// LogSender prints OTP codes to the log; never use in production.
		sender = auth.LogSender{}
		log.Warn("auth: no -smtp-host set, login codes are printed to the log (dev only)")
	}
	authMgr := auth.NewManager(sender, db.Sessions(), auth.Options{})

	webSrv := httpserver.New(httpserver.Config{
		ListenAddr: *webAddr,
		BaseURL:    *baseURL,
		Secret:     secret,
		Dev:        *dev,
		ViteOrigin: *viteOrigin,
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
