package client

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"os"

	"github.com/david-pena/clipboard/ui"
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

// Run connects to the daemon, shows the Fyne picker with live updates,
// and sends the user's selection back.
func Run() {
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
	var init serverMsg
	if err := json.Unmarshal(scanner.Bytes(), &init); err != nil || init.Type != msgInit {
		log.Fatalf("unexpected init message: %s", scanner.Bytes())
	}

	// Stream subsequent add messages into the updates channel.
	updates := make(chan string, 32)
	go func() {
		defer close(updates)
		for scanner.Scan() {
			var msg serverMsg
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				continue
			}
			if msg.Type == msgAdd {
				updates <- msg.Item
			}
		}
	}()

	selections, err := ui.NewFyneUI().Show(init.Items, updates)
	if err != nil {
		msg, _ := json.Marshal(clientMsg{Type: msgCancel})
		conn.Write(append(msg, '\n'))
		return
	}

	for item := range selections {
		msg, _ := json.Marshal(clientMsg{Type: msgSelect, Item: item})
		conn.Write(append(msg, '\n'))
	}

	// Window was closed — notify daemon.
	msg, _ := json.Marshal(clientMsg{Type: msgCancel})
	conn.Write(append(msg, '\n'))
}
