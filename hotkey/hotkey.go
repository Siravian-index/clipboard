package hotkey

type HotkeyListener interface {
	Register(keys string, callback func()) error
	Listen() error
	Stop() error
}
