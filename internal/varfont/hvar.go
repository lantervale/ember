package varfont

import (
	"encoding/binary"
	"errors"
)

var errStore = errors.New("varfont: an item variation store is wrong")

// ItemStore is an ItemVariationStore: regions of the axes, and sets of
// deltas over some of them.
type ItemStore struct {
	// regions are each region's start, peak and end on every axis.
	regions [][][3]float32
	data    []itemData
}

type itemData struct {
	regions []int
	// deltas are itemCount rows of len(regions) deltas.
	deltas []float32
}

// ReadItemStore reads the store at at in b, as GDEF holds GPOS's: over
// as many axes as its regions say.
func ReadItemStore(b []byte, at int) (*ItemStore, error) { return readItemStore(b, at, -1) }

// readItemStore reads the store at at in b, over axes axes, any when
// negative.
func readItemStore(b []byte, at, axes int) (*ItemStore, error) {
	be := binary.BigEndian
	if at <= 0 || at+8 > len(b) {
		return nil, errStore
	}
	s := b[at:]
	regionsAt, count := int(be.Uint32(s[2:])), int(be.Uint16(s[6:]))
	if 8+4*count > len(s) || regionsAt+4 > len(s) {
		return nil, errStore
	}
	r := s[regionsAt:]
	if axes < 0 {
		axes = int(be.Uint16(r))
	}
	if int(be.Uint16(r)) != axes {
		return nil, errStore
	}
	regionCount := int(be.Uint16(r[2:]))
	if 4+6*axes*regionCount > len(r) {
		return nil, errStore
	}
	store := &ItemStore{}
	for i := range regionCount {
		region := make([][3]float32, axes)
		for j := range axes {
			p := r[4+6*(i*axes+j):]
			region[j] = [3]float32{f2dot14(p), f2dot14(p[2:]), f2dot14(p[4:])}
		}
		store.regions = append(store.regions, region)
	}
	for i := range count {
		d, err := readItemData(s, int(be.Uint32(s[8+4*i:])), regionCount)
		if err != nil {
			return nil, err
		}
		store.data = append(store.data, d)
	}
	return store, nil
}

func readItemData(s []byte, at, regionCount int) (itemData, error) {
	be := binary.BigEndian
	if at+6 > len(s) {
		return itemData{}, errStore
	}
	d := s[at:]
	items, words, n := int(be.Uint16(d)), int(be.Uint16(d[2:])), int(be.Uint16(d[4:]))
	long := words&0x8000 != 0
	words &= 0x7fff
	if 6+2*n > len(d) || words > n {
		return itemData{}, errStore
	}
	out := itemData{regions: make([]int, n), deltas: make([]float32, 0, items*n)}
	for i := range n {
		out.regions[i] = int(be.Uint16(d[6+2*i:]))
		if out.regions[i] >= regionCount {
			return itemData{}, errStore
		}
	}
	big, small := 2, 1
	if long {
		big, small = 4, 2
	}
	row := words*big + (n-words)*small
	at = 6 + 2*n
	if at+items*row > len(d) {
		return itemData{}, errStore
	}
	for range items {
		for j := range n {
			size := small
			if j < words {
				size = big
			}
			var v float32
			switch size {
			case 1:
				v = float32(int8(d[at])) //nolint:gosec // signed
			case 2:
				v = float32(int16(be.Uint16(d[at:]))) //nolint:gosec // signed
			default:
				v = float32(int32(be.Uint32(d[at:]))) //nolint:gosec // signed
			}
			out.deltas = append(out.deltas, v)
			at += size
		}
	}
	return out, nil
}

// Delta is item inner of set outer's delta at c.
func (s *ItemStore) Delta(outer, inner int, c Coords) float32 {
	if outer >= len(s.data) {
		return 0
	}
	d := s.data[outer]
	n := len(d.regions)
	if n == 0 || (inner+1)*n > len(d.deltas) {
		return 0
	}
	var sum float32
	for j, r := range d.regions {
		if v := d.deltas[inner*n+j]; v != 0 {
			sum += v * regionScalar(s.regions[r], c)
		}
	}
	return sum
}

// regionScalar is how much of a region's deltas apply at c.
func regionScalar(region [][3]float32, c Coords) float32 {
	s := float32(1)
	for i, a := range region {
		lo, peak, hi := a[0], a[1], a[2]
		if peak == 0 || lo > peak || peak > hi || (lo < 0 && hi > 0) {
			continue
		}
		var v float32
		if i < len(c) {
			v = c[i]
		}
		switch {
		case v == peak:
		case v <= lo || v >= hi:
			return 0
		case v < peak:
			s *= (v - lo) / (peak - lo)
		default:
			s *= (hi - v) / (hi - peak)
		}
	}
	return s
}

// deltaSetMap maps a glyph to its outer and inner indices in a store;
// none maps glyph g to (0, g).
type deltaSetMap struct {
	entries   []uint32
	innerBits uint
}

func readDeltaSetMap(b []byte, at int) deltaSetMap {
	be := binary.BigEndian
	if at <= 0 || at+4 > len(b) {
		return deltaSetMap{}
	}
	m := b[at:]
	format, entry := m[0], m[1]
	var count int
	at = 4
	if format == 0 {
		count = int(be.Uint16(m[2:]))
	} else {
		if len(m) < 6 {
			return deltaSetMap{}
		}
		count, at = int(be.Uint32(m[2:])), 6
	}
	size := int(entry>>4&3) + 1
	if at+count*size > len(m) {
		return deltaSetMap{}
	}
	out := deltaSetMap{innerBits: uint(entry&0x0f) + 1, entries: make([]uint32, count)}
	for i := range count {
		var v uint32
		for k := range size {
			v = v<<8 | uint32(m[at+i*size+k])
		}
		out.entries[i] = v
	}
	return out
}

func (m deltaSetMap) at(g int) (int, int) {
	if len(m.entries) == 0 {
		return 0, g
	}
	v := m.entries[min(g, len(m.entries)-1)]
	return int(v >> m.innerBits), int(v & (1<<m.innerBits - 1))
}
