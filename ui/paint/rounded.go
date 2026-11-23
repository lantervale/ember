package paint

import (
	"cmp"
	"image"
	"math"
	"slices"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/outline"
	"github.com/lantervale/ember/ui/internal/style"
)

// ringBand is a stretch of a rounded border between two curves, each the
// border box inset by so much on every side, in one colour.
type ringBand struct {
	from, to [4]float32
	c        color.Color
}

// look is how one side of a rounded border is painted: its bands, or for
// a dashed or dotted side its colour and the width its dashes are laid by.
type look struct {
	s     style.BorderStyle
	bands []ringBand
	c     color.Color
	w     float32
}

func (l look) same(o look) bool {
	if l.s != o.s || l.c != o.c || len(l.bands) != len(o.bands) {
		return false
	}
	if (l.s == style.BorderDashed || l.s == style.BorderDotted) && l.w != o.w {
		return false
	}
	for i := range l.bands {
		if l.bands[i].c != o.bands[i].c {
			return false
		}
	}
	return true
}

// roundedBorder paints a rounded border whose sides differ, or that is
// not plain solid, as Chrome paints one side by side: each side is the
// ring between the rounded border box and its padding box, cut on the
// lines from its outer corners through its inner ones (to where they
// cross the chord of a rounded inner corner), a run of sides painted
// alike cut as one. A groove or a ridge is two bands, outside and inside
// the middle of the border, a double its outer and inner lines, an inset
// or an outset side its shade; dashes and dots are laid along the middle
// of the border all round from the bottom of the left side, as Skia
// strokes a rounded rectangle, and cut to the ring and to their side.
// Sides are painted as Chrome orders them: the more transparent first,
// then dashed, dotted and double, then the shaded styles, then solid,
// and top, bottom, right, left.
func (p *painter) roundedBorder(r geom.Rect, w [4]float32, c [4]color.Color, styles [4]style.BorderStyle,
	radius radii) {
	k := p.maskScale()
	var looks [4]look
	shown := [4]bool{}
	for i := range 4 {
		shown[i] = w[i] > 0 && styles[i] != style.BorderNone && c[i].A > 0
		if shown[i] {
			looks[i] = lookOf(i, w, c[i], styles[i], k)
		}
	}
	joints := junctions(r, w, radius)
	ring := []maskPart{{path: roundedInset(r, radius, [4]float32{}), evenOdd: true}}
	ring[0].path.Contours = append(ring[0].path.Contours, roundedInset(r, radius, w).Contours...)
	type run struct{ first, n int }
	var runs []run
	start := 0
	for start < 4 && shown[start] && shown[(start+3)%4] && looks[start].same(looks[(start+3)%4]) {
		start++
	}
	if start == 4 {
		runs = append(runs, run{0, 4})
	} else {
		for n := 0; n < 4; {
			first := (start + n) % 4
			m := 1
			for n+m < 4 && shown[first] == shown[(first+m)%4] && looks[first].same(looks[(first+m)%4]) {
				m++
			}
			if shown[first] {
				runs = append(runs, run{first, m})
			}
			n += m
		}
	}
	firstPainted := func(rn run) int {
		least := 4
		for j := range rn.n {
			least = min(least, sidePriority((rn.first+j)%4))
		}
		return least
	}
	slices.SortStableFunc(runs, func(a, b run) int {
		la, lb := looks[a.first], looks[b.first]
		return cmp.Or(cmp.Compare(c[a.first].A, c[b.first].A), cmp.Compare(stylePriority(la.s), stylePriority(lb.s)),
			cmp.Compare(firstPainted(a), firstPainted(b)))
	})
	outer := corners(r)
	for _, rn := range runs {
		var parts []maskPart
		if rn.n < 4 {
			var pts []geom.Vec2
			for j := range rn.n + 1 {
				pts = append(pts, outer[(rn.first+j)%4])
			}
			for j := rn.n; j >= 0; j-- {
				pts = append(pts, joints[(rn.first+j)%4])
			}
			parts = append(parts, maskPart{path: outline.Polygon(pts)})
		}
		l := looks[rn.first]
		if l.s == style.BorderDashed || l.s == style.BorderDotted {
			thick := l.w
			for j := range rn.n {
				i := (rn.first + j) % 4
				thick = max(thick, w[(i+1)%4], w[(i+3)%4])
			}
			dashes := dashPath(r, w, radius, l.w, thick, k, l.s == style.BorderDotted)
			p.maskedRect(append(append(parts, ring...), maskPart{path: dashes}), l.c, k)
			continue
		}
		for _, b := range l.bands {
			band := maskPart{path: roundedInset(r, radius, b.from), evenOdd: true}
			band.path.Contours = append(band.path.Contours, roundedInset(r, radius, b.to).Contours...)
			p.maskedRect(append(parts, band), b.c, k)
		}
	}
}

