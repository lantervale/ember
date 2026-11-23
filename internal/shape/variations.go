package shape

import "sort"

// VariationSelector reports whether r selects a variant of the
// character before it.
func VariationSelector(r rune) bool {
	return r >= 0x180B && r <= 0x180D || r == 0x180F || r >= 0xFE00 && r <= 0xFE0F || r >= 0xE0100 && r <= 0xE01EF
}

// u24 is the 24-bit number at at.
func (d data) u24(at int) rune {
	if at < 0 || at+3 > len(d) {
		return 0
	}
	return rune(d[at])<<16 | rune(d[at+1])<<8 | rune(d[at+2])
}

// variationSequences is the cmap subtable of format 14 in cmap, the
// glyphs of characters followed by variation selectors; nil for none.
func variationSequences(cmap data) data {
	for i := range int(cmap.u16(2)) {
		rec := 4 + 8*i
		if cmap.u16(rec) != 0 || cmap.u16(rec+2) != 5 {
			continue
		}
		if t := cmap.at(int(cmap.u32(rec + 4))); t.u16(0) == 14 {
			return t
		}
	}
	return nil
}

// variant is the glyph the font draws r followed by selector vs with,
// as cmap's format 14 maps them: a glyph of its own, or r's own glyph
// where the sequence is one of its defaults; false when the font does
// not map the sequence.
func (f *Font) variant(r, vs rune) (uint16, bool) {
	t := f.uvs
	n := int(t.u32(6))
	k := sort.Search(n, func(i int) bool { return t.u24(10+11*i) >= vs })
	if k == n || t.u24(10+11*k) != vs {
		return 0, false
	}
	rec := 10 + 11*k
	if d := t.at(int(t.u32(rec + 3))); d != nil {
		ranges := int(d.u32(0))
		i := sort.Search(ranges, func(i int) bool {
			at := 4 + 4*i
			return d.u24(at)+rune(d.u16(at+2)&0xff) >= r
		})
		if i < ranges && d.u24(4+4*i) <= r {
			id := f.nominal(r)
			return id, id != 0
		}
	}
	if d := t.at(int(t.u32(rec + 7))); d != nil {
		maps := int(d.u32(0))
		i := sort.Search(maps, func(i int) bool { return d.u24(4+5*i) >= r })
		if i < maps && d.u24(4+5*i) == r {
			return d.u16(4 + 5*i + 3), true
		}
	}
	return 0, false
}

// variationCluster maps the glyphs of a cluster from start to end
// holding a variation selector as HarfBuzz does, without decomposing
// them: a character and the selector after it make one glyph where the
// font maps the sequence; else each keeps its own and the selector is
// left for substitutions, drawn as nothing.
func (b *buffer) variationCluster(out []Glyph, in []Glyph) []Glyph {
	for i := 0; i < len(in); i++ {
		g := in[i]
		if i+1 < len(in) && VariationSelector(in[i+1].Char) {
			if id, ok := b.f.variant(g.Char, in[i+1].Char); ok {
				g.ID, g.Cluster = id, min(g.Cluster, in[i+1].Cluster)
				out = append(out, g)
				i++
			} else {
				out = append(out, g, in[i+1])
				i++
			}
			for i+1 < len(in) && VariationSelector(in[i+1].Char) {
				out = append(out, in[i+1])
				i++
			}
			continue
		}
		out = append(out, g)
	}
	return out
}
