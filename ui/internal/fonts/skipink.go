package fonts

import "unicode"

// SkipsInk reports whether text-decoration-skip-ink skips the ink of r's
// glyph, as Chrome: auto not for slashes, the low line, CJK ideographs and
// symbols, Hangul and Linear B ideograms (Character::CanTextDecorationSkipInk),
// all for every glyph but the low line's, which an underline runs into.
func SkipsInk(r rune, all bool) bool {
	switch {
	case r == '_':
		return false
	case all:
		return true
	case r == '/' || r == '\\':
		return false
	}
	return !unicode.Is(inkKept, r)
}

// inkKept are the characters whose ink auto leaves whole: Chrome's
// IsCJKIdeographOrSymbol and the Hangul and Linear B blocks, as Chrome on
// macOS draws them, every character of these blocks compared under auto
// and under all; the CJK blocks and the pictographs from U+1F200 are
// whole, though some of their glyphs cross no underline there.
//
//nolint:gochecknoglobals // a fixed table
var inkKept = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 0x1100, Hi: 0x11ff, Stride: 1},
		{Lo: 0x2020, Hi: 0x2021, Stride: 1},
		{Lo: 0x2030, Hi: 0x2030, Stride: 1},
		{Lo: 0x203b, Hi: 0x203c, Stride: 1},
		{Lo: 0x2042, Hi: 0x2042, Stride: 1},
		{Lo: 0x2047, Hi: 0x2049, Stride: 1},
		{Lo: 0x2051, Hi: 0x2051, Stride: 1},
		{Lo: 0x20de, Hi: 0x20de, Stride: 1},
		{Lo: 0x2100, Hi: 0x2100, Stride: 1},
		{Lo: 0x2103, Hi: 0x2103, Stride: 1},
		{Lo: 0x2105, Hi: 0x2105, Stride: 1},
		{Lo: 0x2109, Hi: 0x210a, Stride: 1},
		{Lo: 0x2113, Hi: 0x2113, Stride: 1},
		{Lo: 0x2116, Hi: 0x2116, Stride: 1},
		{Lo: 0x2121, Hi: 0x2121, Stride: 1},
		{Lo: 0x212b, Hi: 0x212b, Stride: 1},
		{Lo: 0x213b, Hi: 0x213b, Stride: 1},
		{Lo: 0x2150, Hi: 0x2152, Stride: 1},
		{Lo: 0x2156, Hi: 0x215a, Stride: 1},
		{Lo: 0x2160, Hi: 0x216b, Stride: 1},
		{Lo: 0x2170, Hi: 0x217b, Stride: 1},
		{Lo: 0x217f, Hi: 0x217f, Stride: 1},
		{Lo: 0x2189, Hi: 0x2189, Stride: 1},
		{Lo: 0x2194, Hi: 0x2195, Stride: 1},
		{Lo: 0x2307, Hi: 0x2307, Stride: 1},
		{Lo: 0x2312, Hi: 0x2312, Stride: 1},
		{Lo: 0x23be, Hi: 0x23cc, Stride: 1},
		{Lo: 0x23ce, Hi: 0x23ce, Stride: 1},
		{Lo: 0x2460, Hi: 0x2492, Stride: 1},
		{Lo: 0x249c, Hi: 0x24ff, Stride: 1},
		{Lo: 0x25a0, Hi: 0x25a2, Stride: 1},
		{Lo: 0x25aa, Hi: 0x25ab, Stride: 1},
		{Lo: 0x25b1, Hi: 0x25b3, Stride: 1},
		{Lo: 0x25b6, Hi: 0x25b7, Stride: 1},
		{Lo: 0x25bc, Hi: 0x25bd, Stride: 1},
		{Lo: 0x25c0, Hi: 0x25c1, Stride: 1},
		{Lo: 0x25c6, Hi: 0x25c7, Stride: 1},
		{Lo: 0x25c9, Hi: 0x25c9, Stride: 1},
		{Lo: 0x25cb, Hi: 0x25cc, Stride: 1},
		{Lo: 0x25ce, Hi: 0x25d3, Stride: 1},
		{Lo: 0x25e2, Hi: 0x25e6, Stride: 1},
		{Lo: 0x25ef, Hi: 0x25ef, Stride: 1},
		{Lo: 0x2600, Hi: 0x2603, Stride: 1},
		{Lo: 0x2605, Hi: 0x2606, Stride: 1},
		{Lo: 0x260e, Hi: 0x260e, Stride: 1},
		{Lo: 0x2616, Hi: 0x2617, Stride: 1},
		{Lo: 0x2620, Hi: 0x2620, Stride: 1},
		{Lo: 0x2640, Hi: 0x2640, Stride: 1},
		{Lo: 0x2642, Hi: 0x2642, Stride: 1},
		{Lo: 0x2660, Hi: 0x266f, Stride: 1},
		{Lo: 0x2672, Hi: 0x267d, Stride: 1},
		{Lo: 0x2695, Hi: 0x2696, Stride: 1},
		{Lo: 0x26a0, Hi: 0x26a0, Stride: 1},
		{Lo: 0x26a7, Hi: 0x26a7, Stride: 1},
		{Lo: 0x26d3, Hi: 0x26d3, Stride: 1},
		{Lo: 0x2708, Hi: 0x2708, Stride: 1},
		{Lo: 0x2713, Hi: 0x2713, Stride: 1},
		{Lo: 0x271a, Hi: 0x271a, Stride: 1},
		{Lo: 0x273f, Hi: 0x2740, Stride: 1},
		{Lo: 0x2744, Hi: 0x2744, Stride: 1},
		{Lo: 0x2756, Hi: 0x2756, Stride: 1},
		{Lo: 0x2763, Hi: 0x2764, Stride: 1},
		{Lo: 0x2776, Hi: 0x277f, Stride: 1},
		{Lo: 0x27a1, Hi: 0x27a1, Stride: 1},
		{Lo: 0x2b1a, Hi: 0x2b1a, Stride: 1},
		{Lo: 0x2e80, Hi: 0x4dbf, Stride: 1},
		{Lo: 0x4e00, Hi: 0x9fff, Stride: 1},
		{Lo: 0xa960, Hi: 0xa97f, Stride: 1},
		{Lo: 0xac00, Hi: 0xd7ff, Stride: 1},
		{Lo: 0xf860, Hi: 0xf862, Stride: 1},
		{Lo: 0xf900, Hi: 0xfaff, Stride: 1},
		{Lo: 0xfe19, Hi: 0xfe19, Stride: 1},
		{Lo: 0xfe30, Hi: 0xfe6f, Stride: 1},
		{Lo: 0xff00, Hi: 0xff0c, Stride: 1},
		{Lo: 0xff0e, Hi: 0xff1a, Stride: 1},
		{Lo: 0xff1d, Hi: 0xff1d, Stride: 1},
		{Lo: 0xff1f, Hi: 0xffef, Stride: 1},
	},
	R32: []unicode.Range32{
		{Lo: 0x10080, Hi: 0x100ff, Stride: 1},
		{Lo: 0x1f100, Hi: 0x1f100, Stride: 1},
		{Lo: 0x1f110, Hi: 0x1f129, Stride: 1},
		{Lo: 0x1f130, Hi: 0x1f149, Stride: 1},
		{Lo: 0x1f150, Hi: 0x1f169, Stride: 1},
		{Lo: 0x1f170, Hi: 0x1f189, Stride: 1},
		{Lo: 0x1f200, Hi: 0x1f6ff, Stride: 1},
		{Lo: 0x1f900, Hi: 0x1f90b, Stride: 1},
		{Lo: 0x1f93b, Hi: 0x1f93b, Stride: 1},
		{Lo: 0x1f946, Hi: 0x1f946, Stride: 1},
		{Lo: 0x20000, Hi: 0x2fa1f, Stride: 1},
		{Lo: 0x30000, Hi: 0x3134f, Stride: 1},
	},
}
