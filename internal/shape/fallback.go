package shape

import (
	"unicode"

	"golang.org/x/text/unicode/norm"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Unicode's combining classes the fallback places marks by.
const (
	ccAttachedBelowLeft  = 200
	ccAttachedBelow      = 202
	ccAttachedAbove      = 214
	ccAttachedAboveRight = 216
	ccBelowLeft          = 218
	ccBelow              = 220
	ccBelowRight         = 222
	ccLeft               = 224
	ccRight              = 226
	ccAboveLeft          = 228
	ccAbove              = 230
	ccAboveRight         = 232
	ccDoubleBelow        = 233
	ccDoubleAbove        = 234
)

// recategorize gives the nonspacing marks the classes the fallback
// places them by: Hebrew's, Arabic's, Thai's, Lao's and Tibetan's
// marks the side of their base they sit on.
func (b *buffer) recategorize() {
	for i := range b.glyphs {
		g := &b.glyphs[i]
		if g.Char != 0 && unicode.Is(unicode.Mn, g.Char) {
			g.cc = fallbackClass(g.Char, g.cc)
		}
	}
}

func fallbackClass(r rune, c uint8) uint8 {
	if c >= 200 {
		return c
	}
	orig := norm.NFD.PropertiesString(string(r)).CCC()
	if r&^0xFF == 0x0E00 {
		switch {
		case orig == 0:
			switch r {
			case 0x0E31, 0x0E34, 0x0E35, 0x0E36, 0x0E37, 0x0E47, 0x0E4C, 0x0E4D, 0x0E4E:
				return ccAboveRight
			case 0x0EB1, 0x0EB4, 0x0EB5, 0x0EB6, 0x0EB7, 0x0EBB, 0x0ECC, 0x0ECD:
				return ccAbove
			case 0x0EBC:
				return ccBelow
			}
		case r == 0x0E3A:
			return ccBelowRight
		}
	}
	switch orig {
	case 10, 11, 12, 13, 14, 15, 16, 17, 18, 20, 22, 29, 32, 118, 129, 132:
		return ccBelow
	case 23:
		return ccAttachedAbove
	case 24, 107:
		return ccAboveRight
	case 25, 19:
		return ccAboveLeft
	case 26, 27, 28, 30, 31, 33, 34, 35, 36, 122, 130:
		return ccAbove
	case 103:
		return ccBelowRight
	}
	return c
}

// extents is a glyph's box, HarfBuzz's way: from its left and top,
// height going down (negative), y up.
type extents struct{ x, y, width, height int32 }

func (f *Font) extents(id uint16) (extents, bool) {
	r, _, err := f.src.GlyphBounds(&f.buf, sfnt.GlyphIndex(id), fixed.I(int(f.upem)), font.HintingNone)
	if err != nil {
		return extents{}, false
	}
	return extents{int32(r.Min.X >> 6), int32(-r.Min.Y >> 6), int32((r.Max.X - r.Min.X) >> 6),
		int32((r.Min.Y - r.Max.Y) >> 6)}, true
}

// placeMarks places each mark on its base by its combining class, as
// HarfBuzz's fallback does for a font with no GPOS to place them.
func (b *buffer) placeMarks(adjust bool) {
	start := 0
	for i := 1; i < len(b.glyphs); i++ {
		g := &b.glyphs[i]
		if !isMarkChar(g.Char) && !hidden(g.Char) && !b.ignorable(g) {
			b.placeCluster(start, i, adjust)
			start = i
		}
	}
	b.placeCluster(start, len(b.glyphs), adjust)
}

func (b *buffer) placeCluster(start, end int, adjust bool) {
	if end-start < 2 {
		return
	}
	for i := start; i < end; i++ {
		if isMarkChar(b.glyphs[i].Char) {
			continue
		}
		j := i + 1
		for ; j < end; j++ {
			g := &b.glyphs[j]
			if hidden(g.Char) || b.ignorable(g) {
				continue
			}
			if !isMarkChar(g.Char) {
				break
			}
		}
		b.placeAround(i, j, adjust)
		i = j - 1
	}
}

// placeAround places the marks from base+1 to end around base.
//
//nolint:nestif // HarfBuzz's steps
func (b *buffer) placeAround(base, end int, adjust bool) {
	g := b.glyphs
	baseBox, ok := b.f.extents(g[base].ID)
	if !ok {
		for i := base + 1; i < end; i++ {
			if g[i].Char != 0 && unicode.Is(unicode.Mn, g[i].Char) {
				if adjust {
					g[i].XOffset -= g[i].Advance
				}
				g[i].Advance = 0
			}
		}
		return
	}
	baseBox.y += g[base].YOffset
	baseBox.x = g[base].XOffset
	if a, err := b.f.src.GlyphAdvance(&b.f.buf, sfnt.GlyphIndex(g[base].ID), fixed.I(int(b.f.upem)),
		font.HintingNone); err == nil {
		baseBox.width = int32(a >> 6)
	}
	comps := int32(1)
	if g[base].ligated {
		comps = int32(g[base].comp)
	}
	var x int32
	if !b.rtl {
		x = -g[base].Advance
	}
	component, cluster := baseBox, baseBox
	lastComp, lastClass := int32(-1), uint8(255)
	for i := base + 1; i < end; i++ {
		if g[i].cc == 0 {
			if !b.rtl {
				x -= g[i].Advance
			} else {
				x += g[i].Advance
			}
			continue
		}
		if comps > 1 {
			this := int32(compOf(g[i])) - 1
			if g[base].lig == 0 || g[base].lig != g[i].lig || this >= comps || this < 0 {
				this = comps - 1
			}
			if lastComp != this {
				lastComp, lastClass = this, 255
				component = baseBox
				if !b.rtl {
					component.x += this * component.width / comps
				} else {
					component.x += (comps - 1 - this) * component.width / comps
				}
				component.width /= comps
			}
		}
		if lastClass != g[i].cc {
			lastClass = g[i].cc
			cluster = component
		}
		b.placeMark(&cluster, i)
		g[i].Advance = 0
		g[i].XOffset += x
	}
}

// placeMark places mark i by its class on box, which grows by it.
func (b *buffer) placeMark(box *extents, i int) {
	g := &b.glyphs[i]
	mark, ok := b.f.extents(g.ID)
	if !ok {
		return
	}
	gap := b.f.upem / 16
	g.XOffset, g.YOffset = 0, 0
	c := g.cc
	switch c {
	case ccDoubleBelow, ccDoubleAbove:
		if !b.rtl {
			g.XOffset = box.x + box.width - mark.width/2 - mark.x
		} else {
			g.XOffset = box.x - mark.width/2 - mark.x
		}
	case ccAttachedBelowLeft, ccBelowLeft, ccAboveLeft:
		g.XOffset = box.x - mark.x
	case ccAttachedAboveRight, ccBelowRight, ccAboveRight:
		g.XOffset = box.x + box.width - mark.width - mark.x
	default:
		g.XOffset = box.x + (box.width-mark.width)/2 - mark.x
	}
	switch c {
	case ccDoubleBelow, ccBelowLeft, ccBelow, ccBelowRight, ccAttachedBelowLeft, ccAttachedBelow:
		if c != ccAttachedBelowLeft && c != ccAttachedBelow {
			box.height -= gap
		}
		g.YOffset = box.y + box.height - mark.y
		if (gap > 0) == (g.YOffset > 0) {
			box.height -= g.YOffset
			g.YOffset = 0
		}
		box.height += mark.height
	case ccDoubleAbove, ccAboveLeft, ccAbove, ccAboveRight, ccAttachedAbove, ccAttachedAboveRight:
		if c != ccAttachedAbove && c != ccAttachedAboveRight {
			box.y += gap
			box.height -= gap
		}
		g.YOffset = box.y - mark.y - mark.height
		if (gap > 0) != (g.YOffset > 0) {
			correction := -g.YOffset / 2
			box.y += correction
			box.height -= correction
			g.YOffset += correction
		}
		box.y -= mark.height
		box.height += mark.height
	}
}
