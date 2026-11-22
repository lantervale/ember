// Package outline holds the outlines a clip-path cuts to (decision
// 0119): closed polylines, curves already flattened, built from CSS's
// basic shapes and SVG path data, tested against a point and drawn into
// a mask. It is pure Go and leaves the svg reader out of ui.
package outline

import (
	"math"

	"github.com/lantervale/ember/geom"
)

// Path is an outline: closed polylines in points, each its last point
// joined to its first.
type Path struct{ Contours [][]geom.Vec2 }

// tolerance is how far, in points, a flattened curve may stray from the
// true one: little enough that a point a hit test finds on a curve's
// edge is on its outline too.
const tolerance = 1.0 / 256

// maxSteps is the most segments one curve is flattened into.
const maxSteps = 1024

// Polygon is the closed outline through pts.
func Polygon(pts []geom.Vec2) Path {
	if len(pts) == 0 {
		return Path{}
	}
	return Path{Contours: [][]geom.Vec2{append([]geom.Vec2(nil), pts...)}}
}

// Ellipse is the outline of the ellipse centred on c with radii rx and
// ry, empty unless both are above zero.
func Ellipse(c geom.Vec2, rx, ry float32) Path {
	if !(rx > 0 && ry > 0) {
		return Path{}
	}
	var pts []geom.Vec2
	pts = arcPoints(pts, c, rx, ry, 0, 0, 2*math.Pi)
	return Path{Contours: [][]geom.Vec2{pts[:len(pts)-1]}}
}

// RoundedRect is the outline of r with its corners rounded by radii, each
// a corner's horizontal and vertical radius clockwise from the top-left;
// radii too long for r's sides are shortened together, as CSS has them.
// An empty r has none.
func RoundedRect(r geom.Rect, radii [4]geom.Vec2) Path {
	if !(r.W > 0 && r.H > 0) {
		return Path{}
	}
	k := float32(1)
	fit := func(side, a, b float32) {
		if a+b > side {
			k = min(k, side/(a+b))
		}
	}
	for i := range radii {
		radii[i] = geom.V2(max(radii[i].X, 0), max(radii[i].Y, 0))
	}
	fit(r.W, radii[0].X, radii[1].X)
	fit(r.W, radii[3].X, radii[2].X)
	fit(r.H, radii[0].Y, radii[3].Y)
	fit(r.H, radii[1].Y, radii[2].Y)
	x0, y0, x1, y1 := r.X, r.Y, r.X+r.W, r.Y+r.H
	centres := [4]geom.Vec2{{X: x0, Y: y0}, {X: x1, Y: y0}, {X: x1, Y: y1}, {X: x0, Y: y1}}
	signs := [4]geom.Vec2{{X: 1, Y: 1}, {X: -1, Y: 1}, {X: -1, Y: -1}, {X: 1, Y: -1}}
	var pts []geom.Vec2
	for i, from := range [4]float64{math.Pi, -math.Pi / 2, 0, math.Pi / 2} {
		rad := radii[i].Scale(k)
		corner := centres[i]
		if rad.X <= 0 || rad.Y <= 0 {
			pts = append(pts, corner)
			continue
		}
		c := geom.V2(corner.X+signs[i].X*rad.X, corner.Y+signs[i].Y*rad.Y)
		pts = arcPoints(pts, c, rad.X, rad.Y, 0, from, math.Pi/2)
	}
	return Path{Contours: [][]geom.Vec2{pts}}
}

