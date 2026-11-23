package svg

import (
	"image/color"
	"math"
	"strings"
)

type paintKind uint8

const (
	propStroke   = "stroke"
	elemSymbol   = "symbol"
	elemCircle   = "circle"
	elemEllipse  = "ellipse"
	elemLine     = "line"
	elemPolyline = "polyline"
	elemUse      = "use"
	elemRect     = "rect"
	elemPolygon  = "polygon"
)

const (
	paintNone paintKind = iota
	paintColor
	paintCurrent
	paintRef
)

// paintSpec is a fill or stroke property as written; a reference keeps
// what to paint with when it leads nowhere.
type paintSpec struct {
	kind     paintKind
	c        color.NRGBA
	ref      string
	fallback paintKind
}

func parsePaint(s string) (paintSpec, bool) {
	s = strings.TrimSpace(s)
	switch {
	case s == none:
		return paintSpec{}, true
	case strings.EqualFold(s, "currentColor"):
		return paintSpec{kind: paintCurrent}, true
	case strings.HasPrefix(s, "url("):
		end := strings.IndexByte(s, ')')
		if end < 0 {
			return paintSpec{}, false
		}
		p := paintSpec{kind: paintRef, ref: strings.Trim(s[4:end], `'" `)}
		if rest := strings.TrimSpace(s[end+1:]); rest != "" {
			if fb, valid := parsePaint(rest); valid && fb.kind != paintRef {
				p.fallback, p.c = fb.kind, fb.c
			}
		}
		return p, true
	}
	c, ok := parseColor(s)
	return paintSpec{kind: paintColor, c: c}, ok
}

// style is what an element inherits and may declare.
type style struct {
	fill, stroke               paintSpec
	color                      color.NRGBA
	fillOpacity, strokeOpacity float64
	width                      float64
	cap                        Cap
	join                       Join
	miter                      float64
	dashes                     []float64
	dashOffset                 float64
	fontSize                   float64
	evenOdd, hidden            bool
	strokeFirst                bool
	clipEvenOdd, pixelated     bool
	// markers are marker-start, marker-mid and marker-end.
	markers    [3]string
	fontFamily string
	bold       bool
	italic     bool
	anchor     textAnchor
}

type builder struct {
	doc    *document
	rules  []rule
	vw, vh float64 // what percentages are of
	color  color.NRGBA
	base   style

	paths     map[*node]Path
	gradients map[*node]*gradientDef
	styles    map[*node]style
	using     []*node
	// referring are the clip paths, masks, patterns and markers being
	// built, which none of them may refer to again.
	referring []*node
	missing   map[string]bool
	// clipping is set while the shapes of a clip path are read.
	clipping bool
	// sized is the symbol or svg a use is drawing, and the size it gives.
	sized *useSize
	// nesting is how many pictures this one is an image inside.
	nesting int

	visits, shapes, segments int
}

type useSize struct {
	n          *node
	w, h       string
	hasW, hasH bool
}

const maxVisits = 4 * MaxElements

func (b *builder) unsupported(what string) {
	if b.missing == nil {
		b.missing = map[string]bool{}
	}
	b.missing[what] = true
}

// props is what an element declares: its attributes, over them the
// style sheets' rules, over them its style attribute.
func (b *builder) props(n *node) map[string]string {
	inline, hasInline := n.attr["style"]
	var matched []*rule
	for i := range b.rules {
		if b.rules[i].matches(n) {
			matched = append(matched, &b.rules[i])
		}
	}
	if matched == nil && !hasInline {
		return n.attr
	}
	out := make(map[string]string, len(n.attr)+8)
	for k, v := range n.attr {
		out[k] = v
	}
	decl, important := declarations(inline)
	for _, r := range matched {
		for k, v := range r.decl {
			if !r.important[k] {
				out[k] = v
			}
		}
	}
	for k, v := range decl {
		if !important[k] {
			out[k] = v
		}
	}
	for _, r := range matched {
		for k := range r.important {
			out[k] = r.decl[k]
		}
	}
	for k := range important {
		out[k] = decl[k]
	}
	return out
}

