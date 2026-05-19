#!/usr/bin/env bash
set -euo pipefail

BINARY_NAME="clipboard-manager"
INSTALL_DIR="$HOME/.local/bin"
SERVICE_SRC="systemd/clipboard-manager.service"
SERVICE_DIR="$HOME/.config/systemd/user"

# --- checks ---
if ! command -v go &>/dev/null; then
  echo "Error: Go is not installed. Install it from https://go.dev/dl/" >&2
  exit 1
fi

if [[ ! -f go.mod ]]; then
  echo "Error: run this script from the repository root." >&2
  exit 1
fi

# --- build ---
echo "Building $BINARY_NAME..."
go build -o "$BINARY_NAME" .

# --- install binary ---
mkdir -p "$INSTALL_DIR"
cp "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
echo "Binary installed to $INSTALL_DIR/$BINARY_NAME"

# --- install systemd service ---
mkdir -p "$SERVICE_DIR"
cp "$SERVICE_SRC" "$SERVICE_DIR/$BINARY_NAME.service"
systemctl --user daemon-reload
systemctl --user enable "$BINARY_NAME"
systemctl --user start "$BINARY_NAME"
echo "Service enabled and started."

# --- PATH reminder ---
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo ""
  echo "Note: $INSTALL_DIR is not in your PATH."
  echo "Add this line to your ~/.bashrc or ~/.zshrc:"
  echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

echo ""
echo "Done. Run 'clipboard-manager show' to open the picker."
