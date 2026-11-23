package shape

import "slices"

// preprocessThai decomposes each sara am of Thai and Lao into nikhahit
// and sara aa, the nikhahit moved before the above-base marks before
// it, as HarfBuzz's Thai shaper does whatever the font; Thai in a font
// whose GSUB lacks the script then takes the glyphs of the Private Use
// Area old fonts place its marks with.
func (b *buffer) preprocessThai() {
	for i := 0; i < len(b.glyphs); i++ {
		g := b.glyphs[i]
		if g.Char&^0x80 != 0x0E33 {
			continue
		}
		nikhahit, aa := g, g
		nikhahit.Char, aa.Char = g.Char-0x0E33+0x0E4D, g.Char-1
		nikhahit.ID, aa.ID = b.f.nominal(nikhahit.Char), b.f.nominal(aa.Char)
		nikhahit.cc, aa.cc = combiningClass(nikhahit.Char), combiningClass(aa.Char)
		b.glyphs = slices.Replace(b.glyphs, i, i+1, nikhahit, aa)
		end := i + 2
		start := i
		for start > 0 && thaiAboveBase(b.glyphs[start-1].Char) {
			start--
		}
		if start+2 < end {
			b.mergeClusters(start, end)
			b.move(end-2, start)
		}
		if start > 0 {
			b.mergeGraphemes(start-1, end)
		}
		i = end - 1
	}
	if b.plan.script == thai && !b.plan.found {
		b.thaiPUA()
	}
}

// thaiAboveBase reports whether r is a Thai or Lao mark above the base.
func thaiAboveBase(r rune) bool {
	r &^= 0x80
	return r >= 0x0E34 && r <= 0x0E37 || r >= 0x0E47 && r <= 0x0E4E || r == 0x0E31 || r == 0x0E3B
}

// mergeGraphemes merges the clusters from start to end and the marks on
// either side that belong to them, as HarfBuzz merges grapheme clusters.
func (b *buffer) mergeGraphemes(start, end int) {
	for start > 0 && isMarkChar(b.glyphs[start].Char) {
		start--
	}
	for end < len(b.glyphs) && isMarkChar(b.glyphs[end].Char) {
		end++
	}
	b.mergeClusters(start, end)
}

// How a mark or base of Thai is moved, as the Private Use Area has its
// glyphs: down, left, down and left, or a base without its descender.
const (
	thaiNop uint8 = iota
	thaiSD
	thaiSL
	thaiSDL
	thaiRD
)

// thaiPUAs are the characters of each move with their glyphs in the
// Private Use Area, Windows's then the Mac's.
//
//nolint:gochecknoglobals // HarfBuzz's tables
var thaiPUAs = [...][][3]rune{
	thaiSD: {
		{0x0E48, 0xF70A, 0xF88B}, {0x0E49, 0xF70B, 0xF88E}, {0x0E4A, 0xF70C, 0xF891}, {0x0E4B, 0xF70D, 0xF894},
		{0x0E4C, 0xF70E, 0xF897}, {0x0E38, 0xF718, 0xF89B}, {0x0E39, 0xF719, 0xF89C}, {0x0E3A, 0xF71A, 0xF89D},
	},
	thaiSDL: {
		{0x0E48, 0xF705, 0xF88C}, {0x0E49, 0xF706, 0xF88F}, {0x0E4A, 0xF707, 0xF892}, {0x0E4B, 0xF708, 0xF895},
		{0x0E4C, 0xF709, 0xF898},
	},
	thaiSL: {
		{0x0E48, 0xF713, 0xF88A}, {0x0E49, 0xF714, 0xF88D}, {0x0E4A, 0xF715, 0xF890}, {0x0E4B, 0xF716, 0xF893},
		{0x0E4C, 0xF717, 0xF896}, {0x0E31, 0xF710, 0xF884}, {0x0E34, 0xF701, 0xF885}, {0x0E35, 0xF702, 0xF886},
		{0x0E36, 0xF703, 0xF887}, {0x0E37, 0xF704, 0xF888}, {0x0E47, 0xF712, 0xF889}, {0x0E4D, 0xF711, 0xF899},
	},
	thaiRD: {{0x0E0D, 0xF70F, 0xF89A}, {0x0E10, 0xF700, 0xF89E}},
}

