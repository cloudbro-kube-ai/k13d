// serve.go - Simple static file server for k13d documentation
//
// Usage:
//   go run serve.go
//   # Then open http://localhost:3000
//
// Or with custom port:
//   go run serve.go -port 8000

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	port := flag.Int("port", 3000, "Port to serve on")
	flag.Parse()

	// Get the directory where this script is located
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// Serve static files
	fs := http.FileServer(http.Dir(dir))

	// Add CORS and cache headers
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

		// Log request
		log.Printf("%s %s", r.Method, r.URL.Path)

		// Handle directory index
		path := filepath.Join(dir, r.URL.Path)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			r.URL.Path = filepath.Join(r.URL.Path, "index.html")
		}

		fs.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("\n")
	fmt.Printf("  ██╗  ██╗ ██╗██████╗ ██████╗  \n")
	fmt.Printf("  ██║ ██╔╝███║╚════██╗██╔══██╗ \n")
	fmt.Printf("  █████╔╝ ╚██║ █████╔╝██║  ██║ \n")
	fmt.Printf("  ██╔═██╗  ██║ ╚═══██╗██║  ██║ \n")
	fmt.Printf("  ██║  ██╗ ██║██████╔╝██████╔╝ \n")
	fmt.Printf("  ╚═╝  ╚═╝ ╚═╝╚═════╝ ╚═════╝  \n")
	fmt.Printf("\n")
	fmt.Printf("  📚 k13d Documentation Server\n")
	fmt.Printf("\n")
	fmt.Printf("  → Local:   http://localhost%s\n", addr)
	fmt.Printf("  → Network: http://0.0.0.0%s\n", addr)
	fmt.Printf("\n")
	fmt.Printf("  Press Ctrl+C to stop\n")
	fmt.Printf("\n")

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
