package history

import (
	"strings"
	"sync"
)

type MemoryHistory struct {
	mu      sync.RWMutex
	entries []string
	maxSize int
}

func NewMemoryHistory(maxSize int) *MemoryHistory {
	return &MemoryHistory{maxSize: maxSize}
}

func (h *MemoryHistory) Add(entry string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry = strings.TrimSpace(entry)
	if entry == "" {
		return
	}
	if len(h.entries) > 0 && h.entries[0] == entry {
		return
	}

	h.entries = append([]string{entry}, h.entries...)
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[:h.maxSize]
	}
}

func (h *MemoryHistory) List() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]string, len(h.entries))
	copy(result, h.entries)
	return result
}

func (h *MemoryHistory) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = nil
}
