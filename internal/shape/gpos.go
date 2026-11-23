package shape

import (
	"math/bits"
	"sort"
)

// valueSize is the bytes a value record of format f takes.
func valueSize(f uint16) int { return 2 * bits.OnesCount16(f) }

// add adds the value record of format f at at in d to g: its placements
// to the offsets and its horizontal advance to the advance. Device and
// variation tables are left out, as at a size without hinting.
func (g *Glyph) add(d data, at int, f uint16) {
	for bit := uint16(1); bit <= 8; bit <<= 1 {
		if f&bit == 0 {
			continue
		}
		v := int32(int16(d.u16(at))) //nolint:gosec // a signed 16-bit field
		at += 2
		switch bit {
		case 1:
			g.XOffset += v
		case 2:
			g.YOffset += v
		case 4:
			g.Advance += v
		}
	}
}

// adjust applies subtable s of a single adjustment to glyph i.
func (b *buffer) adjust(s data, i int) bool {
	g := &b.glyphs[i]
	k := s.sub(2).coverage(g.ID)
	if k < 0 {
		return false
	}
	f := s.u16(4)
	switch s.u16(0) {
	case 1:
		g.add(s, 6, f)
		return true
	case 2:
		if k < int(s.u16(6)) {
			g.add(s, 8+k*valueSize(f), f)
			return true
		}
	}
	return false
}

// pair applies subtable s of a pair adjustment to glyph i and the next
// lk does not pass over, and reports where the glyphs left to adjust
// start: past the second when it was adjusted too.
func (b *buffer) pair(lk *lookup, s data, i int) (int, bool) {
	k := s.sub(2).coverage(b.glyphs[i].ID)
	j := b.next(lk, i)
	if k < 0 || j < 0 {
		return i, false
	}
	f1, f2 := s.u16(4), s.u16(6)
	n1, n2 := valueSize(f1), valueSize(f2)
	second := b.glyphs[j].ID
	switch s.u16(0) {
	case 1:
		if k >= int(s.u16(8)) {
			return i, false
		}
		set := s.sub(10 + 2*k)
		rec := 2 + n1 + n2
		count := int(set.u16(0))
		n := sort.Search(count, func(n int) bool { return set.u16(2+n*rec) >= second })
		if n == count || set.u16(2+n*rec) != second {
			return i, false
		}
		b.glyphs[i].add(set, 2+n*rec+2, f1)
		b.glyphs[j].add(set, 2+n*rec+2+n1, f2)
	case 2:
		c1 := int(s.sub(8).class(b.glyphs[i].ID))
		c2 := int(s.sub(10).class(second))
		count2 := int(s.u16(14))
		if c1 >= int(s.u16(12)) || c2 >= count2 {
			return i, false
		}
		rec := 16 + (c1*count2+c2)*(n1+n2)
		b.glyphs[i].add(s, rec, f1)
		b.glyphs[j].add(s, rec+n1, f2)
	default:
		return i, false
	}
	if f2 != 0 {
		return j + 1, true
	}
	return j, true
}
