package draw

import (
	"image"
	"math"

	"golang.org/x/image/font/sfnt"
)

// miterLimit is how far a stroke's mitred corner may reach from its
// point, in halves of the stroke's width, before it is bevelled: Skia's
// default, which Chrome strokes text with.
const miterLimit = 4

// flatness is how far, in raster pixels, a curve's chords may stray from
// it.
const flatness = 0.1

// stroke draws the stroke of the outline, half pixels on either side of
// it, into t.mask: a quadrilateral for each straight piece of the
// flattened outline and a mitred or bevelled wedge on the outside of each
// corner, all turning the same way, so the rasterizer adds them into one
// shape.
func (t *tracer) stroke(half float32) {
	t.rast.Reset(t.dr.Dx(), t.dr.Dy())
	for _, c := range t.contours() {
		n := len(c)
		if n < 2 {
			continue
		}
		for i := range n {
			t.piece(c[i], c[(i+1)%n], half)
		}
		for i := range n {
			t.join(c[(i+n-1)%n], c[i], c[(i+1)%n], half)
		}
	}
	t.rast.Draw(t.mask, t.mask.Rect, image.Opaque, image.Point{})
}

// contours are the outline's closed contours flattened, in the mask's
// pixels, without repeated points.
func (t *tracer) contours() [][][2]float32 {
	var out [][][2]float32
	var c [][2]float32
	add := func(p [2]float32) {
		if len(c) == 0 || c[len(c)-1] != p {
			c = append(c, p)
		}
	}
	end := func() {
		if len(c) > 1 && c[0] == c[len(c)-1] {
			c = c[:len(c)-1]
		}
		if len(c) > 1 {
			out = append(out, c)
		}
		c = nil
	}
	at := [2]float32{}
	for _, s := range t.segs {
		var p [3][2]float32
		for i, a := range s.Args[:segmentArgs(s.Op)] {
			q := t.pt(a)
			p[i] = [2]float32{q[0] - float32(t.dr.Min.X), q[1] - float32(t.dr.Min.Y)}
		}
		switch s.Op {
		case sfnt.SegmentOpMoveTo:
			end()
			add(p[0])
		case sfnt.SegmentOpLineTo:
			add(p[0])
		case sfnt.SegmentOpQuadTo:
			n := chords(length(sub(add2(at, p[1]), scale(p[0], 2))) / 4)
			for i := 1; i <= n; i++ {
				u := float32(i) / float32(n)
				v := 1 - u
				add([2]float32{v*v*at[0] + 2*v*u*p[0][0] + u*u*p[1][0], v*v*at[1] + 2*v*u*p[0][1] + u*u*p[1][1]})
			}
		case sfnt.SegmentOpCubeTo:
			d := max(length(sub(add2(at, p[1]), scale(p[0], 2))),
				length(sub(add2(p[0], p[2]), scale(p[1], 2)))) * 3 / 4
			n := chords(d)
			for i := 1; i <= n; i++ {
				u := float32(i) / float32(n)
				v := 1 - u
				a, b, cc, e := v*v*v, 3*v*v*u, 3*v*u*u, u*u*u
				add([2]float32{a*at[0] + b*p[0][0] + cc*p[1][0] + e*p[2][0], a*at[1] + b*p[0][1] + cc*p[1][1] + e*p[2][1]})
			}
		}
		if len(c) > 0 {
			at = c[len(c)-1]
		}
	}
	end()
	return out
}

// chords is how many chords a curve straying d from its own chord is cut
// into to keep within flatness.
func chords(d float32) int {
	return min(max(int(math.Ceil(math.Sqrt(float64(d/flatness)))), 1), 64)
}

// piece adds the stroke of the straight piece from a to b.
func (t *tracer) piece(a, b [2]float32, half float32) {
	d := sub(b, a)
	l := length(d)
	if l == 0 {
		return
	}
	n := [2]float32{-d[1] / l * half, d[0] / l * half}
	t.polygon(add2(a, n), add2(b, n), sub(b, n), sub(a, n))
}

// join adds the wedge on the outside of the corner at, between the
// pieces from p and to q: mitred where the mitre reaches no further than
// miterLimit halves, else bevelled.
func (t *tracer) join(p, at, q [2]float32, half float32) {
	d1, d2 := sub(at, p), sub(q, at)
	l1, l2 := length(d1), length(d2)
	if l1 == 0 || l2 == 0 {
		return
	}
	d1, d2 = scale(d1, 1/l1), scale(d2, 1/l2)
	turn := d1[0]*d2[1] - d1[1]*d2[0]
	cos := d1[0]*d2[0] + d1[1]*d2[1]
	if turn == 0 && cos > 0 {
		return
	}
	s := half
	if turn > 0 {
		s = -half
	}
	n1, n2 := [2]float32{-d1[1] * s, d1[0] * s}, [2]float32{-d2[1] * s, d2[0] * s}
	if 1+cos >= 2/(miterLimit*miterLimit) {
		t.polygon(at, add2(at, n1), add2(at, scale(add2(n1, n2), 1/(1+cos))), add2(at, n2))
		return
	}
	t.polygon(at, add2(at, n1), add2(at, n2))
}

// polygon adds the polygon through pts to the rasterizer, turned the
// way every other one is.
func (t *tracer) polygon(pts ...[2]float32) {
	var area float32
	for i, p := range pts {
		q := pts[(i+1)%len(pts)]
		area += p[0]*q[1] - q[0]*p[1]
	}
	if area == 0 {
		return
	}
	if area < 0 {
		for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
			pts[i], pts[j] = pts[j], pts[i]
		}
	}
	t.rast.MoveTo(pts[0][0], pts[0][1])
	for _, p := range pts[1:] {
		t.rast.LineTo(p[0], p[1])
	}
	t.rast.ClosePath()
}

func add2(a, b [2]float32) [2]float32          { return [2]float32{a[0] + b[0], a[1] + b[1]} }
func sub(a, b [2]float32) [2]float32           { return [2]float32{a[0] - b[0], a[1] - b[1]} }
func scale(a [2]float32, k float32) [2]float32 { return [2]float32{a[0] * k, a[1] * k} }
func length(a [2]float32) float32              { return float32(math.Hypot(float64(a[0]), float64(a[1]))) }
