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
		clients:  make(map[chan string]struct{}),
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
	s.hist.Add("first")
	s.hist.Add("second")
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
	if msg.Items[0] != "second" {
		t.Errorf("expected most recent item first, got %q", msg.Items[0])
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
	s.hist.Add("live item")
	s.broadcast("live item")

	done := make(chan serverMsg, 1)
	go func() {
		done <- readMsg(t, scanner)
	}()

	select {
	case msg := <-done:
		if msg.Type != msgAdd {
			t.Errorf("expected add message, got %q", msg.Type)
		}
		if msg.Item != "live item" {
			t.Errorf("expected 'live item', got %q", msg.Item)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for streamed item")
	}
}

func TestServer_HandlesSelectMessage(t *testing.T) {
	s, sockPath := testServer(t)
	s.hist.Add("item one")
	stop := startListening(t, s)
	defer stop()

	conn := dial(t, sockPath)
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	readMsg(t, scanner) // consume init

	sendMsg(t, conn, clientMsg{Type: msgSelect, Item: "item one"})
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
	s.hist.Add("a")
	s.hist.Add("b")
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
	s.hist.Add("one")
	s.hist.Add("two")
	s.hist.Add("three")
	stop := startListening(t, s)
	defer stop()

	conn := dial(t, sockPath)
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	readMsg(t, scanner) // consume init

	sendMsg(t, conn, clientMsg{Type: msgSelect, Item: "one"})
	sendMsg(t, conn, clientMsg{Type: msgSelect, Item: "two"})
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

// noopWatcher satisfies the watcher.Watcher interface without doing anything.
type noopWatcher struct{}

func (n *noopWatcher) Start(_ func(string)) error { return nil }
func (n *noopWatcher) Stop() error                { return nil }

var _ watcher.Watcher = (*noopWatcher)(nil)
