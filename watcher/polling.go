package watcher

import (
	"context"
	"time"

	"golang.design/x/clipboard"
)

type PollingWatcher struct {
	interval time.Duration
	cancel   context.CancelFunc
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
		var last string
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.interval):
				content := string(clipboard.Read(clipboard.FmtText))
				if content != "" && content != last {
					last = content
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
