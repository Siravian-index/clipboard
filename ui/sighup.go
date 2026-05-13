package ui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// sendSIGHUP finds the running daemon PID from the pidfile and sends SIGHUP.
func sendSIGHUP() {
	pidPath := filepath.Join(os.Getenv("HOME"), ".local", "share", "clipboard-manager", "daemon.pid")
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = proc.Signal(syscall.SIGHUP)
}
