package paint

import (
	"math"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/outline"
	"github.com/lantervale/ember/ui/internal/style"
)

// stripe is a stretch across a side's width, from and to, measured from its
// outer edge: what of the side is painted.
type stripe [2]float32

// lines are the stripes a side of width w and style s is painted in, at k
// device pixels a point: a double's two lines, as Chromium cuts them in
// whole device pixels, the outer (w+1)/3 wide and the inner from (2w+1)/3
// on, or the whole width for any other style and a double under 3 device
// pixels wide.
func lines(w float32, s style.BorderStyle, k float32) []stripe {
	if s == style.BorderDouble {
		if outer, inner, ok := doubleCut(w, k); ok {
			return []stripe{{0, outer}, {inner, w}}
		}
	}
	return []stripe{{0, w}}
}

// doubleCut is where a double line of width w ends and where the inner one
// starts, at k device pixels a point; not ok when w is under 3 device
// pixels, too thin for a gap.
func doubleCut(w, k float32) (float32, float32, bool) {
	if k <= 0 {
		k = 1
	}
	px := int(math.Round(float64(w * k)))
	if px < 3 {
		return 0, 0, false
	}
	return float32((px+1)/3) / k, float32((2*px+1)/3) / k, true
}

// doubleBorder paints a border with a double side: as two borders, its
// outer lines and its inner ones past the gap, each mitered or rounded as
// a border is, unless the box is square and its other sides solid, where
// each side is mitered and cut to its lines. It reports false when no
// side is a double wide enough to show its gap.
func (p *painter) doubleBorder(r geom.Rect, w [4]float32, c [4]color.Color, radius radii,
	st style.Sides[style.BorderStyle]) bool {
	styles := [4]style.BorderStyle{st.Top, st.Right, st.Bottom, st.Left}
	var cuts [4][]stripe
	split, solid, other := false, false, false
	for i := range 4 {
		cuts[i] = lines(w[i], styles[i], p.maskScale())
		split = split || len(cuts[i]) == 2
		if w[i] > 0 {
			solid = solid || styles[i] == style.BorderSolid
			other = other || styles[i] != style.BorderSolid && styles[i] != style.BorderDouble
		}
		if styles[i] == style.BorderDouble {
			styles[i] = style.BorderSolid
		}
	}
	if !split {
		return false
	}
	if solid && !other && radius.zero() {
		p.mitres(r, w, c, cuts)
		return true
	}
	lined := style.Computed{BorderStyle: style.Sides[style.BorderStyle]{Top: styles[0], Right: styles[1],
		Bottom: styles[2], Left: styles[3]}}
	lined.BorderColor = style.Sides[color.Color]{Top: c[0], Right: c[1], Bottom: c[2], Left: c[3]}
	var out, gap, in box.Edges
	side := func(e *box.Edges, i int) *float32 { return [4]*float32{&e.Top, &e.Right, &e.Bottom, &e.Left}[i] }
	for i, b := range cuts {
		*side(&out, i), *side(&gap, i) = b[0][1], w[i]
		if len(b) == 2 {
			*side(&gap, i), *side(&in, i) = b[1][0], w[i]-b[1][0]
		}
	}
	p.border(r, out, radius, &lined, 1)
	p.border(inside(r, gap), in, radius.inside(gap), &lined, 1)
	return true
}

