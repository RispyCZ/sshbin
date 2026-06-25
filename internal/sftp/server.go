package sftp

import (
	"context"
	"errors"
	"io"
	"log"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/pkg/sftp"

	"github.com/rispycz/sshbin/internal/sharing"
	"github.com/rispycz/sshbin/internal/storage"
)

const shutdownGrace = 10 * time.Second

type Config struct {
	ListenAddr  string
	HostKeyPath string
	BaseURL     string
}

type Server struct {
	cfg     Config
	storage storage.Storage
	repo    sharing.Repository
}

func New(cfg Config, st storage.Storage, repo sharing.Repository) *Server {
	return &Server{cfg: cfg, storage: st, repo: repo}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	srv, err := wish.NewServer(
		wish.WithAddress(s.cfg.ListenAddr),
		wish.WithHostKeyPath(s.cfg.HostKeyPath),
		wish.WithVersion("sshbin"),
		wish.WithSubsystem("sftp", s.handleSFTP),
	)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		defer cancel()
		srv.Shutdown(shutCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		return err
	}
	return nil
}

// handleSFTP serves the SFTP subsystem for a session. Subsystem handlers run
// outside wish's middleware chain, so panic recovery and logging are applied
// here directly.
func (s *Server) handleSFTP(sess ssh.Session) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("sftp session panic from %s: %v", sess.RemoteAddr(), r)
		}
	}()

	handlers := Handlers(s.storage, s.repo, s.cfg.BaseURL, NewStderrWriter(sess))
	srv := sftp.NewRequestServer(sess, handlers)
	if err := srv.Serve(); err != nil && !errors.Is(err, io.EOF) {
		log.Printf("sftp serve %s: %v", sess.RemoteAddr(), err)
	}
}