// target is the element a reference such as #id or url(#id) names, nil
// for none; a reference out of the file is reported and never followed.
func (b *builder) target(ref string) *node {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil
	}
	if !strings.HasPrefix(ref, "#") {
		b.unsupported("external reference")
		return nil
	}
	return b.doc.ids[ref[1:]]
}

// properties are the inherited properties an element may declare, each
// with how its value is taken; a value that is no value of the property
// leaves what was inherited.
var properties = map[string]func(st *style, v string, ctx lengthContext){ //nolint:gochecknoglobals // a constant table
	"color": func(st *style, v string, _ lengthContext) {
		if c, ok := parseColor(v); ok {
			st.color = c
		}
	},
	"fill": func(st *style, v string, _ lengthContext) {
		if paint, ok := parsePaint(v); ok {
			st.fill = paint
		}
	},
	propStroke: func(st *style, v string, _ lengthContext) {
		if paint, ok := parsePaint(v); ok {
			st.stroke = paint
		}
	},
	"fill-opacity": func(st *style, v string, _ lengthContext) {
		if f, ok := fraction(v); ok {
			st.fillOpacity = f
		}
	},
	"stroke-opacity": func(st *style, v string, _ lengthContext) {
		if f, ok := fraction(v); ok {
			st.strokeOpacity = f
		}
	},
	"stroke-width": func(st *style, v string, ctx lengthContext) {
		if w, ok := length(v, ctx); ok && w >= 0 {
			st.width = w
		}
	},
	"stroke-linecap": func(st *style, v string, _ lengthContext) {
		if c, ok := map[string]Cap{"butt": ButtCap, "round": RoundCap, "square": SquareCap}[v]; ok {
			st.cap = c
		}
	},
	"stroke-linejoin": func(st *style, v string, _ lengthContext) {
		joins := map[string]Join{
			"miter": MiterJoin, "miter-clip": MiterJoin, "arcs": MiterJoin, "round": RoundJoin, "bevel": BevelJoin,
		}
		if j, ok := joins[v]; ok {
			st.join = j
		}
	},
	"stroke-miterlimit": func(st *style, v string, _ lengthContext) {
		if limit, ok := number(v); ok && limit >= 1 {
			st.miter = limit
		}
	},
	"stroke-dasharray": (*style).dasharray,
	"stroke-dashoffset": func(st *style, v string, ctx lengthContext) {
		if off, ok := length(v, ctx); ok {
			st.dashOffset = off
		}
	},
	"fill-rule": func(st *style, v string, _ lengthContext) {
		if rule, ok := map[string]bool{"evenodd": true, "nonzero": false}[v]; ok {
			st.evenOdd = rule
		}
	},
	"visibility": func(st *style, v string, _ lengthContext) {
		if hidden, ok := map[string]bool{valVisible: false, "hidden": true, "collapse": true}[v]; ok {
			st.hidden = hidden
		}
	},
	"clip-rule": func(st *style, v string, _ lengthContext) {
		if rule, ok := map[string]bool{"evenodd": true, "nonzero": false}[v]; ok {
			st.clipEvenOdd = rule
		}
	},
	"image-rendering": func(st *style, v string, _ lengthContext) {
		st.pixelated = v == "pixelated" || v == "optimizeSpeed"
	},
	"font-family": func(st *style, v string, _ lengthContext) { st.fontFamily = strings.ToLower(v) },
	"font-weight": func(st *style, v string, _ lengthContext) {
		if n, ok := number(v); ok {
			st.bold = n >= 600
			return
		}
		switch v {
		case "bold", "bolder":
			st.bold = true
		case valNormal, "lighter":
			st.bold = false
		}
	},
	"font-style": func(st *style, v string, _ lengthContext) {
		if v == valNormal || v == "italic" || strings.HasPrefix(v, "oblique") {
			st.italic = v != valNormal
		}
	},
	"text-anchor": func(st *style, v string, _ lengthContext) {
		if a, ok := map[string]textAnchor{"start": anchorStart, "middle": anchorMiddle, "end": anchorEnd}[v]; ok {
			st.anchor = a
		}
	},
	"paint-order": func(st *style, v string, _ lengthContext) {
		order := strings.Fields(v)
		st.strokeFirst = len(order) > 0 && (order[0] == propStroke ||
			order[0] == "markers" && len(order) > 1 && order[1] == propStroke)
	},
}

