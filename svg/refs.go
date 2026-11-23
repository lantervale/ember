package svg

import (
	"math"
	"strings"
)

const (
	elemText    = "text"
	elemImage   = "image"
	elemPattern = "pattern"
	elemMarker  = "marker"
	unitsBox    = "objectBoundingBox"
	unitsUser   = "userSpaceOnUse"
	elemPath    = "path"
	valAuto     = "auto"
	valVisible  = "visible"
	valNormal   = "normal"
	valRound    = "round"
	valEvenOdd  = "evenodd"
	valNonZero  = "nonzero"
)

// clipsOverflow reports a viewport that cuts what overflows it.
func clipsOverflow(props map[string]string) bool {
	o := strings.TrimSpace(props["overflow"])
	return o != valVisible && o != valAuto
}

// inherited is the style n's content inherits: n's own, over its
// ancestors'.
func (b *builder) inherited(n *node) style {
	if st, ok := b.styles[n]; ok {
		return st
	}
	st := b.base
	if n.parent != nil {
		st = b.inherited(n.parent)
	}
	st.apply(b.props(n), b)
	b.styles[n] = st
	return st
}

// refer runs build with n among the references being built.
func (b *builder) refer(n *node, build func() error) error {
	b.referring = append(b.referring, n)
	err := build()
	b.referring = b.referring[:len(b.referring)-1]
	return err
}

// boxMatrix takes bounding-box units, 0–1, to the box.
func boxMatrix(box Box) Matrix { return Matrix{box.W, 0, 0, box.H, box.X, box.Y} }

// itemsBox is the bounding box of what items draw, their fills' outlines
// alone, in the space m takes user space to.
func itemsBox(items []Item, m Matrix) Box {
	lo, hi := Point{math.Inf(1), math.Inf(1)}, Point{math.Inf(-1), math.Inf(-1)}
	var walk func(items []Item)
	take := func(box Box, t Matrix) {
		x1, y1 := box.X+box.W, box.Y+box.H
		for _, c := range [...]Point{{box.X, box.Y}, {x1, box.Y}, {box.X, y1}, {x1, y1}} {
			q := t.Apply(c)
			lo = Point{math.Min(lo.X, q.X), math.Min(lo.Y, q.Y)}
			hi = Point{math.Max(hi.X, q.X), math.Max(hi.Y, q.Y)}
		}
	}
	walk = func(items []Item) {
		for _, it := range items {
			switch {
			case it.Shape != nil:
				if box, ok := it.Shape.Path.Bounds(); ok {
					take(box, m.Mul(it.Shape.Transform))
				}
			case it.Image != nil && it.Image.Pix != nil:
				r := it.Image.Pix.Rect
				take(Box{0, 0, float64(r.Dx()), float64(r.Dy())}, m.Mul(it.Image.Transform))
			case it.Group != nil:
				walk(it.Group.Items)
			}
		}
	}
	walk(items)
	if !(lo.X <= hi.X) || !lo.finite() || !hi.finite() {
		return Box{}
	}
	return Box{lo.X, lo.Y, hi.X - lo.X, hi.Y - lo.Y}
}

// shapesOf lists the shapes of items, those inside groups too.
func shapesOf(items []Item, out []*Shape) []*Shape {
	for _, it := range items {
		if it.Shape != nil {
			out = append(out, it.Shape)
		} else if it.Group != nil {
			out = shapesOf(it.Group.Items, out)
		}
	}
	return out
}

// clipPath is the clip of the clipPath element n on an element whose
// user space m takes to the picture's, its bounding box bbox there.
func (b *builder) clipPath(n *node, m Matrix, bbox Box) (*Clip, error) {
	props := b.props(n)
	inner := m
	if t, ok := parseTransform(props["transform"]); ok {
		inner = inner.Mul(t)
	}
	c := &Clip{}
	if strings.TrimSpace(props["clipPathUnits"]) == unitsBox {
		if bbox.Empty() {
			return c, nil
		}
		inner = inner.Mul(boxMatrix(bbox))
	}
	err := b.refer(n, func() error {
		var items []Item
		st := b.inherited(n)
		was := b.clipping
		b.clipping = true
		err := b.kids(n, st, inner, &items, 0)
		b.clipping = was
		if err != nil {
			return err
		}
		c.Shapes = shapesOf(items, nil)
		if next, cyclic := b.reference(props["clip-path"], "clip-path", "clipPath"); next != nil {
			c.Clip, err = b.clipPath(next, m, bbox)
		} else if cyclic {
			c.Shapes = nil
		}
		return err
	})
	return c, err
}

