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

	"golang.design/x/clipboard"

	"github.com/david-pena/clipboard/config"
	"github.com/david-pena/clipboard/history"
	"github.com/david-pena/clipboard/watcher"
)

const socketPath = "/.clipboard-manager.sock"

type msgType string

const (
	msgInit         msgType = "init"
	msgAdd          msgType = "add"
	msgRefresh      msgType = "refresh"
	msgSelect       msgType = "select"
	msgCancel       msgType = "cancel"
	msgClear        msgType = "clear"
	msgSearch       msgType = "search"
	msgSearchResult msgType = "search_result"
)

type serverMsg struct {
	Type         msgType                  `json:"type"`
	Items        []history.ClipboardEntry `json:"items,omitempty"`
	Item         *history.ClipboardEntry  `json:"item,omitempty"`
	Query        string                   `json:"query,omitempty"`
	TotalMatches int                      `json:"total_matches,omitempty"`
}

type clientMsg struct {
	Type    msgType `json:"type"`
	EntryID int64   `json:"entry_id,omitempty"`
	Query   string  `json:"query,omitempty"`
}

// Server holds the clipboard history, watcher, and a list of active client channels for streaming.
type Server struct {
	hist     history.History
	watch    watcher.Watcher
	sockPath string

	mu      sync.Mutex
	clients map[chan serverMsg]struct{}
}

// NewServer creates a Server using config from disk, with SQLiteHistory and a 500ms PollingWatcher.
func NewServer() *Server {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("failed to load config, using defaults: %v", err)
	}

	dbPath, imageDir, err := ensureDataPaths()
	if err != nil {
		log.Fatalf("failed to prepare data directory: %v", err)
	}

	hist, err := history.NewSQLiteHistory(dbPath, imageDir, cfg.MaxEntries)
	if err != nil {
		log.Fatalf("failed to open history database: %v", err)
	}

	return &Server{
		hist:     hist,
		watch:    watcher.NewPollingWatcher(500*time.Millisecond, imageDir, cfg.MaxImageSizeMB),
		sockPath: os.Getenv("HOME") + socketPath,
		clients:  make(map[chan serverMsg]struct{}),
	}
}

// Run starts the watcher, listens on the Unix socket, handles SIGHUP for config
// reload, and blocks until SIGINT/SIGTERM.
func (s *Server) Run() {
	if err := s.watch.Start(func(entry history.ClipboardEntry) {
		s.hist.Add(entry)
		log.Printf("captured [%s]: %.40s", entry.Type, entry.Content)
		s.broadcast(entry)
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

// reloadConfig reads config from disk and applies updated settings to the history.
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
	s.broadcastRefresh()
}

// broadcast sends a new entry to all connected clients.
func (s *Server) broadcast(entry history.ClipboardEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg := serverMsg{Type: msgAdd, Item: &entry}
	for ch := range s.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

// broadcastRefresh sends the current history list to all connected clients.
func (s *Server) broadcastRefresh() {
	items := s.hist.List()
	msg := serverMsg{Type: msgRefresh, Items: items}
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (s *Server) maxEntries() int {
	if sh, ok := s.hist.(*history.SQLiteHistory); ok {
		return sh.MaxSize()
	}
	return 50
}

func (s *Server) subscribe() chan serverMsg {
	ch := make(chan serverMsg, 16)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	s.mu.Unlock()
	return ch
}

func (s *Server) unsubscribe(ch chan serverMsg) {
	s.mu.Lock()
	delete(s.clients, ch)
	s.mu.Unlock()
	close(ch)
}

// handleConn sends the current history, streams new entries, then waits for the client's selection.
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	ch := s.subscribe()
	defer s.unsubscribe(ch)

	items := s.hist.List()
	init, _ := json.Marshal(serverMsg{Type: msgInit, Items: items})
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
		case msg, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(msg)
			if _, err := conn.Write(append(data, '\n')); err != nil {
				return
			}
		case msg, ok := <-clientDone:
			if !ok {
				return
			}
			switch msg.Type {
			case msgSelect:
				log.Printf("selected entry id=%d", msg.EntryID)
			case msgClear:
				s.hist.Clear()
				clipboard.Write(clipboard.FmtText, []byte{})
				s.watch.Reset()
				log.Println("history cleared")
			case msgSearch:
				result := s.hist.Search(msg.Query, s.maxEntries())
				data, _ := json.Marshal(serverMsg{
					Type:         msgSearchResult,
					Items:        result.Entries,
					TotalMatches: result.TotalMatches,
				})
				if _, err := conn.Write(append(data, '\n')); err != nil {
					return
				}
			default:
				return
			}
		}
	}
}

func dataDir() string {
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "clipboard-manager")
}

func ensureDataPaths() (dbPath, imageDir string, err error) {
	dir := dataDir()
	if err = os.MkdirAll(dir, 0700); err != nil {
		return
	}
	imageDir, err = history.EnsureImageDir(dir)
	if err != nil {
		return
	}
	dbPath = filepath.Join(dir, "history.db")
	return
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
