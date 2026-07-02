package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSPA_ProdServesShell(t *testing.T) {
	s, err := newSPA(false, "", "example.com")
	if err != nil {
		t.Fatalf("newSPA: %v", err)
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<div id="root">`) {
		t.Error("shell missing SPA mount point")
	}
	if !strings.Contains(body, `src="/assets/`) {
		t.Error("prod shell should reference a hashed /assets/ bundle from the manifest")
	}
	if !strings.Contains(body, `name="sshbin-host" content="example.com"`) {
		t.Error("shell should expose the SSH host for the landing page scp example")
	}
}

func TestSPA_ProdFallbackForClientRoute(t *testing.T) {
	s, _ := newSPA(false, "", "example.com")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest("GET", "/shares/deep/link", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `<div id="root">`) {
		t.Error("deep link should fall back to the shell")
	}
}

func TestSPA_ProdServesHashedAsset(t *testing.T) {
	s, _ := newSPA(false, "", "example.com")
	idx := httptest.NewRecorder()
	s.ServeHTTP(idx, httptest.NewRequest("GET", "/", nil))
	asset := extractAssetPath(idx.Body.String())
	if asset == "" {
		t.Fatal("no hashed asset referenced in shell")
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest("GET", asset, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d for %s, want 200", rec.Code, asset)
	}
	if ct := rec.Header().Get("Content-Type"); strings.HasPrefix(ct, "text/html") {
		t.Errorf("asset %s served as html (fell through to shell)", asset)
	}
}

func TestSPA_ProdBlocksInternalPaths(t *testing.T) {
	s, _ := newSPA(false, "", "example.com")
	// Dotfiles (build metadata) and directories must not be served from the
	// embedded build; they fall back to the SPA shell instead.
	for _, p := range []string{"/.vite/manifest.json", "/.vite/", "/assets/"} {
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, httptest.NewRequest("GET", p, nil))
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("%s served as %q, want shell fallback", p, ct)
		}
		if !strings.Contains(rec.Body.String(), `<div id="root">`) {
			t.Errorf("%s did not fall back to the shell", p)
		}
	}
}

func TestSPA_DevPointsAtViteServer(t *testing.T) {
	s, err := newSPA(true, "http://localhost:5173", "ssh.example.com")
	if err != nil {
		t.Fatalf("newSPA dev: %v", err)
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "http://localhost:5173/@vite/client") {
		t.Error("dev shell should load the Vite client")
	}
	if !strings.Contains(body, "http://localhost:5173/src/main.tsx") {
		t.Error("dev shell should load the entry module from Vite")
	}
	if !strings.Contains(body, "@react-refresh") {
		t.Error("dev shell should install the React Refresh preamble")
	}
}

func TestSPA_DevDoesNotServeAssets(t *testing.T) {
	s, _ := newSPA(true, "http://localhost:5173", "ssh.example.com")
	rec := httptest.NewRecorder()
	// In dev, assets come from Vite, so even an /assets/ path returns the shell.
	s.ServeHTTP(rec, httptest.NewRequest("GET", "/assets/whatever.js", nil))
	if !strings.Contains(rec.Body.String(), `<div id="root">`) {
		t.Error("dev should always return the shell")
	}
}

func extractAssetPath(html string) string {
	const marker = `src="/assets/`
	i := strings.Index(html, marker)
	if i < 0 {
		return ""
	}
	rest := html[i+len(`src="`):]
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return ""
	}
	return rest[:end]
}
