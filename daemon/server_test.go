package daemon

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/david-pena/clipboard/history"
	"github.com/david-pena/clipboard/watcher"
)

// testServer builds a Server wired to an in-memory history and a no-op watcher,
// listening on a temp socket. It returns the server and its socket path.
func testServer(t *testing.T) (*Server, string) {
	t.Helper()

	sockPath := filepath.Join(t.TempDir(), "test.sock")

	s := &Server{
		hist:     history.NewMemoryHistory(50),
		watch:    &noopWatcher{},
		sockPath: sockPath,
		clients:  make(map[chan serverMsg]struct{}),
	}
	return s, sockPath
}

// startListening starts the accept loop on a pre-configured server and returns
// a stop function that closes the listener.
func startListening(t *testing.T, s *Server) func() {
	t.Helper()

	_ = os.Remove(s.sockPath)
	ln, err := net.Listen("unix", s.sockPath)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.handleConn(conn)
		}
	}()

	return func() { ln.Close() }
}

func dial(t *testing.T, sockPath string) net.Conn {
	t.Helper()
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("failed to connect to server: %v", err)
	}
	return conn
}

func readMsg(t *testing.T, scanner *bufio.Scanner) serverMsg {
	t.Helper()
	if !scanner.Scan() {
		t.Fatal("expected a message but scanner returned nothing")
	}
	var msg serverMsg
	if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
		t.Fatalf("failed to unmarshal server message: %v", err)
	}
	return msg
}

func sendMsg(t *testing.T, conn net.Conn, msg clientMsg) {
	t.Helper()
	data, _ := json.Marshal(msg)
	conn.Write(append(data, '\n'))
}

// --- Tests ---

func TestServer_SendsInitOnConnect(t *testing.T) {
	s, sockPath := testServer(t)
	s.hist.Add(history.ClipboardEntry{Type: history.EntryTypeText, Content: "first"})
	s.hist.Add(history.ClipboardEntry{Type: history.EntryTypeText, Content: "second"})
	stop := startListening(t, s)
	defer stop()

	conn := dial(t, sockPath)
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	msg := readMsg(t, scanner)

	if msg.Type != msgInit {
		t.Errorf("expected init message, got %q", msg.Type)
	}
	if len(msg.Items) != 2 {
		t.Errorf("expected 2 items in init, got %d", len(msg.Items))
	}
	if msg.Items[0].Content != "second" {
		t.Errorf("expected most recent item first, got %q", msg.Items[0].Content)
	}
}

