package ui

import (
	"errors"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

type FyneUI struct{}

func NewFyneUI() *FyneUI {
	return &FyneUI{}
}

// Show displays the clipboard history picker. It returns a channel that emits
// each item the user selects; the channel is closed when the window is dismissed.
func (f *FyneUI) Show(items []string, updates <-chan string) (<-chan string, error) {
	if len(items) == 0 && updates == nil {
		return nil, errors.New("no history entries")
	}

	selections := make(chan string, 16)

	a := app.New()
	w := a.NewWindow("Clipboard History")
	w.Resize(fyne.NewSize(500, 420))
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
		}
		list.Unselect(id)
	}

	// Listen for new items streamed from the daemon.
	if updates != nil {
		go func() {
			for item := range updates {
				current, _ := data.Get()
				_ = data.Set(append([]string{item}, current...))
			}
		}()
	}

	w.SetOnClosed(func() {
		close(selections)
	})

	w.SetContent(container.NewBorder(
		widget.NewLabel("Select an entry to copy:"),
		statusLabel,
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