func (st *style) apply(p map[string]string, b *builder) {
	if v, ok := p["font-size"]; ok {
		if size, valid := length(v, lengthContext{whole: st.fontSize, em: st.fontSize}); valid && size >= 0 {
			st.fontSize = size
		}
	}
	ctx := lengthContext{whole: math.Sqrt((b.vw*b.vw + b.vh*b.vh) / 2), em: st.fontSize}
	for k, v := range p {
		if set, ok := properties[k]; ok {
			set(st, strings.TrimSpace(v), ctx)
		}
	}
	if v, ok := p["marker"]; ok {
		v = strings.TrimSpace(v)
		st.markers = [3]string{v, v, v}
	}
	for i, k := range [...]string{"marker-start", "marker-mid", "marker-end"} {
		if v, ok := p[k]; ok {
			st.markers[i] = strings.TrimSpace(v)
		}
	}
}

func (st *style) dasharray(v string, ctx lengthContext) {
	if v == none {
		st.dashes = nil
		return
	}
	var dashes []float64
	sum := 0.0
	for _, f := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' }) {
		d, ok := length(f, ctx)
		if !ok || d < 0 || len(dashes) >= maxDashes {
			return
		}
		dashes = append(dashes, d)
		sum += d
	}
	if len(dashes) == 0 {
		return
	}
	if sum == 0 {
		st.dashes = nil
		return
	}
	if len(dashes)%2 == 1 {
		dashes = append(dashes, dashes...)
	}
	st.dashes = dashes
}

// rootSize reads the root's width or height: absent when it is a
// percentage or no length, since the picture then takes the room given.
func rootSize(s string) (float64, bool) {
	if strings.HasSuffix(strings.TrimSpace(s), "%") {
		return 0, false
	}
	v, ok := length(s, lengthContext{em: 16})
	return math.Max(v, 0), ok
}

func parseAspect(s string) Aspect {
	var a Aspect
	f := strings.Fields(s)
	if len(f) > 0 && f[0] == "defer" {
		f = f[1:]
	}
	if len(f) == 0 {
		return a
	}
	if f[0] == none {
		return Aspect{Stretch: true}
	}
	if len(f[0]) != len("xMidYMid") || f[0][0] != 'x' || f[0][4] != 'Y' {
		return a
	}
	axis := func(s string) (Align, bool) {
		switch s {
		case "Min":
			return AlignMin, true
		case "Mid":
			return AlignMid, true
		case "Max":
			return AlignMax, true
		}
		return AlignMid, false
	}
	x, okX := axis(f[0][1:4])
	y, okY := axis(f[0][5:8])
	if !okX || !okY {
		return a
	}
	return Aspect{X: x, Y: y, Slice: len(f) > 1 && f[1] == "slice"}
}

