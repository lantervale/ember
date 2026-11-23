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
	first := b.glyphs[i]
	for n := range int(set.u16(0)) {
		lig := set.sub(2 + 2*n)
		count := int(lig.u16(2))
		if count == 0 {
			continue
		}
		at := make([]int, 1, count)
		at[0] = i
		j := i
		for c := 1; c < count; c++ {
			j = b.next(lk, j)
			if j < 0 || b.glyphs[j].ID != lig.u16(4+2*(c-1)) || b.glyphs[j].mask&b.mask == 0 {
				break
			}
			// A mark on a component of a ligature joins only marks on the same one.
			g := b.glyphs[j]
			if first.lig != 0 && compOf(first) != 0 {
				if g.lig != first.lig || compOf(g) != compOf(first) {
					break
				}
			} else if g.lig != 0 && compOf(g) != 0 && g.lig != first.lig {
				break
			}
			at = append(at, j)
		}
		if len(at) != count {
			continue
		}
		b.ligate(at, lig.u16(0))
		return true
	}
	return false
}

// ligate makes the glyphs at at, in order, ligature id, the glyphs it
// passed over between them kept after it, and has the marks among
// those and after it sit on the components they sat on, as HarfBuzz
// numbers them.
func (b *buffer) ligate(at []int, id uint16) {
	marks := true
	for _, k := range at[1:] {
		marks = marks && b.f.isMark(b.glyphs[k].ID)
	}
	first := &b.glyphs[at[0]]
	baseLig := marks && b.f.glyphClasses.class(first.ID) == baseGlyph
	markLig := marks && b.f.isMark(first.ID)
	isLig := !baseLig && !markLig
	var lig uint8
	if isLig {
		b.ligs++
		if b.ligs == 0 {
			b.ligs = 1
		}
		lig = b.ligs
	}
	var total uint8
	for _, k := range at {
		total += b.comps(b.glyphs[k])
	}
	lastLig, lastComps := first.lig, b.comps(*first)
	sofar := lastComps
	if isLig {
		first.lig, first.comp, first.ligated = lig, total, true
	}
	first.ID = id
	onto := func(g *Glyph) {
		this := compOf(*g)
		if this == 0 {
			this = lastComps
		}
		g.lig, g.comp, g.ligated = lig, sofar-lastComps+min(this, lastComps), false
	}
	for c := 1; c < len(at); c++ {
		if isLig {
			for k := at[c-1] + 1; k < at[c]; k++ {
				onto(&b.glyphs[k])
			}
		}
		g := b.glyphs[at[c]]
		lastLig, lastComps = g.lig, b.comps(g)
		sofar += lastComps
	}
	if !markLig && lastLig != 0 {
		for k := at[len(at)-1] + 1; k < len(b.glyphs); k++ {
			g := &b.glyphs[k]
			if g.lig != lastLig || compOf(*g) == 0 {
				break
			}
			onto(g)
		}
	}
	b.merge(at[0], at[len(at)-1])
	for c := len(at) - 1; c > 0; c-- {
		b.glyphs = append(b.glyphs[:at[c]], b.glyphs[at[c]+1:]...)
	}
}

// merge gives the glyphs from first to last, and those beside them of
// the same clusters, the first of their clusters, as HarfBuzz merges
// the clusters a ligature joins.
func (b *buffer) merge(first, last int) {
	lo, hi := b.glyphs[first].Cluster, b.glyphs[first].Cluster
	for k := first; k <= last; k++ {
		lo, hi = min(lo, b.glyphs[k].Cluster), max(hi, b.glyphs[k].Cluster)
	}
	for first > 0 && b.glyphs[first-1].Cluster >= lo && b.glyphs[first-1].Cluster <= hi {
		first--
	}
	for last+1 < len(b.glyphs) && b.glyphs[last+1].Cluster >= lo && b.glyphs[last+1].Cluster <= hi {
		last++
	}
	for k := first; k <= last; k++ {
		b.glyphs[k].Cluster = lo
	}
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
