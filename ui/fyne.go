package ui

import (
	"errors"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/david-pena/clipboard/config"
)

type FyneUI struct{}

func NewFyneUI() *FyneUI {
	return &FyneUI{}
}

// Show displays the clipboard history picker. It returns a channel that emits
// each item the user selects; the channel is closed when the window is dismissed.
func (f *FyneUI) Show(items []string, updates <-chan string, onClear func()) (<-chan string, error) {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}

	if len(items) == 0 && updates == nil {
		return nil, errors.New("no history entries")
	}

	selections := make(chan string, 16)

	a := app.New()
	w := a.NewWindow("Clipboard History")
	w.Resize(fyne.NewSize(500, 460))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	data := binding.NewStringList()
	_ = data.Set(items)

	statusLabel := widget.NewLabel("")

	list := widget.NewListWithData(
		data,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(item binding.DataItem, o fyne.CanvasObject) {
			s, _ := item.(binding.String).Get()
			text := strings.ReplaceAll(s, "\n", " ")
			text = strings.Join(strings.Fields(text), " ")
			if len([]rune(text)) > 80 {
				text = string([]rune(text)[:80]) + "…"
			}
			o.(*widget.Label).SetText(text)
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		all, _ := data.Get()
		if id < len(all) {
			item := all[id]
			w.Clipboard().SetContent(item)
			selections <- item
			preview := strings.ReplaceAll(item, "\n", " ")
			preview = strings.Join(strings.Fields(preview), " ")
			if len([]rune(preview)) > 50 {
				preview = string([]rune(preview)[:50]) + "…"
			}
			statusLabel.SetText("Copied: " + preview)
			if !cfg.KeepWindowOpen {
				w.Close()
			}
		}
		list.Unselect(id)
	}

	if updates != nil {
		go func() {
			for item := range updates {
				current, _ := data.Get()
				filtered := current[:0]
				for _, e := range current {
					if e != item {
						filtered = append(filtered, e)
					}
				}
				_ = data.Set(append([]string{item}, filtered...))
			}
		}()
	}

	settingsBtn := widget.NewButton("⚙ Settings", func() {
		showSettings(a, w, cfg, data, onClear)
	})

	w.SetOnClosed(func() {
		close(selections)
	})

	w.SetContent(container.NewBorder(
		widget.NewLabel("Select an entry to copy:"),
		container.NewBorder(nil, nil, nil, settingsBtn, statusLabel),
		nil, nil,
		list,
	))

	w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyEscape {
			w.Close()
		}
	})

	w.Show()
	a.Run()

	return selections, nil
}

func showSettings(a fyne.App, parent fyne.Window, cfg *config.Config, data binding.StringList, onClear func()) {
	sw := a.NewWindow("Settings")
	sw.Resize(fyne.NewSize(360, 280))
	sw.SetFixedSize(true)
	sw.CenterOnScreen()

	maxEntriesVal := binding.NewFloat()
	_ = maxEntriesVal.Set(float64(cfg.MaxEntries))
	maxEntriesLabel := widget.NewLabelWithData(binding.FloatToStringWithFormat(maxEntriesVal, "Max entries: %.0f"))
	maxEntriesSlider := widget.NewSliderWithData(10, 500, maxEntriesVal)
	maxEntriesSlider.Step = 10

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
					_ = data.Set([]string{})
					if onClear != nil {
						onClear()
					}
				}
			},
			sw,
		)
	})

	saveBtn := widget.NewButton("Save", func() {
		maxVal, _ := maxEntriesVal.Get()
		cfg.MaxEntries = int(maxVal)
		if err := cfg.Save(); err != nil {
			dialog.ShowError(err, sw)
			return
		}
		// Send SIGHUP to the daemon so it reloads max_entries.
		sendSIGHUP()
		sw.Close()
	})

	sw.SetContent(container.NewVBox(
		widget.NewLabel("History"),
		maxEntriesLabel,
		maxEntriesSlider,
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