// lookOf is how side i of a rounded border of widths w is painted in
// colour c and style s, at k device pixels a point.
func lookOf(i int, w [4]float32, c color.Color, s style.BorderStyle, k float32) look {
	all := func(f func(w float32) float32) [4]float32 {
		var out [4]float32
		for j := range 4 {
			out[j] = f(w[j])
		}
		return out
	}
	none, whole := [4]float32{}, w
	dark := i == 0 || i == 3
	switch s {
	case style.BorderInset, style.BorderOutset:
		darker, lighter := shades(c)
		if (s == style.BorderInset) == dark {
			return look{s: s, bands: []ringBand{{none, whole, darker}}}
		}
		return look{s: s, bands: []ringBand{{none, whole, lighter}}}
	case style.BorderGroove, style.BorderRidge:
		darker, lighter := shades(c)
		first, second := lighter, darker
		if (s == style.BorderGroove) == dark {
			first, second = darker, lighter
		}
		half := all(func(w float32) float32 { return w / 2 })
		return look{s: s, bands: []ringBand{{none, half, first}, {half, whole, second}}}
	case style.BorderDouble:
		cut := func(inner bool) [4]float32 {
			return all(func(w float32) float32 {
				o, in, ok := doubleCut(w, k)
				switch {
				case !ok:
					return w
				case inner:
					return in
				}
				return o
			})
		}
		return look{s: s, bands: []ringBand{{none, cut(false), c}, {cut(true), whole, c}}}
	case style.BorderDashed, style.BorderDotted:
		return look{s: s, c: c, w: w[i]}
	case style.BorderNone, style.BorderSolid:
	}
	return look{s: s, bands: []ringBand{{none, whole, c}}}
}

// stylePriority and sidePriority order the sides Chrome paints in one
// opacity: the lowest first.
func stylePriority(s style.BorderStyle) int {
	switch s {
	case style.BorderDotted, style.BorderDashed, style.BorderDouble:
		return 1
	case style.BorderInset, style.BorderGroove, style.BorderOutset, style.BorderRidge:
		return 2
	case style.BorderSolid:
		return 3
	case style.BorderNone:
	}
	return 0
}

func sidePriority(i int) int { return [4]int{0, 2, 1, 3}[i] }

// roundedInset is the outline of r less e on each side, its corners those
// of radius less the sides beside them, elliptical where the two differ.
func roundedInset(r geom.Rect, radius radii, e [4]float32) outline.Path {
	in := geom.Rect{X: r.X + e[3], Y: r.Y + e[0], W: r.W - e[1] - e[3], H: r.H - e[0] - e[2]}
	corner := func(i, x, y int) geom.Vec2 {
		return geom.V2(max(radius[i]-e[x], 0), max(radius[i]-e[y], 0))
	}
	return outline.RoundedRect(in, [4]geom.Vec2{corner(0, 3, 0), corner(1, 1, 0), corner(2, 1, 2), corner(3, 3, 2)})
}

// junctions are where the sides of a rounded border of widths w meet
// inside it, corner by corner from the top-left: on the line from the
// outer corner through the inner one, where it crosses the chord of the
// inner corner when that is rounded, kept on the corner's side of the
// box's middle.
func junctions(r geom.Rect, w [4]float32, radius radii) [4]geom.Vec2 {
	outer := corners(r)
	inner := corners(geom.Rect{X: r.X + w[3], Y: r.Y + w[0], W: r.W - w[1] - w[3], H: r.H - w[0] - w[2]})
	sides := [4][2]int{{3, 0}, {1, 0}, {1, 2}, {3, 2}}
	in := [4]geom.Vec2{{X: 1, Y: 1}, {X: -1, Y: 1}, {X: -1, Y: -1}, {X: 1, Y: -1}}
	mid := r.Center()
	var out [4]geom.Vec2
	for i := range 4 {
		o, q := outer[i], inner[i]
		rx, ry := radius[i]-w[sides[i][0]], radius[i]-w[sides[i][1]]
		if rx > 0 && ry > 0 {
			a, b := q.Add(geom.V2(in[i].X*rx, 0)), q.Add(geom.V2(0, in[i].Y*ry))
			if x, ok := crossing(o, q, a, b); ok {
				q = x
			}
		}
		d := q.Sub(o)
		t := float32(1)
		if lim := (mid.X - o.X); d.X != 0 && d.X*in[i].X > lim*in[i].X {
			t = min(t, lim/d.X)
		}
		if lim := (mid.Y - o.Y); d.Y != 0 && d.Y*in[i].Y > lim*in[i].Y {
			t = min(t, lim/d.Y)
		}
		out[i] = o.Add(d.Scale(t))
	}
	return out
}

