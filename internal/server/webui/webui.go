// Package webui embeds the built web front and serves it (ADR-0039).
//
// The assets are committed, not built at release time, so a tagged binary
// carries its own page and `go build ./cmd/serve` needs no Node. NOTHING
// detects a stale dist: run `make web` whenever frontend/src/web/ changes.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var assets embed.FS

// Prefix is where the daemon mounts the page. It is not configurable: a
// movable mount point would be one more thing a deployment can get wrong, and
// the page's own asset links are relative so it only has to be consistent.
const Prefix = "/app/"

// Handler serves the embedded page under Prefix. Unknown paths fall back to
// the page itself so a bookmarked deep link lands on the one-page front.
func Handler() http.Handler {
	sub, err := fs.Sub(assets, "dist")
	if err != nil {
		// Unreachable: the directory is embedded at compile time. Serving
		// nothing is still better than panicking a daemon at startup.
		return http.NotFoundHandler()
	}
	files := http.FileServer(http.FS(sub))
	return http.StripPrefix(strings.TrimSuffix(Prefix, "/"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" || path == "index.html" {
			serveIndex(w, r, sub)
			return
		}
		if _, err := fs.Stat(sub, path); err != nil {
			serveIndex(w, r, sub)
			return
		}
		files.ServeHTTP(w, r)
	}))
}

func serveIndex(w http.ResponseWriter, r *http.Request, sub fs.FS) {
	data, err := fs.ReadFile(sub, "web.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// The page is one file with hashed asset names beside it: it must not be
	// cached, they may be cached forever. Getting this backwards is how a user
	// ends up running last week's page against this week's daemon.
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}
