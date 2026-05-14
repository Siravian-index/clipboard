# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
# Build
go build -o clipboard-manager .

# Run daemon
./clipboard-manager daemon

# Open picker UI
./clipboard-manager show

# Run all tests
go test ./...

# Run tests in a specific package
go test ./history/...
go test -run TestSQLite ./history/...

# Run with race detector
go test -race ./...
```

## Architecture

The app runs as two processes that communicate over a Unix socket at `$HOME/.clipboard-manager.sock`:

**Daemon** (`daemon/`, `watcher/`, `history/`):
- `daemon.Server` listens on the Unix socket and manages clipboard history
- `watcher.PollingWatcher` polls the clipboard every 500ms via `golang.design/x/clipboard`
- History is stored in SQLite at `~/.local/share/clipboard-manager/history.db`
- PID written to `~/.local/share/clipboard-manager/daemon.pid`
- SIGHUP reloads config (max_entries) without restarting
- Broadcasts new clipboard items to all connected clients over the socket

**Client/UI** (`client/`, `ui/`):
- `client.Run()` connects to the daemon socket, receives history + live updates, shows picker
- UI is built with Fyne (`ui/fyne.go`); SIGHUP sent to daemon after selection to sync
- Selection is sent back to the daemon via the socket; daemon logs it

**Protocol** (JSON newline-delimited over Unix socket):
- Server → Client: `{"type":"init","items":[...]}` then `{"type":"add","item":"..."}` for new entries
- Client → Server: `{"type":"select","item":"..."}` or `{"type":"clear"}` or `{"type":"cancel"}`

**Config** (`config/`): JSON at `~/.config/clipboard-manager/config.json`
- `max_entries` (default 50), `keep_window_open` (default true)

**History backends** (`history/`):
- `History` interface: `Add`, `List`, `Clear`
- `SQLiteHistory` — production backend (uses `modernc.org/sqlite`, pure Go, no CGO)
- `MemoryHistory` — used in tests

## After making changes

Always run the following after any code change so the user can test immediately:

```bash
go build -o clipboard-manager . && systemctl --user restart clipboard-manager
```

## Platform

Linux only (X11/XWayland). Requires display server for clipboard access.
