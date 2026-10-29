package contrast

import (
	"image/color"
	"math"
	"testing"
)

func mustHex(t *testing.T, s string) color.RGBA {
	t.Helper()
	c, err := ParseHex(s)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TC-073a
func TestRatio(t *testing.T) {
	tests := []struct {
		fg, bg string
		want   float64
	}{
		{"#000000", "#FFFFFF", 21},
		{"#FFFFFF", "#000000", 21}, // symmetric
		{"#FFFFFF", "#FFFFFF", 1},
		{"#FFFF00", "#FFFFFF", 1.07},
		{"#90EE90", "#FFFFFF", 1.42},
		{"#767676", "#FFFFFF", 4.54},
		{"#757575", "#000000", 4.56},
	}
	for _, tt := range tests {
		got := Ratio(mustHex(t, tt.fg), mustHex(t, tt.bg))
		if math.Abs(got-tt.want) > 0.01 {
			t.Errorf("Ratio(%s, %s) = %.3f, want %.2f", tt.fg, tt.bg, got, tt.want)
		}
	}
}

// TC-074a: the examples from FR-074 at the default minimum.
func TestMeetsDefault(t *testing.T) {
	white := White
	for hex, want := range map[string]bool{
		"#FFFF00": false, // yellow
		"#90EE90": false, // light green
		"#767676": true,
		"#777777": false,
		"#000000": true,
	} {
		if got := Meets(mustHex(t, hex), white, DefaultMin); got != want {
			t.Errorf("Meets(%s on white) = %v, want %v", hex, got, want)
		}
	}
}

// TC-074b
func TestClampMin(t *testing.T) {
	for in, want := range map[float64]float64{1: 3, 3: 3, 4.5: 4.5, 7: 7, 30: 21} {
		if got := ClampMin(in); got != want {
			t.Errorf("ClampMin(%v) = %v, want %v", in, got, want)
		}
	}
}

// TC-075a
func TestBestText(t *testing.T) {
	for hex, want := range map[string]color.RGBA{
		"#FFFFFF": Black,
		"#FFFF00": Black,
		"#000000": White,
		"#00008B": White, // dark blue
	} {
		if got := BestText(mustHex(t, hex)); got != want {
			t.Errorf("BestText(%s) = %s, want %s", hex, Hex(got), Hex(want))
		}
	}
}

// TC-076
func TestEffective(t *testing.T) {
	yellow, navy := mustHex(t, "#FFFF00"), mustHex(t, "#000080")
	if got := Effective(yellow, White, DefaultMin); got != Black {
		t.Errorf("yellow on white -> %s, want black", Hex(got))
	}
	if got := Effective(navy, White, DefaultMin); got != navy {
		t.Errorf("navy on white -> %s, want navy kept", Hex(got))
	}
	// Raising the minimum can turn a passing colour into a fallback.
	gray := mustHex(t, "#767676")
	if got := Effective(gray, White, 7); got != Black {
		t.Errorf("gray on white at 7:1 -> %s, want black", Hex(got))
	}
}

// TC-071a, TC-071b
func TestGray(t *testing.T) {
	g, ok := Gray(White, DefaultMin)
	if !ok || Hex(g) != "#767676" {
		t.Errorf("Gray(white) = %s, %v; want #767676", Hex(g), ok)
	}
	g, ok = Gray(Black, DefaultMin)
	if !ok || Hex(g) != "#757575" {
		t.Errorf("Gray(black) = %s, %v; want #757575", Hex(g), ok)
	}
	for _, bg := range []string{"#FEFEFE", "#808080", "#FFFFE0"} {
		if _, ok := Gray(mustHex(t, bg), DefaultMin); ok {
			t.Errorf("Gray(%s) allowed; only pure black/white backgrounds may use gray", bg)
		}
	}
	// Every gray returned meets the minimum, for any allowed minimum.
	for min := FloorMin; min <= 20; min += 0.5 {
		for _, bg := range []color.RGBA{White, Black} {
			if g, ok := Gray(bg, min); ok && !Meets(g, bg, min) {
				t.Errorf("Gray(%s, %v) = %s fails the minimum", Hex(bg), min, Hex(g))
			}
		}
	}
}

func TestHexRoundTrip(t *testing.T) {
	for _, s := range []string{"#000000", "#FFFFFF", "#1A2B3C"} {
		if got := Hex(mustHex(t, s)); got != s {
			t.Errorf("Hex(ParseHex(%s)) = %s", s, got)
		}
	}
	if c := mustHex(t, "abcdef"); Hex(c) != "#ABCDEF" {
		t.Errorf("lower case without #: %s", Hex(c))
	}
	for _, bad := range []string{"", "#FFF", "#GGGGGG", "#1234567"} {
		if _, err := ParseHex(bad); err == nil {
			t.Errorf("ParseHex(%q) accepted", bad)
		}
	}
}
