// Package web serves the compiled single-page app from inside the API binary.
//
// Shipping the UI and the API on one origin is a deliberate choice, not a
// packaging convenience: the refresh token is an httpOnly cookie, so a split
// origin would force SameSite=None (which Safari's tracking prevention drops)
// plus a CORS allow-list to keep correct. One origin removes both problems, and
// the whole product deploys as a single container with no reverse proxy to own.
//
// Losing the reverse proxy means taking over the two useful things it did:
// cache headers and compression. Both are handled here, and both are decided
// once at startup rather than per request — the bundle is immutable for the life
// of the process, so there is nothing to recompute.
//
// `dist` is filled by the Docker build with the Vite output. A checked-in
// placeholder keeps `go build ./...` and the unit tests working without Node.
package web

import (
	"bytes"
	"compress/gzip"
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strconv"
	"strings"
)

// Plain `dist` rather than `all:dist` on purpose: the pattern without the `all:`
// prefix skips dot-prefixed files, which keeps this directory's .gitignore out
// of the bundle. Vite emits nothing else that starts with a dot.
//
//go:embed dist
var embedded embed.FS

// Bundled reports whether this binary carries a real UI build (as opposed to the
// placeholder page committed for backend-only builds).
func Bundled() bool {
	f, err := dist().Open("assets")
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

func dist() fs.FS {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic("web: dist not embedded: " + err.Error()) // guaranteed at build time
	}
	return sub
}

// asset is one file, prepared for serving.
type asset struct {
	plain        []byte
	gzipped      []byte // nil when compression did not pay for itself
	contentType  string
	cacheControl string
}

const indexFile = "index.html"

// Handler serves static assets and falls back to index.html so client-side
// routes survive a hard refresh. Register it as the router's no-route handler:
// real API routes are matched first, and anything still under /api is answered
// by apiMiss rather than the HTML shell — a JSON client that receives
// `<!doctype html>` is a genuinely confusing thing to debug.
func Handler(apiMiss http.HandlerFunc) http.HandlerFunc {
	assets := load(dist())
	index, hasIndex := assets[indexFile]

	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			apiMiss(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if a, ok := assets[assetName(r.URL.Path)]; ok {
			serve(w, r, a)
			return
		}
		if !hasIndex {
			http.Error(w, "ui not bundled in this build", http.StatusNotFound)
			return
		}
		serve(w, r, index)
	}
}

// load walks the bundle once and prepares every file for serving.
func load(files fs.FS) map[string]asset {
	assets := make(map[string]asset)
	_ = fs.WalkDir(files, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		data, readErr := fs.ReadFile(files, name)
		if readErr != nil {
			return nil
		}
		assets[name] = asset{
			plain:        data,
			gzipped:      precompress(name, data),
			contentType:  contentType(name),
			cacheControl: cacheControl(name),
		}
		return nil
	})
	return assets
}

// precompress gzips text assets at startup, so every request after the first
// costs a map lookup instead of a compression pass. Returns nil when the result
// is not smaller, which is the normal outcome for already-compressed formats.
func precompress(name string, data []byte) []byte {
	if !compressible(name) || len(data) < 1024 {
		return nil
	}
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil
	}
	if _, err := zw.Write(data); err != nil {
		return nil
	}
	if err := zw.Close(); err != nil {
		return nil
	}
	if buf.Len() >= len(data) {
		return nil
	}
	return buf.Bytes()
}

func compressible(name string) bool {
	switch path.Ext(name) {
	case ".html", ".js", ".mjs", ".css", ".json", ".webmanifest", ".svg", ".txt", ".map":
		return true
	default:
		return false
	}
}

// serve writes the response with an explicit 200. Explicit because this handler
// is mounted as the router's no-route fallback, and a router that has already
// decided the answer is 404 keeps that status unless the handler overrides it —
// which would hand every deep link a 404 carrying a perfectly good page.
func serve(w http.ResponseWriter, r *http.Request, a asset) {
	h := w.Header()
	h.Set("Content-Type", a.contentType)
	h.Set("Cache-Control", a.cacheControl)

	body := a.plain
	if a.gzipped != nil {
		// Vary regardless of what this particular client accepts: shared caches
		// must not serve a gzipped body to a client that did not ask for one.
		h.Set("Vary", "Accept-Encoding")
		if acceptsGzip(r) {
			h.Set("Content-Encoding", "gzip")
			body = a.gzipped
		}
	}
	h.Set("Content-Length", strconv.Itoa(len(body)))

	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(body)
}

// acceptsGzip is a substring test on purpose. The correct reading of the header
// would parse q-values, but no browser in the last decade sends `gzip;q=0`, and
// the failure mode of over-eager compression here is a client that must inflate
// a response it can already inflate.
func acceptsGzip(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}

// assetName maps a URL path to a path inside dist. Returns "" for anything that
// cannot name a file (a directory, or an attempt to walk out of the bundle —
// path.Clean collapses the dot-dots, and fs.Sub roots the tree anyway).
func assetName(urlPath string) string {
	clean := path.Clean("/" + urlPath)
	if clean == "/" || strings.HasSuffix(clean, "/") {
		return ""
	}
	return strings.TrimPrefix(clean, "/")
}

func cacheControl(name string) string {
	// Vite fingerprints everything under assets/ with a content hash, so those
	// URLs are immutable by construction. Anything else (service worker,
	// manifest, icons) is served under a stable name and must be revalidated.
	if strings.HasPrefix(name, "assets/") {
		return "public, max-age=31536000, immutable"
	}
	return "no-cache"
}

// contentType covers the extensions a Vite PWA build emits. Go's built-in table
// misses .webmanifest, and distroless images have no /etc/mime.types to fall
// back on, so the mapping is explicit rather than sniffed.
func contentType(name string) string {
	switch path.Ext(name) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".json", ".map":
		return "application/json; charset=utf-8"
	case ".webmanifest":
		return "application/manifest+json; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/vnd.microsoft.icon"
	case ".woff2":
		return "font/woff2"
	case ".txt":
		return "text/plain; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
