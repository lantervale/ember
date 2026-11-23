package shape

import (
	"slices"
	"unicode"

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
	// props is what substitutions did to it, cc its character's combining
	// class as HarfBuzz sorts marks by.
	props uint8
	cc    uint8
	// syllable is the syllable a syllabic shaper found it in, its serial
	// above its kind, 0 for none; cat and pos the category and position
	// the shaper gives it.
	syllable uint8
	cat, pos uint8
}

// What substitutions did to a glyph, as HarfBuzz's glyph properties
// record it.
const (
	substituted uint8 = 1 << iota
	formed
	multiplied
)

// maxGrowth bounds how many glyphs a run may grow to, by how many it
// had, so a font's lookups cannot make it endless.
const maxGrowth = 8

// maxDepth bounds how deep context lookups may call others, and
// attachments chain.
const maxDepth = 6

// Touches reports whether p may do anything to run but give its glyphs
// their advances: one of its lookups may start at one of them, the
// font's old kern table kerns it, one is a mark whose advance shaping
// takes away or a character the font draws from others, or the run is
// of a script whose characters its shaper reorders.
func (p *Plan) Touches(run []Glyph) bool {
	if len(run) > 1 && p.KernTable() || p.shaper >= indicShaper {
		return true
	}
	fallback := p.fallbackMarks()
	for _, g := range run {
		if p.first.has(g.ID) || p.font.isMark(g.ID) || g.ID == 0 && g.Char != 0 ||
			fallback && g.Char >= 0x300 && unicode.Is(unicode.Mn, g.Char) {
			return true
		}
	}
	return false
}

// Inert reports whether p does nothing to any run but give its glyphs
// their advances.
func (p *Plan) Inert() bool {
	return len(p.first) == 0 && !p.KernTable() && p.font.glyphClasses == nil && !p.fallbackMarks() &&
		p.shaper < indicShaper
}

// KernTable reports whether runs are kerned by the font's old kern
// table, its GPOS having no kern feature.
func (p *Plan) KernTable() bool { return p.kerns && p.font.kernTable && p.shaper <= arabicShaper }

// RightToLeft reports whether p's runs are written right to left: Shape
// keeps them in their order, and they are drawn from the last glyph.
func (p *Plan) RightToLeft() bool { return p.rtl }

// fallbackMarks is whether marks are placed by their combining classes,
// the font having no GPOS to place them, as HarfBuzz's default and
// Arabic shapers do (not its Thai and Hangul ones).
func (p *Plan) fallbackMarks() bool {
	return !p.positions && p.shaper <= arabicShaper && p.script != thai && p.script != lao && p.script != hang
}

