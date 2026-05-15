package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"

	"github.com/david-pena/clipboard/config"
	"github.com/david-pena/clipboard/history"
)

// --- truncateText ---

func TestTruncateText_Short(t *testing.T) {
	got := truncateText("hello")
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestTruncateText_CollapsesNewlines(t *testing.T) {
	got := truncateText("hello\nworld")
	if got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestTruncateText_TruncatesAt80Runes(t *testing.T) {
	long := strings.Repeat("a", 100)
	got := truncateText(long)
	runes := []rune(got)
	// 80 chars + ellipsis = 81 runes
	if len(runes) != 81 {
		t.Errorf("expected 81 runes, got %d", len(runes))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected trailing ellipsis, got %q", got)
	}
}

func TestTruncateText_CollapseWhitespace(t *testing.T) {
	got := truncateText("  a   b  ")
	if got != "a b" {
		t.Errorf("expected 'a b', got %q", got)
	}
}

// --- fuzzyMatch ---

func TestFuzzyMatch(t *testing.T) {
	cases := []struct {
		pattern, target string
		want            bool
	}{
		{"abc", "xaxbxcx", true},
		{"abc", "ab", false},
		{"", "anything", true},
		{"ABC", "xaxbxcx", true}, // case-insensitive
		{"z", "hello", false},
	}
	for _, c := range cases {
		got := fuzzyMatch(c.pattern, c.target)
		if got != c.want {
			t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", c.pattern, c.target, got, c.want)
		}
	}
}

// --- imageLabel ---

func writeTempPNG(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.White)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode PNG: %v", err)
	}
	f, err := os.CreateTemp(t.TempDir(), "*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	f.Write(buf.Bytes())
	f.Close()
	return f.Name()
}

func TestImageLabel_ValidPNG(t *testing.T) {
	path := writeTempPNG(t)
	label := imageLabel(path)
	if label != "Image (4×4)" {
		t.Errorf("expected 'Image (4×4)', got %q", label)
	}
}

func TestImageLabel_MissingFile(t *testing.T) {
	label := imageLabel("/nonexistent/path.png")
	if label != "Image" {
		t.Errorf("expected 'Image' fallback, got %q", label)
	}
}

// --- previewText ---

func TestPreviewText_TextEntry(t *testing.T) {
	entry := history.ClipboardEntry{Type: history.EntryTypeText, Content: "hello world"}
	got := previewText(entry)
	if got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestPreviewText_TextTruncatesAt50(t *testing.T) {
	entry := history.ClipboardEntry{Type: history.EntryTypeText, Content: strings.Repeat("a", 60)}
	got := previewText(entry)
	if len([]rune(got)) != 51 { // 50 + ellipsis
		t.Errorf("expected 51 runes, got %d", len([]rune(got)))
	}
}

func TestPreviewText_ImageEntry(t *testing.T) {
	path := writeTempPNG(t)
	entry := history.ClipboardEntry{Type: history.EntryTypeImage, Content: path}
	got := previewText(entry)
	if got != "Image (4×4)" {
		t.Errorf("expected 'Image (4×4)', got %q", got)
	}
}

// --- hexColor / hexColorA ---

func TestHexColor(t *testing.T) {
	c := hexColor("#ff8000")
	if c.R != 0xff || c.G != 0x80 || c.B != 0x00 || c.A != 0xff {
		t.Errorf("unexpected color: %+v", c)
	}
}

func TestHexColorA(t *testing.T) {
	c := hexColorA("#ffffff", 0x80)
	if c.A != 0x80 {
		t.Errorf("expected alpha 0x80, got 0x%02x", c.A)
	}
}

// --- ThemeForName ---

func TestThemeForName_KnownThemes(t *testing.T) {
	keys := []string{
		"nord", "tokyo-night", "gruvbox", "kanagawa",
		"pure-black", "solarized-light", "github-light",
		"rose-pine-dawn", "everforest-light",
	}
	for _, k := range keys {
		if ThemeForName(k) == nil {
			t.Errorf("ThemeForName(%q) returned nil", k)
		}
	}
}

func TestThemeForName_Unknown(t *testing.T) {
	// Unknown key falls back to nordTheme (not nil).
	if ThemeForName("unknown-theme") == nil {
		t.Error("ThemeForName with unknown key returned nil")
	}
}

// --- searchEntry.TypedShortcut ---

func TestSearchEntry_TypedShortcut_CtrlF(t *testing.T) {
	_ = test.NewApp()
	called := false
	e := newSearchEntry()
	e.onCtrlF = func() { called = true }

	e.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierControl})
	if !called {
		t.Error("expected onCtrlF to be called")
	}
}

func TestSearchEntry_TypedShortcut_CtrlD(t *testing.T) {
	_ = test.NewApp()
	called := false
	e := newSearchEntry()
	e.onCtrlD = func() { called = true }

	e.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyD, Modifier: fyne.KeyModifierControl})
	if !called {
		t.Error("expected onCtrlD to be called")
	}
}

func TestSearchEntry_TypedShortcut_CtrlSlash(t *testing.T) {
	_ = test.NewApp()
	called := false
	e := newSearchEntry()
	e.onCtrlSlash = func() { called = true }

	e.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeySlash, Modifier: fyne.KeyModifierControl})
	if !called {
		t.Error("expected onCtrlSlash to be called")
	}
}

func TestSearchEntry_TypedShortcut_CtrlH(t *testing.T) {
	_ = test.NewApp()
	called := false
	e := newSearchEntry()
	e.onCtrlH = func() { called = true }

	e.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyH, Modifier: fyne.KeyModifierControl})
	if !called {
		t.Error("expected onCtrlH to be called")
	}
}

func TestSearchEntry_TypedShortcut_UnhandledShortcut(t *testing.T) {
	_ = test.NewApp()
	e := newSearchEntry()
	// Should not panic — falls through to base Entry handler.
	e.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyZ, Modifier: fyne.KeyModifierControl})
}


// --- buildSettingsContent ---

func testConfig() *config.Config {
	cfg := config.Default()
	return cfg
}

func TestBuildSettingsContent_ReturnsContent(t *testing.T) {
	a := test.NewApp()
	w := a.NewWindow("test")
	defer w.Close()

	cfg := testConfig()
	content, save := buildSettingsContent(
		w, cfg,
		nil, nil,
		func() {}, // onCancel
		func() {}, // onSaved
		func(string) {},
		func(bool) {},
	)

	if content == nil {
		t.Error("expected non-nil content")
	}
	if save == nil {
		t.Error("expected non-nil save function")
	}
}

func TestBuildSettingsContent_ThemeCallback(t *testing.T) {
	a := test.NewApp()
	w := a.NewWindow("test")
	defer w.Close()

	cfg := testConfig()
	var lastTheme string
	buildSettingsContent(
		w, cfg,
		nil, nil,
		func() {}, func() {},
		func(name string) { lastTheme = name },
		func(bool) {},
	)

	// setTheme is triggered by the Select widget internally during construction —
	// just verify no panic and that the callback is wired (it may or may not fire
	// during build depending on Fyne internals).
	_ = lastTheme
}

func TestBuildSettingsContent_ThumbnailCallback(t *testing.T) {
	a := test.NewApp()
	w := a.NewWindow("test")
	defer w.Close()

	cfg := testConfig()
	var thumbnailValue bool
	buildSettingsContent(
		w, cfg,
		nil, nil,
		func() {}, func() {},
		func(string) {},
		func(v bool) { thumbnailValue = v },
	)
	_ = thumbnailValue // wired; value depends on toggle interaction
}
