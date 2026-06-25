package web

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"crypto/rand"

	"github.com/rispycz/sshbin/internal/auth"
	"github.com/rispycz/sshbin/internal/sharing"
	"github.com/rispycz/sshbin/internal/storage"
)

const shutdownGrace = 10 * time.Second

type Config struct {
	ListenAddr string
	BaseURL    string
	// Secret signs password-grant cookies. When empty a random one is generated
	// per start (fine for tests; supply a persisted secret in production so
	// grants survive restarts).
	Secret []byte
}

type Server struct {
	cfg     Config
	repo    sharing.Repository
	storage storage.Storage
	auth    *auth.Manager
}

func New(cfg Config, repo sharing.Repository, st storage.Storage, authMgr *auth.Manager) *Server {
	return &Server{cfg: cfg, repo: repo, storage: st, auth: authMgr}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	tpl, err := parseTemplates()
	if err != nil {
		return err
	}

	secret := s.cfg.Secret
	if len(secret) == 0 {
		secret = make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return err
		}
	}

	h := &handler{
		repo:          s.repo,
		storage:       s.storage,
		auth:          s.auth,
		baseURL:       s.cfg.BaseURL,
		host:          hostFromURL(s.cfg.BaseURL),
		secureCookies: strings.HasPrefix(s.cfg.BaseURL, "https://"),
		secret:        secret,
		tpl:           tpl,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.index)
	mux.HandleFunc("GET /login", h.loginGet)
	mux.HandleFunc("POST /login", h.loginPost)
	mux.HandleFunc("POST /verify", h.verifyPost)
	mux.HandleFunc("POST /logout", h.logout)
	mux.HandleFunc("GET /setup/{id}", h.requireSession(h.setupGet))
	mux.HandleFunc("POST /setup/{id}", h.requireSession(h.setupPost))
	mux.HandleFunc("GET /s/{id}", h.shareView)
	mux.HandleFunc("POST /s/{id}", h.sharePassword)
	mux.HandleFunc("GET /s/{id}/download", h.download)
	mux.Handle("GET /static/", http.FileServerFS(staticFS))

	httpSrv := &http.Server{
		Addr:              s.cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		defer cancel()
		httpSrv.Shutdown(shutCtx)
	}()

	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// hostFromURL extracts the host shown in the example scp command. Falls back to
// the raw base URL if it cannot be parsed.
func hostFromURL(base string) string {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return base
	}
	return u.Hostname()
}
