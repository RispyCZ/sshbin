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
	"github.com/rispycz/sshbin/internal/userprefs"
)

const shutdownGrace = 10 * time.Second

type Config struct {
	ListenAddr string
	BaseURL    string
	// Secret signs password-grant cookies. When empty a random one is generated
	// per start (fine for tests; supply a persisted secret in production so
	// grants survive restarts).
	Secret []byte
	// Dev serves a SPA shell that loads modules from the Vite dev server for
	// HMR. When false, the embedded production build is served.
	Dev bool
	// ViteOrigin is the Vite dev server URL used when Dev is true.
	ViteOrigin string
}

type Server struct {
	cfg     Config
	repo    sharing.Repository
	storage storage.Storage
	auth    *auth.Manager
	prefs   userprefs.Repository
}

func New(cfg Config, repo sharing.Repository, st storage.Storage, authMgr *auth.Manager, prefs userprefs.Repository) *Server {
	return &Server{cfg: cfg, repo: repo, storage: st, auth: authMgr, prefs: prefs}
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
		prefs:         s.prefs,
		baseURL:       s.cfg.BaseURL,
		host:          hostFromURL(s.cfg.BaseURL),
		secureCookies: strings.HasPrefix(s.cfg.BaseURL, "https://"),
		secret:        secret,
		tpl:           tpl,
	}

	viteOrigin := s.cfg.ViteOrigin
	if viteOrigin == "" {
		viteOrigin = "http://localhost:5173"
	}
	spa, err := newSPA(s.cfg.Dev, viteOrigin)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()

	// JSON API consumed by the SPA.
	mux.HandleFunc("GET /api/session", h.apiSession)
	mux.HandleFunc("POST /api/login", h.apiLogin)
	mux.HandleFunc("POST /api/verify", h.apiVerify)
	mux.HandleFunc("POST /api/logout", h.apiLogout)
	mux.HandleFunc("GET /api/shares", h.requireSessionAPI(h.apiShares))
	mux.HandleFunc("DELETE /api/shares/{id}", h.requireSessionAPI(h.apiDeleteShare))

	// Public share consumer pages remain server-rendered.
	mux.HandleFunc("GET /shares/{id}/qr", h.requireSession(h.shareQR))
	mux.HandleFunc("GET /s/{id}", h.shareView)
	mux.HandleFunc("POST /s/{id}", h.sharePassword)
	mux.HandleFunc("GET /s/{id}/download", h.download)
	mux.Handle("GET /static/", http.FileServerFS(staticFS))

	// SPA owns the authenticated admin UI and all client-side routes.
	mux.Handle("/", spa)

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
