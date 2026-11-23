package shape

import (
	"slices"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Hangul's jamo and syllables, as Unicode composes them.
const (
	lBase  = 0x1100
	vBase  = 0x1161
	tBase  = 0x11A7
	lCount = 19
	vCount = 21
	tCount = 28
	sBase  = 0xAC00
	nCount = vCount * tCount
	sCount = lCount * nCount
)

// The jamo feature a glyph of Hangul takes, in its cat.
const (
	jamoNone uint8 = iota
	jamoL
	jamoV
	jamoT
)

func combiningL(r rune) bool { return r >= lBase && r < lBase+lCount }
func combiningV(r rune) bool { return r >= vBase && r < vBase+vCount }
func combiningT(r rune) bool { return r > tBase && r < tBase+tCount }
func syllable(r rune) bool   { return r >= sBase && r < sBase+sCount }
func leadingJamo(r rune) bool {
	return r >= 0x1100 && r <= 0x115F || r >= 0xA960 && r <= 0xA97C
}
func vowelJamo(r rune) bool    { return r >= 0x1160 && r <= 0x11A7 || r >= 0xD7B0 && r <= 0xD7C6 }
func trailingJamo(r rune) bool { return r >= 0x11A8 && r <= 0x11FF || r >= 0xD7CB && r <= 0xD7FB }
func hangulTone(r rune) bool   { return r == 0x302E || r == 0x302F }

// preprocessHangul composes each syllable of jamo the font has whole,
// decomposes into jamo the syllables it lacks, marking the jamo of
// syllables it cannot compose for ljmo, vjmo and tjmo, and puts a tone
// mark before its syllable, as HarfBuzz's Hangul shaper does.
//
//nolint:cyclop,funlen,gocognit,gocyclo // HarfBuzz's cases, in their order
func (b *buffer) preprocessHangul() {
	start, end := 0, 0
	for i := 0; i < len(b.glyphs); {
		u := b.glyphs[i].Char
		if hangulTone(u) {
			switch {
			case start < end && end == i:
				if !b.zeroWidth(u) {
					b.mergeClusters(start, end+1)
					b.move(end, start)
				}
				i++
			case b.f.nominal(dottedCircle) != 0:
				dc := b.glyphs[i]
				dc.Char, dc.ID = dottedCircle, b.f.nominal(dottedCircle)
				at := i + 1
				if b.zeroWidth(u) {
					at = i
				}
				b.glyphs = slices.Insert(b.glyphs, at, dc)
				i += 2
			default:
				i++
			}
			start, end = i, i
			continue
		}
		start = i
		next := func(k int) rune {
			if i+k < len(b.glyphs) {
				return b.glyphs[i+k].Char
			}
			return 0
		}
		switch {
		case leadingJamo(u) && i+1 < len(b.glyphs):
			v := next(1)
			if !vowelJamo(v) {
				break
			}
			t := next(2)
			if !trailingJamo(t) {
				t = 0
			}
			n := 2
			if t != 0 {
				n = 3
			}
			if combiningL(u) && combiningV(v) && (t == 0 || combiningT(t)) {
				s := sBase + (u-lBase)*nCount + (v-vBase)*tCount
				if t != 0 {
					s += t - tBase
				}
				if b.composeHangul(i, n, s) {
					end, i = start+1, start+1
					continue
				}
			}
			for k, cat := range []uint8{jamoL, jamoV, jamoT}[:n] {
				b.glyphs[i+k].cat = cat
			}
			end, i = start+n, start+n
			b.mergeGraphemes(start, end)
			continue
		case syllable(u):
			has := b.f.nominal(u) != 0
			li, ni := (u-sBase)/nCount, (u-sBase)%nCount
			vi, ti := ni/tCount, ni%tCount
			if ti == 0 && combiningT(next(1)) && b.composeHangul(i, 2, u+next(1)-tBase) {
				end, i = start+1, start+1
				continue
			}
			if !has || ti == 0 && trailingJamo(next(1)) {
				jamo := []rune{lBase + li, vBase + vi, tBase + ti}
				if ti == 0 {
					jamo = jamo[:2]
				}
				if !slices.ContainsFunc(jamo, func(r rune) bool { return b.f.nominal(r) == 0 }) {
					g := b.glyphs[i]
					parts := make([]Glyph, len(jamo))
					for k, r := range jamo {
						parts[k] = g
						parts[k].Char, parts[k].ID, parts[k].cc = r, b.f.nominal(r), 0
						parts[k].cat = []uint8{jamoL, jamoV, jamoT}[k]
					}
					b.glyphs = slices.Replace(b.glyphs, i, i+1, parts...)
					n := len(jamo)
					if has && ti == 0 {
						b.glyphs[i+n].cat = jamoT
						n++
					}
					end, i = start+n, start+n
					b.mergeGraphemes(start, end)
					continue
				}
			}
			if has {
				end = start + 1
			}
		}
		i++
	}
}

// composeHangul puts the syllable s in place of the n jamo from i when
// the font has it, their clusters merged.
func (b *buffer) composeHangul(i, n int, s rune) bool {
	id := b.f.nominal(s)
	if id == 0 {
		return false
	}
	b.mergeClusters(i, i+n)
	g := b.glyphs[i]
	g.Char, g.ID, g.cc = s, id, 0
	b.glyphs = slices.Replace(b.glyphs, i, i+n, g)
	return true
}

// zeroWidth reports whether the font draws r with a glyph of no advance.
func (b *buffer) zeroWidth(r rune) bool {
	id := b.f.nominal(r)
	if id == 0 {
		return false
	}
	a, err := b.f.src.GlyphAdvance(&b.f.buf, sfnt.GlyphIndex(id), fixed.I(int(b.f.upem)), font.HintingNone)
	return err == nil && a == 0
}

// hangulMasks gives the jamo of syllables left decomposed the masks of
// their features.
func (b *buffer) hangulMasks() {
	m := b.plan.masks
	masks := [...]uint32{0, m[MakeTag("ljmo")], m[MakeTag("vjmo")], m[MakeTag("tjmo")]}
	for i := range b.glyphs {
		b.glyphs[i].mask |= masks[b.glyphs[i].cat]
	}
}
