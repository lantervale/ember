package varfont

import (
	"encoding/binary"
	"errors"
)

var errGvar = errors.New("varfont: gvar is wrong")

// gvar is the table of the glyphs' variations: the peaks tuples share,
// and each glyph's variation data.
type gvar struct {
	axes   int
	shared [][]float32
	data   []byte
	// offsets are where each glyph's data starts in data, and the last
	// where the last ends.
	offsets []int
}

func readGvar(b []byte, axes, glyphs int) (gvar, error) {
	be := binary.BigEndian
	if len(b) < 20 {
		return gvar{}, errGvar
	}
	if int(be.Uint16(b[4:])) != axes {
		return gvar{}, errGvar
	}
	sharedCount, sharedAt := int(be.Uint16(b[6:])), int(be.Uint32(b[8:]))
	count, long, dataAt := int(be.Uint16(b[12:])), be.Uint16(b[14:])&1 != 0, int(be.Uint32(b[16:]))
	if count != glyphs || sharedAt+2*axes*sharedCount > len(b) || dataAt > len(b) {
		return gvar{}, errGvar
	}
	g := gvar{axes: axes, data: b[dataAt:], offsets: make([]int, count+1)}
	for i := range sharedCount {
		peak := make([]float32, axes)
		for j := range axes {
			peak[j] = f2dot14(b[sharedAt+2*(i*axes+j):])
		}
		g.shared = append(g.shared, peak)
	}
	size := 2
	if long {
		size = 4
	}
	if 20+size*(count+1) > len(b) {
		return gvar{}, errGvar
	}
	for i := range count + 1 {
		if long {
			g.offsets[i] = int(be.Uint32(b[20+4*i:]))
		} else {
			g.offsets[i] = 2 * int(be.Uint16(b[20+2*i:]))
		}
		if g.offsets[i] > len(g.data) || (i > 0 && g.offsets[i] < g.offsets[i-1]) {
			return gvar{}, errGvar
		}
	}
	return g, nil
}

const (
	sharedPointNumbers = 0x8000
	tupleCountMask     = 0x0fff
	embeddedPeak       = 0x8000
	intermediateRegion = 0x4000
	privatePointNumber = 0x2000
	tupleIndexMask     = 0x0fff
)

// vary moves o's points and phantom points by g's variations at c, the
// points no tuple names inferred from those it does, contour by
// contour. A glyph whose data is wrong stays as it is.
func (f *Font) vary(g uint16, c Coords, o *outline) {
	gv := &f.gvar
	if c == nil || gv.offsets == nil || int(g) >= len(gv.offsets)-1 {
		return
	}
	data := gv.data[gv.offsets[g]:gv.offsets[g+1]]
	if len(data) < 4 {
		return
	}
	n := len(o.points) + 4
	dx, dy := make([]float32, n), make([]float32, n)
	if !gv.apply(data, c, n, o, dx, dy) {
		return
	}
	for i := range o.points {
		o.points[i].X += dx[i]
		o.points[i].Y += dy[i]
	}
	for i := range o.phantom {
		o.phantom[i].x += dx[len(o.points)+i]
		o.phantom[i].y += dy[len(o.points)+i]
	}
}

// apply adds each tuple's deltas at c, scaled by how far c is in its
// region, to dx and dy; false when the data is wrong.
func (gv *gvar) apply(data []byte, c Coords, n int, o *outline, dx, dy []float32) bool {
	be := binary.BigEndian
	head := be.Uint16(data)
	tuples, serialAt := int(head&tupleCountMask), int(be.Uint16(data[2:]))
	if serialAt > len(data) {
		return false
	}
	serial := data[serialAt:]
	var shared []int
	if head&sharedPointNumbers != 0 {
		var ok bool
		if shared, serial, ok = pointNumbers(serial); !ok {
			return false
		}
	}
	at := 4
	peak, start, end := make([]float32, gv.axes), make([]float32, gv.axes), make([]float32, gv.axes)
	tx, ty := make([]float32, n), make([]float32, n)
	touched := make([]bool, n)
	for range tuples {
		if at+4 > len(data) {
			return false
		}
		size, index := int(be.Uint16(data[at:])), be.Uint16(data[at+2:])
		at += 4
		if index&embeddedPeak != 0 {
			if at+2*gv.axes > len(data) {
				return false
			}
			for j := range peak {
				peak[j] = f2dot14(data[at+2*j:])
			}
			at += 2 * gv.axes
		} else {
			k := int(index & tupleIndexMask)
			if k >= len(gv.shared) {
				return false
			}
			copy(peak, gv.shared[k])
		}
		inter := index&intermediateRegion != 0
		if inter {
			if at+4*gv.axes > len(data) {
				return false
			}
			for j := range start {
				start[j], end[j] = f2dot14(data[at+2*j:]), f2dot14(data[at+2*(gv.axes+j):])
			}
			at += 4 * gv.axes
		}
		if size > len(serial) {
			return false
		}
		body := serial[:size]
		serial = serial[size:]
		scalar := tupleScalar(c, peak, start, end, inter)
		if scalar == 0 {
			continue
		}
		points := shared
		if index&privatePointNumber != 0 {
			var ok bool
			if points, body, ok = pointNumbers(body); !ok {
				return false
			}
		}
		count := n
		if points != nil {
			count = len(points)
		}
		xs, body, ok := deltas(body, count)
		if !ok {
			return false
		}
		ys, _, ok := deltas(body, count)
		if !ok {
			return false
		}
		clear(tx)
		clear(ty)
		clear(touched)
		for i := range count {
			p := i
			if points != nil {
				p = points[i]
			}
			if p >= n {
				continue
			}
			tx[p], ty[p], touched[p] = xs[i], ys[i], true
		}
		if points != nil && len(o.ends) > 0 {
			infer(o.points, o.ends, tx, ty, touched)
		}
		for i := range n {
			dx[i] += scalar * tx[i]
			dy[i] += scalar * ty[i]
		}
	}
	return true
}

