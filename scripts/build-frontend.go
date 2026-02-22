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
	"css/layout.css",
	"css/login.css",
	"css/dashboard.css",
	"css/ai-panel.css",
	"css/settings.css",
	"css/modals.css",
	"css/terminal.css",
	"css/metrics.css",
}

// JS files in order of inclusion (dependencies first)
var jsFiles = []string{
	"js/core/state.js",
	"js/core/utils.js",
	"js/core/i18n.js",
	"js/core/api.js",
	"js/auth/session.js",
	"js/auth/login.js",
	"js/dashboard/resources.js",
	"js/dashboard/table.js",
	"js/dashboard/sorting.js",
	"js/dashboard/detail.js",
	"js/ai/chat.js",
	"js/ai/streaming.js",
	"js/ai/approval.js",
	"js/ai/history.js",
	"js/settings/index.js",
	"js/settings/llm.js",
	"js/settings/mcp.js",
	"js/settings/admin.js",
	"js/features/terminal.js",
	"js/features/logs.js",
	"js/features/search.js",
	"js/app.js",
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
