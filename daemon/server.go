package daemon

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.design/x/clipboard"

	"github.com/david-pena/clipboard/history"
	"github.com/david-pena/clipboard/watcher"
)

const socketPath = "/.clipboard-manager.sock"

// Server holds the clipboard history and watcher, and serves history over a Unix socket.
type Server struct {
	hist    history.History
	watch   watcher.Watcher
	sockPath string
}

// NewServer creates a Server with a 50-entry MemoryHistory and a 500ms PollingWatcher.
func NewServer() *Server {
	home := os.Getenv("HOME")
	return &Server{
		hist:     history.NewMemoryHistory(50),
		watch:    watcher.NewPollingWatcher(500 * time.Millisecond),
		sockPath: home + socketPath,
	}
}

// Run starts the watcher, listens on the Unix socket, and blocks until SIGINT/SIGTERM.
func (s *Server) Run() {
	if err := s.watch.Start(func(content string) {
		s.hist.Add(content)
		log.Printf("captured: %.40s", content)
	}); err != nil {
		log.Fatalf("failed to start watcher: %v", err)
	}
	defer s.watch.Stop()

	// Remove stale socket file if it exists.
	_ = os.Remove(s.sockPath)

	ln, err := net.Listen("unix", s.sockPath)
	if err != nil {
		log.Fatalf("failed to listen on socket %s: %v", s.sockPath, err)
	}
	defer func() {
		ln.Close()
		os.Remove(s.sockPath)
	}()

	log.Printf("daemon listening on %s", s.sockPath)

	// Accept connections in a separate goroutine so the main goroutine can
	// wait for OS signals.
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				// Listener was closed — stop accepting.
				return
			}
			go s.handleConn(conn)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("daemon shutting down...")
}

// handleConn sends the current history to the client, reads the selected item,
// and writes it to the clipboard if non-empty.
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	// Send history as a JSON array followed by a newline.
	items := s.hist.List()
	data, err := json.Marshal(items)
	if err != nil {
		log.Printf("failed to marshal history: %v", err)
		return
	}
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		log.Printf("failed to send history: %v", err)
		return
	}

	// Read the client's response — a JSON string (selected item or empty string).
	buf := make([]byte, 1<<20) // 1 MiB should be more than enough
	n, err := conn.Read(buf)
	if err != nil {
		log.Printf("failed to read selection: %v", err)
		return
	}

	var selected string
	if err := json.Unmarshal(buf[:n], &selected); err != nil {
		log.Printf("failed to unmarshal selection: %v", err)
		return
	}

	if selected != "" {
		clipboard.Write(clipboard.FmtText, []byte(selected))
		log.Printf("wrote to clipboard: %.40s", selected)
	}
}
