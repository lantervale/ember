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

// hanKey is a character text-spacing-trim reads, standing upright in a
// vertical line or not.
type hanKey struct {
	r       rune
	upright bool
}

// HanGlyph is what text-spacing-trim reads of r's glyph in the face
// drawing it in f, at f's size, standing upright in a vertical line when
// upright: nothing where no face of f draws it.
func (s *Set) HanGlyph(f style.Font, r rune, upright bool) box.HanGlyph {
	c, g := s.drawing(f, r)
	if c == nil || c.upem <= 0 {
		return box.HanGlyph{}
	}
	key := hanKey{r, upright}
	u, ok := c.han[key]
	if !ok {
		u = s.hanOf(c, g, r)
		if upright {
			u.halt, u.shift, u.narrow = c.vhalOf(g, r)
		}
		if c.han == nil {
			c.han = map[hanKey]hanGlyph{}
		}
		c.han[key] = u
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
		return s.systemGlyph(f, r)
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

// vhalOf is what vhal does to glyph g of r in c set upright in its form
// for vertical lines (vert), as Chrome trims punctuation in vertical
// text: whether c has vhal, how far it moves the glyph down the line and
// how much it takes off its advance, in c's units.
func (c *Face) vhalOf(g sfnt.GlyphIndex, r rune) (bool, float32, float32) {
	tagVhal, tagHani, tagKana := shape.MakeTag("vhal"), shape.MakeTag("hani"), shape.MakeTag("kana")
	l := c.layout()
	if !l.Has(tagHani, tagVhal) && !l.Has(tagKana, tagVhal) {
		return false, 0, 0
	}
	run := func() []shape.Glyph { return []shape.Glyph{{ID: uint16(g), Char: r}} }
	plain := l.Plan(tagHani, c.joined(verticalFeatures)).Shape(run())
	vhal := l.Plan(tagHani, c.joined(verticalFeatures+",vhal")).Shape(run())
	if len(plain) != 1 || len(vhal) != 1 {
		return true, 0, 0
	}
	return true, float32(plain[0].YOffset - vhal[0].YOffset), float32(vhal[0].YAdvance - plain[0].YAdvance)
}

// hanPair is what a face's chws does to two characters side by side, in
// its units, ok false where it has no chws.
type hanPair struct {
	a, b box.HanAdjust
	ok   bool
}

// HanPair is what the face drawing a in fa and b in fb does to them side
// by side by its chws feature, which Chrome spaces their punctuation by
// where the face has it and halt: ok is false unless one such face draws
// both at one size.
func (s *Set) HanPair(fa style.Font, a rune, fb style.Font, b rune) (box.HanAdjust, box.HanAdjust, bool) {
	ca, ga := s.drawing(fa, a)
	cb, gb := s.drawing(fb, b)
	if ca == nil || ca != cb || ca.upem <= 0 || fa.Size != fb.Size {
		return box.HanAdjust{}, box.HanAdjust{}, false
	}
	key := [2]rune{a, b}
	p, seen := ca.pairs[key]
	if !seen {
		p = ca.pairOf(ga, a, gb, b)
		if ca.pairs == nil || len(ca.pairs) >= maxShaped {
			ca.pairs = map[[2]rune]hanPair{}
		}
		ca.pairs[key] = p
	}
	if !p.ok {
		return box.HanAdjust{}, box.HanAdjust{}, false
	}
	scale := fa.Size * ca.Look.adjust() / ca.upem
	return box.HanAdjust{Shift: p.a.Shift * scale, Narrow: p.a.Narrow * scale},
		box.HanAdjust{Shift: p.b.Shift * scale, Narrow: p.b.Narrow * scale}, true
}

// pairOf shapes glyph ga of a and gb of b side by side with and without
// chws, in c's units.
func (c *Face) pairOf(ga sfnt.GlyphIndex, a rune, gb sfnt.GlyphIndex, b rune) hanPair {
	tagHalt, tagChws := shape.MakeTag("halt"), shape.MakeTag("chws")
	l := c.layout()
	var script shape.Tag
	for _, t := range []shape.Tag{shape.MakeTag("hani"), shape.MakeTag("kana")} {
		if l.Has(t, tagChws) && l.Has(t, tagHalt) {
			script = t
			break
		}
	}
	if script == 0 {
		return hanPair{}
	}
	run := func() []shape.Glyph {
		return []shape.Glyph{{ID: uint16(ga), Char: a}, {ID: uint16(gb), Cluster: 1, Char: b}}
	}
	plain := l.Plan(script, c.joined("")).Shape(run())
	spaced := l.Plan(script, c.joined("chws")).Shape(run())
	if len(plain) != 2 || len(spaced) != 2 {
		return hanPair{}
	}
	adjust := func(i int) box.HanAdjust {
		return box.HanAdjust{Shift: float32(spaced[i].XOffset - plain[i].XOffset),
			Narrow: float32(spaced[i].Advance - plain[i].Advance)}
	}
	return hanPair{a: adjust(0), b: adjust(1), ok: true}
}