// crossing is where the line through a and b meets the one through c and
// d; not ok when they run side by side.
func crossing(a, b, c, d geom.Vec2) (geom.Vec2, bool) {
	r, s := b.Sub(a), d.Sub(c)
	den := r.X*s.Y - r.Y*s.X
	if abs(den) < 1e-6 {
		return geom.Vec2{}, false
	}
	t := ((c.X-a.X)*s.Y - (c.Y-a.Y)*s.X) / den
	return a.Add(r.Scale(t)), true
}

// dashPath is the dashes or the dots of a rounded border of widths w laid
// as Chrome lays them for a side of width side, at k device pixels a
// point, which Chrome measures them in: along the middle of the border,
// all round, clockwise from the top-left end of the top side, as Skia
// strokes a rounded rectangle. A dash is thick wide either way of the
// middle, for the ring to cut, and so is a dot up to 3 device pixels
// wide, a short dash as long as it is wide; a wider dot is a disc side
// across.
func dashPath(r geom.Rect, w [4]float32, radius radii, side, thick, k float32, dotted bool) outline.Path {
	m, ok := middleOf(r, w, radius)
	if !ok {
		return outline.Path{}
	}
	length := float32(math.Floor(float64(m.total*k))) / k
	var out outline.Path
	if dotted && side*k > 3 {
		if length < 2*side {
			q, _ := m.at(0)
			return outline.Ellipse(q, side/2, side/2)
		}
		step := bestGap(length, side, side) + side - 0.01/k
		for s := float32(0); s < m.total; s += step {
			q, _ := m.at(s)
			out.Contours = append(out.Contours, outline.Ellipse(q, side/2, side/2).Contours...)
		}
		return out
	}
	dash, gap := side, side
	if !dotted {
		dash, gap = 2*side, side
		if side*k < 3 {
			dash, gap = 3*side, 2*side
		}
	}
	switch {
	case length <= 2*dash:
		dash, gap = m.total, 0
	case length <= 2*dash+2*gap:
		f := length / (2*dash + 2*gap)
		dash, gap = dash*f, gap*f
	case !dotted:
		gap = bestGap(length, dash, gap)
	}
	for s := float32(0); s < m.total; s += dash + gap {
		out.Contours = append(out.Contours, m.strip(s, min(s+dash, m.total), thick*1.1))
	}
	return out
}

// middle is the line down the middle of a rounded border, measured from
// where Skia starts stroking it.
type middle struct {
	pts []geom.Vec2
	// along is how far round each point is from the outline's first, and
	// then the whole way round; from is how far the start is.
	along       []float32
	total, from float32
	// turns are how far round each point is from the start, in order.
	turns []float32
}

func middleOf(r geom.Rect, w [4]float32, radius radii) (middle, bool) {
	line := roundedInset(r, radius, [4]float32{w[0] / 2, w[1] / 2, w[2] / 2, w[3] / 2})
	if len(line.Contours) == 0 {
		return middle{}, false
	}
	m := middle{pts: line.Contours[0]}
	m.along = make([]float32, len(m.pts)+1)
	in, _ := line.Bounds()
	top := geom.V2(in.X+max(radius[0]-w[3]/2, 0), in.Y)
	first, near := 0, float32(math.Inf(1))
	for i := range m.pts {
		m.along[i+1] = m.along[i] + m.pts[(i+1)%len(m.pts)].Sub(m.pts[i]).Len()
		if d := m.pts[i].Sub(top).LenSq(); d < near {
			first, near = i, d
		}
	}
	m.total, m.from = m.along[len(m.pts)], m.along[first]
	m.turns = make([]float32, len(m.pts))
	for i := range m.pts {
		m.turns[i] = float32(math.Mod(float64(m.along[i]-m.from+m.total), float64(m.total)))
	}
	slices.Sort(m.turns)
	return m, true
}

