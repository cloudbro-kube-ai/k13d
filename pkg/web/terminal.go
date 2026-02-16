package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// allowedOrigins contains the list of allowed origins for WebSocket connections
var allowedOrigins = getWebSocketAllowedOrigins()

// getWebSocketAllowedOrigins returns allowed origins from environment or defaults
func getWebSocketAllowedOrigins() []string {
	if origins := os.Getenv("K13D_WS_ALLOWED_ORIGINS"); origins != "" {
		return strings.Split(origins, ",")
	}
	// Default: allow localhost for development
	return []string{"http://localhost", "https://localhost", "http://127.0.0.1", "https://127.0.0.1"}
}

// checkOrigin validates WebSocket origin against allowed list
func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // Allow requests without origin (same-origin)
	}

	// In development mode (K13D_DEV=true), allow all origins
	if os.Getenv("K13D_DEV") == "true" {
		return true
	}

	// Check against allowed origins (exact host match to prevent subdomain bypass)
	for _, allowed := range allowedOrigins {
		if origin == allowed || strings.HasPrefix(origin, allowed+":") || strings.HasPrefix(origin, allowed+"/") {
			return true
		}
	}
	return false
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     checkOrigin,
}

// TerminalMessage represents a message to/from the terminal
type TerminalMessage struct {
	Type string `json:"type"` // "input", "output", "resize", "error"
	Data string `json:"data,omitempty"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

// TerminalSession manages a single terminal session
type TerminalSession struct {
	conn     *websocket.Conn
	sizeChan chan remotecommand.TerminalSize
	doneChan chan struct{}
	mu       sync.Mutex
}

// NewTerminalSession creates a new terminal session
func NewTerminalSession(conn *websocket.Conn) *TerminalSession {
	return &TerminalSession{
		conn:     conn,
		sizeChan: make(chan remotecommand.TerminalSize, 1),
		doneChan: make(chan struct{}),
	}
}

// Read implements io.Reader for terminal input
func (t *TerminalSession) Read(p []byte) (int, error) {
	_, message, err := t.conn.ReadMessage()
	if err != nil {
		return 0, err
	}

	var msg TerminalMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		return copy(p, message), nil
	}

	switch msg.Type {
	case "input":
		return copy(p, []byte(msg.Data)), nil
	case "resize":
		t.sizeChan <- remotecommand.TerminalSize{
			Width:  msg.Cols,
			Height: msg.Rows,
		}
		return 0, nil
	}

	return 0, nil
}

// Write implements io.Writer for terminal output
func (t *TerminalSession) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	msg := TerminalMessage{
		Type: "output",
		Data: string(p),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal terminal message: %w", err)
	}

	if err := t.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Next implements remotecommand.TerminalSizeQueue
func (t *TerminalSession) Next() *remotecommand.TerminalSize {
	select {
	case size := <-t.sizeChan:
		return &size
	case <-t.doneChan:
		return nil
	}
}

// Close closes the terminal session
func (t *TerminalSession) Close() {
	close(t.doneChan)
	t.conn.Close()
}

// SendError sends an error message to the client
func (t *TerminalSession) SendError(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	msg := TerminalMessage{
		Type: "error",
		Data: err.Error(),
	}
	data, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		return
	}
	t.conn.WriteMessage(websocket.TextMessage, data)
}

// TerminalHandler handles WebSocket terminal connections
type TerminalHandler struct {
	k8sClient *k8s.Client
}

// NewTerminalHandler creates a new terminal handler
func NewTerminalHandler(k8sClient *k8s.Client) *TerminalHandler {
	return &TerminalHandler{k8sClient: k8sClient}
}

// HandleTerminal handles WebSocket terminal requests
// URL: /api/terminal/{namespace}/{pod}?container={container}
func (h *TerminalHandler) HandleTerminal(w http.ResponseWriter, r *http.Request) {
	// Parse parameters
	path := r.URL.Path
	parts := splitPath(path, "/api/terminal/")
	if len(parts) < 2 {
		http.Error(w, "Invalid path: expected /api/terminal/{namespace}/{pod}", http.StatusBadRequest)
		return
	}

	namespace := parts[0]
	podName := parts[1]
	container := r.URL.Query().Get("container")

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	session := NewTerminalSession(conn)
	defer session.Close()

	// Get pod to find default container if not specified
	if container == "" {
		pod, err := h.k8sClient.Clientset.CoreV1().Pods(namespace).Get(r.Context(), podName, metav1.GetOptions{})
		if err != nil {
			session.SendError(fmt.Errorf("failed to get pod: %v", err))
			return
		}
		if len(pod.Spec.Containers) > 0 {
			container = pod.Spec.Containers[0].Name
		}
	}

	// Create exec request
	req := h.k8sClient.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{"/bin/sh", "-c", "if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(h.k8sClient.Config, "POST", req.URL())
	if err != nil {
		session.SendError(fmt.Errorf("failed to create executor: %v", err))
		return
	}

	// Run the terminal session
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             session,
		Stdout:            session,
		Stderr:            session,
		Tty:               true,
		TerminalSizeQueue: session,
	})

	if err != nil {
		session.SendError(fmt.Errorf("exec error: %v", err))
	}
}

// splitPath splits a URL path and removes the prefix
func splitPath(path, prefix string) []string {
	path = strings.TrimPrefix(path, prefix)
	parts := strings.Split(path, "/")
	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
