package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type paletteTheme struct {
	colors map[fyne.ThemeColorName]color.Color
}

func (t *paletteTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if c, ok := t.colors[name]; ok {
		return c
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *paletteTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *paletteTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *paletteTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

func hexColor(s string) color.NRGBA {
	var r, g, b uint8
	fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	return color.NRGBA{R: r, G: g, B: b, A: 0xff}
}

func hexColorA(s string, a uint8) color.NRGBA {
	c := hexColor(s)
	c.A = a
	return c
}

// darkTheme is a deep navy/purple dark palette (Catppuccin Mocha-inspired).
func darkTheme() fyne.Theme {
	return &paletteTheme{colors: map[fyne.ThemeColorName]color.Color{
		theme.ColorNameBackground:        hexColor("#1e1e2e"),
		theme.ColorNameForeground:        hexColor("#cdd6f4"),
		theme.ColorNamePrimary:           hexColor("#89b4fa"),
		theme.ColorNameButton:            hexColor("#313244"),
		theme.ColorNameHover:             hexColorA("#cdd6f4", 0x18),
		theme.ColorNameInputBackground:   hexColor("#181825"),
		theme.ColorNameInputBorder:       hexColor("#585b70"),
		theme.ColorNameOverlayBackground: hexColor("#1e1e2e"),
		theme.ColorNamePlaceHolder:       hexColor("#6c7086"),
		theme.ColorNamePressed:           hexColorA("#cdd6f4", 0x28),
		theme.ColorNameScrollBar:         hexColorA("#cdd6f4", 0x30),
		theme.ColorNameSeparator:         hexColor("#313244"),
		theme.ColorNameShadow:            hexColorA("#000000", 0x66),
		theme.ColorNameFocus:             hexColor("#89b4fa"),
		theme.ColorNameDisabled:          hexColor("#585b70"),
		theme.ColorNameDisabledButton:    hexColor("#26263e"),
		theme.ColorNameHeaderBackground:  hexColor("#181825"),
		theme.ColorNameMenuBackground:    hexColor("#181825"),
		theme.ColorNameSelection:         hexColorA("#89b4fa", 0x44),
		theme.ColorNameError:             hexColor("#f38ba8"),
		theme.ColorNameSuccess:           hexColor("#a6e3a1"),
		theme.ColorNameWarning:           hexColor("#f9e2af"),
	}}
}

// lightTheme is a clean light palette (Catppuccin Latte-inspired).
func lightTheme() fyne.Theme {
	return &paletteTheme{colors: map[fyne.ThemeColorName]color.Color{
		theme.ColorNameBackground:        hexColor("#eff1f5"),
		theme.ColorNameForeground:        hexColor("#4c4f69"),
		theme.ColorNamePrimary:           hexColor("#1e66f5"),
		theme.ColorNameButton:            hexColor("#dce0e8"),
		theme.ColorNameHover:             hexColorA("#4c4f69", 0x14),
		theme.ColorNameInputBackground:   hexColor("#ffffff"),
		theme.ColorNameInputBorder:       hexColor("#bcc0cc"),
		theme.ColorNameOverlayBackground: hexColor("#eff1f5"),
		theme.ColorNamePlaceHolder:       hexColor("#9ca0b0"),
		theme.ColorNamePressed:           hexColorA("#4c4f69", 0x22),
		theme.ColorNameScrollBar:         hexColorA("#4c4f69", 0x30),
		theme.ColorNameSeparator:         hexColor("#ccd0da"),
		theme.ColorNameShadow:            hexColorA("#000000", 0x20),
		theme.ColorNameFocus:             hexColor("#1e66f5"),
		theme.ColorNameDisabled:          hexColor("#9ca0b0"),
		theme.ColorNameDisabledButton:    hexColor("#e6e9ef"),
		theme.ColorNameHeaderBackground:  hexColor("#e6e9ef"),
		theme.ColorNameMenuBackground:    hexColor("#e6e9ef"),
		theme.ColorNameSelection:         hexColorA("#1e66f5", 0x44),
		theme.ColorNameError:             hexColor("#d20f39"),
		theme.ColorNameSuccess:           hexColor("#40a02b"),
		theme.ColorNameWarning:           hexColor("#df8e1d"),
	}}
}

// nordTheme uses the official Nord color palette (https://www.nordtheme.com).
// Polar Night: #2E3440 #3B4252 #434C5E #4C566A
// Snow Storm:  #D8DEE9 #E5E9F0 #ECEFF4
// Frost:       #8FBCBB #88C0D0 #81A1C1 #5E81AC
// Aurora:      #BF616A #D08770 #EBCB8B #A3BE8C #B48EAD
func nordTheme() fyne.Theme {
	return &paletteTheme{colors: map[fyne.ThemeColorName]color.Color{
		theme.ColorNameBackground:        hexColor("#2E3440"), // Nord0
		theme.ColorNameForeground:        hexColor("#ECEFF4"), // Nord6
		theme.ColorNamePrimary:           hexColor("#88C0D0"), // Nord8 (Frost)
		theme.ColorNameButton:            hexColor("#3B4252"), // Nord1
		theme.ColorNameHover:             hexColorA("#ECEFF4", 0x18),
		theme.ColorNameInputBackground:   hexColor("#3B4252"), // Nord1
		theme.ColorNameInputBorder:       hexColor("#4C566A"), // Nord3
		theme.ColorNameOverlayBackground: hexColor("#2E3440"), // Nord0
		theme.ColorNamePlaceHolder:       hexColorA("#D8DEE9", 0x88), // Nord4 @55% — readable but distinct from real text
		theme.ColorNamePressed:           hexColorA("#ECEFF4", 0x28),
		theme.ColorNameScrollBar:         hexColorA("#ECEFF4", 0x30),
		theme.ColorNameSeparator:         hexColor("#3B4252"), // Nord1
		theme.ColorNameShadow:            hexColorA("#000000", 0x55),
		theme.ColorNameFocus:             hexColor("#88C0D0"), // Nord8
		theme.ColorNameDisabled:          hexColor("#4C566A"), // Nord3
		theme.ColorNameDisabledButton:    hexColor("#2E3440"), // Nord0
		theme.ColorNameHeaderBackground:  hexColor("#242933"), // slightly darker than Nord0
		theme.ColorNameMenuBackground:    hexColor("#2E3440"), // Nord0
		theme.ColorNameSelection:         hexColorA("#88C0D0", 0x44),
		theme.ColorNameError:             hexColor("#BF616A"), // Aurora red
		theme.ColorNameSuccess:           hexColor("#A3BE8C"), // Aurora green
		theme.ColorNameWarning:           hexColor("#EBCB8B"), // Aurora yellow
	}}
}

// ThemeForName returns the fyne.Theme for the given config key.
func ThemeForName(name string) fyne.Theme {
	switch name {
	case "light":
		return lightTheme()
	case "nord":
		return nordTheme()
	default:
		return darkTheme()
	}
}

// ThemeOptions lists available themes in display order.
var ThemeOptions = []struct {
	Label string
	Key   string
}{
	{"Dark", "dark"},
	{"Light", "light"},
	{"Nord", "nord"},
}
