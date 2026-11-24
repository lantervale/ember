package shape

import (
	"encoding/binary"
	"slices"
)

// arabicFallback is whether runs of script in a face whose language
// systems are gsubLS and gposLS are shaped by HarfBuzz's Arabic fallback:
// Arabic in a face with none of the joined forms' features.
func (f *Font) arabicFallback(script Tag, gsubLS, gposLS data) bool {
	if script != arab {
		return false
	}
	for _, t := range []Tag{formTags[isol], formTags[fina], formTags[medi], formTags[initial]} {
		if f.gsub.inLangSys(gsubLS, t) || f.gpos.inLangSys(gposLS, t) {
			return false
		}
	}
	return true
}

// arabicLookups are the lookups HarfBuzz synthesizes from the
// presentation forms of f's cmap, read once: the initial, medial, final
// and isolated forms, then the ligatures of three letters, of two, and of
// a shadda and a mark; nil where the face has none of one.
func (f *Font) arabicLookups() *[7]*lookup {
	if f.arabic != nil {
		return f.arabic
	}
	f.arabic = &[7]*lookup{}
	glyph := func(r rune) (uint16, bool) {
		if r == 0 {
			return 0, false
		}
		g, err := f.src.GlyphIndex(&f.buf, r)
		return uint16(g), err == nil && g != 0
	}
	for form := range 4 {
		var pairs [][2]uint16
		for i, forms := range arabicForms {
			u, ok := glyph(arabicFormsFirst + rune(i))
			s, sok := glyph(rune(forms[form]))
			if ok && sok && u != s {
				pairs = append(pairs, [2]uint16{u, s})
			}
		}
		f.arabic[form] = singleLookup(pairs)
	}
	for k, table := range [][]arabicLigature{arabicLigatures3[:], arabicLigatures[:], arabicMarkLigatures[:]} {
		var sets []ligatureSet
		for _, l := range table {
			first, ok := glyph(l.parts[0])
			if !ok {
				continue
			}
			if n := len(sets); n == 0 || sets[n-1].rune != l.parts[0] {
				sets = append(sets, ligatureSet{rune: l.parts[0], first: first})
			}
			lig, ok := glyph(l.lig)
			comps := []uint16{lig}
			for _, r := range l.parts[1:] {
				if r == 0 {
					break
				}
				g, gok := glyph(r)
				ok = ok && gok
				comps = append(comps, g)
			}
			if ok {
				sets[len(sets)-1].ligs = append(sets[len(sets)-1].ligs, comps)
			}
		}
		flag := ignoreMarks
		if k == 2 {
			flag = 0
		}
		f.arabic[4+k] = ligatureLookup(sets, flag)
	}
	return f.arabic
}

// arabicFallbackStage applies p's Arabic fallback lookups, each to the
// glyphs of its feature's mask, after rlig's stage.
func arabicFallbackStage(p *Plan, b *buffer) {
	for _, l := range p.arabic {
		b.run(l)
	}
}

// arabicPlanned are the fallback lookups of f a plan with masks applies:
// the forms' by their features' masks, the ligatures' by rlig's.
func (f *Font) arabicPlanned(masks map[Tag]uint32) []planned {
	lookups := f.arabicLookups()
	tags := [7]Tag{formTags[initial], formTags[medi], formTags[fina], formTags[isol], rlig, rlig, rlig}
	var out []planned
	for i, lk := range lookups {
		if m := masks[tags[i]]; lk != nil && m != 0 {
			out = append(out, planned{lk: lk, value: 1, mask: m, autoZWJ: true, autoZWNJ: true})
		}
	}
	return out
}

// singleLookup is a lookup of a single substitution of each pair's first
// glyph by its second, nil for none; a glyph twice is substituted by the
// first of its pairs.
func singleLookup(pairs [][2]uint16) *lookup {
	if len(pairs) == 0 {
		return nil
	}
	slices.SortStableFunc(pairs, func(a, b [2]uint16) int { return int(a[0]) - int(b[0]) })
	pairs = slices.CompactFunc(pairs, func(a, b [2]uint16) bool { return a[0] == b[0] })
	n := len(pairs)
	sub := words(2, 6+2*n, n)
	cov := words(1, n)
	for _, p := range pairs {
		sub = append(sub, words(int(p[1]))...)
		cov = append(cov, words(int(p[0]))...)
	}
	return wrapLookup(single, 0, slices.Concat(sub, cov))
}

// ligatureSet is the ligatures that start with a glyph: each its glyph,
// then its components after the first.
type ligatureSet struct {
	rune  rune
	first uint16
	ligs  [][]uint16
}

// ligatureLookup is a lookup of a ligature substitution of sets, nil for
// none; a glyph twice starts the ligatures of its first set.
func ligatureLookup(sets []ligatureSet, flag int) *lookup {
	if len(sets) == 0 {
		return nil
	}
	slices.SortStableFunc(sets, func(a, b ligatureSet) int { return int(a.first) - int(b.first) })
	sets = slices.CompactFunc(sets, func(a, b ligatureSet) bool { return a.first == b.first })
	n := len(sets)
	var offsets, body, cov []byte
	for _, s := range sets {
		offsets = append(offsets, words(6+2*n+len(body))...)
		set := words(len(s.ligs))
		var ligs []byte
		for _, l := range s.ligs {
			set = append(set, words(2+2*len(s.ligs)+len(ligs))...)
			ligs = append(ligs, words(int(l[0]), len(l))...)
			for _, c := range l[1:] {
				ligs = append(ligs, words(int(c))...)
			}
		}
		body = slices.Concat(body, set, ligs)
		cov = append(cov, words(int(s.first))...)
	}
	sub := slices.Concat(words(1, 6+2*n+len(body), n), offsets, body, words(1, n), cov)
	return wrapLookup(ligature, flag, sub)
}

// wrapLookup is the lookup of kind and flag whose one subtable is sub.
func wrapLookup(kind uint16, flag int, sub []byte) *lookup {
	return readLookup(data(slices.Concat(words(int(kind), flag, 1, 8), sub)), false)
}

// words writes vs as a font's big-endian 16-bit words.
func words(vs ...int) []byte {
	out := make([]byte, 0, 2*len(vs))
	for _, v := range vs {
		out = binary.BigEndian.AppendUint16(out, uint16(v)) //nolint:gosec // the sizes of a few hundred glyphs
	}
	return out
}
