// build-frontend.go - Frontend asset bundler for k13d
//
// This script concatenates CSS and JS files into single bundles for production.
// It maintains Go embed compatibility while allowing modular source development.
//
// Usage:
//   go run scripts/build-frontend.go
//
// Output:
//   pkg/web/static/bundle.css - Combined CSS
//   pkg/web/static/bundle.js  - Combined JavaScript

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	staticDir = "pkg/web/static"
	cssDir    = "pkg/web/static/css"
	jsDir     = "pkg/web/static/js"
)

// CSS files in order of inclusion
var cssFiles = []string{
	"css/variables.css",
	"css/base.css",
	"css/components.css",
	"css/layout.css",
	"css/views.css",
	"css/views/overview.css",
	"css/views/custom-views.css",
	"css/views/theme-light.css",
	"css/views/responsive.css",
	"css/animations.css",
}

// JS files in order of inclusion (dependencies first)
var jsFiles = []string{
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

func main() {
	startTime := time.Now()
	fmt.Println("🔨 Building frontend assets...")

	// Check if we have modular source files
	cssModular := hasModularFiles(cssDir)
	jsModular := hasModularFiles(jsDir)

	if !cssModular && !jsModular {
		fmt.Println("ℹ️  No modular source files found.")
		fmt.Println("   CSS/JS are still embedded in index.html")
		fmt.Println("   Run 'make frontend-extract' to extract them first.")
		return
	}

	var errors []string

	// Bundle CSS
	if cssModular {
		if err := bundleFiles(cssFiles, filepath.Join(staticDir, "bundle.css"), "/* ", " */"); err != nil {
			errors = append(errors, fmt.Sprintf("CSS bundle error: %v", err))
		} else {
			fmt.Println("✅ Created bundle.css")
		}
	}

	// Bundle JS
	if jsModular {
		if err := bundleFiles(jsFiles, filepath.Join(staticDir, "bundle.js"), "// ", ""); err != nil {
			errors = append(errors, fmt.Sprintf("JS bundle error: %v", err))
		} else {
			fmt.Println("✅ Created bundle.js")
		}
	}

	// Report results
	duration := time.Since(startTime)
	if len(errors) > 0 {
		fmt.Println("\n❌ Build completed with errors:")
		for _, e := range errors {
			fmt.Printf("   - %s\n", e)
		}
		os.Exit(1)
	}

	fmt.Printf("\n✅ Build completed in %v\n", duration.Round(time.Millisecond))
}

// hasModularFiles checks if directory has any source files
func hasModularFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".css") || strings.HasSuffix(e.Name(), ".js")) {
			return true
		}
		// Check subdirectories for js/
		if e.IsDir() {
			subDir := filepath.Join(dir, e.Name())
			subEntries, err := os.ReadDir(subDir)
			if err == nil {
				for _, se := range subEntries {
					if strings.HasSuffix(se.Name(), ".js") {
						return true
					}
				}
			}
		}
	}
	return false
}

// bundleFiles concatenates multiple files into one bundle
func bundleFiles(files []string, outputPath string, commentStart, commentEnd string) error {
	var buf bytes.Buffer

	// Write header
	buf.WriteString(fmt.Sprintf("%sk13d Frontend Bundle - Generated at %s%s\n",
		commentStart, time.Now().Format(time.RFC3339), commentEnd))
	buf.WriteString(fmt.Sprintf("%sDO NOT EDIT - Modify source files in css/ or js/ directories%s\n\n",
		commentStart, commentEnd))

	filesIncluded := 0
	for _, file := range files {
		fullPath := filepath.Join(staticDir, file)

		// Skip if file doesn't exist
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			continue
		}

		// Read file
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		// Write file marker and content
		buf.WriteString(fmt.Sprintf("\n%s=== %s ===%s\n", commentStart, file, commentEnd))
		buf.Write(content)
		buf.WriteString("\n")
		filesIncluded++
	}

	if filesIncluded == 0 {
		return fmt.Errorf("no source files found")
	}

	// Write output
	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write bundle: %w", err)
	}

	return nil
}
