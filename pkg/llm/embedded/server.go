// Package embedded provides an embedded LLM server using llama.cpp
// This is an optional component that can run a local SLLM for environments
// without external LLM access.
package embedded

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
)

const (
	// DefaultPort is the default port for the embedded LLM server
	DefaultPort = 8081

	// DefaultModel is the recommended model for 2core/4GB environments
	DefaultModel = "qwen2.5-0.5b-instruct-q4_k_m.gguf"

	// ModelURL is the Hugging Face URL for the default model
	ModelURL = "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q4_k_m.gguf"

	// ServerBinary is the name of the llama-server binary
	ServerBinary = "llama-server"

	// LlamaCppVersion is the llama.cpp release version for binary downloads
	LlamaCppVersion = "b4547"

	// Default context sizes for known models (conservative for 4GB RAM)
	DefaultContextSize = 2048
)

// ModelContextInfo contains recommended settings for known models
type ModelContextInfo struct {
	MaxContext     int // Maximum supported context length
	RecommendedCtx int // Recommended context for 4GB RAM
	MinRAM         int // Minimum RAM in GB
}

// KnownModels maps model filename patterns to their context info
var KnownModels = map[string]ModelContextInfo{
	// Qwen2.5 series
	"qwen2.5-0.5b": {MaxContext: 32768, RecommendedCtx: 2048, MinRAM: 2},
	"qwen2.5-1.5b": {MaxContext: 32768, RecommendedCtx: 2048, MinRAM: 4},
	"qwen2.5-3b":   {MaxContext: 32768, RecommendedCtx: 2048, MinRAM: 6},
	"qwen2.5-7b":   {MaxContext: 32768, RecommendedCtx: 1024, MinRAM: 8},

	// Llama 3.2 series
	"llama-3.2-1b": {MaxContext: 131072, RecommendedCtx: 2048, MinRAM: 3},
	"llama-3.2-3b": {MaxContext: 131072, RecommendedCtx: 2048, MinRAM: 6},

	// SmolLM2 series
	"smollm2-135m": {MaxContext: 8192, RecommendedCtx: 2048, MinRAM: 1},
	"smollm2-360m": {MaxContext: 8192, RecommendedCtx: 2048, MinRAM: 2},
	"smollm2-1.7b": {MaxContext: 8192, RecommendedCtx: 2048, MinRAM: 4},

	// Phi series
	"phi-3-mini":   {MaxContext: 4096, RecommendedCtx: 2048, MinRAM: 4},
	"phi-3.5-mini": {MaxContext: 4096, RecommendedCtx: 2048, MinRAM: 4},

	// Mistral series
	"mistral-7b": {MaxContext: 32768, RecommendedCtx: 1024, MinRAM: 8},
}

// GetModelContextInfo returns context info for a model based on filename
func GetModelContextInfo(modelPath string) ModelContextInfo {
	filename := strings.ToLower(filepath.Base(modelPath))

	for pattern, info := range KnownModels {
		if strings.Contains(filename, pattern) {
			return info
		}
	}

	// Default for unknown models
	return ModelContextInfo{
		MaxContext:     4096,
		RecommendedCtx: DefaultContextSize,
		MinRAM:         4,
	}
}

// Server manages the embedded llama.cpp server
type Server struct {
	mu         sync.RWMutex
	cmd        *exec.Cmd
	port       int
	modelPath  string
	dataDir    string
	running    bool
	endpoint   string
	cancelFunc context.CancelFunc

	// Configuration
	contextSize int  // Context window size (default: 2048 for low memory)
	threads     int  // Number of threads (default: 2 for 2-core)
	gpuLayers   int  // GPU layers (0 = CPU only)
	verbose     bool // Verbose logging
}

// Config holds configuration for the embedded server
type Config struct {
	Port        int
	ModelPath   string // Optional: custom model path
	ContextSize int    // Context window size
	Threads     int    // CPU threads
	GPULayers   int    // GPU layers (0 = CPU only)
	Verbose     bool
}

// DefaultConfig returns the default configuration optimized for 2core/4GB
func DefaultConfig() *Config {
	threads := runtime.NumCPU()
	if threads > 2 {
		threads = 2 // Limit to 2 for low-resource environments
	}

	return &Config{
		Port:        DefaultPort,
		ContextSize: 2048, // Small context for memory efficiency
		Threads:     threads,
		GPULayers:   0, // CPU only by default
		Verbose:     false,
	}
}

