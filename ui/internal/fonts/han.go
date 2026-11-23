package fonts

import (
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/lantervale/ember/internal/shape"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// hanGlyph is box.HanGlyph in the face's units.
type hanGlyph struct {
	halt          bool
	shift, narrow float32
	ink           box.Ink
	full          bool
}

// HanGlyph is what text-spacing-trim reads of r's glyph in the face
// drawing it in f, at f's size: nothing where no face of f draws it.
func (s *Set) HanGlyph(f style.Font, r rune) box.HanGlyph {
	c, g := s.drawing(f, r)
	if c == nil || c.upem <= 0 {
		return box.HanGlyph{}
	}
	u, ok := c.han[r]
	if !ok {
		u = s.hanOf(c, g, r)
		if c.han == nil {
			c.han = map[rune]hanGlyph{}
		}
		c.han[r] = u
	}
	scale := f.Size * c.Look.adjust() / c.upem
	return box.HanGlyph{Halt: u.halt, Shift: u.shift * scale, Narrow: u.narrow * scale, Ink: u.ink, Full: u.full}
}

// drawing is the face drawing r in f and its glyph there, nil for none.
func (s *Set) drawing(f style.Font, r rune) (*Face, sfnt.GlyphIndex) {
	if ch := s.chained(f); ch != nil {
		c, g, _ := ch.glyph(s, r)
		return c, g
	}
	c := s.Face(f)
	if c == nil || c.font == nil {
		return nil, 0
	}
	g, err := c.font.GlyphIndex(&s.buf, r)
	if err != nil || g == 0 {
		return nil, 0
	}
	return c, g
}

// hanOf reads glyph g of c, r's: what halt does to it, as Chrome shapes
// the punctuation of Chinese and Japanese, and where its ink stands.
func (s *Set) hanOf(c *Face, g sfnt.GlyphIndex, r rune) hanGlyph {
	tagHalt, tagHani, tagKana := shape.MakeTag("halt"), shape.MakeTag("hani"), shape.MakeTag("kana")
	var u hanGlyph
	l := c.layout()
	u.halt = l.Has(tagHani, tagHalt) || l.Has(tagKana, tagHalt)
	if u.halt {
		run := []shape.Glyph{{ID: uint16(g), Char: r}}
		plain := l.Plan(tagHani, c.joined("")).Shape(run)
		halt := l.Plan(tagHani, c.joined("halt")).Shape([]shape.Glyph{run[0]})
		if len(plain) == 1 && len(halt) == 1 {
			u.shift = float32(halt[0].XOffset - plain[0].XOffset)
			u.narrow = float32(halt[0].Advance - plain[0].Advance)
		}
	}
	b, adv, err := c.font.GlyphBounds(&s.buf, g, fixed.I(int(c.upem)), font.HintingNone)
	if err != nil || adv <= 0 {
		return u
	}
	advance := float32(adv) / 64
	u.full = advance > c.upem*0.99 && advance < c.upem*1.01
	lo, hi := float32(b.Min.X)/64, float32(b.Max.X)/64
	switch {
	case hi <= lo:
	case hi <= advance/2:
		u.ink = box.InkLeft
	case lo >= advance/4 && hi <= advance*3/4:
		u.ink = box.InkCentred
	}
	return u
}
