package ui

// UI displays the clipboard history and streams selections back via the
// returned channel. The channel is closed when the window is dismissed.
type UI interface {
	Show(items []string, updates <-chan string) (<-chan string, error)
}
