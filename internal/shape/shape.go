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
	// Advance is how far the glyph moves the pen, kerning included.
	Advance int32
	// XOffset and YOffset move the glyph from the pen, y up.
	XOffset, YOffset int32
}

// maxGrowth bounds how many glyphs a run may grow to, by how many it
// had, so a font's lookups cannot make it endless.
const maxGrowth = 8

// maxDepth bounds how deep context lookups may call others.
const maxDepth = 6

// Touches reports whether p may do anything to run but give its glyphs
// their advances: one of its lookups may start at one of them, or the
// font's old kern table kerns it.
func (p *Plan) Touches(run []Glyph) bool {
	if len(run) > 1 && p.KernTable() {
		return true
	}
	for _, g := range run {
		if p.first.has(g.ID) {
			return true
		}
	}
	return false
}

// Inert reports whether p does nothing to any run but give its glyphs
// their advances.
func (p *Plan) Inert() bool { return len(p.first) == 0 && !p.KernTable() }

// KernTable reports whether runs are kerned by the font's old kern
// table, its GPOS having no kern feature.
func (p *Plan) KernTable() bool { return p.kerns && p.font.kernTable }

// Shape substitutes and positions run, glyphs of p's font with their
// clusters, and gives each its advance; run's slice may be reused.
func (p *Plan) Shape(run []Glyph) []Glyph {
	f := p.font
	f.mu.Lock()
	defer f.mu.Unlock()
	b := buffer{f: f, glyphs: run, limit: maxGrowth*len(run) + 64}
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
		g.XOffset, g.YOffset = 0, 0
	}
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
	return b.glyphs
}

// buffer is a run being shaped.
type buffer struct {
	f      *Font
	glyphs []Glyph
	limit  int
	depth  int
}

// run applies l to every glyph of the run it may start at, left to
// right, a reverse chaining lookup right to left.
func (b *buffer) run(l planned) {
	lk := l.lk
	if lk.kind == reverse {
		for i := len(b.glyphs) - 1; i >= 0; i-- {
			if g := b.glyphs[i].ID; lk.first.has(g) && !b.f.ignores(lk, g) {
				b.reverse(lk, i)
			}
		}
		return
	}
	for i := 0; i < len(b.glyphs); {
		g := b.glyphs[i].ID
		if !lk.first.has(g) || b.f.ignores(lk, g) {
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

// replace puts glyphs in place of glyph i, each of its cluster, unless
// the run would grow past its limit.
func (b *buffer) replace(i int, ids []uint16) bool {
	if len(b.glyphs)-1+len(ids) > b.limit {
		return false
	}
	g := b.glyphs[i]
	repl := make([]Glyph, len(ids))
	for k, id := range ids {
		repl[k] = Glyph{ID: id, Cluster: g.Cluster}
	}
	b.glyphs = slices.Replace(b.glyphs, i, i+1, repl...)
	return true
}