// mitres paints the sides of r, of widths w and colours c, each a
// trapezoid mitered into its neighbours and cut to its stripes. Where two
// sides of other colours meet, their diagonal is smoothed as Chrome's is:
// across a device pixel each side fades to the colour halfway between the
// two, which the diagonal is drawn in.
func (p *painter) mitres(r geom.Rect, w [4]float32, c [4]color.Color, stripes [4][]stripe) {
	x0, y0, x1, y1 := r.X, r.Y, r.X+r.W, r.Y+r.H
	t, rt, b, l := w[0], w[1], w[2], w[3]
	outer := [4]geom.Vec2{{X: x0, Y: y0}, {X: x1, Y: y0}, {X: x1, Y: y1}, {X: x0, Y: y1}}
	inner := [4]geom.Vec2{{X: x0 + l, Y: y0 + t}, {X: x1 - rt, Y: y0 + t}, {X: x1 - rt, Y: y1 - b}, {X: x0 + l, Y: y1 - b}}
	half := 0.5 / p.maskScale()
	for i := range 4 {
		if w[i] <= 0 {
			continue
		}
		next := (i + 1) % 4
		side := []geom.Vec2{outer[i], outer[next], inner[next], inner[i]}
		// Corner i starts the side and corner next ends it; each is smoothed
		// against the side across it when their colours differ. A clear side
		// paints only its half of a diagonal it shares with one that shows,
		// fading out.
		var ends []smooth
		for _, k := range [2]struct{ corner, across int }{{i, (i + 3) % 4}, {next, next}} {
			if w[k.across] <= 0 || c[k.across] == c[i] || c[k.across].A <= 0 && c[i].A <= 0 ||
				outer[k.corner] == inner[k.corner] {
				continue
			}
			o, in := outer[k.corner], inner[k.corner]
			d := in.Sub(o).Normalize()
			n := geom.V2(-d.Y, d.X)
			if n.Dot(midpoint(side[0], side[1]).Sub(o)) < 0 {
				n = n.Scale(-1)
			}
			own := c[i]
			if own.A <= 0 {
				own = c[k.across].Alpha(0)
			}
			ends = append(ends, smooth{at: o, normal: n, mid: halfway(c[i], c[k.across]), own: own})
		}
		if c[i].A <= 0 && len(ends) == 0 {
			continue
		}
		for _, bd := range stripes[i] {
			piece := clipConvex(side, stripeOf(r, i, bd))
			for _, e := range ends {
				var near []geom.Vec2
				piece, near = splitAt(piece, func(q geom.Vec2) float32 { return e.normal.Dot(q.Sub(e.at)) - half })
				p.fade(near, func(q geom.Vec2) color.Color {
					return e.mid.Lerp(e.own, min(max(e.normal.Dot(q.Sub(e.at))/half, 0), 1))
				})
			}
			if len(piece) >= 3 && c[i].A > 0 {
				p.s.Triangles(piece, fan(len(piece)), c[i])
			}
		}
	}
}

// smooth is a mitre's diagonal from at, the normal pointing into the side
// painted, the colour halfway to the side across it, and the side's own,
// which it fades to.
type smooth struct {
	at, normal geom.Vec2
	mid, own   color.Color
}

func midpoint(a, b geom.Vec2) geom.Vec2 { return a.Add(b).Scale(0.5) }

// halfway is the colour halfway between a and b in encoded sRGB, as Chrome
// blends them; a clear side is a's colour gone clear.
func halfway(a, b color.Color) color.Color {
	if b.A <= 0 {
		b = a.Alpha(0)
	}
	if a.A <= 0 {
		a = b.Alpha(0)
	}
	sa, sb := a.SRGB(), b.SRGB()
	return color.FromSRGB([3]float32{(sa[0] + sb[0]) / 2, (sa[1] + sb[1]) / 2, (sa[2] + sb[2]) / 2}).Alpha((a.A + b.A) / 2)
}

// fade fills the convex poly with each corner's colour by at, blended
// between them.
func (p *painter) fade(poly []geom.Vec2, at func(geom.Vec2) color.Color) {
	if len(poly) < 3 {
		return
	}
	cs := make([]color.Color, len(poly))
	for i, q := range poly {
		cs[i] = at(q)
	}
	p.s.TrianglesColored(poly, cs, fan(len(poly)))
}

// fan is the triangles of a convex polygon of n corners.
func fan(n int) []int {
	out := make([]int, 0, 3*(n-2))
	for i := 1; i+1 < n; i++ {
		out = append(out, 0, i, i+1)
	}
	return out
}

