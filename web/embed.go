package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// DistFS returns the built SPA (Vite output) rooted at the dist directory.
// Callers read ".vite/manifest.json" and hashed assets from it. Run `vp build`
// in this directory so dist exists before compiling the binary.
func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
