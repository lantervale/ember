package shape

// context applies the first rule of contextual subtable s matching the
// glyphs from i, and reports where the glyphs after its input start.
func (b *buffer) context(lk *lookup, s data, i int) (int, bool) {
	first := b.glyphs[i].ID
	switch s.u16(0) {
	case 1, 2:
		k := s.sub(2).coverage(first)
		if k < 0 {
			return i, false
		}
		classes := s.sub(4)
		sets, at := 4, 6
		if s.u16(0) == 2 {
			k = int(classes.class(first))
			sets, at = 6, 8
		}
		if k >= int(s.u16(sets)) {
			return i, false
		}
		set := s.sub(at + 2*k)
		for n := range int(set.u16(0)) {
			rule := set.sub(2 + 2*n)
			count := int(rule.u16(0))
			pos := b.input(lk, i, count, func(c int, g uint16) bool {
				if s.u16(0) == 2 {
					return classes.class(g) == rule.u16(4+2*(c-1))
				}
				return g == rule.u16(4+2*(c-1))
			})
			if pos != nil {
				return b.nested(lk, pos, rule.at(4+2*(count-1)), int(rule.u16(2))), true
			}
		}
	case 3:
		count := int(s.u16(2))
		pos := b.input(lk, i, count, func(c int, g uint16) bool { return s.sub(6+2*c).coverage(g) >= 0 })
		if pos != nil && s.sub(6).coverage(first) >= 0 {
			return b.nested(lk, pos, s.at(6+2*count), int(s.u16(4))), true
		}
	}
	return i, false
}

// chain applies the first rule of chained contextual subtable s whose
// backtrack, input from i and lookahead match, and reports where the
// glyphs after its input start.
func (b *buffer) chain(lk *lookup, s data, i int) (int, bool) {
	first := b.glyphs[i].ID
	switch format := s.u16(0); format {
	case 1, 2:
		k := s.sub(2).coverage(first)
		if k < 0 {
			return i, false
		}
		backs, inputs, aheads := s.sub(4), s.sub(6), s.sub(8)
		sets, at := 4, 6
		if format == 2 {
			k = int(inputs.class(first))
			sets, at = 10, 12
		}
		if k >= int(s.u16(sets)) {
			return i, false
		}
		is := func(set data, c uint16, g uint16) bool {
			if format == 2 {
				return set.class(g) == c
			}
			return g == c
		}
		set := s.sub(at + 2*k)
		for n := range int(set.u16(0)) {
			rule := set.sub(2 + 2*n)
			back := int(rule.u16(0))
			in := 2 + 2*back
			count := int(rule.u16(in))
			ahead := in + 2 + 2*max(count-1, 0)
			after := int(rule.u16(ahead))
			recs := ahead + 2 + 2*after
			pos := b.input(lk, i, count, func(c int, g uint16) bool { return is(inputs, rule.u16(in+2*c), g) })
			if pos == nil || !b.contextAround(lk, pos,
				func(n int, g uint16) bool { return is(backs, rule.u16(2+2*n), g) }, back,
				func(n int, g uint16) bool { return is(aheads, rule.u16(ahead+2+2*n), g) }, after) {
				continue
			}
			return b.nested(lk, pos, rule.at(recs+2), int(rule.u16(recs))), true
		}
	case 3:
		back := int(s.u16(2))
		in := 4 + 2*back
		count := int(s.u16(in))
		ahead := in + 2 + 2*count
		after := int(s.u16(ahead))
		recs := ahead + 2 + 2*after
		if s.sub(in+2).coverage(first) < 0 {
			return i, false
		}
		pos := b.input(lk, i, count, func(c int, g uint16) bool { return s.sub(in+2+2*c).coverage(g) >= 0 })
		if pos == nil || !b.contextAround(lk, pos,
			func(n int, g uint16) bool { return s.sub(4+2*n).coverage(g) >= 0 }, back,
			func(n int, g uint16) bool { return s.sub(ahead+2+2*n).coverage(g) >= 0 }, after) {
			return i, false
		}
		return b.nested(lk, pos, s.at(recs+2), int(s.u16(recs))), true
	}
	return i, false
}

// input is where the count glyphs from i lie, lk passing over those it
// ignores, when each after the first matches and has the lookup's mask;
// nil when they don't.
func (b *buffer) input(lk *lookup, i, count int, match func(c int, g uint16) bool) []int {
	if count <= 0 {
		return nil
	}
	pos := make([]int, 1, count)
	pos[0] = i
	j := i
	for c := 1; c < count; c++ {
		if j = b.next(lk, j); j < 0 || b.glyphs[j].mask&b.mask == 0 || !match(c, b.glyphs[j].ID) {
			return nil
		}
		pos = append(pos, j)
	}
	return pos
}

// contextAround reports whether the back glyphs before pos and the
// after glyphs after it match.
func (b *buffer) contextAround(lk *lookup, pos []int, backMatch func(int, uint16) bool, back int,
	aheadMatch func(int, uint16) bool, after int) bool {
	j := pos[0]
	for n := range back {
		if j = b.prev(lk, j); j < 0 || !backMatch(n, b.glyphs[j].ID) {
			return false
		}
	}
	j = pos[len(pos)-1]
	for n := range after {
		if j = b.next(lk, j); j < 0 || !aheadMatch(n, b.glyphs[j].ID) {
			return false
		}
	}
	return true
}

// nested applies the count lookup records at recs to the input glyphs
// at pos, and reports where the glyphs after the input start.
func (b *buffer) nested(lk *lookup, pos []int, recs data, count int) int {
	end := pos[len(pos)-1] + 1
	if b.depth >= maxDepth {
		return end
	}
	l := &b.f.gsub
	if lk.positions {
		l = &b.f.gpos
	}
	b.depth++
	defer func() { b.depth-- }()
	for r := range count {
		seq, index := int(recs.u16(4*r)), recs.u16(4*r+2)
		if seq >= len(pos) || pos[seq] >= len(b.glyphs) {
			continue
		}
		inner := l.lookup(index)
		if inner == nil || b.f.ignores(inner, b.glyphs[pos[seq]].ID) {
			continue
		}
		n := len(b.glyphs)
		b.apply(inner, 1, pos[seq])
		if d := len(b.glyphs) - n; d != 0 {
			for k := seq + 1; k < len(pos); k++ {
				pos[k] += d
			}
			end += d
		}
	}
	return end
}
