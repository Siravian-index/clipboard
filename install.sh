#!/usr/bin/env bash
set -euo pipefail

BINARY_NAME="clipboard-manager"
INSTALL_DIR="$HOME/.local/bin"
SERVICE_DIR="$HOME/.config/systemd/user"
REPO="Siravian-index/clipboard"

# --- resolve service file location (works both from repo and standalone) ---
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_SRC="$SCRIPT_DIR/systemd/clipboard-manager.service"

install_service() {
  if [[ ! -f "$SERVICE_SRC" ]]; then
    echo "Error: service file not found at $SERVICE_SRC" >&2
    exit 1
  fi
  mkdir -p "$SERVICE_DIR"
  cp "$SERVICE_SRC" "$SERVICE_DIR/$BINARY_NAME.service"
  systemctl --user daemon-reload
  systemctl --user enable "$BINARY_NAME"
  systemctl --user start "$BINARY_NAME"
  echo "Service enabled and started."
}

install_path_reminder() {
  if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "Note: $INSTALL_DIR is not in your PATH."
    echo "Add this line to your ~/.bashrc or ~/.zshrc:"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
  fi
}

# --- option A: build from source (Go available) ---
if command -v go &>/dev/null && [[ -f "$SCRIPT_DIR/go.mod" ]]; then
  echo "Go found — building from source..."
  go build -o "$BINARY_NAME" "$SCRIPT_DIR"
  mkdir -p "$INSTALL_DIR"
  cp "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
  echo "Binary installed to $INSTALL_DIR/$BINARY_NAME"
  install_service
  install_path_reminder
  echo ""
  echo "Done. Run 'clipboard-manager show' to open the picker."
  exit 0
fi

# --- option B: download pre-built binary from GitHub Releases ---
if ! command -v curl &>/dev/null; then
  echo "Error: neither Go nor curl is available. Install one of them and retry." >&2
  exit 1
fi

echo "Go not found — downloading latest release from GitHub..."

LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | head -1 | cut -d'"' -f4)

if [[ -z "$LATEST" ]]; then
  echo "Error: could not fetch latest release from GitHub." >&2
  exit 1
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST/$BINARY_NAME"
echo "Downloading $BINARY_NAME $LATEST..."
mkdir -p "$INSTALL_DIR"
curl -fsSL "$DOWNLOAD_URL" -o "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"
echo "Binary installed to $INSTALL_DIR/$BINARY_NAME"

install_service
install_path_reminder
echo ""
echo "Done. Run 'clipboard-manager show' to open the picker."
