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

// tokyoNightTheme — deep navy with electric blue/violet accents.
func tokyoNightTheme() fyne.Theme {
	return &paletteTheme{colors: map[fyne.ThemeColorName]color.Color{
		theme.ColorNameBackground:        hexColor("#1a1b26"),
		theme.ColorNameForeground:        hexColor("#c0caf5"),
		theme.ColorNamePrimary:           hexColor("#7aa2f7"),
		theme.ColorNameButton:            hexColor("#24283b"),
		theme.ColorNameHover:             hexColorA("#c0caf5", 0x18),
		theme.ColorNameInputBackground:   hexColor("#16161e"),
		theme.ColorNameInputBorder:       hexColor("#414868"),
		theme.ColorNameOverlayBackground: hexColor("#1a1b26"),
		theme.ColorNamePlaceHolder:       hexColorA("#c0caf5", 0x88),
		theme.ColorNamePressed:           hexColorA("#c0caf5", 0x28),
		theme.ColorNameScrollBar:         hexColorA("#c0caf5", 0x30),
		theme.ColorNameSeparator:         hexColor("#24283b"),
		theme.ColorNameShadow:            hexColorA("#000000", 0x66),
		theme.ColorNameFocus:             hexColor("#7aa2f7"),
		theme.ColorNameDisabled:          hexColor("#414868"),
		theme.ColorNameDisabledButton:    hexColor("#16161e"),
		theme.ColorNameHeaderBackground:  hexColor("#16161e"),
		theme.ColorNameMenuBackground:    hexColor("#16161e"),
		theme.ColorNameSelection:         hexColorA("#7aa2f7", 0x44),
		theme.ColorNameError:             hexColor("#f7768e"),
		theme.ColorNameSuccess:           hexColor("#9ece6a"),
		theme.ColorNameWarning:           hexColor("#e0af68"),
	}}
}

// gruvboxTheme — dark hard variant with warm earthy tones.
func gruvboxTheme() fyne.Theme {
	return &paletteTheme{colors: map[fyne.ThemeColorName]color.Color{
		theme.ColorNameBackground:        hexColor("#1d2021"),
		theme.ColorNameForeground:        hexColor("#ebdbb2"),
		theme.ColorNamePrimary:           hexColor("#fabd2f"),
		theme.ColorNameButton:            hexColor("#282828"),
		theme.ColorNameHover:             hexColorA("#ebdbb2", 0x18),
		theme.ColorNameInputBackground:   hexColor("#141617"),
		theme.ColorNameInputBorder:       hexColor("#504945"),
		theme.ColorNameOverlayBackground: hexColor("#1d2021"),
		theme.ColorNamePlaceHolder:       hexColorA("#ebdbb2", 0x88),
		theme.ColorNamePressed:           hexColorA("#ebdbb2", 0x28),
		theme.ColorNameScrollBar:         hexColorA("#ebdbb2", 0x30),
		theme.ColorNameSeparator:         hexColor("#282828"),
		theme.ColorNameShadow:            hexColorA("#000000", 0x66),
		theme.ColorNameFocus:             hexColor("#fabd2f"),
		theme.ColorNameDisabled:          hexColor("#504945"),
		theme.ColorNameDisabledButton:    hexColor("#141617"),
		theme.ColorNameHeaderBackground:  hexColor("#141617"),
		theme.ColorNameMenuBackground:    hexColor("#141617"),
		theme.ColorNameSelection:         hexColorA("#fabd2f", 0x44),
		theme.ColorNameError:             hexColor("#fb4934"),
		theme.ColorNameSuccess:           hexColor("#b8bb26"),
		theme.ColorNameWarning:           hexColor("#fe8019"),
	}}
}