func build(doc *document, p *Picture) error {
	b := &builder{
		doc: doc, color: p.currentColor, nesting: p.nesting,
		paths: map[*node]Path{}, gradients: map[*node]*gradientDef{}, styles: map[*node]style{},
	}
	for _, sheet := range doc.sheets {
		b.rules = parseSheet(sheet, b.rules, MaxRules, b.unsupported)
	}
	root := doc.root
	props := b.props(root)
	p.width, p.hasWidth = rootSize(props["width"])
	p.height, p.hasHeight = rootSize(props["height"])
	p.ViewBox = viewBox(props["viewBox"])
	p.Aspect = parseAspect(props["preserveAspectRatio"])
	b.vw, b.vh = p.ViewBox.W, p.ViewBox.H
	if p.ViewBox.Empty() {
		b.vw, b.vh = p.Size()
	}
	b.base = style{
		fill: paintSpec{kind: paintColor, c: color.NRGBA{A: 255}}, color: p.currentColor,
		fillOpacity: 1, strokeOpacity: 1, width: 1, miter: 4, fontSize: 16,
	}
	p.transform = Identity()
	if t, ok := props["transform"]; ok {
		if tm, valid := parseTransform(t); valid && tm.finite() {
			p.transform = tm
		}
	}
	if err := b.walk(root, b.base, Identity(), &p.Items, 0); err != nil {
		return err
	}
	p.unsupported = sortedKeys(b.missing)
	return nil
}

func viewBox(s string) Box {
	sc := &scanner{s: s}
	if v := sc.list(4); len(v) == 4 && sc.done() && v[2] > 0 && v[3] > 0 {
		return Box{v[0], v[1], v[2], v[3]}
	}
	return Box{}
}

// fold is group as an item: itself, or its one shape, image or group
// faded by its opacity when that needs no layer.
func fold(g *Group) Item {
	if len(g.Items) != 1 || g.Clip != nil || g.Mask != nil {
		return Item{Group: g}
	}
	only := g.Items[0]
	switch {
	case only.Shape != nil:
		s := only.Shape
		switch {
		case s.Stroke.Kind == NoPaint:
			s.Fill.Opacity *= g.Opacity
			return only
		case s.Fill.Kind == NoPaint:
			s.Stroke.Opacity *= g.Opacity
			return only
		}
		return Item{Group: g}
	case only.Image != nil:
		only.Image.Opacity *= g.Opacity
		return only
	}
	only.Group.Opacity *= g.Opacity
	return only
}

// drawn reports whether n is an element that draws; one that this
// package does not draw is noted.
func (b *builder) drawn(n *node) bool {
	if b.clipping {
		switch n.name {
		case elemUse, elemPath, elemRect, elemCircle, elemEllipse, elemLine, elemPolyline, elemPolygon, elemText:
			return true
		}
		return false
	}
	switch n.name {
	case elemSVG, "g", "a", "switch", elemUse, elemPath, elemRect, elemCircle, elemEllipse, elemLine, elemPolyline,
		elemPolygon, elemText, elemImage:
		return true
	case elemSymbol:
		return b.sized != nil && b.sized.n == n
	case "foreignObject":
		b.unsupported(n.name)
	case "animate", "animateTransform", "animateMotion", "set":
		b.unsupported("animation")
	}
	return false
}

// reference is the element a property such as clip-path or mask names
// by url(#id), if it is one of the kind given and not already being
// built: nil for none, and cyclic for one being built.
func (b *builder) reference(v, prop, kind string) (*node, bool) {
	v = strings.TrimSpace(v)
	if v == "" || v == none {
		return nil, false
	}
	if !strings.HasPrefix(v, "url(") {
		b.unsupported(prop)
		return nil, false
	}
	spec, _ := parsePaint(v)
	n := b.target(spec.ref)
	if n == nil || n.name != kind {
		return nil, false
	}
	for _, busy := range b.referring {
		if busy == n {
			return nil, true
		}
	}
	return n, false
}

