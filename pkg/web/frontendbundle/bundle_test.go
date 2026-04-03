package frontendbundle

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestBuildCSSBundle(t *testing.T) {
	fsys := fstest.MapFS{
		"css/variables.css": &fstest.MapFile{Data: []byte(":root { --bg: #000; }\n")},
		"css/base.css":      &fstest.MapFile{Data: []byte("body { background: var(--bg); }\n")},
	}

	bundle, err := BuildCSSBundle(fsys)
	if err != nil {
		t.Fatalf("BuildCSSBundle() error = %v", err)
	}

	got := string(bundle)
	if !strings.Contains(got, "css/variables.css") || !strings.Contains(got, "css/base.css") {
		t.Fatalf("bundle missing expected file markers:\n%s", got)
	}
	if strings.Index(got, "css/variables.css") > strings.Index(got, "css/base.css") {
		t.Fatalf("expected variables.css before base.css:\n%s", got)
	}
}

func TestBuildJSBundle(t *testing.T) {
	fsys := fstest.MapFS{
		"js/app.js":       &fstest.MapFile{Data: []byte("window.app = true;\n")},
		"js/bootstrap.js": &fstest.MapFile{Data: []byte("window.boot = true;\n")},
	}

	bundle, err := BuildJSBundle(fsys)
	if err != nil {
		t.Fatalf("BuildJSBundle() error = %v", err)
	}

	got := string(bundle)
	if !strings.Contains(got, "js/app.js") || !strings.Contains(got, "js/bootstrap.js") {
		t.Fatalf("bundle missing expected file markers:\n%s", got)
	}
	if strings.Index(got, "js/app.js") > strings.Index(got, "js/bootstrap.js") {
		t.Fatalf("expected app.js before bootstrap.js:\n%s", got)
	}
}
