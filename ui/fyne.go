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
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
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

// searchEntry is a custom Entry that forwards our global shortcuts even when focused.
// Fyne routes shortcuts to the focused widget first; without this wrapper those
// shortcuts would be swallowed by the default Entry handler.
type searchEntry struct {
	widget.Entry
	onCtrlF     func()
	onCtrlD     func()
	onCtrlSlash func()
	onCtrlH     func()
}

func newSearchEntry() *searchEntry {
	e := &searchEntry{}
	e.ExtendBaseWidget(e)
	return e
}

func (e *searchEntry) TypedShortcut(s fyne.Shortcut) {
	cs, ok := s.(*desktop.CustomShortcut)
	if ok {
		switch cs.KeyName {
		case fyne.KeyF:
			if e.onCtrlF != nil {
				e.onCtrlF()
				return
			}
		case fyne.KeyD:
			if e.onCtrlD != nil {
				e.onCtrlD()
				return
			}
		case fyne.KeySlash:
			if e.onCtrlSlash != nil {
				e.onCtrlSlash()
				return
			}
		case fyne.KeyH:
			if e.onCtrlH != nil {
				e.onCtrlH()
				return
			}
		}
	}
	e.Entry.TypedShortcut(s)
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

	// filtered holds the current view after applying the search query.
	var filtered []history.ClipboardEntry
	var query string

	applyFilter := func() {
		if query == "" {
			filtered = current
		} else {
			filtered = nil
			for _, e := range current {
				var haystack string
				if e.Type == history.EntryTypeImage {
					haystack = imageLabel(e.Content)
				} else {
					haystack = e.Content
				}
				if fuzzyMatch(query, haystack) {
					filtered = append(filtered, e)
				}
			}
		}
	}
	applyFilter()

	statusLabel := widget.NewLabel("")

	var list *widget.List
	list = widget.NewList(
		func() int {
			mu.Lock()
			defer mu.Unlock()
			return len(filtered)
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
			if id >= len(filtered) {
				mu.Unlock()
				return
			}
			entry := filtered[id]
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
				img.Resource = nil
				img.Hide()
				lbl.SetText(truncateText(entry.Content))
			}
			box.Refresh()
		},
	)

	emptyLabel := widget.NewLabel("No history yet — start copying something!")
	emptyLabel.Alignment = fyne.TextAlignCenter

	noResultsLabel := widget.NewLabel("No results for this search.")
	noResultsLabel.Alignment = fyne.TextAlignCenter
	noResultsLabel.Hide()

	refreshEmpty := func() {
		mu.Lock()
		totalEmpty := len(current) == 0
		filteredEmpty := len(filtered) == 0
		mu.Unlock()

		if totalEmpty {
			emptyLabel.Show()
			noResultsLabel.Hide()
		} else if filteredEmpty {
			emptyLabel.Hide()
			noResultsLabel.Show()
		} else {
			emptyLabel.Hide()
			noResultsLabel.Hide()
		}
	}

	searchVisible := false
	var searchBtn *widget.Button

	updateSearchIcon := func() {
		mu.Lock()
		hasQuery := query != ""
		mu.Unlock()
		if hasQuery && !searchVisible {
			searchBtn.Icon = theme.SearchReplaceIcon()
		} else {
			searchBtn.Icon = theme.SearchIcon()
		}
		searchBtn.Refresh()
	}

	searchField := newSearchEntry()
	searchField.SetPlaceHolder("Search…")
	searchField.Hide()

	searchField.OnChanged = func(q string) {
		mu.Lock()
		query = q
		applyFilter()
		mu.Unlock()
		list.Refresh()
		refreshEmpty()
		updateSearchIcon()
	}

	var showMain func()
	onMainScreen := true

	// helpOpen prevents stacking multiple help dialogs and lets Escape close it.
	helpOpen := false
	var activeHelp dialog.Dialog

	showHelp := func() {
		if helpOpen {
			return
		}
		helpOpen = true

		type row struct{ key, action string }
		shortcuts := []row{
			{"Ctrl+F", "Toggle search bar (filter stays active when closed)"},
			{"Ctrl+D", "Clear search input"},
			{"Ctrl+/", "Open Settings"},
			{"Ctrl+H", "Show this help"},
			{"↑ / ↓", "Navigate entries"},
			{"Space", "Copy selected entry"},
			{"Escape", "Close search / close window"},
		}

		rows := []fyne.CanvasObject{
			container.NewGridWithColumns(2,
				widget.NewLabelWithStyle("Shortcut", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabelWithStyle("Action", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			),
			widget.NewSeparator(),
		}
		for _, s := range shortcuts {
			key := widget.NewLabelWithStyle(s.key, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
			action := widget.NewLabel(s.action)
			action.Wrapping = fyne.TextWrapWord
			rows = append(rows, container.NewGridWithColumns(2, key, action))
		}

		content := container.NewScroll(container.NewVBox(rows...))
		content.SetMinSize(fyne.NewSize(420, 300))

		d := dialog.NewCustom("Keyboard Shortcuts", "Close", content, w)
		d.SetOnClosed(func() {
			helpOpen = false
			activeHelp = nil
		})
		d.Resize(fyne.NewSize(460, 360))
		d.Show()
		activeHelp = d
	}

	openSettings := func() {
		onMainScreen = false
		onClearUI := func() {
			statusLabel.SetText("")
			mu.Lock()
			current = nil
			query = ""
			applyFilter()
			mu.Unlock()
			list.Refresh()
			refreshEmpty()
		}
		settingsContent := buildSettingsContent(w, cfg, onClear, onClearUI, func() { showMain() })
		w.SetTitle("Settings")
		w.SetContent(settingsContent)
		w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
			if ev.Name == fyne.KeyEscape {
				showMain()
			}
		})
	}

	// hideSearch closes the search bar keeping the active filter.
	hideSearch := func() {
		searchVisible = false
		searchField.Hide()
		w.Canvas().Focus(list)
		updateSearchIcon()
	}

	// showSearch opens the search bar and focuses the input.
	showSearch := func() {
		searchVisible = true
		searchField.Show()
		w.Canvas().Focus(searchField)
		updateSearchIcon()
	}

	// clearSearch empties the search input and resets the filter.
	clearSearch := func() {
		searchField.SetText("")
	}

	// Wire searchEntry shortcut handlers — these fire when the input has focus.
	searchField.onCtrlF = func() {
		if onMainScreen {
			hideSearch()
		}
	}
	searchField.onCtrlD = func() {
		clearSearch()
	}
	searchField.onCtrlSlash = func() {
		if onMainScreen {
			openSettings()
		}
	}
	searchField.onCtrlH = func() {
		showHelp()
	}

	searchBtn = widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		if searchVisible {
			hideSearch()
		} else {
			showSearch()
		}
	})

	settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		openSettings()
	})

	header := container.NewBorder(
		nil, nil,
		widget.NewLabel("Select an entry to copy:"),
		searchBtn,
		searchField,
	)

	mainContent := container.NewBorder(
		container.NewVBox(header),
		container.NewBorder(nil, nil, nil, settingsBtn, statusLabel),
		nil, nil,
		container.NewStack(list, container.NewCenter(emptyLabel), container.NewCenter(noResultsLabel)),
	)

	showMain = func() {
		onMainScreen = true
		w.SetTitle("Clipboard History")
		w.SetContent(mainContent)
		w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
			if ev.Name != fyne.KeyEscape {
				return
			}
			if helpOpen {
				activeHelp.Hide()
				return
			}
			if query != "" || searchVisible {
				searchField.SetText("")
				hideSearch()
				mu.Lock()
				query = ""
				applyFilter()
				mu.Unlock()
				list.Refresh()
				refreshEmpty()
			} else {
				w.Close()
			}
		})
		refreshEmpty()
	}

	// Canvas-level shortcuts fire when the list (or nothing) has focus.
	ctrlShortcut := func(key fyne.KeyName) *desktop.CustomShortcut {
		return &desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierControl}
	}

	w.Canvas().AddShortcut(ctrlShortcut(fyne.KeyF), func(_ fyne.Shortcut) {
		if !onMainScreen {
			return
		}
		if searchVisible {
			hideSearch()
		} else {
			showSearch()
		}
	})

	w.Canvas().AddShortcut(ctrlShortcut(fyne.KeyD), func(_ fyne.Shortcut) {
		if !onMainScreen || !searchVisible {
			return
		}
		clearSearch()
	})

	w.Canvas().AddShortcut(ctrlShortcut(fyne.KeySlash), func(_ fyne.Shortcut) {
		if !onMainScreen {
			return
		}
		openSettings()
	})

	w.Canvas().AddShortcut(ctrlShortcut(fyne.KeyH), func(_ fyne.Shortcut) {
		showHelp()
	})

	list.OnSelected = func(id widget.ListItemID) {
		mu.Lock()
		if id >= len(filtered) {
			mu.Unlock()
			return
		}
		entry := filtered[id]
		mu.Unlock()

		writeToClipboard(w, entry)
		selections <- entry
		statusLabel.SetText("Copied: " + previewText(entry))
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
				deduped := current[:0]
				for _, e := range current {
					if !(e.Type == entry.Type && e.Content == entry.Content) {
						deduped = append(deduped, e)
					}
				}
				current = append([]history.ClipboardEntry{entry}, deduped...)
				applyFilter()
				mu.Unlock()
				list.Refresh()
				refreshEmpty()
			case _, ok := <-focusReqs:
				if !ok {
					return
				}
				w.RequestFocus()
			}
		}
	}()

	w.SetOnClosed(func() {
		close(selections)
	})

	w.SetContent(mainContent)
	w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyEscape {
			w.Close()
		}
	})

	w.Show()
	showMain()
	a.Run()

	return selections, nil
}