func (b *builder) walk(n *node, st style, m Matrix, out *[]Item, useDepth int) error {
	if b.visits++; b.visits > maxVisits {
		return ErrTooComplex
	}
	props := b.props(n)
	if strings.TrimSpace(props["display"]) == none || !b.drawn(n) {
		return nil
	}
	st.apply(props, b)
	if t, ok := props["transform"]; ok && n != b.doc.root {
		if tm, valid := parseTransform(t); valid {
			m = m.Mul(tm)
		}
	}
	if b.clipping {
		return b.content(n, props, st, m, out, useDepth)
	}
	for _, k := range [...]string{"filter", "mix-blend-mode", "vector-effect"} {
		if v, ok := props[k]; ok && v != none && v != valNormal && v != "" {
			b.unsupported(k)
		}
	}
	opacity := 1.0
	if f, ok := fraction(props["opacity"]); ok {
		opacity = f
	}
	clip, cyclicClip := b.reference(props["clip-path"], "clip-path", "clipPath")
	mask, cyclicMask := b.reference(props["mask"], "mask", "mask")
	clipped, masked := clip != nil, mask != nil
	if opacity == 0 || cyclicClip || cyclicMask {
		return nil
	}
	if opacity == 1 && !clipped && !masked {
		return b.content(n, props, st, m, out, useDepth)
	}
	group := &Group{Opacity: opacity}
	if err := b.content(n, props, st, m, &group.Items, useDepth); err != nil {
		return err
	}
	if len(group.Items) == 0 {
		return nil
	}
	if !clipped && !masked {
		*out = append(*out, fold(group))
		return nil
	}
	bbox := Box{}
	if inv, ok := m.Inverse(); ok {
		bbox = itemsBox(group.Items, inv)
	}
	var err error
	if clipped {
		if group.Clip, err = b.clipPath(clip, m, bbox); err != nil {
			return err
		}
	}
	if masked {
		if group.Mask, err = b.mask(mask, m, bbox); err != nil {
			return err
		}
	}
	*out = append(*out, Item{Group: group})
	return nil
}

// content appends what n draws, its own style applied, to out.
func (b *builder) content(n *node, props map[string]string, st style, m Matrix, out *[]Item, useDepth int) error {
	switch n.name {
	case elemSVG, elemSymbol:
		if n != b.doc.root {
			return b.viewport(n, props, st, m, out, useDepth)
		}
		return b.kids(n, st, m, out, useDepth)
	case "g", "a":
		return b.kids(n, st, m, out, useDepth)
	case "switch":
		for _, k := range n.kids {
			if _, conditional := k.attr["requiredExtensions"]; !conditional && k.name != "" {
				return b.walk(k, st, m, out, useDepth)
			}
		}
		return nil
	case elemUse:
		return b.use(n, props, st, m, out, useDepth)
	case elemText:
		return b.text(n, st, m, out)
	case elemImage:
		return b.image(props, &st, m, out)
	}
	return b.shape(n, props, &st, m, out)
}

func (b *builder) kids(n *node, st style, m Matrix, out *[]Item, useDepth int) error {
	for _, k := range n.kids {
		if err := b.walk(k, st, m, out, useDepth); err != nil {
			return err
		}
	}
	return nil
}

// viewport appends what a nested svg, or a symbol a use draws, draws:
// its view box fitted into its own viewport, cut to it unless its
// overflow is visible.
func (b *builder) viewport(n *node, props map[string]string, st style, m Matrix, out *[]Item, useDepth int) error {
	ws, hs := props["width"], props["height"]
	if s := b.sized; s != nil && s.n == n {
		if s.hasW {
			ws = s.w
		}
		if s.hasH {
			hs = s.h
		}
	}
	b.sized = nil
	size := func(s string, whole float64) float64 {
		if v, ok := length(s, lengthContext{whole: whole, em: st.fontSize}); ok && strings.TrimSpace(s) != valAuto {
			return v
		}
		return whole
	}
	w, h := size(ws, b.vw), size(hs, b.vh)
	if !(w > 0) || !(h > 0) {
		return nil
	}
	if n.name == elemSVG {
		x, _ := length(props["x"], lengthContext{whole: b.vw, em: st.fontSize})
		y, _ := length(props["y"], lengthContext{whole: b.vh, em: st.fontSize})
		m = m.Mul(translate(x, y))
	}
	inner := m
	vw, vh := w, h
	if box := viewBox(props["viewBox"]); !box.Empty() {
		inner = m.Mul(fit(box, parseAspect(props["preserveAspectRatio"]), w, h))
		vw, vh = box.W, box.H
	}
	group := &Group{Opacity: 1}
	if clipsOverflow(props) {
		group.Clip = rectClip(Box{0, 0, w, h}, m)
	}
	savedW, savedH := b.vw, b.vh
	b.vw, b.vh = vw, vh
	err := b.kids(n, st, inner, &group.Items, useDepth)
	b.vw, b.vh = savedW, savedH
	if err != nil || len(group.Items) == 0 {
		return err
	}
	if group.Clip == nil {
		*out = append(*out, group.Items...)
		return nil
	}
	*out = append(*out, Item{Group: group})
	return nil
}

