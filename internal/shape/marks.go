package shape

import (
	"slices"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// combiningClass is r's canonical combining class as HarfBuzz sorts
// marks by it: Arabic's shadda before the other harakat, Hebrew's points
// in the order fonts expect them, Thai's, Telugu's and Tibetan's vowel
// signs as fonts take them.
func combiningClass(r rune) uint8 {
	switch r {
	case 0:
		return 0
	case 0x1A60, 0x0FC6:
		return 254
	case 0x0F39:
		return 127
	}
	c := norm.NFD.PropertiesString(string(r)).CCC()
	switch {
	case c >= 10 && c <= 35:
		return modifiedClasses[c-10]
	case c == 84:
		return 4
	case c == 91:
		return 5
	case c == 103:
		return 3
	case c == 130:
		return 132
	case c == 132:
		return 131
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

// Characters shaping treats apart.
const (
	zwnj         = 0x200C
	zwj          = 0x200D
	dottedCircle = 0x25CC
)

// DefaultIgnorable reports whether r is a character drawn as nothing
// unless a font says otherwise, as HarfBuzz lists them.
func DefaultIgnorable(r rune) bool {
	switch {
	case r < 0x80:
		return false
	case r < 0x10000:
		switch r >> 8 {
		case 0x00:
			return r == 0x00AD
		case 0x03:
			return r == 0x034F
		case 0x06:
			return r == 0x061C
		case 0x17:
			return r >= 0x17B4 && r <= 0x17B5
		case 0x18:
			return r >= 0x180B && r <= 0x180F
		case 0x20:
			return r >= 0x200B && r <= 0x200F || r >= 0x202A && r <= 0x202E || r >= 0x2060 && r <= 0x206F
		case 0xFE:
			return r >= 0xFE00 && r <= 0xFE0F || r == 0xFEFF
		case 0xFF:
			return r >= 0xFFF0 && r <= 0xFFF8
		}
		return false
	case r>>16 == 1:
		return r >= 0x1D173 && r <= 0x1D17A
	}
	return r >= 0xE0000 && r <= 0xE0FFF
}

// hidden reports whether r is a default ignorable character GSUB does
// not pass over: Mongolian's free variation selectors, tags and the
// combining grapheme joiner.
func hidden(r rune) bool {
	return r >= 0x180B && r <= 0x180D || r == 0x180F || r >= 0xE0020 && r <= 0xE007F || r == 0x034F
}

// ignorable reports whether g is of a default ignorable character no
// substitution has replaced.
func (b *buffer) ignorable(g *Glyph) bool {
	return g.props&substituted == 0 && DefaultIgnorable(g.Char)
}

// isMarkChar reports whether r is a mark, spacing or not.
func isMarkChar(r rune) bool { return r >= 0x300 && unicode.In(r, unicode.Mn, unicode.Mc, unicode.Me) }

// nominal is the font's glyph for r, 0 for none.
func (f *Font) nominal(r rune) uint16 {
	g, err := f.src.GlyphIndex(&f.buf, r)
	if err != nil {
		return 0
	}
	return uint16(g)
}

// Decomposes reports whether the font, which lacks r, draws it from the
// characters r decomposes to, as shaping does.
func (f *Font) Decomposes(r rune) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.nominal(r) == 0 && f.decomposes(r)
}

func (f *Font) decomposes(r rune) bool {
	a, b, ok := decompose(r)
	return ok && (b == 0 || f.nominal(b) != 0) && (f.nominal(a) != 0 || f.decomposes(a))
}

// decompose is r's canonical decomposition in one step, a then b (0 for
// a singleton), as the plan's shaper takes it; false for none.
func (p *Plan) decompose(r rune) (rune, rune, bool) {
	switch p.shaper { //nolint:exhaustive // the shapers that do something here
	case indicShaper:
		switch r {
		case 0x0931, 0x09DC, 0x09DD, 0x0B94:
			return 0, 0, false
		}
	case khmerShaper:
		switch r {
		case 0x17BE, 0x17BF, 0x17C0, 0x17C4, 0x17C5:
			return 0x17C1, r, true
		}
	}
	return decompose(r)
}

// decompose is r's canonical decomposition in one step: the character
// whose own decomposition starts r's and the last of r's.
func decompose(r rune) (rune, rune, bool) {
	s := string(r)
	if norm.NFD.IsNormalString(s) {
		return 0, 0, false
	}
	if c := norm.NFC.String(s); c != s && utf8.RuneCountInString(c) == 1 {
		a, _ := utf8.DecodeRuneInString(c)
		return a, 0, true
	}
	d := []rune(norm.NFD.String(s))
	last := d[len(d)-1]
	head := norm.NFC.String(string(d[:len(d)-1]))
	if utf8.RuneCountInString(head) != 1 {
		return 0, 0, false
	}
	a, _ := utf8.DecodeRuneInString(head)
	return a, last, true
}

// compose is the character a and b compose to, as the plan's shaper
// takes it; false for none.
func (p *Plan) compose(a, b rune) (rune, bool) {
	if p.shaper >= indicShaper && p.shaper != myanmarShaper && isMarkChar(a) {
		return 0, false
	}
	if p.shaper == indicShaper && a == 0x09AF && b == 0x09BC {
		return 0x09DF, true
	}
	s := norm.NFC.String(string([]rune{a, b}))
	if utf8.RuneCountInString(s) != 1 {
		if p.shaper == hebrewShaper && !p.marks {
			return hebrewForm(a, b)
		}
		return 0, false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r, true
}

// dageshForms are the presentation forms of the letters from alef to
// tav with a dagesh, 0 for those Unicode has none of.
//
//nolint:gochecknoglobals // a fixed table
var dageshForms = [...]rune{
	0xFB30, 0xFB31, 0xFB32, 0xFB33, 0xFB34, 0xFB35, 0xFB36, 0, 0xFB38, 0xFB39, 0xFB3A, 0xFB3B, 0xFB3C, 0, 0xFB3E,
	0, 0xFB40, 0xFB41, 0, 0xFB43, 0xFB44, 0, 0xFB46, 0xFB47, 0xFB48, 0xFB49, 0xFB4A,
}

// hebrewForm is the presentation form a Hebrew letter and point make,
// which normalisation leaves out but old fonts without mark positioning
// draw them with, as HarfBuzz's Hebrew shaper composes them.
func hebrewForm(a, b rune) (rune, bool) {
	var r rune
	switch b {
	case 0x05B4:
		if a == 0x05D9 {
			r = 0xFB1D
		}
	case 0x05B7:
		switch a {
		case 0x05F2:
			r = 0xFB1F
		case 0x05D0:
			r = 0xFB2E
		}
	case 0x05B8:
		if a == 0x05D0 {
			r = 0xFB2F
		}
	case 0x05B9:
		if a == 0x05D5 {
			r = 0xFB4B
		}
	case 0x05BC:
		switch {
		case a >= 0x05D0 && a <= 0x05EA:
			r = dageshForms[a-0x05D0]
		case a == 0xFB2A:
			r = 0xFB2C
		case a == 0xFB2B:
			r = 0xFB2D
		}
	case 0x05BF:
		switch a {
		case 0x05D1:
			r = 0xFB4C
		case 0x05DB:
			r = 0xFB4D
		case 0x05E4:
			r = 0xFB4E
		}
	case 0x05C1:
		switch a {
		case 0x05E9:
			r = 0xFB2A
		case 0xFB49:
			r = 0xFB2C
		}
	case 0x05C2:
		switch a {
		case 0x05E9:
			r = 0xFB2B
		case 0xFB49:
			r = 0xFB2D
		}
	}
	return r, r != 0
}

// normalize decomposes the characters the font lacks, and those of
// syllabic scripts even where it has them, into the ones it has, sorts
// each sequence of marks by their combining classes, and composes the
// marks that compose with their base into a character the font has,
// as HarfBuzz's normalisation does before shaping. Hangul's composes
// nothing and decomposes only what the font lacks; a cluster with a
// variation selector is left as it is (variationCluster).
func (b *buffer) normalize() {
	p := b.plan
	shortCircuit := p.shaper < indicShaper
	in := b.glyphs
	out := make([]Glyph, 0, len(in))
	simple := true
	for i := 0; i < len(in); {
		end := i + 1
		for end < len(in) && !isMarkChar(in[end].Char) {
			end++
		}
		if end < len(in) {
			end--
		}
		for ; i < end; i++ {
			out = b.decomposeChar(out, in[i], shortCircuit)
		}
		if i == len(in) {
			break
		}
		simple = false
		end = i + 1
		for end < len(in) && isMarkChar(in[end].Char) {
			end++
		}
		if slices.ContainsFunc(in[i:end], func(g Glyph) bool { return VariationSelector(g.Char) }) {
			out, i = b.variationCluster(out, in[i:end]), end
			continue
		}
		for ; i < end; i++ {
			out = b.decomposeChar(out, in[i], p.shaper == hangulShaper)
		}
	}
	if len(out) != len(in) {
		simple = false
	}
	b.glyphs = out
	if simple {
		return
	}
	for i := 0; i < len(b.glyphs); {
		if b.glyphs[i].cc == 0 {
			i++
			continue
		}
		end := i + 1
		for end < len(b.glyphs) && b.glyphs[end].cc != 0 {
			end++
		}
		if end-i <= maxCombiningMarks {
			b.sort(i, end, func(x, y *Glyph) int { return int(x.cc) - int(y.cc) })
			switch p.shaper { //nolint:exhaustive // the shapers that reorder marks
			case arabicShaper:
				b.reorderArabicMarks(i, end)
			case hebrewShaper:
				b.reorderHebrewMarks(i, end)
			}
		}
		i = end
	}
	if p.shaper != hangulShaper {
		b.compose()
	}
}

// maxCombiningMarks is the longest sequence of marks normalisation
// sorts, as HarfBuzz bounds it.
const maxCombiningMarks = 32

// modifierMarks are the marks Arabic's shaper moves before the others
// of their class, which modify the letter rather than sit on it.
//
//nolint:gochecknoglobals // a fixed list
var modifierMarks = []rune{
	0x0654, 0x0655, 0x0658, 0x06DC, 0x06E3, 0x06E7, 0x06E8, 0x08CA, 0x08CB, 0x08CD, 0x08CE, 0x08CF, 0x08D3, 0x08F3,
}

// reorderArabicMarks moves the modifier marks of classes 220 and 230
// in the sorted marks from start to end before the others, renumbered
// so the sequence stays sorted, as HarfBuzz's Arabic shaper does.
func (b *buffer) reorderArabicMarks(start, end int) {
	g := b.glyphs
	i := start
	for cc := uint8(220); cc <= 230; cc += 10 {
		for i < end && g[i].cc < cc {
			i++
		}
		if i == end {
			break
		}
		if g[i].cc > cc {
			continue
		}
		j := i
		for j < end && g[j].cc == cc && slices.Contains(modifierMarks, g[j].Char) {
			j++
		}
		if i == j {
			continue
		}
		b.mergeClusters(start, j)
		moved := slices.Clone(g[i:j])
		copy(g[start+j-i:j], g[start:i])
		copy(g[start:], moved)
		renumbered := uint8(25)
		if cc == 230 {
			renumbered = 26
		}
		for moved := start + j - i; start < moved; start++ {
			g[start].cc = renumbered
		}
	}
}

// reorderHebrewMarks puts a meteg or a mark below before the sheva or
// hiriq it follows after a patah or qamats, the order fonts expect, as
// HarfBuzz's Hebrew shaper does.
func (b *buffer) reorderHebrewMarks(start, end int) {
	g := b.glyphs
	for i := start + 2; i < end; i++ {
		c0, c1, c2 := g[i-2].cc, g[i-1].cc, g[i].cc
		if (c0 == 20 || c0 == 21) && (c1 == 22 || c1 == 23) && (c2 == 25 || c2 == 220) {
			b.mergeClusters(i-1, i+1)
			g[i-1], g[i] = g[i], g[i-1]
			return
		}
	}
}

// decomposeChar appends g to out, or what it decomposes to: the
// shortest decomposition the font has all of when it has g's character
// and shortest is set, else the longest.
func (b *buffer) decomposeChar(out []Glyph, g Glyph, shortest bool) []Glyph {
	if g.Char == 0 || shortest && g.ID != 0 {
		return append(out, g)
	}
	if parts := b.decomposed(g.Char, shortest); parts != nil {
		for _, r := range parts {
			c := g
			c.Char, c.ID, c.cc = r, b.f.nominal(r), combiningClass(r)
			out = append(out, c)
		}
		return out
	}
	if g.ID == 0 && g.Char == 0x2011 {
		if id := b.f.nominal(0x2010); id != 0 {
			g.ID = id
		}
	}
	return append(out, g)
}

// decomposed is what r decomposes to in characters the font has, nil
// for nothing.
func (b *buffer) decomposed(r rune, shortest bool) []rune {
	a, c, ok := b.plan.decompose(r)
	if !ok || c != 0 && b.f.nominal(c) == 0 {
		return nil
	}
	tail := func(head []rune) []rune {
		if c != 0 {
			return append(head, c)
		}
		return head
	}
	has := b.f.nominal(a) != 0
	if shortest && has {
		return tail([]rune{a})
	}
	if deeper := b.decomposed(a, shortest); deeper != nil {
		return tail(deeper)
	}
	if has {
		return tail([]rune{a})
	}
	return nil
}

// compose composes each mark with the character it follows when
// nothing between them has a combining class as high as its, and the
// font has the character they make.
func (b *buffer) compose() {
	if len(b.glyphs) == 0 {
		return
	}
	out := b.glyphs[:1]
	starter := 0
	for _, g := range b.glyphs[1:] {
		if isMarkChar(g.Char) && (starter == len(out)-1 || out[len(out)-1].cc < g.cc) {
			if c, ok := b.plan.compose(out[starter].Char, g.Char); ok {
				if id := b.f.nominal(c); id != 0 {
					s := &out[starter]
					for k := starter; k < len(out); k++ {
						s.Cluster = min(s.Cluster, out[k].Cluster)
					}
					s.Cluster = min(s.Cluster, g.Cluster)
					for k := starter + 1; k < len(out); k++ {
						out[k].Cluster = s.Cluster
					}
					s.Char, s.ID, s.cc = c, id, combiningClass(c)
					continue
				}
			}
		}
		out = append(out, g)
		if g.cc == 0 {
			starter = len(out) - 1
		}
	}
	b.glyphs = out
}

// sort sorts the glyphs from start to end stably by cmp, merging the
// clusters of each glyph it moves and those it moves over, as
// HarfBuzz's buffer sorts.
func (b *buffer) sort(start, end int, cmp func(x, y *Glyph) int) {
	for i := start + 1; i < end; i++ {
		j := i
		for j > start && cmp(&b.glyphs[j-1], &b.glyphs[i]) > 0 {
			j--
		}
		if i == j {
			continue
		}
		b.mergeClusters(j, i+1)
		b.move(i, j)
	}
}

// move moves glyph from to index to, those between shifting over.
func (b *buffer) move(from, to int) {
	g := b.glyphs[from]
	if from > to {
		copy(b.glyphs[to+1:from+1], b.glyphs[to:from])
	} else {
		copy(b.glyphs[from:to], b.glyphs[from+1:to+1])
	}
	b.glyphs[to] = g
}

// mergeClusters gives the glyphs from start to end, and those beside
// them of the clusters at either end, the least of their clusters, as
// HarfBuzz merges clusters.
func (b *buffer) mergeClusters(start, end int) {
	if end-start < 2 {
		return
	}
	c := b.glyphs[start].Cluster
	for i := start + 1; i < end; i++ {
		c = min(c, b.glyphs[i].Cluster)
	}
	if c != b.glyphs[end-1].Cluster {
		for end < len(b.glyphs) && b.glyphs[end-1].Cluster == b.glyphs[end].Cluster {
			end++
		}
	}
	if c != b.glyphs[start].Cluster {
		for start > 0 && b.glyphs[start-1].Cluster == b.glyphs[start].Cluster {
			start--
		}
	}
	for i := start; i < end; i++ {
		b.glyphs[i].Cluster = c
	}
}

// hideIgnorables draws default ignorable characters as the font's
// space, or leaves them out, their clusters merged into their
// neighbours', when it has none, as HarfBuzz hides them.
func (b *buffer) hideIgnorables() {
	space := b.f.nominal(' ')
	g := b.glyphs
	out := g[:0]
	for i := range g {
		if !b.ignorable(&g[i]) {
			out = append(out, g[i])
			continue
		}
		if space != 0 {
			g[i].ID = space
			out = append(out, g[i])
			continue
		}
		c := g[i].Cluster
		switch {
		case i+1 < len(g) && c == g[i+1].Cluster:
		case len(out) > 0:
			if old := out[len(out)-1].Cluster; c < old {
				for k := len(out); k > 0 && out[k-1].Cluster == old; k-- {
					out[k-1].Cluster = c
				}
			}
		case i+1 < len(g):
			old := g[i+1].Cluster
			for k := i + 1; k < len(g) && g[k].Cluster == old; k++ {
				g[k].Cluster = min(c, old)
			}
		}
	}
	b.glyphs = out
}

// reverse reverses the glyphs from start to end.
func (b *buffer) reverseRange(start, end int) { slices.Reverse(b.glyphs[start:end]) }
