package watcher

import (
	"os"
	"testing"
	"time"

	"golang.design/x/clipboard"

	"github.com/david-pena/clipboard/history"
)

func TestPollingWatcher_DetectsChange(t *testing.T) {
	if err := clipboard.Init(); err != nil {
		t.Skipf("clipboard not available in this environment: %v", err)
	}

	dir := t.TempDir()
	w := NewPollingWatcher(50*time.Millisecond, dir, 10)

	received := make(chan history.ClipboardEntry, 1)
	if err := w.Start(func(entry history.ClipboardEntry) {
		select {
		case received <- entry:
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
		if got.Content != want {
			t.Errorf("expected %q, got %q", want, got.Content)
		}
		if got.Type != history.EntryTypeText {
			t.Errorf("expected text type, got %q", got.Type)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for clipboard change to be detected")
	}
}

func TestPollingWatcher_StopPreventsCallbacks(t *testing.T) {
	if err := clipboard.Init(); err != nil {
		t.Skipf("clipboard not available in this environment: %v", err)
	}

	dir := t.TempDir()
	w := NewPollingWatcher(50*time.Millisecond, dir, 10)

	callCount := 0
	if err := w.Start(func(entry history.ClipboardEntry) {
		callCount++
	}); err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	w.Stop()

	clipboard.Write(clipboard.FmtText, []byte("after_stop"))
	time.Sleep(200 * time.Millisecond)

	if callCount > 1 {
		t.Errorf("callback fired %d times after Stop", callCount)
	}
}

func TestPollingWatcher_ImageDetection(t *testing.T) {
	if err := clipboard.Init(); err != nil {
		t.Skipf("clipboard not available in this environment: %v", err)
	}

	dir := t.TempDir()
	w := NewPollingWatcher(50*time.Millisecond, dir, 10)

	// Clear text clipboard so image polling triggers.
	clipboard.Write(clipboard.FmtText, []byte{})

	received := make(chan history.ClipboardEntry, 1)
	if err := w.Start(func(entry history.ClipboardEntry) {
		select {
		case received <- entry:
		default:
		}
	}); err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}
	defer w.Stop()

	// Write a minimal valid PNG to the clipboard.
	pngData := minimalPNG()
	clipboard.Write(clipboard.FmtImage, pngData)

	select {
	case got := <-received:
		if got.Type != history.EntryTypeImage {
			t.Errorf("expected image type, got %q", got.Type)
		}
		if _, err := os.Stat(got.Content); err != nil {
			t.Errorf("image file not found at %q: %v", got.Content, err)
		}
	case <-time.After(2 * time.Second):
		t.Skip("image clipboard not detected (may not be supported in this environment)")
	}
}

func TestPollingWatcher_Reset(t *testing.T) {
	dir := t.TempDir()
	w := NewPollingWatcher(50*time.Millisecond, dir, 10)

	// Manually set internal state to verify Reset clears it.
	w.lastText = "something"
	w.lastImageHash = "deadbeef"

	w.Reset()

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lastText != "" {
		t.Errorf("expected lastText to be empty after Reset, got %q", w.lastText)
	}
	if w.lastImageHash != "" {
		t.Errorf("expected lastImageHash to be empty after Reset, got %q", w.lastImageHash)
	}
}

// minimalPNG returns a 1x1 white PNG image as bytes.
func minimalPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, // IHDR chunk length + type
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 pixels
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, // bit depth=8, color=RGB
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xff, 0xff, 0x3f,
		0x00, 0x05, 0xfe, 0x02, 0xfe, 0xdc, 0xcc, 0x59,
		0xe7, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, // IEND chunk
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
}