// region is the rectangle of a mask or pattern: x, y, width and height,
// each defaulting to def's, in bounding-box units unless units says
// userSpaceOnUse.
func (b *builder) region(props map[string]string, units string, bbox Box, def Box) Box {
	user := strings.TrimSpace(props[units]) == unitsUser
	get := func(k string, whole, d float64) float64 {
		if !user {
			whole = 1
		}
		if v, ok := length(props[k], lengthContext{whole: whole, em: 16}); ok {
			return v
		}
		return d
	}
	r := Box{get("x", b.vw, def.X), get("y", b.vh, def.Y), get("width", b.vw, def.W), get("height", b.vh, def.H)}
	if user {
		return r
	}
	return Box{bbox.X + r.X*bbox.W, bbox.Y + r.Y*bbox.H, r.W * bbox.W, r.H * bbox.H}
}

// mask is the mask of the mask element n on an element whose user space
// m takes to the picture's, its bounding box bbox there.
func (b *builder) mask(n *node, m Matrix, bbox Box) (*Mask, error) {
	props := b.props(n)
	mk := &Mask{Alpha: strings.TrimSpace(props["mask-type"]) == "alpha"}
	userRegion := strings.TrimSpace(props["maskUnits"]) == unitsUser
	if !userRegion && bbox.Empty() {
		mk.Region = &Clip{}
		return mk, nil
	}
	def := Box{-0.1, -0.1, 1.2, 1.2}
	if userRegion {
		def = Box{-0.1 * b.vw, -0.1 * b.vh, 1.2 * b.vw, 1.2 * b.vh}
	}
	region := b.region(props, "maskUnits", bbox, def)
	if region.Empty() {
		mk.Region = &Clip{}
		return mk, nil
	}
	mk.Region = rectClip(region, m)
	content := m
	if strings.TrimSpace(props["maskContentUnits"]) == unitsBox {
		if bbox.Empty() {
			return mk, nil
		}
		content = m.Mul(boxMatrix(bbox))
	}
	err := b.refer(n, func() error { return b.kids(n, b.inherited(n), content, &mk.Items, 0) })
	return mk, err
}

// patternDef is a pattern element's attributes with those of the
// patterns it refers to, and the one whose children it draws.
func (b *builder) patternDef(n *node) (map[string]string, *node) {
	chain := []*node{n}
	for at := n; len(chain) < 16; {
		next := b.target(at.attr["href"])
		if next == nil || next.name != elemPattern {
			break
		}
		for _, seen := range chain {
			if seen == next {
				next = nil
				break
			}
		}
		if next == nil {
			break
		}
		chain = append(chain, next)
		at = next
	}
	attr := map[string]string{}
	for i := len(chain) - 1; i >= 0; i-- {
		for k, v := range b.props(chain[i]) {
			if k != "transform" {
				attr[k] = v
			}
		}
	}
	for _, c := range chain {
		for _, k := range c.kids {
			if k.name != "" {
				return attr, c
			}
		}
	}
	return attr, n
}

