// Package contrast checks text/background colour combinations against a
// minimum WCAG 2 contrast ratio (FR-071–FR-077).
package contrast

import (
	"fmt"
	"image/color"
	"math"
	"strings"
)

const (
	// DefaultMin is the default minimum contrast ratio (FR-074).
	DefaultMin = 4.5
	// FloorMin is the lowest minimum the user may configure (FR-074).
	FloorMin = 3.0
	// CeilingMin is the highest minimum the user may configure (FR-074);
	// above it coloured backgrounds cannot pass with black or white text.
	CeilingMin = 7.0
	// MaxRatio is the contrast of black on white.
	MaxRatio = 21.0
)

var (
	Black = color.RGBA{0, 0, 0, 0xff}
	White = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

// ClampMin limits a configured minimum ratio to [FloorMin, CeilingMin].
func ClampMin(min float64) float64 {
	return math.Max(FloorMin, math.Min(min, CeilingMin))
}

// Luminance returns the WCAG relative luminance of c, from 0 (black) to 1 (white).
// Alpha is ignored.
func Luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
}

func linear(v uint32) float64 {
	s := float64(v) / 0xffff
	if s <= 0.04045 {
		return s / 12.92
	}
	return math.Pow((s+0.055)/1.055, 2.4)
}

// Ratio returns the WCAG contrast ratio between two colours, from 1 to 21.
func Ratio(a, b color.Color) float64 {
	la, lb := Luminance(a), Luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// Meets reports whether fg on bg reaches the minimum ratio.
func Meets(fg, bg color.Color, min float64) bool {
	return Ratio(fg, bg) >= min
}

// BestText returns black or white, whichever contrasts more with bg (FR-075).
func BestText(bg color.Color) color.RGBA {
	if Ratio(Black, bg) >= Ratio(White, bg) {
		return Black
	}
	return White
}

// Effective returns fg if it meets the minimum on bg, otherwise BestText(bg) (FR-076).
func Effective(fg, bg color.Color, min float64) color.Color {
	if Meets(fg, bg, min) {
		return fg
	}
	return BestText(bg)
}

// IsPureBlackOrWhite reports whether bg is exactly #000000 or #FFFFFF.
func IsPureBlackOrWhite(bg color.Color) bool {
	r, g, b, _ := bg.RGBA()
	return (r == 0 && g == 0 && b == 0) || (r == 0xffff && g == 0xffff && b == 0xffff)
}

// Gray returns the gray closest to bg that still meets the minimum, for
// "greyed out" text (FR-071): the lightest passing gray on white, the
// darkest on black. It returns false if bg is not pure black or white, or
// if no gray other than black/white meets the minimum.
func Gray(bg color.Color, min float64) (color.RGBA, bool) {
	if !IsPureBlackOrWhite(bg) {
		return color.RGBA{}, false
	}
	onWhite := Luminance(bg) > 0.5
	for i := 1; i < 0xff; i++ {
		v := uint8(i) // on black: step lighter from black
		if onWhite {
			v = uint8(0xff - i) // on white: step darker from white
		}
		g := color.RGBA{v, v, v, 0xff}
		if Meets(g, bg, min) {
			return g, true
		}
	}
	return color.RGBA{}, false
}

// ParseHex parses "#RRGGBB" (case-insensitive, '#' optional).
func ParseHex(s string) (color.RGBA, error) {
	var c color.RGBA
	h := strings.TrimPrefix(s, "#")
	if len(h) != 6 {
		return c, fmt.Errorf("colour %q: want #RRGGBB", s)
	}
	if _, err := fmt.Sscanf(h, "%02x%02x%02x", &c.R, &c.G, &c.B); err != nil {
		return c, fmt.Errorf("colour %q: %w", s, err)
	}
	c.A = 0xff
	return c, nil
}

// Hex formats c as "#RRGGBB".
func Hex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
}
