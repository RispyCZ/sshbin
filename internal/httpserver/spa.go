package httpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/rispycz/sshbin/web"
)

// spaServer serves the React single-page app. In dev it emits a shell that
// loads modules from the Vite dev server (HMR); in prod it reads the embedded
// build manifest and serves hashed assets from the embedded dist directory.
type spaServer struct {
	shell  []byte // fully rendered index document
	assets fs.FS  // embedded dist, nil in dev (Vite serves assets)
	files  http.Handler
}

type viteChunk struct {
	File string   `json:"file"`
	CSS  []string `json:"css"`
}

// newSPA builds the SPA server. When dev is true, viteOrigin is the Vite dev
// server URL (e.g. http://localhost:5173) and assets are served by Vite. host
// is the SSH host shown in the landing page's example scp command.
func newSPA(dev bool, viteOrigin, host string) (*spaServer, error) {
	shellTpl, err := template.ParseFS(templateFS, "templates/spa.html")
	if err != nil {
		return nil, err
	}

	var data struct {
		HeadExtra template.HTML
		ScriptSrc string
		Host      string
	}
	data.Host = host

	if dev {
		viteOrigin = strings.TrimRight(viteOrigin, "/")
		data.HeadExtra = template.HTML(fmt.Sprintf(`<script type="module">
      import RefreshRuntime from "%s/@react-refresh";
      RefreshRuntime.injectIntoGlobalHook(window);
      window.$RefreshReg$ = () => {};
      window.$RefreshSig$ = () => (type) => type;
      window.__vite_plugin_react_preamble_installed__ = true;
    </script>
    <script type="module" src="%s/@vite/client"></script>`, viteOrigin, viteOrigin))
		data.ScriptSrc = viteOrigin + "/src/main.tsx"

		shell, err := renderShell(shellTpl, data)
		if err != nil {
			return nil, err
		}
		return &spaServer{shell: shell}, nil
	}

	sub, err := web.DistFS()
	if err != nil {
		return nil, err
	}
	entry, err := readEntry(sub)
	if err != nil {
		return nil, err
	}
	var head strings.Builder
	for _, css := range entry.CSS {
		fmt.Fprintf(&head, `<link rel="stylesheet" href="/%s" />`+"\n    ", css)
	}
	data.HeadExtra = template.HTML(strings.TrimSpace(head.String()))
	data.ScriptSrc = "/" + entry.File

	shell, err := renderShell(shellTpl, data)
	if err != nil {
		return nil, err
	}
	return &spaServer{shell: shell, assets: sub, files: http.FileServer(http.FS(sub))}, nil
}

func renderShell(tpl *template.Template, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// readEntry resolves the main entry chunk from the Vite build manifest.
func readEntry(sub fs.FS) (viteChunk, error) {
	raw, err := fs.ReadFile(sub, ".vite/manifest.json")
	if err != nil {
		return viteChunk{}, fmt.Errorf("read vite manifest (run `vp build` first): %w", err)
	}
	var manifest map[string]viteChunk
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return viteChunk{}, fmt.Errorf("parse vite manifest: %w", err)
	}
	entry, ok := manifest["src/main.tsx"]
	if !ok {
		return viteChunk{}, fmt.Errorf("vite manifest missing entry src/main.tsx")
	}
	return entry, nil
}

func (s *spaServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// In prod, serve a real built file when one exists; otherwise fall back to
	// the shell so client-side routes (e.g. /shares) resolve on deep links.
	// Skip dotfiles (e.g. .vite/manifest.json) and directories so the embedded
	// build's internal metadata and file listings aren't exposed.
	if s.assets != nil {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" && !strings.HasPrefix(name, ".") && !strings.Contains(name, "/.") {
			if f, err := s.assets.Open(name); err == nil {
				info, statErr := f.Stat()
				f.Close()
				if statErr == nil && !info.IsDir() {
					s.files.ServeHTTP(w, r)
					return
				}
			}
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(s.shell); err != nil {
		log.Error(err)
	}

}