// pattern is the paint of the pattern element n on a shape whose path is
// within bounds; ok is false when n is one being built.
func (b *builder) pattern(n *node, opacity float64, bounds Box) (Paint, bool) {
	for _, busy := range b.referring {
		if busy == n {
			return Paint{}, false
		}
	}
	if opacity == 0 {
		return Paint{}, true
	}
	attr, content := b.patternDef(n)
	boxUnits := strings.TrimSpace(attr["patternUnits"]) != unitsUser
	if boxUnits && bounds.Empty() {
		return Paint{}, true
	}
	tile := b.region(attr, "patternUnits", bounds, Box{})
	if tile.Empty() {
		return Paint{}, true
	}
	pat := &Pattern{Tile: Box{0, 0, tile.W, tile.H}, Transform: translate(tile.X, tile.Y)}
	if t, ok := parseTransform(attr["patternTransform"]); ok {
		pat.Transform = t.Mul(pat.Transform)
	}
	if _, ok := pat.Transform.Inverse(); !ok || !pat.Transform.finite() {
		return Paint{}, true
	}
	inner := Identity()
	if box := viewBox(attr["viewBox"]); !box.Empty() {
		inner = fit(box, parseAspect(attr["preserveAspectRatio"]), tile.W, tile.H)
	} else if strings.TrimSpace(attr["patternContentUnits"]) == unitsBox {
		if bounds.Empty() {
			return Paint{}, true
		}
		inner = scale(bounds.W, bounds.H)
	}
	saved := b.clipping
	b.clipping = false
	err := b.refer(n, func() error {
		return b.refer(content, func() error { return b.kids(content, b.inherited(content), inner, &pat.Items, 0) })
	})
	b.clipping = saved
	if err != nil || len(pat.Items) == 0 {
		return Paint{}, true
	}
	return Paint{Kind: PatternPaint, Pattern: pat, Opacity: opacity}, true
}

func rotation(deg float64) Matrix {
	s, c := math.Sincos(deg * math.Pi / 180)
	return Matrix{c, s, -s, c, 0, 0}
}

// vertex is where a marker may be drawn on a path: the point, and the
// directions of the segments into and out of it, when there are some.
type vertex struct {
	at          Point
	in, out     Point
	hasIn, hasO bool
}

// angle is the marker's direction at v, in degrees: the bisector of the
// segments in and out of it.
func (v vertex) angle() float64 {
	in := math.Atan2(v.in.Y, v.in.X) * 180 / math.Pi
	out := math.Atan2(v.out.Y, v.out.X) * 180 / math.Pi
	switch {
	case v.hasIn && v.hasO:
		if math.Abs(in-out) > 180 {
			out += 360
		}
		return (in + out) / 2
	case v.hasIn:
		return in
	case v.hasO:
		return out
	}
	return 0
}

func firstNonZero(ds ...Point) Point {
	for _, d := range ds {
		if d != (Point{}) {
			return d
		}
	}
	return Point{}
}

// vertices are the places of a path markers go: every point a step of
// it ends at.
func vertices(p Path) []vertex {
	var out []vertex
	var cur, start Point
	startAt := -1
	i := 0
	segment := func(to, in, out0 Point) {
		if n := len(out); n > 0 {
			out[n-1].out, out[n-1].hasO = out0, true
		}
		out = append(out, vertex{at: to, in: in, hasIn: true})
		cur = to
	}
	for _, v := range p.Verbs {
		switch v {
		case MoveTo:
			cur, start = p.Points[i], p.Points[i]
			startAt = len(out)
			out = append(out, vertex{at: cur})
			i++
		case LineTo:
			d := p.Points[i].sub(cur)
			segment(p.Points[i], d, d)
			i++
		case QuadTo:
			c, to := p.Points[i], p.Points[i+1]
			segment(to, firstNonZero(to.sub(c), to.sub(cur)), firstNonZero(c.sub(cur), to.sub(cur)))
			i += 2
		case CubicTo:
			c1, c2, to := p.Points[i], p.Points[i+1], p.Points[i+2]
			segment(to, firstNonZero(to.sub(c2), to.sub(c1), to.sub(cur)), firstNonZero(c1.sub(cur), c2.sub(cur), to.sub(cur)))
			i += 3
		case Close:
			if startAt < 0 {
				continue
			}
			d := start.sub(cur)
			segment(start, d, d)
			last := &out[len(out)-1]
			first := &out[startAt]
			if first.hasO {
				last.out, last.hasO = first.out, true
				first.in, first.hasIn = d, true
			}
		}
	}
	return out
}