// NewServer creates a new embedded LLM server
func NewServer(cfg *Config) (*Server, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Use XDG data directory for storing models and binaries
	dataDir, err := xdg.DataFile("k13d/llm")
	if err != nil {
		dataDir = filepath.Join(os.TempDir(), "k13d", "llm")
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	modelPath := cfg.ModelPath
	if modelPath == "" {
		modelPath = filepath.Join(dataDir, "models", DefaultModel)
	}

	return &Server{
		port:        cfg.Port,
		modelPath:   modelPath,
		dataDir:     dataDir,
		contextSize: cfg.ContextSize,
		threads:     cfg.Threads,
		gpuLayers:   cfg.GPULayers,
		verbose:     cfg.Verbose,
	}, nil
}

// DataDir returns the data directory path
func (s *Server) DataDir() string {
	return s.dataDir
}

// ModelPath returns the model file path
func (s *Server) ModelPath() string {
	return s.modelPath
}

// Endpoint returns the server endpoint URL
func (s *Server) Endpoint() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.endpoint
}

// IsRunning returns whether the server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// ServerBinaryPath returns the path to the llama-server binary
func (s *Server) ServerBinaryPath() string {
	binaryName := ServerBinary
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	return filepath.Join(s.dataDir, "bin", binaryName)
}

// EnsureModel downloads the model if it doesn't exist
func (s *Server) EnsureModel(ctx context.Context, progressFn func(downloaded, total int64)) error {
	if _, err := os.Stat(s.modelPath); err == nil {
		// Model already exists
		return nil
	}

	// Create models directory
	modelsDir := filepath.Dir(s.modelPath)
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return fmt.Errorf("failed to create models directory: %w", err)
	}

	// Download model
	return downloadFile(ctx, ModelURL, s.modelPath, progressFn)
}

// EnsureBinary checks if the llama-server binary exists, downloads if not
func (s *Server) EnsureBinary() error {
	binaryPath := s.ServerBinaryPath()
	if _, err := os.Stat(binaryPath); err == nil {
		// Binary exists, make sure it's executable
		if runtime.GOOS != "windows" {
			if err := os.Chmod(binaryPath, 0755); err != nil {
				return fmt.Errorf("failed to make binary executable: %w", err)
			}
		}
		return nil
	}

	// Binary doesn't exist, download it
	fmt.Println("llama-server binary not found. Downloading...")
	if err := s.downloadBinary(); err != nil {
		return fmt.Errorf("failed to download llama-server: %w", err)
	}

	return nil
}

