package ui

// UI displays the clipboard history and streams selections back via the
// returned channel. The channel is closed when the window is dismissed.
// onClear is called when the user confirms clearing the history.
type UI interface {
	Show(items []string, updates <-chan string, onClear func()) (<-chan string, error)
}
