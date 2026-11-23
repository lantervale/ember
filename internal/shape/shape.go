package shape

import (
	"slices"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Glyph is a glyph of a shaped run, in the font's units.
type Glyph struct {
	ID uint16
	// Cluster is the index, in what the run was made of, of the first
	// character the glyph draws.
	Cluster int32
	// Char is the character the glyph is of, 0 where it is not known:
	// the letters of a script that joins them take their forms by it.
	Char rune
	// Advance is how far the glyph moves the pen, kerning included.
	Advance int32
	// XOffset and YOffset move the glyph from the pen, y up.
	XOffset, YOffset int32

	// mask is the features that apply to it, by their masks.
	mask uint32
	// lig is the ligature it is, or the one a mark of it sits on, 0 for
	// none, and comp how many characters that ligature joined (ligated
	// set) or the one of them the mark sits on, from 1.
	lig     uint8
	comp    uint8
	ligated bool
	// attach is how far the glyph it is attached to lies, cursive whether
	// it is attached as the next letter rather than as a mark.
	attach  int32
	cursive bool
}

// maxGrowth bounds how many glyphs a run may grow to, by how many it
// had, so a font's lookups cannot make it endless.
const maxGrowth = 8

// maxDepth bounds how deep context lookups may call others, and
// attachments chain.
const maxDepth = 6

// Touches reports whether p may do anything to run but give its glyphs
// their advances: one of its lookups may start at one of them, the
// font's old kern table kerns it, or one is a mark whose advance
// shaping takes away.
func (p *Plan) Touches(run []Glyph) bool {
	if len(run) > 1 && p.KernTable() {
		return true
	}
	for _, g := range run {
		if p.first.has(g.ID) || p.font.isMark(g.ID) {
			return true
		}
	}
	return false
}

// Inert reports whether p does nothing to any run but give its glyphs
// their advances.
func (p *Plan) Inert() bool { return len(p.first) == 0 && !p.KernTable() && p.font.glyphClasses == nil }

// KernTable reports whether runs are kerned by the font's old kern
// table, its GPOS having no kern feature.
func (p *Plan) KernTable() bool { return p.kerns && p.font.kernTable }

// RightToLeft reports whether p's runs are written right to left: Shape
// keeps them in their order, and they are drawn from the last glyph.
func (p *Plan) RightToLeft() bool { return p.rtl }

// Shape substitutes and positions run, glyphs of p's font with their
// clusters in the order of the text, and gives each its advance; run's
// slice may be reused.
func (p *Plan) Shape(run []Glyph) []Glyph {
	f := p.font
	f.mu.Lock()
	defer f.mu.Unlock()
	b := buffer{f: f, glyphs: run, limit: maxGrowth*len(run) + 64}
	for i := range b.glyphs {
		g := &b.glyphs[i]
		g.mask, g.lig, g.comp, g.ligated = globalMask, 0, 0, false
	}
	b.reorderMarks()
	if p.joins {
		b.join(&p.forms)
	}
	for _, l := range p.gsub {
		b.run(l)
	}
	ppem := fixed.I(int(f.upem))
	for i := range b.glyphs {
		g := &b.glyphs[i]
		a, err := f.src.GlyphAdvance(&f.buf, sfnt.GlyphIndex(g.ID), ppem, font.HintingNone)
		if err == nil {
			g.Advance = int32(a >> 6)
		}
		g.XOffset, g.YOffset, g.attach, g.cursive = 0, 0, 0, false
	}
	b.rtl = p.rtl
	for _, l := range p.gpos {
		b.run(l)
	}
	if p.KernTable() {
		for i := 1; i < len(b.glyphs); i++ {
			k, err := f.src.Kern(&f.buf, sfnt.GlyphIndex(b.glyphs[i-1].ID), sfnt.GlyphIndex(b.glyphs[i].ID), ppem,
				font.HintingNone)
			if err == nil {
				b.glyphs[i-1].Advance += int32(k >> 6)
			}
		}
	}
	b.zeroMarks(f.gpos.lookups == nil && !p.rtl)
	for i := range b.glyphs {
		b.propagate(i, maxDepth)
	}
	return b.glyphs
}

// buffer is a run being shaped.
type buffer struct {
	f      *Font
	glyphs []Glyph
	limit  int
	depth  int
	// mask is the mask of the lookup applied: the glyphs it takes in.
	mask uint32
	// ligs is the last ligature made.
	ligs uint8
	rtl  bool
}

// run applies l to every glyph of the run it may start at, left to
// right, a reverse chaining lookup right to left.
func (b *buffer) run(l planned) {
	lk := l.lk
	b.mask = l.mask
	at := func(i int) bool {
		g := b.glyphs[i]
		return g.mask&l.mask != 0 && lk.first.has(g.ID) && !b.f.ignores(lk, g.ID)
	}
	if lk.kind == reverse {
		for i := len(b.glyphs) - 1; i >= 0; i-- {
			if at(i) {
				b.reverse(lk, i)
			}
		}
		return
	}
	for i := 0; i < len(b.glyphs); {
		if !at(i) {
			i++
			continue
		}
		if next, ok := b.apply(lk, l.value, i); ok {
			i = max(next, i+1)
		} else {
			i++
		}
	}
}

// apply applies the first of lk's subtables that applies at glyph i,
// and reports where the glyphs after what it changed start.
func (b *buffer) apply(lk *lookup, value uint32, i int) (int, bool) {
	for _, s := range lk.subs {
		var next int
		var ok bool
		switch lk.kind {
		case single:
			ok = b.single(s, i)
			next = i + 1
		case multiple:
			next, ok = b.multiple(s, i)
		case alternate:
			ok = b.alternate(s, i, value)
			next = i + 1
		case ligature:
			ok = b.ligature(lk, s, i)
			next = i + 1
		case context:
			next, ok = b.context(lk, s, i)
		case chain:
			next, ok = b.chain(lk, s, i)
		case adjust:
			ok = b.adjust(s, i)
			next = i + 1
		case pair:
			next, ok = b.pair(lk, s, i)
		case cursive:
			ok = b.cursive(lk, s, i)
			next = i + 1
		case markBase, markLigature:
			ok = b.markToBase(lk.kind == markLigature, s, i)
			next = i + 1
		case markMark:
			ok = b.markToMark(lk, s, i)
			next = i + 1
		}
		if ok {
			return next, true
		}
	}
	return i, false
}

// next is the first glyph after i lk does not pass over, -1 for none.
func (b *buffer) next(lk *lookup, i int) int {
	for j := i + 1; j < len(b.glyphs); j++ {
		if !b.f.ignores(lk, b.glyphs[j].ID) {
			return j
		}
	}
	return -1
}

// prev is the first glyph before i lk does not pass over, -1 for none.
func (b *buffer) prev(lk *lookup, i int) int {
	for j := i - 1; j >= 0; j-- {
		if !b.f.ignores(lk, b.glyphs[j].ID) {
			return j
		}
	}
	return -1
}

// replace puts glyphs in place of glyph i, each of its cluster and
// character, unless the run would grow past its limit.
func (b *buffer) replace(i int, ids []uint16) bool {
	if len(b.glyphs)-1+len(ids) > b.limit {
		return false
	}
	g := b.glyphs[i]
	repl := make([]Glyph, len(ids))
	for k, id := range ids {
		repl[k] = g
		repl[k].ID = id
	}
	b.glyphs = slices.Replace(b.glyphs, i, i+1, repl...)
	return true
}

// comps is how many characters g joins: a ligature's, else 1.
func (b *buffer) comps(g Glyph) uint8 {
	if g.ligated && (b.f.glyphClasses == nil || b.f.glyphClasses.class(g.ID) == ligatureGlyph) {
		return g.comp
	}
	return 1
}

// compOf is the character of the ligature g, a mark, sits on, 0 for
// none.
func compOf(g Glyph) uint8 {
	if g.ligated {
		return 0
	}
	return g.comp
}