func rectClip(r Box, m Matrix) *Clip {
	return &Clip{Shapes: []*Shape{{Path: rectPath(r.X, r.Y, r.W, r.H, 0, 0), Transform: m}}}
}

func (b *builder) use(n *node, props map[string]string, st style, m Matrix, out *[]Item, depth int) error {
	ref := b.target(props["href"])
	if ref == nil {
		return nil
	}
	if b.clipping {
		switch ref.name {
		case elemPath, elemRect, elemCircle, elemEllipse, elemLine, elemPolyline, elemPolygon, elemText:
		default:
			return nil
		}
	}
	for _, busy := range b.using {
		if busy == ref {
			return nil
		}
	}
	for at := n.parent; at != nil; at = at.parent {
		if at == ref {
			return nil
		}
	}
	if depth >= MaxUseDepth {
		return ErrTooComplex
	}
	x, _ := length(props["x"], lengthContext{whole: b.vw, em: st.fontSize})
	y, _ := length(props["y"], lengthContext{whole: b.vh, em: st.fontSize})
	if ref.name == elemSymbol || ref.name == elemSVG {
		w, hasW := props["width"]
		h, hasH := props["height"]
		b.sized = &useSize{n: ref, w: w, h: h, hasW: hasW, hasH: hasH}
	}
	b.using = append(b.using, ref)
	err := b.walk(ref, st, m.Mul(translate(x, y)), out, depth+1)
	b.using = b.using[:len(b.using)-1]
	b.sized = nil
	return err
}

func (b *builder) path(n *node, props map[string]string, em float64) Path {
	if p, ok := b.paths[n]; ok {
		return p
	}
	budget := MaxSegments - b.segments + 1
	x := func(k string) float64 {
		v, _ := length(props[k], lengthContext{whole: b.vw, em: em})
		return v
	}
	y := func(k string) float64 {
		v, _ := length(props[k], lengthContext{whole: b.vh, em: em})
		return v
	}
	var p Path
	switch n.name {
	case elemPath:
		p = parsePath(props["d"], budget)
	case elemRect:
		rx, okX := length(props["rx"], lengthContext{whole: b.vw, em: em})
		ry, okY := length(props["ry"], lengthContext{whole: b.vh, em: em})
		okX, okY = okX && rx >= 0, okY && ry >= 0
		switch {
		case !okX && !okY:
			rx, ry = 0, 0
		case !okX:
			rx = ry
		case !okY:
			ry = rx
		}
		p = rectPath(x("x"), y("y"), x("width"), y("height"), rx, ry)
	case elemCircle:
		r, _ := length(props["r"], lengthContext{whole: math.Sqrt((b.vw*b.vw + b.vh*b.vh) / 2), em: em})
		p = ellipsePath(Point{x("cx"), y("cy")}, r, r)
	case elemEllipse:
		p = ellipsePath(Point{x("cx"), y("cy")}, x("rx"), y("ry"))
	case elemLine:
		p.moveTo(Point{x("x1"), y("y1")})
		p.lineTo(Point{x("x2"), y("y2")})
	case elemPolyline, elemPolygon:
		p = pointsPath(props["points"], n.name == elemPolygon, budget)
	}
	for _, q := range p.Points {
		if !q.finite() {
			p = Path{}
			break
		}
	}
	if _, styled := n.attr["style"]; !styled && len(b.rules) == 0 {
		b.paths[n] = p
	}
	return p
}

