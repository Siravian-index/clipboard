package watcher

import (
	"testing"
	"time"

	"golang.design/x/clipboard"
)

func TestPollingWatcher_DetectsChange(t *testing.T) {
	if err := clipboard.Init(); err != nil {
		t.Skipf("clipboard not available in this environment: %v", err)
	}

	w := NewPollingWatcher(50 * time.Millisecond)

	received := make(chan string, 1)
	if err := w.Start(func(content string) {
		select {
		case received <- content:
		default:
		}
	}); err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}
	defer w.Stop()

	want := "watcher_test_value"
	clipboard.Write(clipboard.FmtText, []byte(want))

	select {
	case got := <-received:
		if got != want {
			t.Errorf("expected %q, got %q", want, got)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for clipboard change to be detected")
	}
}

func TestPollingWatcher_StopPreventsCallbacks(t *testing.T) {
	if err := clipboard.Init(); err != nil {
		t.Skipf("clipboard not available in this environment: %v", err)
	}

	w := NewPollingWatcher(50 * time.Millisecond)

	callCount := 0
	if err := w.Start(func(content string) {
		callCount++
	}); err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	w.Stop()

	// Write after stop — callback should not fire.
	clipboard.Write(clipboard.FmtText, []byte("after_stop"))
	time.Sleep(200 * time.Millisecond)

	if callCount > 1 {
		t.Errorf("callback fired %d times after Stop", callCount)
	}
}
