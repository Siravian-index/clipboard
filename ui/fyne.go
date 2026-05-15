package ui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
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
	"golang.org/x/image/draw"

	"golang.design/x/clipboard"

	"github.com/david-pena/clipboard/config"
	"github.com/david-pena/clipboard/history"
)

type FyneUI struct{}

func NewFyneUI() *FyneUI {
	return &FyneUI{}
}

// SearchResponse carries search results from the daemon.
type SearchResponse struct {
	Items        []history.ClipboardEntry
	TotalMatches int
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
func (f *FyneUI) Show(items []history.ClipboardEntry, initialTotal int, updates <-chan history.ClipboardEntry, refreshes <-chan []history.ClipboardEntry, searches <-chan SearchResponse, counts <-chan int, sendSearch func(string), onClear func(), focusReqs <-chan struct{}) (<-chan history.ClipboardEntry, error) {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}

	if len(items) == 0 && updates == nil {
		return nil, errors.New("no history entries")
	}

	selections := make(chan history.ClipboardEntry, 16)

	a := app.New()
	a.Settings().SetTheme(ThemeForName(cfg.Theme))
	w := a.NewWindow("Clipboard History")
	w.Resize(fyne.NewSize(500, 520))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	var mu sync.Mutex
	current := make([]history.ClipboardEntry, len(items))
	copy(current, items)

	// imageLabelCache avoids re-reading + decoding image files on every scroll tick.
	imageLabelCache := make(map[string]string)
	// truncateCache avoids recomputing the same string on every scroll tick.
	truncateCache := make(map[string]string)

	// filtered holds the current view after applying the search query.
	var filtered []history.ClipboardEntry
	var activeQuery string
	var totalMatches int

	cachedImageLabel := func(path string) string {
		if label, ok := imageLabelCache[path]; ok {
			return label
		}
		label := imageLabel(path)
		imageLabelCache[path] = label
		return label
	}

	cachedTruncate := func(s string) string {
		if t, ok := truncateCache[s]; ok {
			return t
		}
		t := truncateText(s)
		truncateCache[s] = t
		return t
	}

	applyFilter := func() {
		if activeQuery == "" {
			filtered = current
		}
		// When activeQuery != "", filtered is set from search results received from daemon.
	}
	applyFilter()

	statusLabel := widget.NewLabel("")
	totalCount := initialTotal
	countLabel := widget.NewLabel(fmt.Sprintf("(%d)", totalCount))

	countFn := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(filtered)
	}

	const thumbSize = 100

	// sourceCache holds fully decoded images keyed by path.
	// Populated before w.Show() so UpdateItem never does I/O or decode.
	sourceCache := make(map[string]image.Image)
	cachedSource := func(path string) image.Image {
		if src, ok := sourceCache[path]; ok {
			return src
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		src, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil
		}
		sourceCache[path] = src
		return src
	}

	// scaledCache avoids re-scaling when the Generator is called repeatedly
	// with the same dimensions. Keyed by "path:w:h".
	scaledCache := make(map[string]*image.NRGBA)
	scaleThumbnail := func(src image.Image, path string, w, h int) *image.NRGBA {
		key := fmt.Sprintf("%s:%d:%d", path, w, h)
		if t, ok := scaledCache[key]; ok {
			return t
		}
		dst := scaleContain(src, w, h)
		scaledCache[key] = dst
		return dst
	}

	// rasterPaths tracks which image path is currently displayed in each
	// recycled Raster cell so Refresh() is only called when the path changes.
	rasterPaths := make(map[*canvas.Raster]string)

	// transparent placeholder returned by Raster cells before an image is assigned.
	placeholder := image.NewUniform(color.Transparent)

	// showThumbnails is read by UpdateItem and can be flipped at runtime.
	showThumbnails := cfg.ShowImageThumbnails

	list := widget.NewList(
		countFn,
		func() fyne.CanvasObject {
			r := canvas.NewRaster(func(w, h int) image.Image { return placeholder })
			r.SetMinSize(fyne.NewSize(thumbSize, thumbSize))
			r.Hide()
			lbl := widget.NewLabel("")
			return container.NewBorder(nil, nil, r, nil, lbl)
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
			r := box.Objects[1].(*canvas.Raster)
			lbl := box.Objects[0].(*widget.Label)

			if entry.Type == history.EntryTypeImage && showThumbnails {
				if rasterPaths[r] != entry.Content {
					if src := cachedSource(entry.Content); src != nil {
						path := entry.Content
						r.Generator = func(w, h int) image.Image {
							return scaleThumbnail(src, path, w, h)
						}
						r.Refresh()
						rasterPaths[r] = entry.Content
					}
				}
				if !r.Visible() {
					r.Show()
				}
				if t := cachedImageLabel(entry.Content); lbl.Text != t {
					lbl.SetText(t)
				}
			} else {
				if r.Visible() {
					r.Generator = func(w, h int) image.Image { return placeholder }
					rasterPaths[r] = ""
					r.Hide()
				}
				var t string
				if entry.Type == history.EntryTypeImage {
					t = cachedImageLabel(entry.Content)
				} else {
					t = cachedTruncate(entry.Content)
				}
				if lbl.Text != t {
					lbl.SetText(t)
				}
			}
		},
	)

	emptyLabel := widget.NewLabel("No history yet — start copying something!")
	emptyLabel.Alignment = fyne.TextAlignCenter

	noResultsLabel := widget.NewLabel("No results for this search.")
	noResultsLabel.Alignment = fyne.TextAlignCenter
	noResultsLabel.Hide()

	moreResultsLine := canvas.NewLine(theme.WarningColor())
	moreResultsLine.StrokeWidth = 2
	moreResultsLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	moreResultsLabel.Importance = widget.LowImportance
	moreResultsContainer := container.NewVBox(moreResultsLine, moreResultsLabel)
	moreResultsContainer.Hide()

	var listArea *fyne.Container
	var refreshMore func()

	refreshEmpty := func() {
		mu.Lock()
		totalEmpty := len(current) == 0
		filteredEmpty := len(filtered) == 0
		mu.Unlock()

		countLabel.SetText(fmt.Sprintf("(%d)", totalCount))
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

	refreshMore = func() {
		mu.Lock()
		extra := totalMatches - len(filtered)
		q := activeQuery
		mu.Unlock()
		if q != "" && extra > 0 {
			moreResultsLabel.SetText(fmt.Sprintf("↓ %d more results — refine your search", extra))
			moreResultsContainer.Show()
		} else {
			moreResultsContainer.Hide()
		}
		if listArea != nil {
			listArea.Refresh()
		}
	}

	searchVisible := false
	var searchBtn *widget.Button

	updateSearchIcon := func() {
		mu.Lock()
		hasQuery := activeQuery != ""
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
		activeQuery = q
		if q == "" {
			filtered = current
			totalMatches = 0
		}
		mu.Unlock()
		if q != "" {
			sendSearch(q)
		} else {
			list.Refresh()
			refreshEmpty()
			refreshMore()
		}
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
			{"Ctrl+F", "Toggle search bar"},
			{"Ctrl+D", "Clear search input"},
			{"Ctrl+/", "Open Settings"},
			{"Ctrl+H", "Show this help"},
			{"Ctrl+S", "Save settings"},
			{"↑ / ↓", "Navigate entries"},
			{"Space", "Confirm selection"},
			{"Escape", "Close dialogs"},
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

	// settingsSave holds the active save function while settings is open.
	var settingsSave func()

	openSettings := func() {
		onMainScreen = false
		onClearUI := func() {
			statusLabel.SetText("")
			totalCount = 0
			mu.Lock()
			current = nil
			activeQuery = ""
			totalMatches = 0
			filtered = nil
			mu.Unlock()
			list.Refresh()
			refreshEmpty()
			refreshMore()
		}
		setTheme := func(name string) { a.Settings().SetTheme(ThemeForName(name)) }
		setThumbnails := func(v bool) {
			showThumbnails = v
			if v {
				mu.Lock()
				snap := make([]history.ClipboardEntry, len(current))
				copy(snap, current)
				mu.Unlock()
				for _, item := range snap {
					if item.Type == history.EntryTypeImage {
						cachedSource(item.Content)
					}
				}
			}
			list.Refresh()
		}
		settingsContent, save := buildSettingsContent(w, cfg, onClear, onClearUI, func() { showMain() }, setTheme, setThumbnails)
		settingsSave = save
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

	listArea = container.NewBorder(nil, moreResultsContainer, nil, nil, list)

	mainContent := container.NewBorder(
		container.NewVBox(header),
		container.NewBorder(nil, nil, nil, container.NewHBox(countLabel, settingsBtn), statusLabel),
		nil, nil,
		container.NewStack(listArea, container.NewCenter(emptyLabel), container.NewCenter(noResultsLabel)),
	)

	showMain = func() {
		onMainScreen = true
		w.SetTitle("Clipboard History")
		w.SetContent(mainContent)
		w.Canvas().Focus(list)
		w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
			if ev.Name != fyne.KeyEscape {
				return
			}
			if helpOpen {
				activeHelp.Hide()
				return
			}
			if activeQuery != "" || searchVisible {
				searchField.SetText("")
				hideSearch()
				mu.Lock()
				activeQuery = ""
				totalMatches = 0
				filtered = current
				mu.Unlock()
				list.Refresh()
				refreshEmpty()
				refreshMore()
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

	// Ctrl+S — save in any screen that exposes a save action (e.g. settings).
	w.Canvas().AddShortcut(ctrlShortcut(fyne.KeyS), func(_ fyne.Shortcut) {
		if !onMainScreen && settingsSave != nil {
			settingsSave()
		}
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
				if cfg.MaxEntries > 0 && len(current) > cfg.MaxEntries {
					current = current[:cfg.MaxEntries]
				}
				q := activeQuery
				if q == "" {
					filtered = current
				}
				mu.Unlock()
				if q != "" {
					sendSearch(q)
				} else {
					fyne.Do(func() {
						list.Refresh()
						refreshEmpty()
						refreshMore()
					})
				}
			case newItems, ok := <-refreshes:
				if !ok {
					refreshes = nil
					continue
				}
				mu.Lock()
				current = newItems
				q := activeQuery
				if q == "" {
					filtered = current
				}
				mu.Unlock()
				if q != "" {
					sendSearch(q)
				} else {
					fyne.Do(func() {
						list.Refresh()
						refreshEmpty()
						refreshMore()
					})
				}
			case sr, ok := <-searches:
				if !ok {
					return
				}
				mu.Lock()
				filtered = sr.Items
				totalMatches = sr.TotalMatches
				mu.Unlock()
				fyne.Do(func() {
					list.Refresh()
					refreshEmpty()
					refreshMore()
				})
			case n, ok := <-counts:
				if !ok {
					counts = nil
					continue
				}
				totalCount = n
				fyne.Do(func() { countLabel.SetText(fmt.Sprintf("(%d)", totalCount)) })
			case _, ok := <-focusReqs:
				if !ok {
					return
				}
				fyne.Do(w.RequestFocus)
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

	// Pre-decode all image thumbnails before the window shows so UpdateItem
	// never does I/O or PNG decode on the main thread during render.
	if cfg.ShowImageThumbnails {
		for _, item := range current {
			if item.Type == history.EntryTypeImage {
				cachedSource(item.Content)
			}
		}
	}

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

// buildSettingsContent returns the settings screen content and a save function for in-window navigation.
func buildSettingsContent(w fyne.Window, cfg *config.Config, onClear func(), onClearUI func(), goBack func(), setTheme func(string), setThumbnails func(bool)) (fyne.CanvasObject, func()) {
	selectedTheme := cfg.Theme
	themeLabels := make([]string, len(ThemeOptions))
	themeKeys := make([]string, len(ThemeOptions))
	for i, opt := range ThemeOptions {
		themeLabels[i] = opt.Label
		themeKeys[i] = opt.Key
	}
	themeSelect := widget.NewSelect(themeLabels, func(label string) {
		for i, l := range themeLabels {
			if l == label {
				selectedTheme = themeKeys[i]
				setTheme(selectedTheme)
				break
			}
		}
	})
	for i, k := range themeKeys {
		if k == cfg.Theme {
			themeSelect.SetSelected(themeLabels[i])
			break
		}
	}
	if themeSelect.Selected == "" {
		themeSelect.SetSelected(themeLabels[0])
	}

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

	thumbnailsCheck := widget.NewCheck("Show image thumbnails in list", func(v bool) {
		cfg.ShowImageThumbnails = v
		setThumbnails(v)
	})
	thumbnailsCheck.SetChecked(cfg.ShowImageThumbnails)

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

	save := func() {
		cfg.MaxEntries = int(*maxEntriesVal)
		cfg.MaxImageSizeMB = int(*maxImageVal)
		cfg.Theme = selectedTheme
		if err := cfg.Save(); err != nil {
			dialog.ShowError(err, w)
			return
		}
		sendSIGHUP()
		goBack()
	}

	saveBtn := widget.NewButton("Save", save)

	backBtn := widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), func() {
		goBack()
	})

	body := container.NewVBox(
		widget.NewSeparator(),
		widget.NewLabel("Appearance"),
		widget.NewLabel("Theme"),
		themeSelect,
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
		thumbnailsCheck,
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
	), save
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

// scaleContain scales src to fit within w×h preserving aspect ratio,
// centering the result on a transparent w×h canvas.
func scaleContain(src image.Image, w, h int) *image.NRGBA {
	sb := src.Bounds()
	sw, sh := float64(sb.Dx()), float64(sb.Dy())
	scale := float64(w) / sw
	if sh/sw > float64(h)/float64(w) {
		scale = float64(h) / sh
	}
	fitW, fitH := int(sw*scale), int(sh*scale)
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	offsetX, offsetY := (w-fitW)/2, (h-fitH)/2
	draw.BiLinear.Scale(dst, image.Rect(offsetX, offsetY, offsetX+fitW, offsetY+fitH), src, sb, draw.Over, nil)
	return dst
}