// stripeOf is the rectangle of side i of r (top, right, bottom, left) that
// bd spans, its corners clockwise on the screen.
func stripeOf(r geom.Rect, i int, bd stripe) []geom.Vec2 {
	var b geom.Rect
	switch i {
	case 0:
		b = geom.Rect{X: r.X, Y: r.Y + bd[0], W: r.W, H: bd[1] - bd[0]}
	case 1:
		b = geom.Rect{X: r.X + r.W - bd[1], Y: r.Y, W: bd[1] - bd[0], H: r.H}
	case 2:
		b = geom.Rect{X: r.X, Y: r.Y + r.H - bd[1], W: r.W, H: bd[1] - bd[0]}
	default:
		b = geom.Rect{X: r.X + bd[0], Y: r.Y, W: bd[1] - bd[0], H: r.H}
	}
	return []geom.Vec2{{X: b.X, Y: b.Y}, {X: b.X + b.W, Y: b.Y}, {X: b.X + b.W, Y: b.Y + b.H}, {X: b.X, Y: b.Y + b.H}}
}

// splitAt cuts the convex poly by the line where f is 0: the part where f
// is above it, and the part where it is below.
func splitAt(poly []geom.Vec2, f func(geom.Vec2) float32) ([]geom.Vec2, []geom.Vec2) {
	var above, below []geom.Vec2
	for j := range poly {
		a, b := poly[j], poly[(j+1)%len(poly)]
		fa, fb := f(a), f(b)
		if fa >= 0 {
			above = append(above, a)
		}
		if fa <= 0 {
			below = append(below, a)
		}
		if fa < 0 && fb > 0 || fa > 0 && fb < 0 {
			x := a.Add(b.Sub(a).Scale(fa / (fa - fb)))
			above, below = append(above, x), append(below, x)
		}
	}
	return above, below
}

// roundedSides paints a rounded border as wide all round whose sides
// differ in colour: a ring in each colour, cut by a mask to its run of
// sides, each run meeting the next on the line from their outer corner to
// the inner one, at 45 degrees, blended across it by the mask's edge as
// Chrome blends them. When every side is opaque, the first side's colour
// is a whole ring the others are drawn over.
func (p *painter) roundedSides(r geom.Rect, w float32, c [4]color.Color, radius radii) {
	ring := func(col color.Color) {
		if radius.uniform() {
			p.s.Border(r, radius[0], w, col)
		} else {
			p.s.BorderCorners(r, radius, w, col)
		}
	}
	opaque := c[0].A >= 1 && c[1].A >= 1 && c[2].A >= 1 && c[3].A >= 1
	start := 0
	for start < 4 && c[start] == c[(start+3)%4] {
		start++
	}
	if start == 4 {
		if w > 0 && c[0].A > 0 {
			ring(c[0])
		}
		return
	}
	outer := [4]geom.Vec2{{X: r.X, Y: r.Y}, {X: r.X + r.W, Y: r.Y}, {X: r.X + r.W, Y: r.Y + r.H}, {X: r.X, Y: r.Y + r.H}}
	d := min(r.W, r.H) / 2
	deep := [4]geom.Vec2{
		outer[0].Add(geom.V2(d, d)), outer[1].Add(geom.V2(-d, d)),
		outer[2].Add(geom.V2(-d, -d)), outer[3].Add(geom.V2(d, -d)),
	}
	if opaque {
		ring(c[0])
	}
	for n := 0; n < 4; {
		first := (start + n) % 4
		last := first
		for n++; n < 4 && c[(start+n)%4] == c[first]; n++ {
			last = (start + n) % 4
		}
		if c[first].A <= 0 || opaque && c[first] == c[0] {
			continue
		}
		var pts []geom.Vec2
		for k := first; ; k = (k + 1) % 4 {
			pts = append(pts, outer[k])
			if k == last {
				pts = append(pts, outer[(k+1)%4])
				break
			}
		}
		for k := (last + 1) % 4; ; k = (k + 3) % 4 {
			pts = append(pts, deep[k])
			if k == first {
				break
			}
		}
		shape := &clipShape{path: outline.Polygon(pts)}
		shape.bounds, _ = shape.path.Bounds()
		p.s.PushClipMask(shape.bounds, masks.mask(shape, p.maskScale()))
		ring(c[first])
		p.s.PopClip()
	}
}
