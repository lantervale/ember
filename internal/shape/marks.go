package shape

import (
	"slices"

	"golang.org/x/text/unicode/norm"
)

// combiningClass is r's canonical combining class as HarfBuzz sorts
// marks by it: Arabic's shadda before the other harakat, and Hebrew's
// points in the order fonts expect them.
func combiningClass(r rune) uint8 {
	if r == 0 {
		return 0
	}
	c := norm.NFD.PropertiesString(string(r)).CCC()
	if c >= 10 && c <= 35 {
		return modifiedClasses[c-10]
	}
	return c
}

// modifiedClasses are HarfBuzz's classes for the canonical ones from 10
// to 35, Hebrew's then Arabic's.
//
//nolint:gochecknoglobals // a fixed table
var modifiedClasses = [...]uint8{
	22, 15, 16, 17, 23, 18, 19, 20, 21, 14, 24, 12, 25, 25, 10, 11, 26,
	28, 29, 30, 31, 32, 33, 27, 34, 35,
}

// reorderMarks sorts each sequence of marks after a character by their
// combining classes, as HarfBuzz's normalisation does before shaping.
func (b *buffer) reorderMarks() {
	for i := 0; i < len(b.glyphs); {
		if combiningClass(b.glyphs[i].Char) == 0 {
			i++
			continue
		}
		j := i + 1
		for j < len(b.glyphs) && combiningClass(b.glyphs[j].Char) != 0 {
			j++
		}
		if j-i > 1 {
			slices.SortStableFunc(b.glyphs[i:j], func(x, y Glyph) int {
				return int(combiningClass(x.Char)) - int(combiningClass(y.Char))
			})
		}
		i = j
	}
}
