package sysprefs

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/platform"
)

// System is what the system says of its user's preferences at one time,
// what it doesn't say unknown.
type System struct {
	Appearance   platform.Appearance
	Motion       platform.Motion
	Contrast     platform.Contrast
	ForcedColors platform.ForcedColors
	Highlight    platform.Highlight
	Languages    []string
}

// Equal reports whether s and o say the same.
func (s System) Equal(o System) bool {
	return s.Appearance == o.Appearance && s.Motion == o.Motion && s.Contrast == o.Contrast &&
		s.ForcedColors == o.ForcedColors && s.Highlight == o.Highlight && slices.Equal(s.Languages, o.Languages)
}

// ForcedContrast is the contrast a system forcing its colours asks for,
// as Chromium words it: more where the forced text stands at least 7:1
// from its background, less at 2.5:1 or under, custom between.
func ForcedContrast(back, text color.Color) platform.Contrast {
	switch ratio := contrastRatio(back, text); {
	case ratio >= 7:
		return platform.ContrastMore
	case ratio <= 2.5:
		return platform.ContrastLess
	}
	return platform.ContrastCustom
}

// ComputedColor reads a colour as a browser's getComputedStyle gives it,
// rgb(r, g, b) or rgba(r, g, b, a), its channels 0 to 255.
func ComputedColor(s string) (color.Color, bool) {
	s = strings.TrimSpace(s)
	open, end := strings.IndexByte(s, '('), len(s)-1
	if open < 0 || end < open || s[end] != ')' {
		return color.Color{}, false
	}
	if name := s[:open]; name != "rgb" && name != "rgba" {
		return color.Color{}, false
	}
	fields := strings.FieldsFunc(s[open+1:end], func(r rune) bool { return r == ',' || r == '/' || r == ' ' })
	if len(fields) != 3 && len(fields) != 4 {
		return color.Color{}, false
	}
	var ch [3]uint8
	for i := range ch {
		v, err := strconv.ParseFloat(fields[i], 64)
		if err != nil || v < 0 || v > 255 {
			return color.Color{}, false
		}
		ch[i] = uint8(math.Round(v))
	}
	c := color.RGB8(ch[0], ch[1], ch[2])
	if len(fields) == 4 {
		a, err := strconv.ParseFloat(fields[3], 64)
		if err != nil || a < 0 || a > 1 {
			return color.Color{}, false
		}
		c.A = float32(a)
	}
	return c, true
}

// contrastRatio is WCAG's contrast ratio of two opaque colours.
func contrastRatio(a, b color.Color) float64 {
	la, lb := luminance(a), luminance(b)
	return (math.Max(la, lb) + 0.05) / (math.Min(la, lb) + 0.05)
}

// luminance is c's relative luminance: color.Color is linear already.
func luminance(c color.Color) float64 {
	return 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
}
