package svg

import "math"

// tolerance is how far, in pixels, a flattened curve may stray from the
// true one.
const tolerance = 0.05

// subpath is one run of a flattened path: points no two of which follow
// each other equal. drawn tells a single point that a segment or a close
// was drawn there, which caps turn into a dot, from a bare move.
type subpath struct {
	pts    []Point
	closed bool
	drawn  bool
}

// flattener turns a Path into polylines, in the path's own space.
type flattener struct {
	// scale is pixels per unit of the path.
	scale float64
	// reach is half the stroke's width in pixels, 0 for a fill: the
	// outer edge of a stroked curve turns on a wider circle than the
	// curve, and needs the steps of that one.
	reach    float64
	maxSteps int
	subs     []subpath
}

func (f *flattener) moveTo(p Point) {
	f.subs = append(f.subs, subpath{pts: []Point{p}})
}

func (f *flattener) lineTo(p Point) {
	s := &f.subs[len(f.subs)-1]
	s.drawn = true
	if p != s.pts[len(s.pts)-1] {
		s.pts = append(s.pts, p)
	}
}

func (f *flattener) close() {
	s := &f.subs[len(f.subs)-1]
	if n := len(s.pts); n > 1 && s.pts[n-1] == s.pts[0] {
		s.pts = s.pts[:n-1]
	}
	s.closed, s.drawn = true, true
}

func turn(a, b Point) float64 {
	if a == (Point{}) || b == (Point{}) {
		return 0
	}
	return math.Abs(math.Atan2(a.cross(b), a.dot(b)))
}

func (f *flattener) cubicTo(p0, c1, c2, p Point) {
	bend := math.Max(p0.sub(c1.mul(2)).add(c2).len(), c1.sub(c2.mul(2)).add(p).len())
	steps := math.Ceil(math.Sqrt(0.75 * bend * f.scale / tolerance))
	if f.reach > tolerance {
		d0, d1, d2 := c1.sub(p0), c2.sub(c1), p.sub(c2)
		if d0 == (Point{}) {
			d0 = d1
		}
		if d2 == (Point{}) {
			d2 = d1
		}
		steps = math.Max(steps, math.Ceil((turn(d0, d1)+turn(d1, d2))/(2*math.Acos(1-tolerance/f.reach))))
	}
	n := 1
	if steps > 1 { // false for NaN too
		n = int(math.Min(steps, float64(f.maxSteps)))
	}
	for i := 1; i < n; i++ {
		f.lineTo(cubicAt(p0, c1, c2, p, float64(i)/float64(n)))
	}
	f.lineTo(p)
}

// flatten is path as polylines for a drawing at scale pixels per unit,
// its stroke reach pixels to each side.
func flatten(path Path, scale, reach float64) []subpath {
	curves := 0
	for _, v := range path.Verbs {
		if v == QuadTo || v == CubicTo {
			curves++
		}
	}
	f := &flattener{scale: scale, reach: reach, maxSteps: min(256, max(1, maxFlatPoints/max(curves, 1)))}
	var cur, start Point
	open := false
	i := 0
	pts := path.Points
	for _, v := range path.Verbs {
		if v != MoveTo && v != Close && !open {
			f.moveTo(start)
			open = true
		}
		switch v {
		case MoveTo:
			cur = pts[i]
			start, open = cur, true
			f.moveTo(cur)
			i++
		case LineTo:
			cur = pts[i]
			f.lineTo(cur)
			i++
		case QuadTo:
			c, to := pts[i], pts[i+1]
			f.cubicTo(cur, lerp(cur, c, 2.0/3), lerp(to, c, 2.0/3), to)
			cur = to
			i += 2
		case CubicTo:
			f.cubicTo(cur, pts[i], pts[i+1], pts[i+2])
			cur = pts[i+2]
			i += 3
		case Close:
			if open {
				f.close()
			}
			cur, open = start, false
		}
	}
	return f.subs
}