// downloadBinary downloads the llama-server binary from GitHub releases
func (s *Server) downloadBinary() error {
	// Determine the correct archive name based on OS and architecture
	archiveName, binarySubPath := getLlamaCppArchiveInfo()
	if archiveName == "" {
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	downloadURL := fmt.Sprintf("https://github.com/ggerganov/llama.cpp/releases/download/%s/%s", LlamaCppVersion, archiveName)
	fmt.Printf("Downloading from: %s\n", downloadURL)

	// Create bin directory
	binDir := filepath.Dir(s.ServerBinaryPath())
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Download to temp file
	tmpDir, err := os.MkdirTemp("", "llama-cpp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, archiveName)

	// Download the archive
	ctx := context.Background()
	if err := downloadFile(ctx, downloadURL, archivePath, func(downloaded, total int64) {
		if total > 0 {
			pct := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rDownloading llama-server: %.1f%% (%d / %d MB)", pct, downloaded/1024/1024, total/1024/1024)
		}
	}); err != nil {
		return fmt.Errorf("failed to download archive: %w", err)
	}
	fmt.Println()

	// Extract the archive
	fmt.Println("Extracting llama-server binary...")
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}

	if err := extractArchive(archivePath, extractDir); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// Find and copy the llama-server binary
	var srcBinary string
	if binarySubPath != "" {
		srcBinary = filepath.Join(extractDir, binarySubPath)
	} else {
		// Search for llama-server in extracted files
		srcBinary, err = findBinary(extractDir, "llama-server")
		if err != nil {
			return fmt.Errorf("failed to find llama-server in archive: %w", err)
		}
	}

	if _, err := os.Stat(srcBinary); err != nil {
		// Try alternative paths
		alternatives := []string{
			filepath.Join(extractDir, "llama-server"),
			filepath.Join(extractDir, "build", "bin", "llama-server"),
			filepath.Join(extractDir, "bin", "llama-server"),
		}
		for _, alt := range alternatives {
			if _, err := os.Stat(alt); err == nil {
				srcBinary = alt
				break
			}
		}
	}

	// Copy binary to destination
	if err := copyFile(srcBinary, s.ServerBinaryPath()); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	// Make executable
	if runtime.GOOS != "windows" {
		if err := os.Chmod(s.ServerBinaryPath(), 0755); err != nil {
			return fmt.Errorf("failed to make binary executable: %w", err)
		}
	}

	// Copy required dynamic libraries (.dylib, .so, .dll) from the same directory
	srcDir := filepath.Dir(srcBinary)
	destDir := filepath.Dir(s.ServerBinaryPath())

	libsCopied := 0
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		name := info.Name()
		// Copy dynamic libraries
		if strings.HasSuffix(name, ".dylib") ||
			strings.HasSuffix(name, ".so") ||
			strings.Contains(name, ".so.") ||
			strings.HasSuffix(name, ".dll") {
			destPath := filepath.Join(destDir, name)
			if err := copyFile(path, destPath); err != nil {
				fmt.Printf("Warning: failed to copy %s: %v\n", name, err)
			} else {
				libsCopied++
				// Make library executable on Unix
				if runtime.GOOS != "windows" {
					_ = os.Chmod(destPath, 0755)
				}
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Warning: error scanning for libraries: %v\n", err)
	}

	if libsCopied > 0 {
		fmt.Printf("Copied %d dynamic libraries\n", libsCopied)
	}

	fmt.Printf("llama-server installed to: %s\n", s.ServerBinaryPath())
	return nil
}

// getLlamaCppArchiveInfo returns the archive name and binary subpath for the current platform
func getLlamaCppArchiveInfo() (archiveName, binarySubPath string) {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return fmt.Sprintf("llama-%s-bin-macos-arm64.zip", LlamaCppVersion), "build/bin/llama-server"
		}
		return fmt.Sprintf("llama-%s-bin-macos-x64.zip", LlamaCppVersion), "build/bin/llama-server"
	case "linux":
		if runtime.GOARCH == "arm64" {
			return fmt.Sprintf("llama-%s-bin-ubuntu-arm64.zip", LlamaCppVersion), "build/bin/llama-server"
		}
		return fmt.Sprintf("llama-%s-bin-ubuntu-x64.zip", LlamaCppVersion), "build/bin/llama-server"
	case "windows":
		if runtime.GOARCH == "arm64" {
			return fmt.Sprintf("llama-%s-bin-win-arm64.zip", LlamaCppVersion), "llama-server.exe"
		}
		return fmt.Sprintf("llama-%s-bin-win-avx2-x64.zip", LlamaCppVersion), "llama-server.exe"
	}
	return "", ""
}

// extractArchive extracts a zip archive to the destination directory
func extractArchive(archivePath, destDir string) error {
	return unzip(archivePath, destDir)
}

// unzip extracts a zip file
func unzip(src, dest string) error {
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	stat, err := r.Stat()
	if err != nil {
		return err
	}

	// Use archive/zip
	zipReader, err := newZipReader(r, stat.Size())
	if err != nil {
		return err
	}

	for _, f := range zipReader.File {
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// newZipReader creates a zip reader from a file
func newZipReader(r io.ReaderAt, size int64) (*zipArchive, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, err
	}
	return &zipArchive{Reader: zr}, nil
}

type zipArchive struct {
	*zip.Reader
}

// findBinary searches for a binary in the directory tree
func findBinary(root, name string) (string, error) {
	var found string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}
		baseName := filepath.Base(path)
		if baseName == name || baseName == name+".exe" {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil && err != filepath.SkipAll {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("binary %s not found", name)
	}
	return found, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// Start starts the embedded LLM server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil // Already running
	}

	// Check model exists
	if _, err := os.Stat(s.modelPath); err != nil {
		return fmt.Errorf("model not found: %w", err)
	}

	// Check binary exists
	binaryPath := s.ServerBinaryPath()
	if _, err := os.Stat(binaryPath); err != nil {
		return fmt.Errorf("server binary not found: %w", err)
	}

	// Find available port
	port := s.port
	if !isPortAvailable(port) {
		var err error
		port, err = findAvailablePort(port, port+100)
		if err != nil {
			return fmt.Errorf("no available port: %w", err)
		}
	}

	// Create context with cancel
	serverCtx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel

	// Build command arguments
	args := []string{
		"--model", s.modelPath,
		"--port", fmt.Sprintf("%d", port),
		"--ctx-size", fmt.Sprintf("%d", s.contextSize),
		"--threads", fmt.Sprintf("%d", s.threads),
		"--host", "127.0.0.1",
		"--jinja", // Enable Jinja templates for tool calling support
	}

	if s.gpuLayers > 0 {
		args = append(args, "--n-gpu-layers", fmt.Sprintf("%d", s.gpuLayers))
	}

	// Start server process
	s.cmd = exec.CommandContext(serverCtx, binaryPath, args...)

	// Capture output - show startup messages, capture errors
	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	stderr, err := s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Monitor both stdout and stderr for startup and errors
	go s.monitorOutput(stdout, s.verbose)
	go s.monitorOutput(stderr, true) // Always show stderr for errors

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	s.port = port
	s.endpoint = fmt.Sprintf("http://127.0.0.1:%d", port)
	s.running = true

	// Wait for server to be ready
	if err := s.waitForReady(ctx, 30*time.Second); err != nil {
		_ = s.Stop()
		return fmt.Errorf("server failed to start: %w", err)
	}

	// Monitor process in background
	go s.monitorProcess()

	return nil
}

