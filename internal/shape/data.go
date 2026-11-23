package shape

import "sort"

// data is a part of a font file. Every read is checked: a font is a
// page's and may lie, so what lies outside reads as 0, and an offset
// out of range as no table.
type data []byte

func (d data) u16(at int) uint16 {
	if at < 0 || at+2 > len(d) {
		return 0
	}
	return uint16(d[at])<<8 | uint16(d[at+1])
}

func (d data) u32(at int) uint32 {
	if at < 0 || at+4 > len(d) {
		return 0
	}
	return uint32(d[at])<<24 | uint32(d[at+1])<<16 | uint32(d[at+2])<<8 | uint32(d[at+3])
}

// at is the table at offset off of d, nil for a null or bad offset.
func (d data) at(off int) data {
	if off <= 0 || off >= len(d) {
		return nil
	}
	return d[off:]
}

// sub is the table the 16-bit offset at at points to.
func (d data) sub(at int) data { return d.at(int(d.u16(at))) }

// coverage is g's index in coverage table d, -1 when d does not cover it.
func (d data) coverage(g uint16) int {
	switch d.u16(0) {
	case 1:
		n := int(d.u16(2))
		i := sort.Search(n, func(i int) bool { return d.u16(4+2*i) >= g })
		if i < n && d.u16(4+2*i) == g {
			return i
		}
	case 2:
		n := int(d.u16(2))
		i := sort.Search(n, func(i int) bool { return d.u16(4+6*i+2) >= g })
		if i < n && d.u16(4+6*i) <= g {
			return int(d.u16(4+6*i+4)) + int(g-d.u16(4+6*i))
		}
	}
	return -1
}

// eachCovered calls fn with every glyph coverage table d covers.
func (d data) eachCovered(fn func(g uint16)) {
	switch d.u16(0) {
	case 1:
		for i := range int(d.u16(2)) {
			fn(d.u16(4 + 2*i))
		}
	case 2:
		for i := range int(d.u16(2)) {
			lo, hi := d.u16(4+6*i), d.u16(4+6*i+2)
			for g := int(lo); g <= int(hi); g++ {
				fn(uint16(g)) //nolint:gosec // at most hi, a 16-bit glyph
			}
		}
	}
}

// class is g's class in class definition table d, 0 for none.
func (d data) class(g uint16) uint16 {
	switch d.u16(0) {
	case 1:
		start := d.u16(2)
		if g >= start && int(g-start) < int(d.u16(4)) {
			return d.u16(6 + 2*int(g-start))
		}
	case 2:
		n := int(d.u16(2))
		i := sort.Search(n, func(i int) bool { return d.u16(4+6*i+2) >= g })
		if i < n && d.u16(4+6*i) <= g {
			return d.u16(4 + 6*i + 4)
		}
	}
	return 0
}

// eachClassed calls fn with every glyph class table d gives a class
// other than 0.
func (d data) eachClassed(fn func(g uint16)) {
	switch d.u16(0) {
	case 1:
		start := d.u16(2)
		for i := range int(d.u16(4)) {
			if d.u16(6+2*i) != 0 {
				fn(start + uint16(i))
			}
		}
	case 2:
		for i := range int(d.u16(2)) {
			if d.u16(4+6*i+4) == 0 {
				continue
			}
			for g := int(d.u16(4 + 6*i)); g <= int(d.u16(4+6*i+2)); g++ {
				fn(uint16(g)) //nolint:gosec // a glyph of a 16-bit range
			}
		}
	}
}

// glyphSet is a set of glyphs, one bit each.
type glyphSet []uint64

func (s glyphSet) has(g uint16) bool {
	i := int(g >> 6)
	return i < len(s) && s[i]&(1<<(g&63)) != 0
}

func (s *glyphSet) add(g uint16) {
	i := int(g >> 6)
	if i >= len(*s) {
		*s = append(*s, make(glyphSet, i+1-len(*s))...)
	}
	(*s)[i] |= 1 << (g & 63)
}

func (s *glyphSet) union(o glyphSet) {
	if len(o) > len(*s) {
		*s = append(*s, make(glyphSet, len(o)-len(*s))...)
	}
	for i, v := range o {
		(*s)[i] |= v
	}
}
