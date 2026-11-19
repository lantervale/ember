package svg

import "math"

// stroker turns each subpath into the outline of its stroke, in the
// path's space: for an open one, a side forward, a cap, the other side
// back, a cap; for a closed one, each side as a loop. At a corner the
// outer side takes the join and the inner side passes through the
// corner itself, which makes the outline the sum of each segment's
// rectangle and each join's wedge, all turning the same way: filled
// non-zero, it is their union, with no hole where they overlap.
type stroker struct {
	hw    float64 // half the stroke's width, in the path's units
	scale float64 // pixels per unit
	cap   Cap
	join  Join
	miter float64
	out   [][]Point
}

// arc appends the points of the stroke's disc round c from c+u over
// sweep radians.
func (s *stroker) arc(poly []Point, c, u Point, sweep float64) []Point {
	n := 1
	if r := s.hw * s.scale; r > tolerance {
		n = max(n, int(math.Ceil(math.Abs(sweep)/(2*math.Acos(1-tolerance/r)))))
	}
	n = min(n, 1024)
	a0 := math.Atan2(u.Y, u.X)
	for i := 0; i <= n; i++ {
		sin, cos := math.Sincos(a0 + sweep*float64(i)/float64(n))
		poly = append(poly, Point{c.X + s.hw*cos, c.Y + s.hw*sin})
	}
	return poly
}