// Thai consonants by their shapes, and marks by where they sit.
const (
	thaiNC uint8 = iota
	thaiAC
	thaiRC
	thaiDC
	thaiNotConsonant
)

const (
	thaiAV uint8 = iota
	thaiBV
	thaiT
	thaiNotMark
)

func thaiConsonant(r rune) uint8 {
	switch {
	case r == 0x0E1B || r == 0x0E1D || r == 0x0E1F:
		return thaiAC
	case r == 0x0E0D || r == 0x0E10:
		return thaiRC
	case r == 0x0E0E || r == 0x0E0F:
		return thaiDC
	case r >= 0x0E01 && r <= 0x0E2E:
		return thaiNC
	}
	return thaiNotConsonant
}

func thaiMark(r rune) uint8 {
	switch {
	case r == 0x0E31 || r >= 0x0E34 && r <= 0x0E37 || r == 0x0E47 || r == 0x0E4D || r == 0x0E4E:
		return thaiAV
	case r >= 0x0E38 && r <= 0x0E3A:
		return thaiBV
	case r >= 0x0E48 && r <= 0x0E4C:
		return thaiT
	}
	return thaiNotMark
}

// thaiEdge is a step of HarfBuzz's machines of the marks above and below
// a Thai base: the move it makes and the state after it.
type thaiEdge struct{ action, next uint8 }

//nolint:gochecknoglobals // HarfBuzz's machines
var (
	thaiAboveStart = [...]uint8{0, 1, 0, 0, 3}
	thaiAbove      = [4][3]thaiEdge{
		{{thaiNop, 3}, {thaiNop, 0}, {thaiSD, 3}},
		{{thaiSL, 2}, {thaiNop, 1}, {thaiSDL, 2}},
		{{thaiNop, 3}, {thaiNop, 2}, {thaiSL, 3}},
		{{thaiNop, 3}, {thaiNop, 3}, {thaiNop, 3}},
	}
	thaiBelowStart = [...]uint8{0, 0, 1, 2, 2}
	thaiBelow      = [3][3]thaiEdge{
		{{thaiNop, 0}, {thaiNop, 2}, {thaiNop, 0}},
		{{thaiNop, 1}, {thaiRD, 2}, {thaiNop, 1}},
		{{thaiNop, 2}, {thaiSD, 2}, {thaiNop, 2}},
	}
)

// thaiPUA gives the marks, and the bases whose descender a mark below
// takes away, the glyphs of the Private Use Area the font has for them.
func (b *buffer) thaiPUA() {
	above, below := thaiAboveStart[thaiNotConsonant], thaiBelowStart[thaiNotConsonant]
	base := 0
	for i := range b.glyphs {
		mt := thaiMark(b.glyphs[i].Char)
		if mt == thaiNotMark {
			ct := thaiConsonant(b.glyphs[i].Char)
			above, below, base = thaiAboveStart[ct], thaiBelowStart[ct], i
			continue
		}
		ae, be := thaiAbove[above][mt], thaiBelow[below][mt]
		above, below = ae.next, be.next
		action := ae.action
		if action == thaiNop {
			action = be.action
		}
		at := i
		if action == thaiRD {
			at = base
		}
		if id := b.thaiGlyph(b.glyphs[at].Char, action); id != 0 {
			b.glyphs[at].ID = id
		}
	}
}

// thaiGlyph is the font's glyph in the Private Use Area of r moved by
// action, 0 for none.
func (b *buffer) thaiGlyph(r rune, action uint8) uint16 {
	if action == thaiNop {
		return 0
	}
	for _, m := range thaiPUAs[action] {
		if m[0] != r {
			continue
		}
		if id := b.f.nominal(m[1]); id != 0 {
			return id
		}
		return b.f.nominal(m[2])
	}
	return 0
}
