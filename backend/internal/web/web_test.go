package web

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// The handler is only ever mounted as a no-route fallback, and that is where the
// interesting failure lives: gin pre-sets 404 before running no-route handlers,
// so a fallback that forgets to override the status serves every client-side
// route as "404 Not Found" with a perfectly good page in the body. Browsers
// render it, so it looks fine — until a crawler, an uptime check or a service
// worker reads the status. These tests go through gin for exactly that reason.
func testRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	r.NoRoute(gin.WrapF(Handler(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	})))
	return r
}

func get(t *testing.T, r http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestClientSideRouteServesShellWith200(t *testing.T) {
	r := testRouter()
	for _, path := range []string{"/", "/login", "/inventory", "/dashboard"} {
		rec := get(t, r, http.MethodGet, path)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s: status = %d, want 200", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("GET %s: content-type = %q, want html", path, ct)
		}
		if rec.Body.Len() == 0 {
			t.Errorf("GET %s: empty body", path)
		}
	}
}

func TestShellIsNotCached(t *testing.T) {
	// The shell names the hashed asset files; a cached copy outlives the deploy
	// that removed them.
	rec := get(t, testRouter(), http.MethodGet, "/dashboard")
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cc)
	}
}

func TestUnknownAPIPathStaysJSON(t *testing.T) {
	rec := get(t, testRouter(), http.MethodGet, "/api/v1/does-not-exist")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if got := rec.Body.String(); got != `{"error":"not found"}` {
		t.Errorf("body = %q, want a JSON error (an API client must never be handed HTML)", got)
	}
}

func TestRealRoutesAreNotShadowed(t *testing.T) {
	rec := get(t, testRouter(), http.MethodGet, "/api/v1/ping")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"ok":true}` {
		t.Errorf("status = %d body = %q, want the registered handler to win", rec.Code, rec.Body.String())
	}
}

func TestHeadServesNoBody(t *testing.T) {
	rec := get(t, testRouter(), http.MethodHead, "/login")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("HEAD returned %d bytes of body", rec.Body.Len())
	}
	if cl := rec.Header().Get("Content-Length"); cl == "" || cl == "0" {
		t.Errorf("Content-Length = %q, want the length the GET would return", cl)
	}
}

func TestWriteMethodsAreRejected(t *testing.T) {
	// A stray POST to a client-side route must not be answered with the shell,
	// which would read as success to whatever sent it.
	rec := get(t, testRouter(), http.MethodPost, "/login")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// There is no reverse proxy in front of this binary, so compression is the
// handler's job. Two things have to hold: a client that asks for gzip gets bytes
// that inflate to exactly the uncompressed response, and a shared cache is told
// the body varies by encoding — otherwise it will eventually hand a gzipped body
// to a client that never asked for one.
func TestGzipNegotiation(t *testing.T) {
	r := testRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	r.ServeHTTP(rec, req)

	if enc := rec.Header().Get("Content-Encoding"); enc != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", enc)
	}
	if v := rec.Header().Get("Vary"); v != "Accept-Encoding" {
		t.Errorf("Vary = %q, want Accept-Encoding", v)
	}
	if cl := rec.Header().Get("Content-Length"); cl != strconv.Itoa(rec.Body.Len()) {
		t.Errorf("Content-Length = %q but body is %d bytes", cl, rec.Body.Len())
	}

	zr, err := gzip.NewReader(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatalf("body is not valid gzip: %v", err)
	}
	inflated, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("inflate: %v", err)
	}
	plain := get(t, r, http.MethodGet, "/login")
	if !bytes.Equal(inflated, plain.Body.Bytes()) {
		t.Error("gzipped response does not inflate to the uncompressed one")
	}
	if rec.Body.Len() >= plain.Body.Len() {
		t.Errorf("compressed %d bytes >= plain %d bytes", rec.Body.Len(), plain.Body.Len())
	}
	if plain.Header().Get("Content-Encoding") != "" {
		t.Error("served an encoded body to a client that did not ask for one")
	}
	if plain.Header().Get("Vary") != "Accept-Encoding" {
		t.Error("missing Vary on the uncompressed response — caches will mix the two up")
	}
}

func TestAssetCachePolicy(t *testing.T) {
	// Vite content-hashes everything under assets/, so only those URLs may be
	// cached forever. Getting this backwards is invisible in development and
	// permanent in production.
	cases := map[string]string{
		"assets/index-a1b2c3.js":  "public, max-age=31536000, immutable",
		"assets/index-a1b2c3.css": "public, max-age=31536000, immutable",
		"sw.js":                   "no-cache",
		"manifest.webmanifest":    "no-cache",
		"icon.svg":                "no-cache",
		"index.html":              "no-cache",
	}
	for name, want := range cases {
		if got := cacheControl(name); got != want {
			t.Errorf("cacheControl(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestAssetNameCannotEscapeTheBundle(t *testing.T) {
	// Two layers hold here: path.Clean collapses the dot-dots before they reach
	// the filesystem, and fs.Sub roots the tree at dist/ regardless. Assert the
	// property that matters — the resulting name is always relative and
	// dot-dot-free — rather than a specific rewrite.
	for _, in := range []string{"/", "/inventory/", "/../go.mod", "/a/../../../etc/passwd", "//..//.."} {
		got := assetName(in)
		if strings.HasPrefix(got, "/") || strings.Contains(got, "..") {
			t.Errorf("assetName(%q) = %q, want a bundle-relative name", in, got)
		}
	}
}

func TestTraversalAttemptGetsTheShell(t *testing.T) {
	// End to end: a traversal attempt is just an unknown client-side route.
	rec := get(t, testRouter(), http.MethodGet, "/../go.mod")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<!doctype html>") {
		t.Errorf("status = %d, body starts %.40q; want the app shell", rec.Code, rec.Body.String())
	}
}

func TestManifestContentType(t *testing.T) {
	// Go's built-in table has no entry for .webmanifest and distroless images
	// carry no /etc/mime.types, so the wrong answer here silently breaks
	// "install to home screen".
	if got := contentType("manifest.webmanifest"); got != "application/manifest+json; charset=utf-8" {
		t.Errorf("contentType(.webmanifest) = %q", got)
	}
}
