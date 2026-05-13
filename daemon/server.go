package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/david-pena/clipboard/config"
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
	msgClear  msgType = "clear"
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
	hist    history.History
	watch   watcher.Watcher
	sockPath string

	mu      sync.Mutex
	clients map[chan string]struct{}
}

// NewServer creates a Server using config from disk, with SQLiteHistory and a 500ms PollingWatcher.
func NewServer() *Server {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("failed to load config, using defaults: %v", err)
	}

	dbPath, err := ensureDataPath()
	if err != nil {
		log.Fatalf("failed to prepare data directory: %v", err)
	}

	hist, err := history.NewSQLiteHistory(dbPath, cfg.MaxEntries)
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

// Run starts the watcher, listens on the Unix socket, handles SIGHUP for config
// reload, and blocks until SIGINT/SIGTERM.
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

	if err := writePIDFile(); err != nil {
		log.Printf("failed to write pid file: %v", err)
	}
	defer removePIDFile()

	log.Printf("daemon listening on %s (PID %d)", s.sockPath, os.Getpid())

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.handleConn(conn)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for sig := range sigs {
		switch sig {
		case syscall.SIGHUP:
			s.reloadConfig()
		default:
			log.Println("daemon shutting down...")
			return
		}
	}
}

// reloadConfig reads config from disk and applies updated max_entries to the history.
func (s *Server) reloadConfig() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("SIGHUP: failed to reload config: %v", err)
		return
	}
	if sh, ok := s.hist.(*history.SQLiteHistory); ok {
		sh.SetMaxSize(cfg.MaxEntries)
	}
	log.Printf("SIGHUP: config reloaded (max_entries=%d)", cfg.MaxEntries)
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

	init, _ := json.Marshal(serverMsg{Type: msgInit, Items: s.hist.List()})
	if _, err := conn.Write(append(init, '\n')); err != nil {
		log.Printf("failed to send init: %v", err)
		return
	}

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
			switch msg.Type {
			case msgSelect:
				log.Printf("selected: %.40s", msg.Item)
			case msgClear:
				s.hist.Clear()
				s.watch.Reset()
				log.Println("history cleared")
			default:
				return
			}
		}
	}
}

func dataDir() string {
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "clipboard-manager")
}

func ensureDataPath() (string, error) {
	dir := dataDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.db"), nil
}

func pidFilePath() string {
	return filepath.Join(dataDir(), "daemon.pid")
}

func writePIDFile() error {
	return os.WriteFile(pidFilePath(), []byte(fmt.Sprintf("%d", os.Getpid())), 0600)
}

func removePIDFile() {
	_ = os.Remove(pidFilePath())
}

// ReadPID returns the PID stored in the daemon pidfile, or 0 if not found.
func ReadPID() int {
	data, err := os.ReadFile(pidFilePath())
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(string(data))
	return pid
}
