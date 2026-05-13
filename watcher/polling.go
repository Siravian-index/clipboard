package watcher

import (
	"context"
	"sync"
	"time"

	"golang.design/x/clipboard"
)

type PollingWatcher struct {
	interval time.Duration
	cancel   context.CancelFunc
	mu       sync.Mutex
	last     string
}

func NewPollingWatcher(interval time.Duration) *PollingWatcher {
	return &PollingWatcher{interval: interval}
}

func (w *PollingWatcher) Start(onChange func(content string)) error {
	if err := clipboard.Init(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.interval):
				content := string(clipboard.Read(clipboard.FmtText))
				w.mu.Lock()
				seen := w.last
				w.mu.Unlock()
				if content != "" && content != seen {
					w.mu.Lock()
					w.last = content
					w.mu.Unlock()
					onChange(content)
				}
			}
		}
	}()

	return nil
}

func (w *PollingWatcher) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

// Reset clears the last-seen value. Call this after clearing the system
// clipboard so the watcher picks up the next copy correctly.
func (w *PollingWatcher) Reset() {
	w.mu.Lock()
	w.last = ""
	w.mu.Unlock()
}