func TestServer_StreamsNewItemsToClient(t *testing.T) {
	s, sockPath := testServer(t)
	stop := startListening(t, s)
	defer stop()

	conn := dial(t, sockPath)
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	readMsg(t, scanner) // consume init

	// Simulate a new clipboard entry arriving while the client is connected.
	entry := history.ClipboardEntry{Type: history.EntryTypeText, Content: "live item"}
	s.hist.Add(entry)
	s.broadcast(entry)

	done := make(chan serverMsg, 1)
	go func() {
		done <- readMsg(t, scanner)
	}()

	select {
	case msg := <-done:
		if msg.Type != msgAdd {
			t.Errorf("expected add message, got %q", msg.Type)
		}
		if msg.Item == nil || msg.Item.Content != "live item" {
			t.Errorf("expected 'live item', got %v", msg.Item)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for streamed item")
	}
}

func TestServer_HandlesSelectMessage(t *testing.T) {
	s, sockPath := testServer(t)
	s.hist.Add(history.ClipboardEntry{Type: history.EntryTypeText, Content: "item one"})
	stop := startListening(t, s)
	defer stop()

	conn := dial(t, sockPath)
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	readMsg(t, scanner) // consume init

	sendMsg(t, conn, clientMsg{Type: msgSelect, EntryID: 1})
	sendMsg(t, conn, clientMsg{Type: msgCancel})

	// Server should close the connection cleanly — verify by waiting for EOF.
	done := make(chan struct{})
	go func() {
		for scanner.Scan() {
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for connection to close after cancel")
	}
}

func TestServer_HandlesClearMessage(t *testing.T) {
	s, sockPath := testServer(t)
	s.hist.Add(history.ClipboardEntry{Type: history.EntryTypeText, Content: "a"})
	s.hist.Add(history.ClipboardEntry{Type: history.EntryTypeText, Content: "b"})
	stop := startListening(t, s)
	defer stop()

	conn := dial(t, sockPath)
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	readMsg(t, scanner) // consume init

	sendMsg(t, conn, clientMsg{Type: msgClear})
	sendMsg(t, conn, clientMsg{Type: msgCancel})

	// Give the server time to process the clear.
	time.Sleep(50 * time.Millisecond)

	if items := s.hist.List(); len(items) != 0 {
		t.Errorf("expected empty history after clear, got %d items", len(items))
	}
}

func TestServer_MultipleSelectsBeforeCancel(t *testing.T) {
	s, sockPath := testServer(t)
	s.hist.Add(history.ClipboardEntry{Type: history.EntryTypeText, Content: "one"})
	s.hist.Add(history.ClipboardEntry{Type: history.EntryTypeText, Content: "two"})
	s.hist.Add(history.ClipboardEntry{Type: history.EntryTypeText, Content: "three"})
	stop := startListening(t, s)
	defer stop()

	conn := dial(t, sockPath)
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	readMsg(t, scanner) // consume init

	sendMsg(t, conn, clientMsg{Type: msgSelect, EntryID: 1})
	sendMsg(t, conn, clientMsg{Type: msgSelect, EntryID: 2})
	sendMsg(t, conn, clientMsg{Type: msgCancel})

	done := make(chan struct{})
	go func() {
		for scanner.Scan() {
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for connection to close")
	}
}

func TestServer_HandlesSearchMessage(t *testing.T) {
	s, sockPath := testServer(t)
	s.hist.Add(history.ClipboardEntry{Type: history.EntryTypeText, Content: "hello world"})
	s.hist.Add(history.ClipboardEntry{Type: history.EntryTypeText, Content: "foo bar"})
	s.hist.Add(history.ClipboardEntry{Type: history.EntryTypeText, Content: "hello go"})
	stop := startListening(t, s)
	defer stop()

	conn := dial(t, sockPath)
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	readMsg(t, scanner) // consume init

	sendMsg(t, conn, clientMsg{Type: msgSearch, Query: "hello"})

	done := make(chan serverMsg, 1)
	go func() { done <- readMsg(t, scanner) }()

	select {
	case msg := <-done:
		if msg.Type != msgSearchResult {
			t.Errorf("expected search_result, got %q", msg.Type)
		}
		if len(msg.Items) != 2 {
			t.Errorf("expected 2 results, got %d", len(msg.Items))
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for search result")
	}
}

func TestServer_BroadcastRefresh(t *testing.T) {
	s, sockPath := testServer(t)
	s.hist.Add(history.ClipboardEntry{Type: history.EntryTypeText, Content: "a"})
	stop := startListening(t, s)
	defer stop()

	conn := dial(t, sockPath)
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	readMsg(t, scanner) // consume init

	s.hist.Add(history.ClipboardEntry{Type: history.EntryTypeText, Content: "b"})
	s.broadcastRefresh()

	done := make(chan serverMsg, 1)
	go func() { done <- readMsg(t, scanner) }()

	select {
	case msg := <-done:
		if msg.Type != msgRefresh {
			t.Errorf("expected refresh, got %q", msg.Type)
		}
		if len(msg.Items) != 2 {
			t.Errorf("expected 2 items in refresh, got %d", len(msg.Items))
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for broadcast refresh")
	}
}

func TestServer_MaxEntries_MemoryHistory(t *testing.T) {
	s, _ := testServer(t) // testServer uses MemoryHistory
	if got := s.maxEntries(); got != 50 {
		t.Errorf("expected default 50 for MemoryHistory, got %d", got)
	}
}

func TestServer_ReloadConfig_MemoryHistory(t *testing.T) {
	s, _ := testServer(t)
	// reloadConfig with MemoryHistory is a no-op (type assertion to *SQLiteHistory fails).
	// Verify it doesn't panic.
	s.reloadConfig()
}

// --- filesystem helpers ---

func TestDataDir(t *testing.T) {
	got := dataDir()
	if got == "" {
		t.Error("expected non-empty dataDir")
	}
}

func TestWriteAndReadPIDFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// ensureDataPaths creates the directory that writePIDFile requires.
	if _, _, err := ensureDataPaths(); err != nil {
		t.Fatalf("ensureDataPaths failed: %v", err)
	}

	if err := writePIDFile(); err != nil {
		t.Fatalf("writePIDFile failed: %v", err)
	}

	pid := ReadPID()
	if pid != os.Getpid() {
		t.Errorf("expected PID %d, got %d", os.Getpid(), pid)
	}
}

func TestRemovePIDFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	_ = writePIDFile()
	removePIDFile()

	if ReadPID() != 0 {
		t.Error("expected ReadPID to return 0 after remove")
	}
}

func TestReadPID_MissingFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if pid := ReadPID(); pid != 0 {
		t.Errorf("expected 0 for missing PID file, got %d", pid)
	}
}

func TestEnsureDataPaths(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	dbPath, imageDir, err := ensureDataPaths()
	if err != nil {
		t.Fatalf("ensureDataPaths failed: %v", err)
	}
	if dbPath == "" {
		t.Error("expected non-empty dbPath")
	}
	if info, err := os.Stat(imageDir); err != nil || !info.IsDir() {
		t.Errorf("expected imageDir to exist as directory: %v", err)
	}
	// Idempotent — second call should not error.
	if _, _, err := ensureDataPaths(); err != nil {
		t.Errorf("second ensureDataPaths call failed: %v", err)
	}
}

// noopWatcher satisfies the watcher.Watcher interface without doing anything.
type noopWatcher struct{}

func (n *noopWatcher) Start(_ func(history.ClipboardEntry)) error { return nil }
func (n *noopWatcher) Stop() error                                { return nil }
func (n *noopWatcher) Reset()                                     {}

var _ watcher.Watcher = (*noopWatcher)(nil)
