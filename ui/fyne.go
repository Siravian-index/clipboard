package ui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"golang.design/x/clipboard"

	"github.com/david-pena/clipboard/config"
	"github.com/david-pena/clipboard/history"
)

type FyneUI struct{}

func NewFyneUI() *FyneUI {
	return &FyneUI{}
}

// Show displays the clipboard history picker. It returns a channel that emits
// each entry the user selects; the channel is closed when the window is dismissed.
func (f *FyneUI) Show(items []history.ClipboardEntry, updates <-chan history.ClipboardEntry, onClear func(), focusReqs <-chan struct{}) (<-chan history.ClipboardEntry, error) {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}

	if len(items) == 0 && updates == nil {
		return nil, errors.New("no history entries")
	}

	selections := make(chan history.ClipboardEntry, 16)

	a := app.New()
	w := a.NewWindow("Clipboard History")
	w.Resize(fyne.NewSize(500, 460))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	var mu sync.Mutex
	current := make([]history.ClipboardEntry, len(items))
	copy(current, items)

	statusLabel := widget.NewLabel("")

	var list *widget.List
	list = widget.NewList(
		func() int {
			mu.Lock()
			defer mu.Unlock()
			return len(current)
		},
		func() fyne.CanvasObject {
			img := &canvas.Image{}
			img.FillMode = canvas.ImageFillContain
			img.SetMinSize(fyne.NewSize(60, 60))
			lbl := widget.NewLabel("")
			lbl.Truncation = fyne.TextTruncateEllipsis
			return container.NewBorder(nil, nil, img, nil, lbl)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			mu.Lock()
			if id >= len(current) {
				mu.Unlock()
				return
			}
			entry := current[id]
			mu.Unlock()

			box := obj.(*fyne.Container)
			img := box.Objects[1].(*canvas.Image)
			lbl := box.Objects[0].(*widget.Label)

			if entry.Type == history.EntryTypeImage {
				img.File = entry.Content
				img.Resource = nil
				img.Show()
				lbl.SetText(imageLabel(entry.Content))
			} else {
				img.File = ""
				img.Resource = theme.FileTextIcon()
				img.Hide()
				lbl.SetText(truncateText(entry.Content))
			}
			img.Refresh()
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		mu.Lock()
		if id >= len(current) {
			mu.Unlock()
			return
		}
		entry := current[id]
		mu.Unlock()

		writeToClipboard(w, entry)
		selections <- entry

		preview := previewText(entry)
		statusLabel.SetText("Copied: " + preview)
		if !cfg.KeepWindowOpen {
			w.Close()
		}
		list.Unselect(id)
	}

	go func() {
		for {
			select {
			case entry, ok := <-updates:
				if !ok {
					return
				}
				mu.Lock()
				filtered := current[:0]
				for _, e := range current {
					if !(e.Type == entry.Type && e.Content == entry.Content) {
						filtered = append(filtered, e)
					}
				}
				current = append([]history.ClipboardEntry{entry}, filtered...)
				mu.Unlock()
				list.Refresh()
			case _, ok := <-focusReqs:
				if !ok {
					return
				}
				w.RequestFocus()
			}
		}
	}()

	emptyLabel := widget.NewLabel("No history yet — start copying something!")
	emptyLabel.Alignment = fyne.TextAlignCenter

	refreshEmpty := func() {
		mu.Lock()
		empty := len(current) == 0
		mu.Unlock()
		if empty {
			emptyLabel.Show()
		} else {
			emptyLabel.Hide()
		}
	}

	settingsBtn := widget.NewButton("⚙", func() {
		onClearUI := func() {
			statusLabel.SetText("")
			mu.Lock()
			current = nil
			mu.Unlock()
			list.Refresh()
			refreshEmpty()
		}
		showSettings(a, w, cfg, onClear, onClearUI)
	})

	w.SetOnClosed(func() {
		close(selections)
	})

	w.SetContent(container.NewBorder(
		widget.NewLabel("Select an entry to copy:"),
		container.NewBorder(nil, nil, nil, settingsBtn, statusLabel),
		nil, nil,
		container.NewStack(list, container.NewCenter(emptyLabel)),
	))

	w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyEscape {
			w.Close()
		}
	})

	w.Show()
	refreshEmpty()
	a.Run()

	return selections, nil
}