// corner appends what joins the side at offset u, before the corner p,
// to the side at offset v after it; d0 and d1 are the directions in and
// out.
func (s *stroker) corner(poly []Point, p, d0, d1, u, v Point) []Point {
	cross, dot := d0.cross(d1), d0.dot(d1)
	r := s.hw * s.scale
	if dot > 0 && r*cross*cross < tolerance {
		return poly // too slight a turn for its join to show: the flattened steps of a curve
	}
	if dot <= -1+1e-12 {
		// The path turns straight back: a cap's worth of join.
		if s.join == RoundJoin {
			return s.arc(poly, p, u, sign(u.cross(d0))*math.Pi)
		}
		return poly
	}
	if d0.perp().dot(u)*cross > 0 {
		return append(poly, p)
	}
	switch half := math.Sqrt((1 + dot) / 2); {
	case s.join == RoundJoin:
		return s.arc(poly, p, u, math.Atan2(u.cross(v), u.dot(v)))
	case s.join == MiterJoin && half > 0 && 1/half <= s.miter:
		return append(poly, p.add(u.add(v).mul(1/(1+dot))))
	}
	return poly
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

// end appends the cap at p, where the path leaves in direction d, from
// the side at offset u to the one at -u.
func (s *stroker) end(poly []Point, p, d, u Point) []Point {
	switch s.cap {
	case RoundCap:
		return s.arc(poly, p, u, sign(u.cross(d))*math.Pi)
	case SquareCap:
		e := d.mul(s.hw)
		return append(poly, p.add(u).add(e), p.sub(u).add(e))
	case ButtCap:
	}
	return poly
}

func (s *stroker) subpath(sub subpath) {
	pts := sub.pts
	n := len(pts)
	if n == 1 {
		if sub.drawn && s.cap != ButtCap {
			u := Point{0, s.hw}
			dot := s.end(nil, pts[0], Point{1, 0}, u)
			s.out = append(s.out, s.end(dot, pts[0], Point{-1, 0}, u.mul(-1)))
		}
		return
	}
	segs := n - 1
	if sub.closed {
		segs = n
	}
	dirs := make([]Point, segs)
	for i := range dirs {
		d := pts[(i+1)%n].sub(pts[i])
		dirs[i] = d.mul(1 / d.len())
	}
	side := func(poly []Point, dir int, loop bool) []Point {
		for k := range segs {
			i := k
			if dir < 0 {
				i = segs - 1 - k
			}
			a, b, d := pts[i], pts[(i+1)%n], dirs[i]
			if dir < 0 {
				a, b, d = b, a, d.mul(-1)
			}
			w := d.perp().mul(s.hw)
			poly = append(poly, a.add(w), b.add(w))
			next := i + dir
			if loop {
				next = (next + segs) % segs
			}
			if next < 0 || next >= segs {
				continue
			}
			nd := dirs[next].mul(float64(dir))
			poly = s.corner(poly, b, d, nd, w, nd.perp().mul(s.hw))
		}
		return poly
	}
	if sub.closed {
		s.out = append(s.out, side(make([]Point, 0, 3*segs), 1, true), side(make([]Point, 0, 3*segs), -1, true))
		return
	}
	last, first := dirs[segs-1], dirs[0].mul(-1)
	poly := s.end(side(make([]Point, 0, 6*segs+8), 1, false), pts[n-1], last, last.perp().mul(s.hw))
	poly = side(poly, -1, false)
	s.out = append(s.out, s.end(poly, pts[0], first, first.perp().mul(s.hw)))
}

// strokeOutline is the outline of the stroke of subs, to fill non-zero.
func strokeOutline(subs []subpath, shape *Shape, scale float64) [][]Point {
	s := &stroker{hw: shape.StrokeWidth / 2, scale: scale, cap: shape.Cap, join: shape.Join, miter: shape.MiterLimit}
	for _, sub := range subs {
		if len(sub.pts) > 0 {
			s.subpath(sub)
		}
	}
	return s.out
}

// maxDashPieces is the most dashes one shape is cut into; a pattern
// finer than that is drawn solid.
const maxDashPieces = 1 << 16

// dasher cuts polylines into the dashes of a pattern, an even count of
// lengths, dash and gap in turn.
type dasher struct {
	pattern []float64
	offset  float64
	out     []subpath
}

// line cuts the polyline pts, the pattern starting anew.
func (d *dasher) line(pts []Point) {
	at, left := 0, d.offset
	for left > d.pattern[at] || left == d.pattern[at] && left > 0 {
		left -= d.pattern[at]
		at = (at + 1) % len(d.pattern)
	}
	left = d.pattern[at] - left // what remains of the dash or gap under way
	var cur []Point
	if at%2 == 0 {
		cur = []Point{pts[0]}
	}
	for i := 1; i < len(pts); i++ {
		a, b := pts[i-1], pts[i]
		seg := b.sub(a).len()
		done := 0.0
		for seg-done > left {
			done += left
			p := lerp(a, b, done/seg)
			if at%2 == 0 {
				if p != cur[len(cur)-1] {
					cur = append(cur, p)
				}
				d.out = append(d.out, subpath{pts: cur, drawn: true})
				cur = nil
			} else {
				cur = []Point{p}
			}
			at = (at + 1) % len(d.pattern)
			left = d.pattern[at]
		}
		left -= seg - done
		if at%2 == 0 && b != cur[len(cur)-1] {
			cur = append(cur, b)
		}
	}
	if at%2 == 0 && len(cur) > 0 {
		d.out = append(d.out, subpath{pts: cur, drawn: true})
	}
}

// dash cuts subs into the dashes of pattern, starting offset into it.
func dash(subs []subpath, pattern []float64, offset float64) []subpath {
	period := 0.0
	for _, d := range pattern {
		period += d
	}
	total := 0.0
	for _, sub := range subs {
		for i := 1; i < len(sub.pts); i++ {
			total += sub.pts[i].sub(sub.pts[i-1]).len()
		}
	}
	if !(period > 0) || math.IsInf(period, 0) || total/period*float64(len(pattern)) > maxDashPieces {
		return subs
	}
	offset = math.Mod(offset, period)
	if offset < 0 {
		offset += period
	}
	d := &dasher{pattern: pattern, offset: offset}
	for _, sub := range subs {
		pts := sub.pts
		if sub.closed && len(pts) > 1 {
			pts = append(pts[:len(pts):len(pts)], pts[0])
		}
		if len(pts) > 1 {
			d.line(pts)
		}
	}
	return d.out
}