// kanagawaTheme — ink-black with muted Japanese watercolor accents.
func kanagawaTheme() fyne.Theme {
	return &paletteTheme{colors: map[fyne.ThemeColorName]color.Color{
		theme.ColorNameBackground:        hexColor("#1f1f28"),
		theme.ColorNameForeground:        hexColor("#dcd7ba"),
		theme.ColorNamePrimary:           hexColor("#7fb4ca"),
		theme.ColorNameButton:            hexColor("#2a2a37"),
		theme.ColorNameHover:             hexColorA("#dcd7ba", 0x18),
		theme.ColorNameInputBackground:   hexColor("#16161d"),
		theme.ColorNameInputBorder:       hexColor("#54546d"),
		theme.ColorNameOverlayBackground: hexColor("#1f1f28"),
		theme.ColorNamePlaceHolder:       hexColorA("#dcd7ba", 0x88),
		theme.ColorNamePressed:           hexColorA("#dcd7ba", 0x28),
		theme.ColorNameScrollBar:         hexColorA("#dcd7ba", 0x30),
		theme.ColorNameSeparator:         hexColor("#2a2a37"),
		theme.ColorNameShadow:            hexColorA("#000000", 0x66),
		theme.ColorNameFocus:             hexColor("#7fb4ca"),
		theme.ColorNameDisabled:          hexColor("#54546d"),
		theme.ColorNameDisabledButton:    hexColor("#16161d"),
		theme.ColorNameHeaderBackground:  hexColor("#16161d"),
		theme.ColorNameMenuBackground:    hexColor("#16161d"),
		theme.ColorNameSelection:         hexColorA("#7fb4ca", 0x44),
		theme.ColorNameError:             hexColor("#c34043"),
		theme.ColorNameSuccess:           hexColor("#76946a"),
		theme.ColorNameWarning:           hexColor("#dca561"),
	}}
}

// pureBlackTheme — maximum minimalism, true black for OLED displays.
func pureBlackTheme() fyne.Theme {
	return &paletteTheme{colors: map[fyne.ThemeColorName]color.Color{
		theme.ColorNameBackground:        hexColor("#000000"),
		theme.ColorNameForeground:        hexColor("#e0e0e0"),
		theme.ColorNamePrimary:           hexColor("#555555"),
		theme.ColorNameButton:            hexColor("#1a1a1a"),
		theme.ColorNameHover:             hexColorA("#e0e0e0", 0x18),
		theme.ColorNameInputBackground:   hexColor("#0d0d0d"),
		theme.ColorNameInputBorder:       hexColor("#333333"),
		theme.ColorNameOverlayBackground: hexColor("#000000"),
		theme.ColorNamePlaceHolder:       hexColorA("#e0e0e0", 0x88),
		theme.ColorNamePressed:           hexColorA("#e0e0e0", 0x28),
		theme.ColorNameScrollBar:         hexColorA("#e0e0e0", 0x30),
		theme.ColorNameSeparator:         hexColor("#1a1a1a"),
		theme.ColorNameShadow:            hexColorA("#000000", 0x00),
		theme.ColorNameFocus:             hexColor("#555555"),
		theme.ColorNameDisabled:          hexColor("#444444"),
		theme.ColorNameDisabledButton:    hexColor("#0d0d0d"),
		theme.ColorNameHeaderBackground:  hexColor("#0d0d0d"),
		theme.ColorNameMenuBackground:    hexColor("#0d0d0d"),
		theme.ColorNameSelection:         hexColorA("#e0e0e0", 0x22),
		theme.ColorNameError:             hexColor("#ff5555"),
		theme.ColorNameSuccess:           hexColor("#50fa7b"),
		theme.ColorNameWarning:           hexColor("#ffb86c"),
	}}
}