// writeToClipboard writes the entry content to the system clipboard.
func writeToClipboard(w fyne.Window, entry history.ClipboardEntry) {
	if entry.Type == history.EntryTypeImage {
		data, err := os.ReadFile(entry.Content)
		if err != nil {
			return
		}
		if err := clipboard.Init(); err != nil {
			return
		}
		clipboard.Write(clipboard.FmtImage, data)
		return
	}
	w.Clipboard().SetContent(entry.Content)
}

// imageLabel returns a short display string for an image entry.
func imageLabel(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "Image"
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "Image"
	}
	return fmt.Sprintf("Image (%d×%d)", cfg.Width, cfg.Height)
}

// truncateText collapses whitespace and caps at 80 runes.
func truncateText(s string) string {
	text := strings.ReplaceAll(s, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")
	if len([]rune(text)) > 80 {
		text = string([]rune(text)[:80]) + "…"
	}
	return text
}

// previewText returns a short preview string for the status bar.
func previewText(entry history.ClipboardEntry) string {
	if entry.Type == history.EntryTypeImage {
		return imageLabel(entry.Content)
	}
	text := strings.ReplaceAll(entry.Content, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")
	if len([]rune(text)) > 50 {
		text = string([]rune(text)[:50]) + "…"
	}
	return text
}

func showSettings(a fyne.App, parent fyne.Window, cfg *config.Config, onClear func(), onClearUI func()) {
	sw := a.NewWindow("Settings")
	sw.Resize(fyne.NewSize(360, 320))
	sw.SetFixedSize(true)
	sw.CenterOnScreen()

	maxEntriesVal := newFloat(float64(cfg.MaxEntries))
	maxEntriesLabel := widget.NewLabel(fmt.Sprintf("Max entries: %d", cfg.MaxEntries))
	maxEntriesSlider := widget.NewSlider(10, 500)
	maxEntriesSlider.Step = 10
	maxEntriesSlider.Value = float64(cfg.MaxEntries)
	maxEntriesSlider.OnChanged = func(v float64) {
		*maxEntriesVal = v
		maxEntriesLabel.SetText(fmt.Sprintf("Max entries: %d", int(v)))
	}

	maxImageVal := newFloat(float64(cfg.MaxImageSizeMB))
	maxImageLabel := widget.NewLabel(fmt.Sprintf("Max image size: %d MB", cfg.MaxImageSizeMB))
	maxImageSlider := widget.NewSlider(1, 50)
	maxImageSlider.Step = 1
	maxImageSlider.Value = float64(cfg.MaxImageSizeMB)
	maxImageSlider.OnChanged = func(v float64) {
		*maxImageVal = v
		maxImageLabel.SetText(fmt.Sprintf("Max image size: %d MB", int(v)))
	}

	keepOpenCheck := widget.NewCheck("Keep window open after selection", func(v bool) {
		cfg.KeepWindowOpen = v
	})
	keepOpenCheck.SetChecked(cfg.KeepWindowOpen)

	clearBtn := widget.NewButton("🗑 Clear History", func() {
		dialog.ShowConfirm(
			"Clear History",
			"Delete all clipboard history entries?",
			func(confirmed bool) {
				if confirmed {
					if onClear != nil {
						onClear()
					}
					if onClearUI != nil {
						onClearUI()
					}
				}
			},
			sw,
		)
	})

	saveBtn := widget.NewButton("Save", func() {
		cfg.MaxEntries = int(*maxEntriesVal)
		cfg.MaxImageSizeMB = int(*maxImageVal)
		if err := cfg.Save(); err != nil {
			dialog.ShowError(err, sw)
			return
		}
		sendSIGHUP()
		sw.Close()
	})

	sw.SetContent(container.NewVBox(
		widget.NewLabel("History"),
		maxEntriesLabel,
		maxEntriesSlider,
		widget.NewSeparator(),
		widget.NewLabel("Images"),
		maxImageLabel,
		maxImageSlider,
		widget.NewSeparator(),
		widget.NewLabel("Behavior"),
		keepOpenCheck,
		widget.NewSeparator(),
		widget.NewLabel("Danger zone"),
		clearBtn,
		widget.NewSeparator(),
		container.NewHBox(widget.NewLabel(""), saveBtn),
	))

	sw.Show()
}

func newFloat(v float64) *float64 {
	return &v
}
