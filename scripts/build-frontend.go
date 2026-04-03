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
	"fmt"
	"os"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/web/frontendbundle"
)

const (
	staticDir = "pkg/web/static"
)

func main() {
	startTime := time.Now()
	fmt.Println("🔨 Building frontend assets...")

	if err := frontendbundle.WriteBundles(staticDir); err != nil {
		fmt.Printf("\n❌ Build completed with error: %v\n", err)
		os.Exit(1)
	}

	duration := time.Since(startTime)
	fmt.Println("✅ Created bundle.css")
	fmt.Println("✅ Created bundle.js")
	fmt.Printf("\n✅ Build completed in %v\n", duration.Round(time.Millisecond))
}