// fuzzyMatch returns true if all runes of pattern appear in target in order,
// case-insensitive.
func fuzzyMatch(pattern, target string) bool {
	pattern = strings.ToLower(pattern)
	target = strings.ToLower(target)
	pi := 0
	prunes := []rune(pattern)
	for _, r := range target {
		if pi < len(prunes) && unicode.ToLower(r) == prunes[pi] {
			pi++
		}
	}
	return pi == len(prunes)
}

// buildSettingsContent returns the settings screen content for in-window navigation.
func buildSettingsContent(w fyne.Window, cfg *config.Config, onClear func(), onClearUI func(), goBack func()) fyne.CanvasObject {
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
					goBack()
				}
			},
			w,
		)
	})

	saveBtn := widget.NewButton("Save", func() {
		cfg.MaxEntries = int(*maxEntriesVal)
		cfg.MaxImageSizeMB = int(*maxImageVal)
		if err := cfg.Save(); err != nil {
			dialog.ShowError(err, w)
			return
		}
		sendSIGHUP()
		goBack()
	})

	backBtn := widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), func() {
		goBack()
	})

	body := container.NewVBox(
		widget.NewSeparator(),
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
	)

	return container.NewBorder(
		container.NewBorder(nil, nil, backBtn, saveBtn,
			widget.NewLabelWithStyle("Settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		),
		nil, nil, nil,
		container.NewScroll(body),
	)
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

func newFloat(v float64) *float64 {
	return &v
}
