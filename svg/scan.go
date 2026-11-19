package svg

import (
	"image"
	"math"
	"slices"
)

// subScanlines is how many lines each row of pixels is sampled on. Along
// a line the coverage is exact; down a row it is the share of lines.
const subScanlines = 1 << subScanlineShift

const subScanlineShift = 4

// scanEdge is an edge of a polygon in pixels, as the lines it crosses: x
// where it meets line k0, dx more for each line down to k1, which it
// does not reach. dir is +1 for an edge going down, -1 up.
type scanEdge struct {
	x, dx  float64
	k0, k1 int32
	dir    int32
}

// coverage finds how much of each pixel the polygons it is given cover,
// under either fill rule. It counts how often the outline winds round
// each sample, so a part covered twice is covered, no more: outlines may
// overlap themselves freely.
type coverage struct {
	clip   image.Rectangle
	edges  []scanEdge
	active []scanEdge
	next   int // the first edge not yet under way
	// acc is each pixel's partial cover on the row under way, and diff the
	// steps of full cover, summed along the row.
	acc, diff []float32
	row       []float32
}

func (c *coverage) reset(clip image.Rectangle) {
	c.clip = clip
	c.edges = c.edges[:0]
}

// add takes the closed polygon pts, moved by m.
func (c *coverage) add(pts []Point, m Matrix) {
	if len(pts) < 3 {
		return
	}
	lo, hi := float64(c.clip.Min.Y*subScanlines), float64(c.clip.Max.Y*subScanlines)
	a := m.Apply(pts[len(pts)-1])
	for _, p := range pts {
		b := m.Apply(p)
		from, to, dir := a, b, int32(1)
		a = b
		if from.Y > to.Y {
			from, to, dir = to, from, -1
		}
		// Line k samples at (k + ½) / subScanlines; an edge owns the
		// lines from its top, included, to its bottom, excluded.
		k0 := math.Ceil(from.Y*subScanlines - 0.5)
		k1 := math.Ceil(to.Y*subScanlines - 0.5)
		if !(k0 < k1) || !(k1 > lo) || !(k0 < hi) { // false for NaN
			continue
		}
		dx := (to.X - from.X) / (to.Y - from.Y) / subScanlines
		x := from.X + ((k0+0.5)/subScanlines-from.Y)*(to.X-from.X)/(to.Y-from.Y)
		if k0 < lo {
			x += (lo - k0) * dx
			k0 = lo
		}
		if math.IsNaN(x) || math.IsNaN(dx) || math.IsInf(x, 0) || math.IsInf(dx, 0) {
			continue
		}
		c.edges = append(c.edges, scanEdge{x: x, dx: dx, k0: int32(k0), k1: int32(math.Min(k1, hi)), dir: dir})
	}
}

// line samples line k: it takes the edges that start there, drops those
// that ended, and adds what lies between the others to the row's cover.
// It returns the first and last pixel touched, given those so far.
func (c *coverage) line(k int32, evenOdd bool, lo, hi int) (int, int) {
	for c.next < len(c.edges) && c.edges[c.next].k0 <= k {
		c.active = append(c.active, c.edges[c.next])
		c.next++
	}
	// Edges cross seldom from one line to the next, so the ones under way
	// stay nearly sorted along x.
	active := c.active
	keep := 0
	for i := range active {
		e := active[i]
		if e.k1 <= k {
			continue
		}
		j := keep
		for ; j > 0 && active[j-1].x > e.x; j-- {
			active[j] = active[j-1]
		}
		active[j] = e
		keep++
	}
	active = active[:keep]
	c.active = active
	left, width := float64(c.clip.Min.X), float64(c.clip.Dx())
	winding := int32(0)
	for i := range active {
		e := &active[i]
		from := e.x
		e.x += e.dx
		winding += e.dir
		if evenOdd && winding&1 == 0 || winding == 0 || i+1 == len(active) {
			continue
		}
		a, b := from-left, active[i+1].x-left
		if a < 0 {
			a = 0
		}
		if b > width {
			b = width
		}
		if !(a < b) {
			continue
		}
		ia, ib := int(a), int(b)
		lo, hi = min(lo, ia), max(hi, ib+1)
		if ia == ib {
			c.acc[ia] += float32(b - a)
			continue
		}
		c.acc[ia] += float32(float64(ia+1) - a)
		c.acc[ib] += float32(b - float64(ib))
		c.diff[ia+1]++
		c.diff[ib]--
	}
	return lo, hi
}

// fill gives emit each row the polygons cover: y, the first pixel x0,
// and the cover, 0–1, of the pixels from there.
func (c *coverage) fill(evenOdd bool, emit func(y, x0 int, cover []float32)) {
	if len(c.edges) == 0 {
		return
	}
	slices.SortFunc(c.edges, func(a, b scanEdge) int { return int(a.k0) - int(b.k0) })
	w := c.clip.Dx()
	if cap(c.acc) < w+1 {
		c.acc, c.diff, c.row = make([]float32, w+1), make([]float32, w+2), make([]float32, w)
	}
	c.acc, c.diff = c.acc[:w+1], c.diff[:w+2]
	c.next, c.active = 0, c.active[:0]
	for y := int(c.edges[0].k0) >> subScanlineShift; y < c.clip.Max.Y; y++ {
		if len(c.active) == 0 {
			if c.next == len(c.edges) {
				return
			}
			y = max(y, int(c.edges[c.next].k0)>>subScanlineShift)
		}
		lo, hi := w, 0
		for sub := range subScanlines {
			lo, hi = c.line(int32(y<<subScanlineShift+sub), evenOdd, lo, hi) //nolint:gosec // rows are within an image
		}
		if lo >= hi {
			continue
		}
		hi = min(hi, w)
		row := c.row[:hi-lo]
		full := float32(0)
		for i := range row {
			x := lo + i
			full += c.diff[x]
			row[i] = min((c.acc[x]+full)/subScanlines, 1)
			c.acc[x], c.diff[x] = 0, 0
		}
		c.acc[hi], c.diff[hi], c.diff[hi+1] = 0, 0, 0
		emit(y, c.clip.Min.X+lo, row)
	}
}