// markers appends the markers of a path drawn with st through m.
func (b *builder) markers(path Path, st *style, m Matrix, out *[]Item) error {
	if st.markers == [3]string{} {
		return nil
	}
	var refs [3]*node
	found := false
	for i, v := range st.markers {
		if n, _ := b.reference(v, elemMarker, elemMarker); n != nil {
			refs[i], found = n, true
		}
	}
	if !found {
		return nil
	}
	vs := vertices(path)
	for i, v := range vs {
		kind := 1
		switch i {
		case 0:
			kind = 0
		case len(vs) - 1:
			kind = 2
		}
		if refs[kind] == nil {
			continue
		}
		if err := b.marker(refs[kind], v, kind == 0, st.width, m, out); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) marker(n *node, v vertex, start bool, strokeWidth float64, m Matrix, out *[]Item) error {
	props := b.props(n)
	size := func(k string) float64 {
		if s, ok := length(props[k], lengthContext{whole: b.vw, em: 16}); ok {
			return s
		}
		return 3
	}
	w, h := size("markerWidth"), size("markerHeight")
	if !(w > 0) || !(h > 0) {
		return nil
	}
	refX, _ := length(props["refX"], lengthContext{whole: b.vw, em: 16})
	refY, _ := length(props["refY"], lengthContext{whole: b.vh, em: 16})
	inner := Identity()
	if box := viewBox(props["viewBox"]); !box.Empty() {
		inner = fit(box, parseAspect(props["preserveAspectRatio"]), w, h)
	}
	ref := inner.Apply(Point{refX, refY})
	angle := 0.0
	switch orient := strings.TrimSpace(props["orient"]); orient {
	case valAuto:
		angle = v.angle()
	case "auto-start-reverse":
		angle = v.angle()
		if start {
			angle += 180
		}
	default:
		if a, ok := angleValue(orient); ok {
			angle = a
		}
	}
	units := 1.0
	if strings.TrimSpace(props["markerUnits"]) != unitsUser {
		units = strokeWidth
	}
	if units == 0 {
		return nil
	}
	base := m.Mul(translate(v.at.X, v.at.Y)).Mul(rotation(angle)).Mul(scale(units, units)).Mul(translate(-ref.X, -ref.Y))
	group := &Group{Opacity: 1}
	if clipsOverflow(props) {
		group.Clip = rectClip(Box{0, 0, w, h}, base)
	}
	saved := b.clipping
	b.clipping = false
	err := b.refer(n, func() error { return b.kids(n, b.inherited(n), base.Mul(inner), &group.Items, 0) })
	b.clipping = saved
	if err != nil || len(group.Items) == 0 {
		return err
	}
	*out = append(*out, Item{Group: group})
	return nil
}

// angleValue reads an angle in degrees, or in the CSS units of one.
func angleValue(s string) (float64, bool) {
	sc := &scanner{s: s}
	v, ok := sc.number()
	if !ok {
		return 0, false
	}
	switch strings.ToLower(strings.TrimSpace(sc.s[sc.i:])) {
	case "", "deg":
		return v, true
	case "rad":
		return v * 180 / math.Pi, true
	case "grad":
		return v * 0.9, true
	case "turn":
		return v * 360, true
	}
	return 0, false
}

// relocate is a copy of items moved by m.
func relocate(items []Item, m Matrix) []Item {
	out := make([]Item, len(items))
	for i, it := range items {
		switch {
		case it.Shape != nil:
			s := *it.Shape
			s.Transform = m.Mul(s.Transform)
			out[i].Shape = &s
		case it.Image != nil:
			im := *it.Image
			im.Transform = m.Mul(im.Transform)
			out[i].Image = &im
		case it.Group != nil:
			g := *it.Group
			g.Items = relocate(g.Items, m)
			g.Clip = relocateClip(g.Clip, m)
			if g.Filters != nil {
				g.Filters = make([]*Filter, len(it.Group.Filters))
				for k, f := range it.Group.Filters {
					moved := *f
					moved.Transform = m.Mul(f.Transform)
					g.Filters[k] = &moved
				}
			}
			if g.Mask != nil {
				mk := *g.Mask
				mk.Items = relocate(mk.Items, m)
				mk.Region = relocateClip(mk.Region, m)
				g.Mask = &mk
			}
			out[i].Group = &g
		}
	}
	return out
}

func relocateClip(c *Clip, m Matrix) *Clip {
	if c == nil {
		return nil
	}
	out := &Clip{Clip: relocateClip(c.Clip, m)}
	for _, s := range c.Shapes {
		moved := *s
		moved.Transform = m.Mul(s.Transform)
		out.Shapes = append(out.Shapes, &moved)
	}
	return out
}
