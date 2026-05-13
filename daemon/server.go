package daemon

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/david-pena/clipboard/history"
	"github.com/david-pena/clipboard/watcher"
)

const socketPath = "/.clipboard-manager.sock"

type msgType string

const (
	msgInit   msgType = "init"
	msgAdd    msgType = "add"
	msgSelect msgType = "select"
	msgCancel msgType = "cancel"
)

type serverMsg struct {
	Type  msgType  `json:"type"`
	Items []string `json:"items,omitempty"`
	Item  string   `json:"item,omitempty"`
}

type clientMsg struct {
	Type msgType `json:"type"`
	Item string  `json:"item,omitempty"`
}

// Server holds the clipboard history, watcher, and a list of active client channels for streaming.
type Server struct {
	hist     history.History
	watch    watcher.Watcher
	sockPath string

	mu      sync.Mutex
	clients map[chan string]struct{}
}

// NewServer creates a Server with a SQLiteHistory and a 500ms PollingWatcher.
// The database is stored at ~/.local/share/clipboard-manager/history.db.
func NewServer() *Server {
	dbPath, err := ensureDBPath()
	if err != nil {
		log.Fatalf("failed to prepare database directory: %v", err)
	}

	hist, err := history.NewSQLiteHistory(dbPath, 50)
	if err != nil {
		log.Fatalf("failed to open history database: %v", err)
	}

	return &Server{
		hist:     hist,
		watch:    watcher.NewPollingWatcher(500 * time.Millisecond),
		sockPath: os.Getenv("HOME") + socketPath,
		clients:  make(map[chan string]struct{}),
	}
}

func ensureDBPath() (string, error) {
	dir := filepath.Join(os.Getenv("HOME"), ".local", "share", "clipboard-manager")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.db"), nil
}

// Run starts the watcher, listens on the Unix socket, and blocks until SIGINT/SIGTERM.
func (s *Server) Run() {
	if err := s.watch.Start(func(content string) {
		s.hist.Add(content)
		log.Printf("captured: %.40s", content)
		s.broadcast(content)
	}); err != nil {
		log.Fatalf("failed to start watcher: %v", err)
	}
	defer s.watch.Stop()

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

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
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

// broadcast sends a new item to all connected clients.
func (s *Server) broadcast(item string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.clients {
		select {
		case ch <- item:
		default:
		}
	}
}

func (s *Server) subscribe() chan string {
	ch := make(chan string, 16)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	s.mu.Unlock()
	return ch
}

func (s *Server) unsubscribe(ch chan string) {
	s.mu.Lock()
	delete(s.clients, ch)
	s.mu.Unlock()
	close(ch)
}

// handleConn sends the current history, streams new items, then waits for the client's selection.
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	ch := s.subscribe()
	defer s.unsubscribe(ch)

	// Send initial history.
	init, _ := json.Marshal(serverMsg{Type: msgInit, Items: s.hist.List()})
	if _, err := conn.Write(append(init, '\n')); err != nil {
		log.Printf("failed to send init: %v", err)
		return
	}

	// Read all messages from the client (multiple selects, then cancel).
	clientDone := make(chan clientMsg, 16)
	go func() {
		scanner := bufio.NewScanner(conn)
		scanner.Buffer(make([]byte, 1<<20), 1<<20)
		for scanner.Scan() {
			var msg clientMsg
			if err := json.Unmarshal(scanner.Bytes(), &msg); err == nil {
				clientDone <- msg
			}
		}
		close(clientDone)
	}()

	for {
		select {
		case item, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(serverMsg{Type: msgAdd, Item: item})
			if _, err := conn.Write(append(data, '\n')); err != nil {
				return
			}
		case msg, ok := <-clientDone:
			if !ok {
				return
			}
			if msg.Type == msgSelect && msg.Item != "" {
				log.Printf("selected: %.40s", msg.Item)
				// Keep looping — client may send more selections.
			} else {
				return
			}
		}
	}
}