func (b *builder) shape(n *node, props map[string]string, st *style, m Matrix, out *[]Item) error {
	if st.hidden {
		return nil
	}
	path := b.path(n, props, st.fontSize)
	if len(path.Verbs) == 0 || !m.finite() || m.det() == 0 {
		return nil
	}
	if err := b.count(len(path.Verbs)); err != nil {
		return err
	}
	if b.clipping {
		*out = append(*out, Item{Shape: b.clipShape(path, st, m)})
		return nil
	}
	bounds, _ := path.Bounds()
	if s := b.paintShape(path, st, m, bounds, n.name == elemLine); s != nil {
		*out = append(*out, Item{Shape: s})
	}
	if n.name != elemRect && n.name != elemCircle && n.name != elemEllipse {
		return b.markers(path, st, m, out)
	}
	return nil
}

func (b *builder) count(segments int) error {
	b.shapes++
	b.segments += segments
	if b.shapes > MaxShapes || b.segments > MaxSegments {
		return ErrTooComplex
	}
	return nil
}

// clipShape is path as a shape of a clip path: its outline alone, under
// the clip rule.
func (b *builder) clipShape(path Path, st *style, m Matrix) *Shape {
	s := &Shape{Path: path, Transform: m, Fill: Paint{Kind: Solid, Color: color.NRGBA{A: 255}, Opacity: 1}}
	if st.clipEvenOdd {
		s.FillRule = EvenOdd
	}
	return s
}

// paintShape is path with the paints of st, those in bounding-box units
// taken from bounds; nil when it paints nothing.
func (b *builder) paintShape(path Path, st *style, m Matrix, bounds Box, unfilled bool) *Shape {
	s := &Shape{
		Path: path, Transform: m,
		StrokeWidth: st.width, Cap: st.cap, Join: st.join, MiterLimit: st.miter,
		StrokeFirst: st.strokeFirst,
	}
	if !unfilled {
		s.Fill = b.paint(st.fill, st.fillOpacity, st, bounds)
	}
	if st.evenOdd {
		s.FillRule = EvenOdd
	}
	if st.width > 0 {
		s.Stroke = b.paint(st.stroke, st.strokeOpacity, st, bounds)
	}
	if s.Stroke.Kind == NoPaint {
		s.StrokeWidth = 0
	} else {
		s.Dashes, s.DashOffset = st.dashes, st.dashOffset
	}
	if s.Fill.Kind == NoPaint && s.Stroke.Kind == NoPaint {
		return nil
	}
	return s
}

func (b *builder) paint(spec paintSpec, opacity float64, st *style, bounds Box) Paint {
	solid := func(c color.NRGBA) Paint {
		if c.A == 0 || opacity == 0 {
			return Paint{}
		}
		return Paint{Kind: Solid, Color: c, Opacity: opacity}
	}
	switch spec.kind {
	case paintNone:
	case paintColor:
		return solid(spec.c)
	case paintCurrent:
		return solid(st.color)
	case paintRef:
		ref := b.target(spec.ref)
		if ref != nil && ref.name == elemPattern {
			if paint, ok := b.pattern(ref, opacity, bounds); ok {
				return paint
			}
		} else if paint, ok := b.gradient(ref, opacity, bounds); ok {
			return paint
		}
		switch spec.fallback {
		case paintColor:
			return solid(spec.c)
		case paintCurrent:
			return solid(st.color)
		case paintNone, paintRef:
		}
	}
	return Paint{}
}
