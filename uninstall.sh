#!/usr/bin/env bash
set -euo pipefail

BINARY_NAME="clipboard-manager"
INSTALL_DIR="$HOME/.local/bin"
SERVICE_DIR="$HOME/.config/systemd/user"

# --- stop and disable service ---
if systemctl --user is-active --quiet "$BINARY_NAME" 2>/dev/null; then
  systemctl --user stop "$BINARY_NAME"
  echo "Service stopped."
fi

if systemctl --user is-enabled --quiet "$BINARY_NAME" 2>/dev/null; then
  systemctl --user disable "$BINARY_NAME"
  echo "Service disabled."
fi

# --- remove service file ---
SERVICE_FILE="$SERVICE_DIR/$BINARY_NAME.service"
if [[ -f "$SERVICE_FILE" ]]; then
  rm "$SERVICE_FILE"
  systemctl --user daemon-reload
  echo "Service file removed."
fi

# --- remove binary ---
BINARY_FILE="$INSTALL_DIR/$BINARY_NAME"
if [[ -f "$BINARY_FILE" ]]; then
  rm "$BINARY_FILE"
  echo "Binary removed from $BINARY_FILE"
fi

echo ""
echo "Uninstall complete."
echo "Note: clipboard history data at ~/.local/share/clipboard-manager/ was NOT removed."
echo "Delete it manually if you want a fully clean slate:"
echo "  rm -rf ~/.local/share/clipboard-manager/"
