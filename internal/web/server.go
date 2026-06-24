package web

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/rispycz/securedrop/internal/sharing"
)

const shutdownGrace = 10 * time.Second

type Config struct {
	ListenAddr string
	BaseURL    string
}

type Server struct {
	cfg  Config
	repo sharing.Repository
}

func New(cfg Config, repo sharing.Repository) *Server {
	return &Server{cfg: cfg, repo: repo}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	tpl, err := parseTemplates()
	if err != nil {
		return err
	}

	h := &handler{
		repo:    s.repo,
		baseURL: s.cfg.BaseURL,
		host:    hostFromURL(s.cfg.BaseURL),
		tpl:     tpl,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.index)
	mux.HandleFunc("GET /setup/{id}", h.setupGet)
	mux.HandleFunc("POST /setup/{id}", h.setupPost)
	mux.HandleFunc("GET /s/{id}", h.shareView)
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
