package client

import (
	"encoding/json"
	"log"
	"net"
	"os"

	"github.com/david-pena/clipboard/ui"
)

const socketPath = "/.clipboard-manager.sock"

// Run connects to the daemon socket, retrieves history, shows the Fyne UI,
// and sends the user's selection (or an empty string) back to the daemon.
func Run() {
	sockPath := os.Getenv("HOME") + socketPath

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		log.Fatalf("failed to connect to daemon at %s: %v", sockPath, err)
	}
	defer conn.Close()

	// Read the JSON array sent by the daemon.
	buf := make([]byte, 1<<20)
	n, err := conn.Read(buf)
	if err != nil {
		log.Fatalf("failed to read history from daemon: %v", err)
	}

	var items []string
	if err := json.Unmarshal(buf[:n], &items); err != nil {
		log.Fatalf("failed to unmarshal history: %v", err)
	}

	// Show the Fyne UI and get the user's selection.
	fyneUI := ui.NewFyneUI()
	selected, _ := fyneUI.Show(items)
	// Show returns an error when nothing is selected — we still need to
	// respond to the daemon with an empty string in that case.

	// Send selection back as a JSON string.
	response, err := json.Marshal(selected)
	if err != nil {
		log.Fatalf("failed to marshal selection: %v", err)
	}
	response = append(response, '\n')
	if _, err := conn.Write(response); err != nil {
		log.Fatalf("failed to send selection to daemon: %v", err)
	}
}
