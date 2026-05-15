package client

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/david-pena/clipboard/history"
	"github.com/david-pena/clipboard/ui"
)

const socketPath = "/.clipboard-manager.sock"
const showSocketPath = "/.clipboard-manager-show.sock"

type msgType string

const (
	msgInit   msgType = "init"
	msgAdd    msgType = "add"
	msgSelect msgType = "select"
	msgCancel msgType = "cancel"
	msgClear  msgType = "clear"
	msgFocus  msgType = "focus"
)

type serverMsg struct {
	Type  msgType                  `json:"type"`
	Items []history.ClipboardEntry `json:"items,omitempty"`
	Item  *history.ClipboardEntry  `json:"item,omitempty"`
}

type clientMsg struct {
	Type    msgType `json:"type"`
	EntryID int64   `json:"entry_id,omitempty"`
}

// Run connects to the daemon, shows the Fyne picker with live updates,
// and sends the user's selection back.
func Run() {
	go func() { log.Println(http.ListenAndServe("localhost:6060", nil)) }()
	showSock := os.Getenv("HOME") + showSocketPath

	// If another instance is already showing, forward a focus request and exit.
	if focusExistingInstance(showSock) {
		return
	}

	// Claim the show socket before opening the UI so concurrent invocations
	// can detect this instance immediately.
	ln, focusReqs := listenForFocus(showSock)
	defer func() {
		ln.Close()
		os.Remove(showSock)
	}()

	conn, err := net.Dial("unix", os.Getenv("HOME")+socketPath)
	if err != nil {
		log.Fatalf("failed to connect to daemon: %v", err)
	}
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	// First message is always the initial history list.
	if !scanner.Scan() {
		log.Fatalf("failed to read init message from daemon")
	}
	var initMsg serverMsg
	if err := json.Unmarshal(scanner.Bytes(), &initMsg); err != nil || initMsg.Type != msgInit {
		log.Fatalf("unexpected init message: %s", scanner.Bytes())
	}

	// Stream subsequent add messages into the updates channel.
	updates := make(chan history.ClipboardEntry, 32)
	go func() {
		defer close(updates)
		for scanner.Scan() {
			var msg serverMsg
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				continue
			}
			if msg.Type == msgAdd && msg.Item != nil {
				updates <- *msg.Item
			}
		}
	}()

	onClear := func() {
		msg, _ := json.Marshal(clientMsg{Type: msgClear})
		conn.Write(append(msg, '\n'))
	}

	selections, err := ui.NewFyneUI().Show(initMsg.Items, updates, onClear, focusReqs)
	if err != nil {
		msg, _ := json.Marshal(clientMsg{Type: msgCancel})
		conn.Write(append(msg, '\n'))
		return
	}

	for entry := range selections {
		msg, _ := json.Marshal(clientMsg{Type: msgSelect, EntryID: entry.ID})
		conn.Write(append(msg, '\n'))
	}

	// Window was closed — notify daemon.
	msg, _ := json.Marshal(clientMsg{Type: msgCancel})
	conn.Write(append(msg, '\n'))
}

// focusExistingInstance tries to connect to an already-running picker and send
// a focus request. Returns true if a running instance was found.
func focusExistingInstance(sockPath string) bool {
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return false
	}
	defer conn.Close()
	// focusMsg uses a separate local type to avoid import cycle with daemon.
	type focusMsg struct {
		Type string `json:"type"`
	}
	msg, _ := json.Marshal(focusMsg{Type: "focus"})
	conn.Write(append(msg, '\n'))
	return true
}

// listenForFocus opens the show socket and starts a goroutine that accepts
// connections and signals focusReqs for each valid focus message received.
func listenForFocus(sockPath string) (net.Listener, <-chan struct{}) {
	os.Remove(sockPath) // clean up any leftover socket from a previous crash
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		log.Fatalf("failed to listen on show socket %s: %v", sockPath, err)
	}

	focusReqs := make(chan struct{}, 8)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener was closed
			}
			go handleFocusConn(conn, focusReqs)
		}
	}()

	return ln, focusReqs
}

func handleFocusConn(conn net.Conn, focusReqs chan<- struct{}) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var msg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if msg.Type == "focus" {
			select {
			case focusReqs <- struct{}{}:
			default:
			}
		}
	}
}