// solarizedLightTheme — warm cream background, scientifically balanced muted palette.
func solarizedLightTheme() fyne.Theme {
	return &paletteTheme{colors: map[fyne.ThemeColorName]color.Color{
		theme.ColorNameBackground:        hexColor("#fdf6e3"),
		theme.ColorNameForeground:        hexColor("#657b83"),
		theme.ColorNamePrimary:           hexColor("#268bd2"),
		theme.ColorNameButton:            hexColor("#eee8d5"),
		theme.ColorNameHover:             hexColorA("#657b83", 0x14),
		theme.ColorNameInputBackground:   hexColor("#eee8d5"),
		theme.ColorNameInputBorder:       hexColor("#ccc4a8"),
		theme.ColorNameOverlayBackground: hexColor("#fdf6e3"),
		theme.ColorNamePlaceHolder:       hexColorA("#657b83", 0x88),
		theme.ColorNamePressed:           hexColorA("#657b83", 0x22),
		theme.ColorNameScrollBar:         hexColorA("#657b83", 0x30),
		theme.ColorNameSeparator:         hexColor("#e8e1c8"),
		theme.ColorNameShadow:            hexColorA("#000000", 0x18),
		theme.ColorNameFocus:             hexColor("#268bd2"),
		theme.ColorNameDisabled:          hexColor("#93a1a1"),
		theme.ColorNameDisabledButton:    hexColor("#f4edd6"),
		theme.ColorNameHeaderBackground:  hexColor("#eee8d5"),
		theme.ColorNameMenuBackground:    hexColor("#eee8d5"),
		theme.ColorNameSelection:         hexColorA("#268bd2", 0x44),
		theme.ColorNameError:             hexColor("#dc322f"),
		theme.ColorNameSuccess:           hexColor("#859900"),
		theme.ColorNameWarning:           hexColor("#b58900"),
	}}
}

// githubLightTheme — clean white, minimal, familiar GitHub UI aesthetic.
func githubLightTheme() fyne.Theme {
	return &paletteTheme{colors: map[fyne.ThemeColorName]color.Color{
		theme.ColorNameBackground:        hexColor("#ffffff"),
		theme.ColorNameForeground:        hexColor("#24292f"),
		theme.ColorNamePrimary:           hexColor("#0969da"),
		theme.ColorNameButton:            hexColor("#f6f8fa"),
		theme.ColorNameHover:             hexColorA("#24292f", 0x0e),
		theme.ColorNameInputBackground:   hexColor("#f6f8fa"),
		theme.ColorNameInputBorder:       hexColor("#d0d7de"),
		theme.ColorNameOverlayBackground: hexColor("#ffffff"),
		theme.ColorNamePlaceHolder:       hexColorA("#24292f", 0x88),
		theme.ColorNamePressed:           hexColorA("#24292f", 0x1a),
		theme.ColorNameScrollBar:         hexColorA("#24292f", 0x28),
		theme.ColorNameSeparator:         hexColor("#d8dee4"),
		theme.ColorNameShadow:            hexColorA("#000000", 0x18),
		theme.ColorNameFocus:             hexColor("#0969da"),
		theme.ColorNameDisabled:          hexColor("#8c959f"),
		theme.ColorNameDisabledButton:    hexColor("#eaeef2"),
		theme.ColorNameHeaderBackground:  hexColor("#f6f8fa"),
		theme.ColorNameMenuBackground:    hexColor("#f6f8fa"),
		theme.ColorNameSelection:         hexColorA("#0969da", 0x44),
		theme.ColorNameError:             hexColor("#cf222e"),
		theme.ColorNameSuccess:           hexColor("#1a7f37"),
		theme.ColorNameWarning:           hexColor("#9a6700"),
	}}
}