// at is the point s round from the start, and the normal there.
func (m middle) at(s float32) (geom.Vec2, geom.Vec2) {
	s = float32(math.Mod(float64(s+m.from), float64(m.total)))
	i, _ := slices.BinarySearchFunc(m.along[1:], s, cmp.Compare[float32])
	i = min(i, len(m.pts)-1)
	a, b := m.pts[i], m.pts[(i+1)%len(m.pts)]
	seg := m.along[i+1] - m.along[i]
	if seg <= 0 {
		return a, geom.V2(0, 1)
	}
	d := b.Sub(a).Scale(1 / seg)
	return a.Add(d.Scale(s - m.along[i])), geom.V2(-d.Y, d.X)
}

// strip is the outline of the stretch of the middle from a to b round,
// half wide either way of it.
func (m middle) strip(a, b, half float32) []geom.Vec2 {
	var left, right []geom.Vec2
	add := func(s float32) {
		q, n := m.at(s)
		left, right = append(left, q.Add(n.Scale(half))), append(right, q.Sub(n.Scale(half)))
	}
	add(a)
	for _, s := range m.turns {
		if s > a && s < b {
			add(s)
		}
	}
	add(b)
	slices.Reverse(right)
	return append(left, right...)
}

// bestGap is the gap between dashes of length dash round a closed path of
// length length, as Chrome picks it: of the whole numbers of dashes on
// either side of what gap would fit, the one whose gap comes nearer it.
func bestGap(length, dash, gap float32) float32 {
	fewer := float32(math.Floor(float64(length / (dash + gap))))
	more := fewer + 1
	if fewer < 1 {
		return (length - more*dash) / more
	}
	few, many := (length-fewer*dash)/fewer, (length-more*dash)/more
	if many <= 0 || abs(few-gap) < abs(many-gap) {
		return few
	}
	return many
}

// maskPart is an outline a mask covers, filled by the even-odd rule when
// evenOdd is set.
type maskPart struct {
	path    outline.Path
	evenOdd bool
}

// maskedRect fills with c what every one of parts covers, smoothed at
// their edges, through a mask at k device pixels a point over whole
// device pixels.
func (p *painter) maskedRect(parts []maskPart, c color.Color, k float32) {
	var bounds geom.Rect
	for i, m := range parts {
		b, ok := m.path.Bounds()
		if !ok {
			return
		}
		if i == 0 {
			bounds = b
			continue
		}
		bounds = intersect(bounds, b)
	}
	if bounds.W <= 0 || bounds.H <= 0 {
		return
	}
	x0, y0 := float32(math.Floor(float64(bounds.X*k)))/k, float32(math.Floor(float64(bounds.Y*k)))/k
	x1 := float32(math.Ceil(float64((bounds.X+bounds.W)*k))) / k
	y1 := float32(math.Ceil(float64((bounds.Y+bounds.H)*k))) / k
	bounds = geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
	shape := &clipShape{bounds: bounds, parts: parts}
	p.s.PushClipMask(bounds, masks.mask(shape, k))
	p.s.Rect(bounds, c)
	p.s.PopClip()
}

// partsMask is the mask of what every one of parts covers over r, w×h.
func partsMask(parts []maskPart, r geom.Rect, w, h int) *image.RGBA {
	out := parts[0].path.Mask(r, w, h, parts[0].evenOdd)
	for _, m := range parts[1:] {
		more := m.path.Mask(r, w, h, m.evenOdd)
		for i := 3; i < len(out.Pix); i += 4 {
			v := uint8((uint16(out.Pix[i])*uint16(more.Pix[i]) + 127) / 255) //nolint:gosec // at most 255
			out.Pix[i-3], out.Pix[i-2], out.Pix[i-1], out.Pix[i] = v, v, v, v
		}
	}
	return out
}

// oneRing reports a rounded border drawn as rings round the box: as wide
// all round, solid or double on every side, and of one colour.
func oneRing(w [4]float32, c [4]color.Color, s [4]style.BorderStyle) bool {
	for i := range 4 {
		if w[i] != w[0] || c[i] != c[0] || s[i] != s[0] {
			return false
		}
	}
	return s[0] == style.BorderSolid || s[0] == style.BorderDouble
}
