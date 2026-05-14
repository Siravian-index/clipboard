package ui

import (
	"github.com/david-pena/clipboard/history"
)

// UI displays the clipboard history and streams selections back via the
// returned channel. The channel is closed when the window is dismissed.
// onClear is called when the user confirms clearing the history.
// focusReqs receives a signal each time another instance requests focus.
type UI interface {
	Show(items []history.ClipboardEntry, updates <-chan history.ClipboardEntry, onClear func(), focusReqs <-chan struct{}) (<-chan history.ClipboardEntry, error)
}