// rosePineDawnTheme — warm pinkish-beige with rose and gold accents.
func rosePineDawnTheme() fyne.Theme {
	return &paletteTheme{colors: map[fyne.ThemeColorName]color.Color{
		theme.ColorNameBackground:        hexColor("#faf4ed"),
		theme.ColorNameForeground:        hexColor("#575279"),
		theme.ColorNamePrimary:           hexColor("#d7827e"),
		theme.ColorNameButton:            hexColor("#f2e9e1"),
		theme.ColorNameHover:             hexColorA("#575279", 0x14),
		theme.ColorNameInputBackground:   hexColor("#f2e9e1"),
		theme.ColorNameInputBorder:       hexColor("#dfd9d0"),
		theme.ColorNameOverlayBackground: hexColor("#faf4ed"),
		theme.ColorNamePlaceHolder:       hexColorA("#575279", 0x88),
		theme.ColorNamePressed:           hexColorA("#575279", 0x22),
		theme.ColorNameScrollBar:         hexColorA("#575279", 0x30),
		theme.ColorNameSeparator:         hexColor("#e4dfda"),
		theme.ColorNameShadow:            hexColorA("#000000", 0x18),
		theme.ColorNameFocus:             hexColor("#d7827e"),
		theme.ColorNameDisabled:          hexColor("#9893a5"),
		theme.ColorNameDisabledButton:    hexColor("#f4ede8"),
		theme.ColorNameHeaderBackground:  hexColor("#f2e9e1"),
		theme.ColorNameMenuBackground:    hexColor("#f2e9e1"),
		theme.ColorNameSelection:         hexColorA("#d7827e", 0x44),
		theme.ColorNameError:             hexColor("#b4637a"),
		theme.ColorNameSuccess:           hexColor("#56949f"),
		theme.ColorNameWarning:           hexColor("#ea9d34"),
	}}
}

// everforestLightTheme — warm nature greens, easy on the eyes.
func everforestLightTheme() fyne.Theme {
	return &paletteTheme{colors: map[fyne.ThemeColorName]color.Color{
		theme.ColorNameBackground:        hexColor("#fff9ef"),
		theme.ColorNameForeground:        hexColor("#5c6a72"),
		theme.ColorNamePrimary:           hexColor("#8da101"),
		theme.ColorNameButton:            hexColor("#f3ead3"),
		theme.ColorNameHover:             hexColorA("#5c6a72", 0x14),
		theme.ColorNameInputBackground:   hexColor("#f3ead3"),
		theme.ColorNameInputBorder:       hexColor("#d8cdb4"),
		theme.ColorNameOverlayBackground: hexColor("#fff9ef"),
		theme.ColorNamePlaceHolder:       hexColorA("#5c6a72", 0x88),
		theme.ColorNamePressed:           hexColorA("#5c6a72", 0x22),
		theme.ColorNameScrollBar:         hexColorA("#5c6a72", 0x30),
		theme.ColorNameSeparator:         hexColor("#e8e1cc"),
		theme.ColorNameShadow:            hexColorA("#000000", 0x18),
		theme.ColorNameFocus:             hexColor("#8da101"),
		theme.ColorNameDisabled:          hexColor("#a6b0a0"),
		theme.ColorNameDisabledButton:    hexColor("#f0e8d0"),
		theme.ColorNameHeaderBackground:  hexColor("#f3ead3"),
		theme.ColorNameMenuBackground:    hexColor("#f3ead3"),
		theme.ColorNameSelection:         hexColorA("#8da101", 0x44),
		theme.ColorNameError:             hexColor("#f85552"),
		theme.ColorNameSuccess:           hexColor("#8da101"),
		theme.ColorNameWarning:           hexColor("#dfa000"),
	}}
}

// ThemeForName returns the fyne.Theme for the given config key.
func ThemeForName(name string) fyne.Theme {
	switch name {
	case "light":
		return lightTheme()
	case "nord":
		return nordTheme()
	case "tokyo-night":
		return tokyoNightTheme()
	case "gruvbox":
		return gruvboxTheme()
	case "kanagawa":
		return kanagawaTheme()
	case "pure-black":
		return pureBlackTheme()
	case "solarized-light":
		return solarizedLightTheme()
	case "github-light":
		return githubLightTheme()
	case "rose-pine-dawn":
		return rosePineDawnTheme()
	case "everforest-light":
		return everforestLightTheme()
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
	{"Tokyo Night", "tokyo-night"},
	{"Gruvbox", "gruvbox"},
	{"Kanagawa", "kanagawa"},
	{"Pure Black", "pure-black"},
	{"Solarized Light", "solarized-light"},
	{"GitHub Light", "github-light"},
	{"Rosé Pine Dawn", "rose-pine-dawn"},
	{"Everforest Light", "everforest-light"},
}