// Shape substitutes and positions run, glyphs of p's font with their
// clusters in the order of the text, and gives each its advance; run's
// slice may be reused. A glyph 0 of a known character is one the font
// lacks, which it may draw from the characters it decomposes to.
//
//nolint:funlen // HarfBuzz's steps, in their order
func (p *Plan) Shape(run []Glyph) []Glyph {
	f := p.font
	f.mu.Lock()
	defer f.mu.Unlock()
	b := buffer{f: f, plan: p, glyphs: run, limit: maxGrowth*len(run) + 64}
	for i := range b.glyphs {
		g := &b.glyphs[i]
		g.mask, g.lig, g.comp, g.ligated = globalMask, 0, 0, false
		g.props, g.syllable, g.cat, g.pos = 0, 0, 0, 0
		g.cc = combiningClass(g.Char)
	}
	if len(b.glyphs) > 0 && isMarkChar(b.glyphs[0].Char) {
		if id := f.nominal(dottedCircle); id != 0 {
			dc := Glyph{ID: id, Char: dottedCircle, Cluster: b.glyphs[0].Cluster, mask: globalMask}
			b.glyphs = slices.Insert(b.glyphs, 0, dc)
		}
	}
	if p.shaper == indicShaper || p.shaper == useShaper {
		b.vowelConstraints()
	}
	b.normalize()
	switch p.shaper { //nolint:exhaustive // the shapers that do something here
	case arabicShaper:
		b.join(&p.forms)
	case indicShaper, khmerShaper, myanmarShaper:
		b.indicCategories()
	case useShaper:
		b.useCategories()
	}
	if p.fallbackMarks() {
		b.recategorize()
	}
	b.substitute()
	b.hideIgnorables()
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
	adjust := !p.positions && !p.rtl
	early := p.shaper == myanmarShaper || p.shaper == useShaper
	if early {
		b.zeroMarks(adjust)
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
	if p.shaper <= arabicShaper {
		b.zeroMarks(adjust)
	}
	for i := range b.glyphs {
		if g := &b.glyphs[i]; b.ignorable(g) {
			g.Advance, g.XOffset, g.YOffset = 0, 0, 0
		}
	}
	for i := range b.glyphs {
		b.propagate(i, maxDepth)
	}
	if p.fallbackMarks() {
		b.placeMarks(adjust)
	}
	return b.glyphs
}

// substitute applies the plan's GSUB lookups stage by stage, each
// stage's pause after its lookups.
func (b *buffer) substitute() {
	p := b.plan
	last := 0
	for s := range p.pauses {
		last = max(last, s)
	}
	if n := len(p.gsub); n > 0 {
		last = max(last, p.gsub[n-1].stage)
	}
	at := 0
	for s := 0; s <= last; s++ {
		for at < len(p.gsub) && p.gsub[at].stage == s {
			b.run(p.gsub[at])
			at++
		}
		if fn := p.pauses[s]; fn != nil {
			fn(p, b)
		}
	}
}

// buffer is a run being shaped.
type buffer struct {
	f      *Font
	plan   *Plan
	glyphs []Glyph
	limit  int
	depth  int
	// mask is the mask of the lookup applied: the glyphs it takes in.
	mask uint32
	// autoZWJ, autoZWNJ and perSyllable are its planned's, positions
	// whether it is GPOS's, and cur the glyph it applies at.
	autoZWJ, autoZWNJ, perSyllable, positions bool
	cur                                       int
	// ligs is the last ligature made.
	ligs uint8
	rtl  bool
}

// run applies l to every glyph of the run it may start at, left to
// right, a reverse chaining lookup right to left.
func (b *buffer) run(l planned) {
	lk := l.lk
	b.mask, b.autoZWJ, b.autoZWNJ, b.perSyllable, b.positions = l.mask, l.autoZWJ, l.autoZWNJ, l.perSyllable,
		lk.positions
	at := func(i int) bool {
		g := b.glyphs[i]
		return g.mask&l.mask != 0 && lk.first.has(g.ID) && !b.ignores(lk, i)
	}
	if lk.kind == reverse {
		for i := len(b.glyphs) - 1; i >= 0; i-- {
			if at(i) {
				b.cur = i
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
	defer func(cur int) { b.cur = cur }(b.cur)
	b.cur = i
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

// seek is the first glyph after glyph i (dir 1) or before it (dir -1)
// that a match of lk takes, as HarfBuzz's skipping iterator finds it:
// glyphs lk ignores are passed over, and default ignorable characters
// that do not match unless they are joiners the lookup matches; -1 when
// the first other glyph does not match, or is of another syllable. The
// glyph must have the lookup's mask unless it is of the context around
// the input; match, nil for any glyph, says whether it matches.
func (b *buffer) seek(lk *lookup, i, dir int, around bool, match func(g *Glyph) bool) int {
	return b.seekBy(lk.flag, lk.markSet, i, dir, around, match)
}

// seekBy is seek for a lookup of flag and markSet.
func (b *buffer) seekBy(flag, markSet uint16, i, dir int, around bool, match func(g *Glyph) bool) int {
	ignoreZWNJ := b.positions || around && b.autoZWNJ
	ignoreZWJ := around || b.autoZWJ
	syllable := uint8(0)
	if b.perSyllable && !b.positions && b.cur < len(b.glyphs) {
		syllable = b.glyphs[b.cur].syllable
	}
	for j := i + dir; j >= 0 && j < len(b.glyphs); j += dir {
		if b.skips(flag, markSet, j) {
			continue
		}
		g := &b.glyphs[j]
		maybeSkip := b.ignorable(g) && (ignoreZWNJ || g.Char != zwnj) && (ignoreZWJ || g.Char != zwj) &&
			(b.positions || !hidden(g.Char))
		matches, maybe := false, false
		switch {
		case !around && g.mask&b.mask == 0, syllable != 0 && g.syllable != syllable:
		case match != nil:
			matches = match(g)
		default:
			maybe = true
		}
		if matches || maybe && !maybeSkip {
			return j
		}
		if !maybeSkip {
			return -1
		}
	}
	return -1
}

// next is the first glyph after i the input of lk takes, -1 for none.
func (b *buffer) next(lk *lookup, i int) int { return b.seek(lk, i, 1, false, nil) }

// prev is the first glyph before i the input of lk takes, -1 for none.
func (b *buffer) prev(lk *lookup, i int) int { return b.seek(lk, i, -1, false, nil) }

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

// class is glyph i's class: GDEF's, else a mark for a nonspacing mark
// character that is not default ignorable and a base for the rest, as
// HarfBuzz synthesizes classes for a font without them.
func (b *buffer) class(i int) uint16 {
	g := &b.glyphs[i]
	if b.f.glyphClasses != nil {
		return b.f.glyphClasses.class(g.ID)
	}
	if g.Char != 0 && unicode.Is(unicode.Mn, g.Char) && !b.ignorable(g) {
		return markGlyph
	}
	if g.ligated {
		return ligatureGlyph
	}
	return baseGlyph
}

// isMark reports whether glyph i is of the mark class.
func (b *buffer) isMark(i int) bool { return b.class(i) == markGlyph }

// ignores reports whether lk passes over glyph i, as its flag says.
func (b *buffer) ignores(lk *lookup, i int) bool { return b.skips(lk.flag, lk.markSet, i) }

// skips reports whether a lookup of flag, and markSet where the flag
// says to use one, passes over glyph i.
func (b *buffer) skips(flag, markSet uint16, i int) bool {
	if flag&(ignoreBase|ignoreLigatures|ignoreMarks|useMarkSet|markAttachType) == 0 {
		return false
	}
	return b.f.skipsClass(flag, markSet, b.glyphs[i].ID, b.class(i))
}
