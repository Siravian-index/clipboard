# Clipboard Manager

A Linux clipboard history manager built in Go. Runs as a background daemon and shows a picker UI on demand.

## Requirements

- Linux with X11 or XWayland
- Go 1.21+ (to build from source)
- System graphics libraries (already present on any desktop Linux):
  `libgl1-mesa-dev xorg-dev`

## Install

Clone the repo and run the install script from the repository root:

```bash
git clone https://github.com/david-pena/clipboard.git
cd clipboard
./install.sh
```

This will:
1. Compile the binary
2. Install it to `~/.local/bin/clipboard-manager`
3. Install and start the systemd user service (autostart on login)

Make sure `~/.local/bin` is in your `PATH`. If not, add this to your `~/.bashrc` or `~/.zshrc`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

## Usage

```bash
# Open the history picker
clipboard-manager show

# The daemon starts automatically via systemd; to manage it manually:
systemctl --user start clipboard-manager
systemctl --user stop clipboard-manager
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

## Systemd service

The install script sets up autostart automatically. Useful commands:

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
