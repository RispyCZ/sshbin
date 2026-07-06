package sftp

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/charmbracelet/log"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"

	"github.com/rispycz/sshbin/internal/sharing"
	"github.com/rispycz/sshbin/internal/storage"
)

const shutdownGrace = 10 * time.Second

type Config struct {
	ListenAddr  string
	HostKeyPath string
	BaseURL     string
	// AllowAnonymous permits uploads from unrecognized keys. When false, only
	// keys registered by a web user may connect.
	AllowAnonymous bool
}

// KeyAuthenticator resolves an SSH public key to the web user that owns it.
type KeyAuthenticator interface {
	FindOwnerByMarshaledKey(ctx context.Context, marshaled []byte) (email string, found bool, err error)
	TouchLastUsed(ctx context.Context, fingerprint string) error
}

type Server struct {
	cfg     Config
	storage storage.Storage
	repo    sharing.Repository
	keys    KeyAuthenticator
}

func New(cfg Config, st storage.Storage, repo sharing.Repository, keys KeyAuthenticator) *Server {
	return &Server{cfg: cfg, storage: st, repo: repo, keys: keys}
}

// ctxKey namespaces values stashed on the ssh.Context during auth.
type ctxKey string

const (
	ownerEmailKey  ctxKey = "sshbin_owner_email"
	fingerprintKey ctxKey = "sshbin_fingerprint"
)

// publicKeyHandler authenticates a connecting client by its offered public key.
// It is installed unconditionally so the SSH "none" method is always rejected
// and every client is forced to offer a key (see architecture note below);
// AllowAnonymous only decides whether an unregistered key is admitted.
func (s *Server) publicKeyHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	email, found, err := s.keys.FindOwnerByMarshaledKey(ctx, key.Marshal())
	if err != nil {
		// Fail closed: a DB error must not silently grant anonymous access.
		log.Error("ssh key lookup", "err", err)
		return false
	}
	if found {
		ctx.SetValue(ownerEmailKey, email)
		ctx.SetValue(fingerprintKey, gossh.FingerprintSHA256(key))
		return true
	}
	return s.cfg.AllowAnonymous
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	srv, err := wish.NewServer(
		wish.WithAddress(s.cfg.ListenAddr),
		wish.WithHostKeyPath(s.cfg.HostKeyPath),
		wish.WithVersion("sshbin"),
		wish.WithPublicKeyAuth(s.publicKeyHandler),
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
			log.Error("sftp session panic", "addr", sess.RemoteAddr(), "err", r)
		}
	}()

	owner, _ := sess.Context().Value(ownerEmailKey).(string)
	handlers := Handlers(s.storage, s.repo, s.cfg.BaseURL, NewStderrWriter(sess), owner)
	srv := sftp.NewRequestServer(sess, handlers)
	if err := srv.Serve(); err != nil && !errors.Is(err, io.EOF) {
		log.Error("sftp serve", "addr", sess.RemoteAddr(), "err", err)
	}

	// Record key usage best-effort after the session so registered users can see
	// when their key was last used.
	if fp, ok := sess.Context().Value(fingerprintKey).(string); ok && fp != "" {
		if err := s.keys.TouchLastUsed(context.Background(), fp); err != nil {
			log.Error("ssh touch last used", "err", err)
		}
	}
}
