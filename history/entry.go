package history

type EntryType string

const (
	EntryTypeText  EntryType = "text"
	EntryTypeImage EntryType = "image"
)

type ClipboardEntry struct {
	ID      int64
	Type    EntryType
	Content string // text for text entries; absolute file path for image entries
}
