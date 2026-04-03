package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestServeGeneratedBundle_FallbackCSS(t *testing.T) {
	resetGeneratedBundleCache()

	req := httptest.NewRequest(http.MethodGet, "/bundle.css", nil)
	w := httptest.NewRecorder()

	staticFS := fstest.MapFS{
		"css/variables.css": &fstest.MapFile{Data: []byte(":root { --bg: #000; }\n")},
		"css/base.css":      &fstest.MapFile{Data: []byte("body { background: var(--bg); }\n")},
	}

	if !serveGeneratedBundle(w, req, staticFS) {
		t.Fatal("serveGeneratedBundle() = false, want true")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := w.Header().Get("Content-Type"); got != "text/css; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/css; charset=utf-8", got)
	}
	if body := w.Body.String(); body == "" {
		t.Fatal("expected generated CSS bundle body")
	}
}

func TestServeGeneratedBundle_UsesPrebuiltBundle(t *testing.T) {
	resetGeneratedBundleCache()

	req := httptest.NewRequest(http.MethodGet, "/bundle.js", nil)
	w := httptest.NewRecorder()

	staticFS := fstest.MapFS{
		"bundle.js": &fstest.MapFile{Data: []byte("console.log('prebuilt');\n")},
		"js/app.js": &fstest.MapFile{Data: []byte("console.log('fallback');\n")},
	}

	if !serveGeneratedBundle(w, req, staticFS) {
		t.Fatal("serveGeneratedBundle() = false, want true")
	}
	if got := w.Header().Get("Content-Type"); got != "application/javascript; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/javascript; charset=utf-8", got)
	}
	if got := w.Body.String(); got != "console.log('prebuilt');\n" {
		t.Fatalf("body = %q, want prebuilt bundle", got)
	}
}

func resetGeneratedBundleCache() {
	embeddedBundleMu.Lock()
	defer embeddedBundleMu.Unlock()
	embeddedBundles = map[string][]byte{}
}
