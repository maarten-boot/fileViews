package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/maarten-boot/fileViews/internal/contrast"
)

// lightTheme is always light, whatever the desktop prefers (FR-070), and
// every text colour meets the contrast minimum on the background it is
// drawn on (FR-077). Gray text (disabled, placeholder) is only ever drawn on
// pure white (FR-071), so every background that can carry it is white.
type lightTheme struct {
	base fyne.Theme
	gray color.Color
}

func newLightTheme(min float64) *lightTheme {
	gray, ok := contrast.Gray(contrast.White, min)
	if !ok {
		gray = contrast.Black
	}
	return &lightTheme{base: theme.LightTheme(), gray: gray}
}

var (
	colorButton     = color.RGBA{0xf0, 0xf0, 0xf0, 0xff} // black text only
	colorHeader     = color.RGBA{0xf0, 0xf0, 0xf0, 0xff} // black text only
	colorBorder     = color.RGBA{0x76, 0x76, 0x76, 0xff}
	colorSeparator  = color.RGBA{0xc8, 0xc8, 0xc8, 0xff}
	colorScrollBack = color.RGBA{0xe8, 0xe8, 0xe8, 0xff}
	colorPrimary    = color.RGBA{0x00, 0x4a, 0xb8, 0xff} // white on it: 7.8:1
	colorError      = color.RGBA{0xa8, 0x00, 0x1c, 0xff} // white on it: 7.8:1
	colorSuccess    = color.RGBA{0x1b, 0x5e, 0x20, 0xff} // white on it: 7.9:1
	colorWarning    = color.RGBA{0x85, 0x47, 0x00, 0xff} // white on it: 7.0:1
	colorFocus      = color.NRGBA{0x00, 0x4a, 0xb8, 0x40}
	colorSelection  = color.NRGBA{0x00, 0x6c, 0xff, 0x40}
)

func (t *lightTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNameBackground, theme.ColorNameInputBackground, theme.ColorNameMenuBackground,
		theme.ColorNameOverlayBackground, theme.ColorNameDisabledButton,
		theme.ColorNameInnerWindowBorder, theme.ColorNameInnerWindowBorderInactive:
		return contrast.White
	case theme.ColorNameForeground:
		return contrast.Black
	case theme.ColorNameDisabled, theme.ColorNamePlaceHolder:
		return t.gray
	case theme.ColorNameButton:
		return colorButton
	case theme.ColorNameHeaderBackground:
		return colorHeader
	case theme.ColorNameInputBorder:
		return colorBorder
	case theme.ColorNameSeparator:
		return colorSeparator
	case theme.ColorNameScrollBarBackground:
		return colorScrollBack
	case theme.ColorNamePrimary, theme.ColorNameHyperlink:
		return colorPrimary
	case theme.ColorNameError:
		return colorError
	case theme.ColorNameSuccess:
		return colorSuccess
	case theme.ColorNameWarning:
		return colorWarning
	case theme.ColorNameForegroundOnPrimary, theme.ColorNameForegroundOnError,
		theme.ColorNameForegroundOnSuccess, theme.ColorNameForegroundOnWarning:
		return contrast.White
	case theme.ColorNameFocus:
		return colorFocus
	case theme.ColorNameSelection:
		return colorSelection
	}
	return t.base.Color(n, theme.VariantLight)
}

func (t *lightTheme) Font(s fyne.TextStyle) fyne.Resource     { return t.base.Font(s) }
func (t *lightTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return t.base.Icon(n) }
func (t *lightTheme) Size(n fyne.ThemeSizeName) float32       { return t.base.Size(n) }
