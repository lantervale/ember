package colourfont

import (
	"image"
	"math"
	"slices"

	"golang.org/x/image/vector"
)

// stop is a colour line's colour at offset t, premultiplied.
type stop struct {
	t   float32
	rgb [4]float32
}

// colourLine is a gradient's colours along its line.
type colourLine struct {
	extend uint8
	stops  []stop
}

const (
	extendPad uint8 = iota
	extendRepeat
	extendReflect
)

// line reads the colour line at at, its stops variable ones when
// variable.
func (c *colrCtx) line(at int, variable bool) colourLine {
	t := c.f.colr
	if at == 0 {
		return colourLine{}
	}
	size := 6
	if variable {
		size = 10
	}
	l := colourLine{extend: t.u8(at)}
	n := int(t.u16(at + 1))
	for i := range n {
		rec := at + 3 + size*i
		if !t.in(rec, size) {
			break
		}
		l.stops = append(l.stops, stop{t.f2dot14(rec), c.f.colour(t.u16(rec+2), t.f2dot14(rec+4), c.req)})
	}
	slices.SortStableFunc(l.stops, func(a, b stop) int {
		switch {
		case a.t < b.t:
			return -1
		case a.t > b.t:
			return 1
		}
		return 0
	})
	return l
}

// at is the line's colour at t, extended past its first and last stops.
func (l colourLine) at(t float32) [4]float32 {
	first, last := l.stops[0].t, l.stops[len(l.stops)-1].t
	if span := last - first; span > 0 {
		switch l.extend {
		case extendRepeat:
			u := (t - first) / span
			t = first + (u-float32(math.Floor(float64(u))))*span
		case extendReflect:
			u := (t - first) / span
			k := float32(math.Mod(float64(u), 2))
			if k < 0 {
				k += 2
			}
			if k > 1 {
				k = 2 - k
			}
			t = first + k*span
		}
	}
	if t <= first {
		return l.stops[0].rgb
	}
	if t >= last {
		return l.stops[len(l.stops)-1].rgb
	}
	for i := 1; i < len(l.stops); i++ {
		b := l.stops[i]
		if t > b.t {
			continue
		}
		a := l.stops[i-1]
		if b.t <= a.t {
			return b.rgb
		}
		k := (t - a.t) / (b.t - a.t)
		var out [4]float32
		for j := range out {
			out[j] = a.rgb[j] + (b.rgb[j]-a.rgb[j])*k
		}
		return out
	}
	return l.stops[len(l.stops)-1].rgb
}

// position is where on its colour line a point of a gradient's space
// falls, false where the gradient draws nothing.
type position func(x, y float32) (float32, bool)

// gradient fills a layer with line laid out by pos in the space m maps
// to the layer's pixels.
func (c *colrCtx) gradient(m affine, line colourLine, pos position) *layer {
	if len(line.stops) == 0 || pos == nil {
		return nil
	}
	inv, ok := m.invert()
	if !ok {
		return nil
	}
	l := newLayer(c.w, c.h)
	for y := range c.h {
		for x := range c.w {
			gx, gy := inv.apply(float32(x)+0.5, float32(y)+0.5)
			t, on := pos(gx, gy)
			if !on {
				continue
			}
			col := line.at(t)
			copy(l.px[4*(y*c.w+x):], col[:])
		}
	}
	return l
}

// linear is a linear gradient from p0 to p1, its colour constant along
// lines parallel to p0–p2: p1 is moved onto the normal of p0–p2.
func linear(p [6]float32) position {
	x0, y0, x1, y1, x2, y2 := p[0], p[1], p[2], p[3], p[4], p[5]
	nx, ny := y2-y0, -(x2 - x0)
	if nn := nx*nx + ny*ny; nn > 0 {
		k := ((x1-x0)*nx + (y1-y0)*ny) / nn
		x1, y1 = x0+nx*k, y0+ny*k
	}
	dx, dy := x1-x0, y1-y0
	d := dx*dx + dy*dy
	if d == 0 {
		return nil
	}
	return func(x, y float32) (float32, bool) { return ((x-x0)*dx + (y-y0)*dy) / d, true }
}

// radial is the gradient between circles (x0, y0, r0) and (x1, y1, r1):
// a point takes the largest t whose circle passes through it with a
// radius of at least 0.
func radial(x0, y0, r0, x1, y1, r1 float32) position {
	cdx, cdy, dr := float64(x1-x0), float64(y1-y0), float64(r1-r0)
	a := cdx*cdx + cdy*cdy - dr*dr
	return func(x, y float32) (float32, bool) {
		pdx, pdy := float64(x-x0), float64(y-y0)
		b := pdx*cdx + pdy*cdy + float64(r0)*dr
		c := pdx*pdx + pdy*pdy - float64(r0)*float64(r0)
		ok := func(t float64) bool { return float64(r0)+t*dr >= 0 }
		if math.Abs(a) < 1e-9 {
			if b == 0 {
				return 0, false
			}
			t := c / (2 * b)
			return float32(t), ok(t)
		}
		disc := b*b - a*c
		if disc < 0 {
			return 0, false
		}
		s := math.Sqrt(disc)
		t1, t2 := (b+s)/a, (b-s)/a
		if t1 < t2 {
			t1, t2 = t2, t1
		}
		if ok(t1) {
			return float32(t1), true
		}
		if ok(t2) {
			return float32(t2), true
		}
		return 0, false
	}
}

// sweep is a gradient round (cx, cy) from start to end degrees,
// counter-clockwise in the font's space, y up.
func sweep(cx, cy, start, end float32) position {
	if start == end {
		return nil
	}
	return func(x, y float32) (float32, bool) {
		a := float32(math.Atan2(float64(y-cy), float64(x-cx)) * 180 / math.Pi)
		if a < 0 {
			a += 360
		}
		return (a - start) / (end - start), true
	}
}

// polygonMask covers the polygon of pts, in pixels.
func polygonMask(pts [][2]float32, w, h int) *image.Alpha {
	r := vector.NewRasterizer(w, h)
	r.MoveTo(pts[0][0], pts[0][1])
	for _, p := range pts[1:] {
		r.LineTo(p[0], p[1])
	}
	r.ClosePath()
	mask := image.NewAlpha(image.Rect(0, 0, w, h))
	r.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
	return mask
}
