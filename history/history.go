package history

type History interface {
	Add(entry ClipboardEntry)
	List() []ClipboardEntry
	Clear()
}
