package history

type SearchResult struct {
	Entries      []ClipboardEntry
	TotalMatches int
}

type History interface {
	Add(entry ClipboardEntry)
	List() []ClipboardEntry
	Search(query string, limit int) SearchResult
	Clear()
}
