package outline

import (
	"cmp"
	"image"
	"math"
	"slices"

	"github.com/lantervale/ember/geom"
)

// subRows is how many scanlines a row of pixels is sampled at.
const subRows = 16

// edge is one segment of an outline in pixels, from top to bottom; dir
// is +1 where the outline runs down, -1 where it runs up.
type edge struct {
	x0, y0, x1, y1 float32
	dir            int
}

// crossing is where a scanline meets an edge.
type crossing struct {
	x   float32
	dir int
}

// Mask draws p into a w×h picture of r, white and opaque inside, clear
// outside, its edges smoothed: each row sampled at subRows scanlines,
// each scanline covering its pixels by the exact length inside, filled
// by the even-odd rule or else the non-zero one.
func (p Path) Mask(r geom.Rect, w, h int, evenOdd bool) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	if !(r.W > 0 && r.H > 0) {
		return out
	}
	s := scan{edges: p.edges(r, float32(w)/r.W, float32(h)/r.H), cover: make([]float32, w+1), evenOdd: evenOdd}
	for y := range h {
		clear(s.cover)
		for k := range subRows {
			s.line(float32(y)+(float32(k)+0.5)/subRows, w)
		}
		row := out.Pix[y*out.Stride:]
		for x := range w {
			v := uint8(math.Round(float64(min(s.cover[x]/subRows, 1)) * 255))
			row[4*x], row[4*x+1], row[4*x+2], row[4*x+3] = v, v, v, v
		}
	}
	return out
}

// edges are p's segments, scaled by sx and sy from r's corner, sorted
// by their tops; level ones are left out.
func (p Path) edges(r geom.Rect, sx, sy float32) []edge {
	var edges []edge
	for _, c := range p.Contours {
		for i := range c {
			a, b := c[i], c[(i+1)%len(c)]
			e := edge{(a.X - r.X) * sx, (a.Y - r.Y) * sy, (b.X - r.X) * sx, (b.Y - r.Y) * sy, 1}
			if e.y0 == e.y1 {
				continue
			}
			if e.y0 > e.y1 {
				e = edge{e.x1, e.y1, e.x0, e.y0, -1}
			}
			edges = append(edges, e)
		}
	}
	slices.SortFunc(edges, func(a, b edge) int { return cmp.Compare(a.y0, b.y0) })
	return edges
}

// scan is a mask's rows being covered: the edges, those a scanline
// meets, and the row's coverage summed over its scanlines.
type scan struct {
	edges, active []edge
	next          int
	xs            []crossing
	cover         []float32
	evenOdd       bool
}

// line covers the row by the scanline at y, across w pixels.
func (s *scan) line(y float32, w int) {
	for s.next < len(s.edges) && s.edges[s.next].y0 <= y {
		s.active = append(s.active, s.edges[s.next])
		s.next++
	}
	s.active = slices.DeleteFunc(s.active, func(e edge) bool { return e.y1 <= y })
	s.xs = s.xs[:0]
	for _, e := range s.active {
		if e.y0 <= y && y < e.y1 {
			s.xs = append(s.xs, crossing{e.x0 + (y-e.y0)/(e.y1-e.y0)*(e.x1-e.x0), e.dir})
		}
	}
	slices.SortFunc(s.xs, func(a, b crossing) int { return cmp.Compare(a.x, b.x) })
	winding := 0
	for i, c := range s.xs {
		winding += c.dir
		inside := winding != 0
		if s.evenOdd {
			inside = winding&1 != 0
		}
		if inside && i+1 < len(s.xs) {
			span(s.cover, c.x, s.xs[i+1].x, w)
		}
	}
}

// span adds to cover, pixel by pixel, how much of each the run from a to
// b covers on one scanline.
func span(cover []float32, a, b float32, w int) {
	a, b = max(a, 0), min(b, float32(w))
	if b <= a {
		return
	}
	i, j := int(a), int(b)
	if i == j {
		cover[i] += b - a
		return
	}
	cover[i] += float32(i+1) - a
	for k := i + 1; k < j; k++ {
		cover[k]++
	}
	if j < w {
		cover[j] += b - float32(j)
	}
}
