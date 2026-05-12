package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

// HandleTUIShell handles a WebSocket connection that provides a local shell on the host.
// This is used for the "TUI Mode" in the AI panel (experimental feature).
func (s *Server) HandleTUIShell(w http.ResponseWriter, r *http.Request) {
	if !s.experimental {
		http.Error(w, "experimental features not enabled", http.StatusForbidden)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Choose shell — validate against known safe shells to prevent command injection (gosec G204)
	shell := resolveShell(os.Getenv("SHELL"))

	// Create command
	cmd := exec.Command(shell) // #nosec G204 -- shell is validated against an allowlist
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Start the command with a pty
	f, err := pty.Start(cmd)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\nFailed to start shell: %v\r\n", err)))
		return
	}
	defer func() {
		_ = f.Close()
		_ = cmd.Process.Kill()
	}()

	var mu sync.Mutex
	writeMsg := func(msgType string, data string) error {
		mu.Lock()
		defer mu.Unlock()
		m := TerminalMessage{
			Type: msgType,
			Data: data,
		}
		b, err := json.Marshal(m)
		if err != nil {
			return err
		}
		return conn.WriteMessage(websocket.TextMessage, b)
	}

	// Read from PTY and send to WebSocket
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := f.Read(buf)
			if n > 0 {
				if err := writeMsg("output", string(buf[:n])); err != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// Read from WebSocket and send to PTY
loop:
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg TerminalMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "input":
			if _, err := f.WriteString(msg.Data); err != nil {
				break loop
			}
		case "resize":
			_ = pty.Setsize(f, &pty.Winsize{
				Rows: msg.Rows,
				Cols: msg.Cols,
			})
		case "ping":
			_ = writeMsg("pong", "")
		}
	}
}

// resolveShell returns a validated shell path from the given candidate.
// Only known safe absolute paths are accepted; falls back to /bin/sh.
func resolveShell(candidate string) string {
	allowed := []string{
		"/bin/sh",
		"/bin/bash",
		"/bin/zsh",
		"/bin/fish",
		"/usr/bin/sh",
		"/usr/bin/bash",
		"/usr/bin/zsh",
		"/usr/bin/fish",
		"/usr/local/bin/bash",
		"/usr/local/bin/zsh",
		"/usr/local/bin/fish",
	}
	for _, s := range allowed {
		if candidate == s {
			return s
		}
	}
	// fallback
	for _, s := range []string{"/bin/bash", "/bin/sh"} {
		if _, err := os.Stat(s); err == nil {
			return s
		}
	}
	return "/bin/sh"
}