// tupleScalar is how much of a tuple's deltas apply at c: 1 at its
// peak, falling to 0 at the edges of its region, which runs from 0 to
// the peak on each axis unless it is intermediate (start to end).
func tupleScalar(c Coords, peak, start, end []float32, inter bool) float32 {
	s := float32(1)
	for i, p := range peak {
		if p == 0 {
			continue
		}
		var v float32
		if i < len(c) {
			v = c[i]
		}
		if v == p {
			continue
		}
		if inter {
			lo, hi := start[i], end[i]
			if lo > p || p > hi || (lo < 0 && hi > 0) {
				continue
			}
			if v < lo || v > hi {
				return 0
			}
			if v < p {
				s *= (v - lo) / (p - lo)
			} else {
				s *= (hi - v) / (hi - p)
			}
			continue
		}
		if v == 0 || v < min(0, p) || v > max(0, p) {
			return 0
		}
		s *= v / p
	}
	return s
}

// pointNumbers reads packed point numbers: nil for all the points.
func pointNumbers(b []byte) ([]int, []byte, bool) {
	if len(b) < 1 {
		return nil, b, false
	}
	count := int(b[0])
	b = b[1:]
	if count&0x80 != 0 {
		if len(b) < 1 {
			return nil, b, false
		}
		count = (count&0x7f)<<8 | int(b[0])
		b = b[1:]
	}
	if count == 0 {
		return nil, b, true
	}
	out := make([]int, 0, count)
	last := 0
	for len(out) < count {
		if len(b) < 1 {
			return nil, b, false
		}
		ctl := b[0]
		b = b[1:]
		run := int(ctl&0x7f) + 1
		words := ctl&0x80 != 0
		for k := 0; k < run && len(out) < count; k++ {
			var d int
			if words {
				if len(b) < 2 {
					return nil, b, false
				}
				d = int(binary.BigEndian.Uint16(b))
				b = b[2:]
			} else {
				if len(b) < 1 {
					return nil, b, false
				}
				d = int(b[0])
				b = b[1:]
			}
			last += d
			out = append(out, last)
		}
	}
	return out, b, true
}

// deltas reads count packed deltas.
func deltas(b []byte, count int) ([]float32, []byte, bool) {
	out := make([]float32, 0, count)
	for len(out) < count {
		if len(b) < 1 {
			return nil, b, false
		}
		ctl := b[0]
		b = b[1:]
		run := int(ctl&0x3f) + 1
		for k := 0; k < run && len(out) < count; k++ {
			switch ctl & 0xc0 {
			case 0x80:
				out = append(out, 0)
			case 0x40:
				if len(b) < 2 {
					return nil, b, false
				}
				out = append(out, float32(int16(binary.BigEndian.Uint16(b)))) //nolint:gosec // signed
				b = b[2:]
			case 0xc0:
				if len(b) < 4 {
					return nil, b, false
				}
				out = append(out, float32(int32(binary.BigEndian.Uint32(b)))) //nolint:gosec // signed
				b = b[4:]
			default:
				if len(b) < 1 {
					return nil, b, false
				}
				out = append(out, float32(int8(b[0]))) //nolint:gosec // signed
				b = b[1:]
			}
		}
	}
	return out, b, true
}

// infer gives each point of points a tuple names none of the deltas of
// the touched points beside it in its contour: between two, in
// proportion along each axis, past them, the nearer's; a contour with
// one touched point moves whole with it.
func infer(points []Point, ends []int, dx, dy []float32, touched []bool) {
	start := 0
	for _, end := range ends {
		if end >= len(points) {
			return
		}
		first, last := -1, -1
		for i := start; i <= end; i++ {
			if touched[i] {
				if first < 0 {
					first = i
				}
				last = i
			}
		}
		if first < 0 {
			start = end + 1
			continue
		}
		if first == last {
			for i := start; i <= end; i++ {
				if i != first {
					dx[i], dy[i] = dx[first], dy[first]
				}
			}
			start = end + 1
			continue
		}
		for i := first; ; {
			next := i + 1
			if next > end {
				next = start
			}
			for !touched[next] {
				next++
				if next > end {
					next = start
				}
			}
			between(points, start, end, i, next, dx, dy)
			i = next
			if i == first {
				break
			}
		}
		start = end + 1
	}
}

// between infers the points after a and before b, touched both, round
// the contour from start to end.
func between(points []Point, start, end, a, b int, dx, dy []float32) {
	for i := a + 1; ; i++ {
		if i > end {
			i = start
		}
		if i == b {
			return
		}
		dx[i] = interp(points[a].X, points[b].X, dx[a], dx[b], points[i].X)
		dy[i] = interp(points[a].Y, points[b].Y, dy[a], dy[b], points[i].Y)
	}
}

// interp is the delta of a point at v, between points at c1 and c2
// moved by d1 and d2.
func interp(c1, c2, d1, d2, v float32) float32 {
	if c1 == c2 {
		if d1 == d2 {
			return d1
		}
		return 0
	}
	if c1 > c2 {
		c1, c2, d1, d2 = c2, c1, d2, d1
	}
	switch {
	case v <= c1:
		return d1
	case v >= c2:
		return d2
	}
	return d1 + (v-c1)*(d2-d1)/(c2-c1)
}
