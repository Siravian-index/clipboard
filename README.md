# Clipboard Manager

A Linux clipboard history manager built in Go. Runs as a background daemon and shows a picker UI on demand.

## Requirements

- Linux with X11 or XWayland
- Go 1.21+ (to build)
- System graphics libraries (already present on any desktop Linux):
  `libgl1-mesa-dev xorg-dev`

## Build

```bash
go build -o clipboard-manager .
```

## Usage

```bash
# Start the background daemon
./clipboard-manager daemon

# Open the history picker
./clipboard-manager show
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./history/...
go test ./ui/...
go test ./client/...

# Run a specific test by name
go test -run TestSQLite ./history/...

# Run with race detector (recommended before committing)
go test -race ./...

# Run with coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Open coverage in browser
go tool cover -html=coverage.out
```

## GNOME Shortcut Setup

1. Go to **Settings → Keyboard → View and Customize Shortcuts → Custom Shortcuts**
2. Click **Add Shortcut**
3. Fill in:
   - **Name:** `Clipboard History`
   - **Command:** `/full/path/to/clipboard-manager show`
   - **Shortcut:** your preferred key combo (e.g. `Ctrl+Alt+V`)

## Autostart with systemd (runs daemon on login)

### Install

```bash
cp systemd/clipboard-manager.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable clipboard-manager
systemctl --user start clipboard-manager
```

### Useful commands

```bash
# Check status
systemctl --user status clipboard-manager

# View live logs
journalctl --user -u clipboard-manager -f

# Restart after rebuilding the binary
systemctl --user restart clipboard-manager

# Stop the daemon
systemctl --user stop clipboard-manager

# Disable autostart
systemctl --user disable clipboard-manager
```