// Stop stops the embedded LLM server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	if s.cmd != nil && s.cmd.Process != nil {
		// Try graceful shutdown first
		if runtime.GOOS != "windows" {
			_ = s.cmd.Process.Signal(os.Interrupt)
			time.Sleep(500 * time.Millisecond)
		}
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}

	s.running = false
	s.cmd = nil
	return nil
}

// waitForReady waits for the server to respond to health checks
func (s *Server) waitForReady(ctx context.Context, timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)

	fmt.Print("Waiting for llama-server to be ready")

	// Try multiple endpoints - llama.cpp uses different ones depending on version
	healthEndpoints := []string{
		"/health",
		"/v1/models",
		"/",
	}

	attempts := 0
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			fmt.Println(" cancelled")
			return ctx.Err()
		default:
		}

		for _, endpoint := range healthEndpoints {
			resp, err := client.Get(s.endpoint + endpoint)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
					// Server is responding
					fmt.Println(" ready!")
					return nil
				}
			}
		}

		attempts++
		if attempts%2 == 0 {
			fmt.Print(".")
		}
		time.Sleep(1 * time.Second)
	}

	fmt.Println(" timeout")
	return fmt.Errorf("server did not become ready within %v", timeout)
}

// monitorProcess monitors the server process and updates state
func (s *Server) monitorProcess() {
	if s.cmd == nil {
		return
	}

	err := s.cmd.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = false
	if err != nil && s.verbose {
		fmt.Printf("embedded LLM server exited: %v\n", err)
	}
}

// monitorOutput reads server output for error detection
func (s *Server) monitorOutput(r io.Reader, showAll bool) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		lower := strings.ToLower(line)

		// Always show errors and important messages
		if strings.Contains(lower, "error") ||
			strings.Contains(lower, "fatal") ||
			strings.Contains(lower, "failed") ||
			strings.Contains(lower, "listening") ||
			strings.Contains(lower, "model loaded") ||
			showAll {
			fmt.Printf("[llama-server] %s\n", line)
		}
	}
}

// Status returns the current server status
type Status struct {
	Running     bool   `json:"running"`
	Endpoint    string `json:"endpoint,omitempty"`
	Model       string `json:"model"`
	ModelExists bool   `json:"model_exists"`
	Port        int    `json:"port"`
}

// Status returns the current server status
func (s *Server) Status() *Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, modelErr := os.Stat(s.modelPath)

	return &Status{
		Running:     s.running,
		Endpoint:    s.endpoint,
		Model:       filepath.Base(s.modelPath),
		ModelExists: modelErr == nil,
		Port:        s.port,
	}
}

// Helper functions

func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func findAvailablePort(start, end int) (int, error) {
	for port := start; port <= end; port++ {
		if isPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port in range %d-%d", start, end)
}

func downloadFile(ctx context.Context, url, dest string, progressFn func(downloaded, total int64)) error {
	// Create temporary file
	tmpFile := dest + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	defer out.Close()

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	total := resp.ContentLength

	// Copy with progress
	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			os.Remove(tmpFile)
			return ctx.Err()
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				os.Remove(tmpFile)
				return writeErr
			}
			downloaded += int64(n)
			if progressFn != nil {
				progressFn(downloaded, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			os.Remove(tmpFile)
			return err
		}
	}

	// Rename to final destination
	if err := os.Rename(tmpFile, dest); err != nil {
		os.Remove(tmpFile)
		return err
	}

	return nil
}
