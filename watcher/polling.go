package watcher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.design/x/clipboard"

	"github.com/david-pena/clipboard/history"
)

type PollingWatcher struct {
	interval      time.Duration
	imageDir      string
	maxImageBytes int64
	cancel        context.CancelFunc
	mu            sync.Mutex
	lastText      string
	lastImageHash string
}

func NewPollingWatcher(interval time.Duration, imageDir string, maxImageMB int) *PollingWatcher {
	return &PollingWatcher{
		interval:      interval,
		imageDir:      imageDir,
		maxImageBytes: int64(maxImageMB) * 1024 * 1024,
	}
}

func (w *PollingWatcher) Start(onChange func(entry history.ClipboardEntry)) error {
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
				w.poll(onChange)
			}
		}
	}()

	return nil
}

func (w *PollingWatcher) poll(onChange func(entry history.ClipboardEntry)) {
	// Check text first.
	text := string(clipboard.Read(clipboard.FmtText))
	w.mu.Lock()
	lastText := w.lastText
	w.mu.Unlock()

	if text != "" && text != lastText {
		w.mu.Lock()
		w.lastText = text
		w.lastImageHash = ""
		w.mu.Unlock()
		onChange(history.ClipboardEntry{Type: history.EntryTypeText, Content: text})
		return
	}

	// Check image when no new text detected.
	imgBytes := clipboard.Read(clipboard.FmtImage)
	if len(imgBytes) == 0 {
		return
	}

	// Validate it's a PNG before hashing.
	if _, err := png.DecodeConfig(bytes.NewReader(imgBytes)); err != nil {
		return
	}

	hash := sha256Hash(imgBytes)
	w.mu.Lock()
	lastHash := w.lastImageHash
	w.mu.Unlock()

	if hash == lastHash {
		return
	}

	if w.maxImageBytes > 0 && int64(len(imgBytes)) > w.maxImageBytes {
		return
	}

	path, err := w.saveImage(imgBytes, hash)
	if err != nil {
		return
	}

	w.mu.Lock()
	w.lastImageHash = hash
	w.lastText = ""
	w.mu.Unlock()

	onChange(history.ClipboardEntry{Type: history.EntryTypeImage, Content: path})
}

// saveImage writes PNG bytes to imageDir/<hash>.png and returns the path.
// If the file already exists (dedup by hash), it returns the existing path.
func (w *PollingWatcher) saveImage(imgBytes []byte, hash string) (string, error) {
	if err := os.MkdirAll(w.imageDir, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(w.imageDir, hash+".png")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	return path, os.WriteFile(path, imgBytes, 0600)
}

func (w *PollingWatcher) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

// Reset clears the last-seen values so the watcher picks up the next copy correctly.
func (w *PollingWatcher) Reset() {
	w.mu.Lock()
	w.lastText = ""
	w.lastImageHash = ""
	w.mu.Unlock()
}

func sha256Hash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
