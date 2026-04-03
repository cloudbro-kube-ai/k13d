package frontendbundle

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// CSSFiles defines the bundle order for stylesheet assets under pkg/web/static.
var CSSFiles = []string{
	"css/variables.css",
	"css/base.css",
	"css/components.css",
	"css/layout.css",
	"css/views.css",
	"css/views/overview.css",
	"css/views/custom-views.css",
	"css/views/settings.css",
	"css/views/theme-light.css",
	"css/views/responsive.css",
	"css/animations.css",
}

// JSFiles defines the bundle order for JavaScript assets under pkg/web/static.
var JSFiles = []string{
	"js/core/utils.js",
	"js/modules/i18n.js",
	"js/modules/VirtualScroller.js",
	"js/modules/api.js",
	"js/modules/swr.js",
	"js/app.js",
	"js/features/custom-views/shared.js",
	"js/features/custom-views/metrics.js",
	"js/features/custom-views/topology-tree.js",
	"js/features/custom-views/applications.js",
	"js/features/custom-views/cluster-context.js",
	"js/features/ai-settings.js",
	"js/features/settings.js",
	"js/features/notifications.js",
	"js/features/insights.js",
	"js/features/workspace.js",
	"js/bootstrap.js",
}

// BuildCSSBundle builds bundle.css content from a static asset filesystem.
func BuildCSSBundle(fsys fs.FS) ([]byte, error) {
	return buildBundle(fsys, CSSFiles, "/* ", " */")
}

// BuildJSBundle builds bundle.js content from a static asset filesystem.
func BuildJSBundle(fsys fs.FS) ([]byte, error) {
	return buildBundle(fsys, JSFiles, "// ", "")
}

// WriteBundles writes bundle.css and bundle.js into the given static directory.
func WriteBundles(staticDir string) error {
	fsys := os.DirFS(staticDir)

	cssBundle, err := BuildCSSBundle(fsys)
	if err != nil {
		return fmt.Errorf("build CSS bundle: %w", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "bundle.css"), cssBundle, 0o644); err != nil {
		return fmt.Errorf("write bundle.css: %w", err)
	}

	jsBundle, err := BuildJSBundle(fsys)
	if err != nil {
		return fmt.Errorf("build JS bundle: %w", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "bundle.js"), jsBundle, 0o644); err != nil {
		return fmt.Errorf("write bundle.js: %w", err)
	}

	return nil
}

func buildBundle(fsys fs.FS, files []string, commentStart, commentEnd string) ([]byte, error) {
	var buf bytes.Buffer
	filesIncluded := 0

	buf.WriteString(fmt.Sprintf("%sk13d Frontend Bundle%s\n", commentStart, commentEnd))
	buf.WriteString(fmt.Sprintf("%sGenerated from embedded modular assets%s\n\n", commentStart, commentEnd))

	for _, file := range files {
		content, err := fs.ReadFile(fsys, file)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", file, err)
		}

		buf.WriteString(fmt.Sprintf("%s=== %s ===%s\n", commentStart, file, commentEnd))
		buf.Write(content)
		buf.WriteByte('\n')
		filesIncluded++
	}

	if filesIncluded == 0 {
		return nil, fmt.Errorf("no source files found")
	}

	return buf.Bytes(), nil
}
