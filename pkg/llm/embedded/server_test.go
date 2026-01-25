package embedded

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Port != DefaultPort {
		t.Errorf("expected port %d, got %d", DefaultPort, cfg.Port)
	}

	if cfg.ContextSize != 2048 {
		t.Errorf("expected context size 2048, got %d", cfg.ContextSize)
	}

	if cfg.Threads < 1 || cfg.Threads > 2 {
		t.Errorf("expected threads 1-2, got %d", cfg.Threads)
	}

	if cfg.GPULayers != 0 {
		t.Errorf("expected GPU layers 0, got %d", cfg.GPULayers)
	}
}

func TestNewServer(t *testing.T) {
	cfg := DefaultConfig()
	server, err := NewServer(cfg)

	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if server.port != DefaultPort {
		t.Errorf("expected port %d, got %d", DefaultPort, server.port)
	}

	if server.dataDir == "" {
		t.Error("expected non-empty data directory")
	}

	// Check model path ends with default model
	if filepath.Base(server.modelPath) != DefaultModel {
		t.Errorf("expected model %s, got %s", DefaultModel, filepath.Base(server.modelPath))
	}
}

func TestServerStatus(t *testing.T) {
	cfg := DefaultConfig()
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	status := server.Status()

	if status.Running {
		t.Error("expected server to not be running")
	}

	if status.Model != DefaultModel {
		t.Errorf("expected model %s, got %s", DefaultModel, status.Model)
	}

	if status.Port != DefaultPort {
		t.Errorf("expected port %d, got %d", DefaultPort, status.Port)
	}
}

func TestIsPortAvailable(t *testing.T) {
	// Port 0 is always available (OS assigns)
	// Testing with common unused port
	available := isPortAvailable(59999)
	if !available {
		t.Skip("port 59999 is in use")
	}
}

func TestFindAvailablePort(t *testing.T) {
	port, err := findAvailablePort(59990, 59999)
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}

	if port < 59990 || port > 59999 {
		t.Errorf("port %d is outside expected range", port)
	}
}

func TestServerPaths(t *testing.T) {
	cfg := DefaultConfig()
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// DataDir should exist
	dataDir := server.DataDir()
	if dataDir == "" {
		t.Error("expected non-empty data directory")
	}

	// ModelPath should be within dataDir
	modelPath := server.ModelPath()
	if !filepath.HasPrefix(modelPath, dataDir) {
		t.Errorf("model path %s should be within data dir %s", modelPath, dataDir)
	}

	// ServerBinaryPath should contain "llama-server"
	binaryPath := server.ServerBinaryPath()
	if filepath.Base(binaryPath) != "llama-server" && filepath.Base(binaryPath) != "llama-server.exe" {
		t.Errorf("unexpected binary name: %s", filepath.Base(binaryPath))
	}
}

func TestServerIsRunning(t *testing.T) {
	cfg := DefaultConfig()
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if server.IsRunning() {
		t.Error("server should not be running initially")
	}
}

func TestDownloadFile_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "test.txt")

	err := downloadFile(ctx, "https://example.com/nonexistent", dest, nil)
	if err == nil {
		t.Error("expected error for cancelled context")
	}

	// File should not exist
	if _, err := os.Stat(dest); err == nil {
		t.Error("file should not exist after cancelled download")
	}
}

func TestCustomModelPath(t *testing.T) {
	customPath := "/custom/path/to/model.gguf"
	cfg := &Config{
		Port:        8081,
		ModelPath:   customPath,
		ContextSize: 2048,
		Threads:     2,
		GPULayers:   0,
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if server.ModelPath() != customPath {
		t.Errorf("expected model path %s, got %s", customPath, server.ModelPath())
	}
}

func TestGetModelContextInfo(t *testing.T) {
	tests := []struct {
		modelPath      string
		expectedMax    int
		expectedRec    int
		expectedMinRAM int
	}{
		{
			modelPath:      "/path/to/qwen2.5-0.5b-instruct-q4_k_m.gguf",
			expectedMax:    32768,
			expectedRec:    2048,
			expectedMinRAM: 2,
		},
		{
			modelPath:      "/path/to/Qwen2.5-1.5B-Instruct.gguf",
			expectedMax:    32768,
			expectedRec:    2048,
			expectedMinRAM: 4,
		},
		{
			modelPath:      "/path/to/llama-3.2-1b-instruct.gguf",
			expectedMax:    131072,
			expectedRec:    2048,
			expectedMinRAM: 3,
		},
		{
			modelPath:      "/path/to/smollm2-360m-instruct.gguf",
			expectedMax:    8192,
			expectedRec:    2048,
			expectedMinRAM: 2,
		},
		{
			modelPath:      "/path/to/unknown-model.gguf",
			expectedMax:    4096,
			expectedRec:    2048,
			expectedMinRAM: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.modelPath, func(t *testing.T) {
			info := GetModelContextInfo(tt.modelPath)

			if info.MaxContext != tt.expectedMax {
				t.Errorf("MaxContext: expected %d, got %d", tt.expectedMax, info.MaxContext)
			}
			if info.RecommendedCtx != tt.expectedRec {
				t.Errorf("RecommendedCtx: expected %d, got %d", tt.expectedRec, info.RecommendedCtx)
			}
			if info.MinRAM != tt.expectedMinRAM {
				t.Errorf("MinRAM: expected %d, got %d", tt.expectedMinRAM, info.MinRAM)
			}
		})
	}
}
