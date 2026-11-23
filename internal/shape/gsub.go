package shape

// single substitutes glyph i by subtable s of a single substitution.
func (b *buffer) single(s data, i int) bool {
	g := &b.glyphs[i]
	k := s.sub(2).coverage(g.ID)
	if k < 0 {
		return false
	}
	switch s.u16(0) {
	case 1:
		g.ID += s.u16(4)
		return true
	case 2:
		if k < int(s.u16(4)) {
			g.ID = s.u16(6 + 2*k)
			return true
		}
	}
	return false
}

// multiple puts the sequence subtable s has for glyph i in its place.
func (b *buffer) multiple(s data, i int) (int, bool) {
	k := s.sub(2).coverage(b.glyphs[i].ID)
	if s.u16(0) != 1 || k < 0 || k >= int(s.u16(4)) {
		return i, false
	}
	seq := s.sub(6 + 2*k)
	ids := make([]uint16, seq.u16(0))
	for n := range ids {
		ids[n] = seq.u16(2 + 2*n)
	}
	if !b.replace(i, ids) {
		return i, false
	}
	return i + len(ids), true
}

// alternate takes the value-th of the alternates subtable s has for
// glyph i, the first for 1.
func (b *buffer) alternate(s data, i int, value uint32) bool {
	g := &b.glyphs[i]
	k := s.sub(2).coverage(g.ID)
	if s.u16(0) != 1 || k < 0 || k >= int(s.u16(4)) || value == 0 {
		return false
	}
	set := s.sub(6 + 2*k)
	if int(value) > int(set.u16(0)) {
		return false
	}
	g.ID = set.u16(2 + 2*int(value-1))
	return true
}

// ligature joins glyph i and those after it into the first ligature of
// subtable s whose components they are, the glyphs lk passes over
// between them kept after it.
func (b *buffer) ligature(lk *lookup, s data, i int) bool {
	k := s.sub(2).coverage(b.glyphs[i].ID)
	if s.u16(0) != 1 || k < 0 || k >= int(s.u16(4)) {
		return false
	}
	set := s.sub(6 + 2*k)
	for n := range int(set.u16(0)) {
		lig := set.sub(2 + 2*n)
		count := int(lig.u16(2))
		if count == 0 {
			continue
		}
		at := make([]int, 0, count-1)
		j := i
		for c := 1; c < count; c++ {
			j = b.next(lk, j)
			if j < 0 || b.glyphs[j].ID != lig.u16(4+2*(c-1)) {
				break
			}
			at = append(at, j)
		}
		if len(at) != count-1 {
			continue
		}
		b.glyphs[i].ID = lig.u16(0)
		for c := len(at) - 1; c >= 0; c-- {
			b.glyphs = append(b.glyphs[:at[c]], b.glyphs[at[c]+1:]...)
		}
		return true
	}
	return false
}

// reverse applies subtable format 1 of a reverse chaining single
// substitution at glyph i, its context read from the glyphs as they
// stand.
func (b *buffer) reverse(lk *lookup, i int) {
	for _, s := range lk.subs {
		if s.u16(0) != 1 {
			continue
		}
		k := s.sub(2).coverage(b.glyphs[i].ID)
		if k < 0 {
			continue
		}
		back := int(s.u16(4))
		ahead := 6 + 2*back
		count := int(s.u16(ahead))
		subs := ahead + 2 + 2*count
		if !b.around(lk, i, i, func(n int) data { return s.sub(6 + 2*n) }, back,
			func(n int) data { return s.sub(ahead + 2 + 2*n) }, count) || k >= int(s.u16(subs)) {
			continue
		}
		b.glyphs[i].ID = s.u16(subs + 2 + 2*k)
		return
	}
}

// around reports whether the back glyphs before first and the ahead
// glyphs after last, lk passing over those it ignores, are covered by
// the coverage tables backCov and aheadCov give, the first back one
// nearest first.
func (b *buffer) around(lk *lookup, first, last int, backCov func(int) data, back int,
	aheadCov func(int) data, ahead int) bool {
	j := first
	for n := range back {
		if j = b.prev(lk, j); j < 0 || backCov(n).coverage(b.glyphs[j].ID) < 0 {
			return false
		}
	}
	j = last
	for n := range ahead {
		if j = b.next(lk, j); j < 0 || aheadCov(n).coverage(b.glyphs[j].ID) < 0 {
			return false
		}
	}
	return true
}
