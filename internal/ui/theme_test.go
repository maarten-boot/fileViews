package ui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"github.com/maarten-boot/fileViews/internal/contrast"
)

// over blends a translucent colour over an opaque background.
func over(top, bg color.Color) color.Color {
	tr, tg, tb, ta := top.RGBA() // premultiplied
	br, bg2, bb, _ := bg.RGBA()
	mix := func(t, b uint32) uint8 { return uint8((t + b*(0xffff-ta)/0xffff) >> 8) }
	return color.RGBA{mix(tr, br), mix(tg, bg2), mix(tb, bb), 0xff}
}

// textPairs lists every text colour and the backgrounds it is drawn on.
var textPairs = []struct {
	text fyne.ThemeColorName
	bgs  []fyne.ThemeColorName
}{
	{theme.ColorNameForeground, []fyne.ThemeColorName{
		theme.ColorNameBackground, theme.ColorNameButton, theme.ColorNameInputBackground,
		theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground, theme.ColorNameHeaderBackground,
	}},
	{theme.ColorNameDisabled, []fyne.ThemeColorName{
		theme.ColorNameBackground, theme.ColorNameDisabledButton, theme.ColorNameInputBackground,
		theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground,
	}},
	{theme.ColorNamePlaceHolder, []fyne.ThemeColorName{theme.ColorNameInputBackground}},
	{theme.ColorNameHyperlink, []fyne.ThemeColorName{theme.ColorNameBackground}},
	{theme.ColorNameError, []fyne.ThemeColorName{theme.ColorNameBackground}},
	{theme.ColorNameForegroundOnPrimary, []fyne.ThemeColorName{theme.ColorNamePrimary}},
	{theme.ColorNameForegroundOnError, []fyne.ThemeColorName{theme.ColorNameError}},
	{theme.ColorNameForegroundOnSuccess, []fyne.ThemeColorName{theme.ColorNameSuccess}},
	{theme.ColorNameForegroundOnWarning, []fyne.ThemeColorName{theme.ColorNameWarning}},
}

// TC-070a, TC-071c, TC-077: all theme text colours pass the minimum, for the
// default and a stricter minimum, and gray text only sits on pure white.
func TestThemeContrast(t *testing.T) {
	test.NewTempApp(t)
	for _, min := range []float64{contrast.FloorMin, contrast.DefaultMin, 7} {
		th := newLightTheme(min)
		c := func(n fyne.ThemeColorName) color.Color { return th.Color(n, theme.VariantDark) }
		for _, p := range textPairs {
			for _, bg := range p.bgs {
				if r := contrast.Ratio(c(p.text), c(bg)); r < min {
					t.Errorf("min %v: %s on %s = %.2f:1", min, p.text, bg, r)
				}
			}
		}
		// Black text on hover, pressed and selection highlights over white.
		for _, hl := range []fyne.ThemeColorName{theme.ColorNameHover, theme.ColorNamePressed, theme.ColorNameSelection} {
			bg := over(c(hl), c(theme.ColorNameBackground))
			if r := contrast.Ratio(c(theme.ColorNameForeground), bg); r < min {
				t.Errorf("min %v: foreground on %s = %.2f:1", min, hl, r)
			}
		}
		for _, p := range textPairs[1:3] { // disabled and placeholder are gray
			for _, bg := range p.bgs {
				if !contrast.IsPureBlackOrWhite(c(bg)) {
					t.Errorf("gray %s drawn on non-white %s", p.text, bg)
				}
			}
		}
	}
}

// TC-070a, TC-070b: light colours whatever variant the desktop asks for.
func TestThemeIgnoresDarkMode(t *testing.T) {
	test.NewTempApp(t)
	th := newLightTheme(contrast.DefaultMin)
	for _, v := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {
		if th.Color(theme.ColorNameBackground, v) != contrast.White || th.Color(theme.ColorNameForeground, v) != contrast.Black {
			t.Errorf("variant %v: not white/black", v)
		}
	}
	// Colours we do not override come from the light theme, not the dark one.
	if got, want := th.Color(theme.ColorNameShadow, theme.VariantDark), theme.LightTheme().Color(theme.ColorNameShadow, theme.VariantLight); got != want {
		t.Errorf("shadow = %v, want light %v", got, want)
	}
}

// TC-077
func TestThemeGrayFollowsMinimum(t *testing.T) {
	test.NewTempApp(t)
	if got := contrast.Hex(newLightTheme(4.5).Color(theme.ColorNameDisabled, theme.VariantLight)); got != "#767676" {
		t.Errorf("disabled at 4.5 = %s", got)
	}
	if got := contrast.Hex(newLightTheme(7).Color(theme.ColorNameDisabled, theme.VariantLight)); got != "#595959" {
		t.Errorf("disabled at 7 = %s", got)
	}
}
