package shape

// wouldSubstitute reports whether one of lks would substitute glyphs,
// in that order and nothing else, as HarfBuzz asks of a feature's
// lookups: zeroContext leaves out a chained rule that looks before or
// after them.
func (f *Font) wouldSubstitute(lks []*lookup, glyphs []uint16, zeroContext bool) bool {
	for _, lk := range lks {
		if len(glyphs) == 0 || !lk.first.has(glyphs[0]) {
			continue
		}
		for _, s := range lk.subs {
			if lk.would(s, glyphs, zeroContext) {
				return true
			}
		}
	}
	return false
}

// would reports whether subtable s of lk would apply to glyphs.
//
//nolint:gocognit,gocyclo,cyclop,funlen // a case for each kind and format of lookup
func (lk *lookup) would(s data, glyphs []uint16, zeroContext bool) bool {
	n := len(glyphs)
	first := glyphs[0]
	rest := func(count int, match func(c int, g uint16) bool) bool {
		if count != n {
			return false
		}
		for c := 1; c < count; c++ {
			if !match(c, glyphs[c]) {
				return false
			}
		}
		return true
	}
	switch lk.kind {
	case single, multiple, alternate, reverse:
		return n == 1 && s.sub(2).coverage(first) >= 0
	case ligature:
		k := s.sub(2).coverage(first)
		if s.u16(0) != 1 || k < 0 || k >= int(s.u16(4)) {
			return false
		}
		set := s.sub(6 + 2*k)
		for i := range int(set.u16(0)) {
			lig := set.sub(2 + 2*i)
			if rest(int(lig.u16(2)), func(c int, g uint16) bool { return g == lig.u16(4+2*(c-1)) }) {
				return true
			}
		}
	case context:
		switch format := s.u16(0); format {
		case 1, 2:
			k, sets, at := s.sub(2).coverage(first), 4, 6
			classes := s.sub(4)
			if format == 2 {
				k, sets, at = int(classes.class(first)), 6, 8
			}
			if k < 0 || k >= int(s.u16(sets)) {
				return false
			}
			set := s.sub(at + 2*k)
			for i := range int(set.u16(0)) {
				rule := set.sub(2 + 2*i)
				if rest(int(rule.u16(0)), func(c int, g uint16) bool {
					if format == 2 {
						return classes.class(g) == rule.u16(4+2*(c-1))
					}
					return g == rule.u16(4+2*(c-1))
				}) {
					return true
				}
			}
		case 3:
			return rest(int(s.u16(2)), func(c int, g uint16) bool { return s.sub(6+2*c).coverage(g) >= 0 })
		}
	case chain:
		switch format := s.u16(0); format {
		case 1, 2:
			k, sets, at := s.sub(2).coverage(first), 4, 6
			inputs := s.sub(6)
			if format == 2 {
				k, sets, at = int(inputs.class(first)), 10, 12
			}
			if k < 0 || k >= int(s.u16(sets)) {
				return false
			}
			set := s.sub(at + 2*k)
			for i := range int(set.u16(0)) {
				rule := set.sub(2 + 2*i)
				back := int(rule.u16(0))
				in := 2 + 2*back
				count := int(rule.u16(in))
				after := int(rule.u16(in + 2 + 2*max(count-1, 0)))
				if zeroContext && (back != 0 || after != 0) {
					continue
				}
				if rest(count, func(c int, g uint16) bool {
					if format == 2 {
						return inputs.class(g) == rule.u16(in+2*c)
					}
					return g == rule.u16(in+2*c)
				}) {
					return true
				}
			}
		case 3:
			back := int(s.u16(2))
			in := 4 + 2*back
			count := int(s.u16(in))
			after := int(s.u16(in + 2 + 2*count))
			if zeroContext && (back != 0 || after != 0) {
				return false
			}
			return rest(count, func(c int, g uint16) bool { return s.sub(in+2+2*c).coverage(g) >= 0 })
		}
	}
	return false
}