// arcPoints appends to pts the points of the arc of an ellipse centred
// on c and turned by phi, from angle a0 over sweep, both ends included:
// a whole number of them a quarter turn, so a whole ellipse reaches its
// sides.
func arcPoints(pts []geom.Vec2, c geom.Vec2, rx, ry float32, phi, a0, sweep float64) []geom.Vec2 {
	r := float64(max(rx, ry))
	step := 2 * math.Acos(max(1-tolerance/r, -1))
	n := maxSteps
	if step > 0 {
		n = min(max(int(math.Ceil(math.Abs(sweep)/step)), 2), maxSteps)
	}
	if quarters := math.Abs(sweep) / (math.Pi / 2); math.Abs(quarters-math.Round(quarters)) < 1e-9 && quarters >= 1 {
		q := int(math.Round(quarters))
		n = (n + q - 1) / q * q
	}
	sp, cp := math.Sincos(phi)
	for i := range n + 1 {
		s, co := math.Sincos(a0 + sweep*float64(i)/float64(n))
		x, y := co*float64(rx), s*float64(ry)
		pts = append(pts, geom.V2(c.X+float32(x*cp-y*sp), c.Y+float32(x*sp+y*cp)))
	}
	return pts
}

// cubicPoints appends to pts the points of the cubic curve from p0, its
// first left out.
func cubicPoints(pts []geom.Vec2, p0, c1, c2, p3 geom.Vec2) []geom.Vec2 {
	bend := max(p0.Sub(c1.Scale(2)).Add(c2).Len(), c1.Sub(c2.Scale(2)).Add(p3).Len())
	n := min(max(int(math.Ceil(math.Sqrt(0.75*float64(bend)/tolerance))), 1), maxSteps)
	for i := 1; i <= n; i++ {
		t := float32(i) / float32(n)
		u := 1 - t
		a, b, c, d := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
		pts = append(pts, p0.Scale(a).Add(c1.Scale(b)).Add(c2.Scale(c)).Add(p3.Scale(d)))
	}
	return pts
}

// Translate is p moved by d.
func (p Path) Translate(d geom.Vec2) Path {
	out := Path{Contours: make([][]geom.Vec2, len(p.Contours))}
	for i, c := range p.Contours {
		out.Contours[i] = make([]geom.Vec2, len(c))
		for j, q := range c {
			out.Contours[i][j] = q.Add(d)
		}
	}
	return out
}

// Bounds is the rectangle round p, ok false when it holds no area.
func (p Path) Bounds() (geom.Rect, bool) {
	lo, hi := geom.V2(math.MaxFloat32, math.MaxFloat32), geom.V2(-math.MaxFloat32, -math.MaxFloat32)
	for _, c := range p.Contours {
		for _, q := range c {
			lo, hi = geom.V2(min(lo.X, q.X), min(lo.Y, q.Y)), geom.V2(max(hi.X, q.X), max(hi.Y, q.Y))
		}
	}
	if !(hi.X > lo.X && hi.Y > lo.Y) {
		return geom.Rect{}, false
	}
	return geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}, true
}

// Contains reports whether q is inside p, filled by the even-odd rule or
// else the non-zero one; a point on the outline is inside, as browsers
// hit it.
func (p Path) Contains(q geom.Vec2, evenOdd bool) bool {
	winding := 0
	for _, c := range p.Contours {
		n := len(c)
		for i := range n {
			a, b := c[i], c[(i+1)%n]
			cross := (b.X-a.X)*(q.Y-a.Y) - (b.Y-a.Y)*(q.X-a.X)
			if onSegment(q, a, b, cross) {
				return true
			}
			switch {
			case a.Y <= q.Y && q.Y < b.Y && cross > 0:
				winding++
			case b.Y <= q.Y && q.Y < a.Y && cross < 0:
				winding--
			}
		}
	}
	if evenOdd {
		return winding&1 != 0
	}
	return winding != 0
}

// onSegment reports whether q lies on the segment from a to b, give or
// take how far a flattened curve strays from the true one.
func onSegment(q, a, b geom.Vec2, cross float32) bool {
	const e = tolerance
	if abs(cross) > e*max(b.Sub(a).Len(), 1) {
		return false
	}
	return q.X >= min(a.X, b.X)-e && q.X <= max(a.X, b.X)+e && q.Y >= min(a.Y, b.Y)-e && q.Y <= max(a.Y, b.Y)+e
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
