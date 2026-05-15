package client

import (
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"
)

// --- focusExistingInstance ---

func TestFocusExistingInstance_NoServer(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "nonexistent.sock")
	if focusExistingInstance(sockPath) {
		t.Error("expected false when no server is listening")
	}
}

func TestFocusExistingInstance_ServerPresent(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "show.sock")

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	received := make(chan []byte, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 256)
		n, _ := conn.Read(buf)
		received <- buf[:n]
	}()

	if !focusExistingInstance(sockPath) {
		t.Error("expected true when server is listening")
	}

	select {
	case data := <-received:
		var msg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(data[:len(data)-1], &msg); err != nil {
			// Try without stripping newline.
			json.Unmarshal(data, &msg)
		}
		if msg.Type != "focus" {
			t.Errorf("expected focus message, got %q", string(data))
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for focus message")
	}
}

// --- handleFocusConn ---

func TestHandleFocusConn_ValidFocusMessage(t *testing.T) {
	server, client := net.Pipe()
	focusReqs := make(chan struct{}, 1)

	go handleFocusConn(server, focusReqs)

	msg, _ := json.Marshal(struct {
		Type string `json:"type"`
	}{"focus"})
	client.Write(append(msg, '\n'))

	select {
	case <-focusReqs:
		// ok
	case <-time.After(time.Second):
		t.Error("timed out waiting for focus signal")
	}

	client.Close()
}

func TestHandleFocusConn_IgnoresNonFocusMessage(t *testing.T) {
	server, client := net.Pipe()
	focusReqs := make(chan struct{}, 1)

	go handleFocusConn(server, focusReqs)

	msg, _ := json.Marshal(struct {
		Type string `json:"type"`
	}{"other"})
	client.Write(append(msg, '\n'))
	client.Close()

	select {
	case <-focusReqs:
		t.Error("expected no focus signal for non-focus message")
	case <-time.After(100 * time.Millisecond):
		// ok — nothing received
	}
}

func TestHandleFocusConn_IgnoresInvalidJSON(t *testing.T) {
	server, client := net.Pipe()
	focusReqs := make(chan struct{}, 1)

	go handleFocusConn(server, focusReqs)

	client.Write([]byte("not json\n"))
	client.Close()

	select {
	case <-focusReqs:
		t.Error("expected no focus signal for invalid JSON")
	case <-time.After(100 * time.Millisecond):
		// ok
	}
}

// --- listenForFocus ---

func TestListenForFocus_ReceivesFocusRequest(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "focus.sock")

	ln, focusReqs := listenForFocus(sockPath)
	defer ln.Close()

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	msg, _ := json.Marshal(struct {
		Type string `json:"type"`
	}{"focus"})
	conn.Write(append(msg, '\n'))

	select {
	case <-focusReqs:
		// ok
	case <-time.After(time.Second):
		t.Error("timed out waiting for focus signal from listenForFocus")
	}
}

func TestListenForFocus_MultipleFocusRequests(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "focus.sock")

	ln, focusReqs := listenForFocus(sockPath)
	defer ln.Close()

	sendFocus := func() {
		conn, err := net.Dial("unix", sockPath)
		if err != nil {
			t.Errorf("failed to connect: %v", err)
			return
		}
		defer conn.Close()
		msg, _ := json.Marshal(struct {
			Type string `json:"type"`
		}{"focus"})
		conn.Write(append(msg, '\n'))
	}

	sendFocus()
	sendFocus()

	for i := 0; i < 2; i++ {
		select {
		case <-focusReqs:
		case <-time.After(time.Second):
			t.Errorf("timed out waiting for focus signal %d", i+1)
		}
	}
}
