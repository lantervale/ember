package layout

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// A vertical writing mode is laid out turned: the algorithms, all written for horizontal
// lines, lay a box whose lines run down it out in its frame, where x runs along its lines
// (down, or up for sideways-lr) and y across them, from the side its blocks start on. Each
// node's style is its own turned into the frame of its parent's content (turnStyle), and
// a node of another frame than its parent's, an orthogonal flow, lays its content out in
// its own (node.inner): its size is asked for and given across. Writing the boxes turns
// each one back from its parent's frame into the page's (turnRect).

// turnKey is a style turned into a frame.
type turnKey struct {
	st   *style.Computed
	mode style.WritingMode
}

// turned is st in the frame of mode, one copy each, kept from layout to layout with the
// nodes, so a node's style stays the same pointer while its element's does.
func (t *tree) turned(st *style.Computed, mode style.WritingMode) *style.Computed {
	if !mode.Vertical() {
		return st
	}
	k := turnKey{st, mode}
	if s, ok := t.turns[k]; ok {
		return s
	}
	if t.turns == nil {
		t.turns = map[turnKey]*style.Computed{}
	}
	s, ok := t.prevTurns[k]
	if !ok {
		s = turnStyle(st, mode)
	}
	t.turns[k] = s
	return s
}

// turnStyle is st as the frame of mode has it: its sizes across, its sides those of the
// frame, where top is the side its blocks start from and left the one its lines do.
func turnStyle(st *style.Computed, mode style.WritingMode) *style.Computed {
	c := *st
	c.Inset = turnSides(st.Inset, mode)
	c.Margin = turnSides(st.Margin, mode)
	c.Padding = turnSides(st.Padding, mode)
	c.Border = turnSides(st.Border, mode)
	c.Size = style.Size[style.Length]{Width: st.Size.Height, Height: st.Size.Width}
	c.MinSize = style.Size[style.Length]{Width: st.MinSize.Height, Height: st.MinSize.Width}
	c.MaxSize = style.Size[style.Length]{Width: st.MaxSize.Height, Height: st.MaxSize.Width}
	c.Overflow = style.Size[style.Overflow]{Width: st.Overflow.Height, Height: st.Overflow.Width}
	c.ScrollbarWidth, c.ScrollbarHeight = st.ScrollbarHeight, st.ScrollbarWidth
	c.BorderSpacing = [2]float32{st.BorderSpacing[1], st.BorderSpacing[0]}
	if st.AspectRatio > 0 {
		c.AspectRatio = 1 / st.AspectRatio
	}
	c.Anchoring.Area = turnArea(st.Anchoring.Area, mode)
	if len(st.Anchoring.Tries) > 0 {
		c.Anchoring.Tries = make([]*style.Computed, len(st.Anchoring.Tries))
		for i, try := range st.Anchoring.Tries {
			c.Anchoring.Tries[i] = turnStyle(try, mode)
		}
	}
	return &c
}

// frameSide is the page's side or size s an anchor function names in the frame of mode.
func frameSide(s style.AnchorSide, mode style.WritingMode) style.AnchorSide {
	if !mode.Vertical() {
		return s
	}
	if s == style.AnchorWidth || s == style.AnchorHeight {
		return style.AnchorWidth + style.AnchorHeight - s
	}
	page := style.Sides[style.AnchorSide]{Top: style.AnchorTop, Right: style.AnchorRight, Bottom: style.AnchorBottom,
		Left: style.AnchorLeft}
	framed := turnSides(page, mode)
	sides := [4]style.AnchorSide{style.AnchorTop, style.AnchorRight, style.AnchorBottom, style.AnchorLeft}
	for i, side := range [4]style.AnchorSide{framed.Top, framed.Right, framed.Bottom, framed.Left} {
		if s == side {
			return sides[i]
		}
	}
	return s
}

// turnArea is the page's position-area a in the frame of mode.
func turnArea(a style.PositionArea, mode style.WritingMode) style.PositionArea {
	if !a.Set {
		return a
	}
	flip := func(s style.AreaSpan) style.AreaSpan { return style.AreaSpan{From: 2 - s.To, To: 2 - s.From} }
	switch mode {
	case style.VerticalRL, style.SidewaysRL:
		a.X, a.Y = a.Y, flip(a.X)
	case style.VerticalLR:
		a.X, a.Y = a.Y, a.X
	case style.SidewaysLR:
		a.X, a.Y = flip(a.Y), a.X
	case style.HorizontalTB:
	}
	return a
}

// turnSides is the page's sides s in the frame of mode.
func turnSides[T any](s style.Sides[T], mode style.WritingMode) style.Sides[T] {
	switch mode {
	case style.VerticalRL, style.SidewaysRL:
		return style.Sides[T]{Top: s.Right, Bottom: s.Left, Left: s.Top, Right: s.Bottom}
	case style.VerticalLR:
		return style.Sides[T]{Top: s.Left, Bottom: s.Right, Left: s.Top, Right: s.Bottom}
	case style.SidewaysLR:
		return style.Sides[T]{Top: s.Left, Bottom: s.Right, Left: s.Bottom, Right: s.Top}
	case style.HorizontalTB:
	}
	return s
}

// outerFrame is the frame of the content of parent, where a node of physical style st
// stands: the root's own, as the initial containing block takes its writing mode.
func outerFrame(parent *node, st *style.Computed) style.WritingMode {
	if parent == nil {
		return st.Mode
	}
	return frameOf(parent)
}

// frameOf is the frame n lays its content out in: its writing mode's.
func frameOf(n *node) style.WritingMode {
	if n == nil || n.box.Style == nil {
		return style.HorizontalTB
	}
	return n.box.Style.Mode
}

// pageEdges are edges e of the frame of mode on the page.
func pageEdges(e rect, mode style.WritingMode) rect {
	switch mode {
	case style.VerticalRL, style.SidewaysRL:
		return rect{t: e.l, b: e.r, r: e.t, l: e.b}
	case style.VerticalLR:
		return rect{t: e.l, b: e.r, l: e.t, r: e.b}
	case style.SidewaysLR:
		return rect{b: e.l, t: e.r, l: e.t, r: e.b}
	case style.HorizontalTB:
	}
	return e
}

// frameEdges are the page's edges e in the frame of mode.
func frameEdges(e rect, mode style.WritingMode) rect {
	s := turnSides(style.Sides[float32]{Top: e.t, Right: e.r, Bottom: e.b, Left: e.l}, mode)
	return rect{t: s.Top, r: s.Right, b: s.Bottom, l: s.Left}
}

// turnRect is r, in a box of size in of the frame of mode, on the page.
func turnRect(r geom.Rect, in size, mode style.WritingMode) geom.Rect {
	switch mode {
	case style.VerticalRL, style.SidewaysRL:
		return geom.Rect{X: in.h - r.Y - r.H, Y: r.X, W: r.H, H: r.W}
	case style.VerticalLR:
		return geom.Rect{X: r.Y, Y: r.X, W: r.H, H: r.W}
	case style.SidewaysLR:
		return geom.Rect{X: r.Y, Y: in.w - r.X - r.W, W: r.H, H: r.W}
	case style.HorizontalTB:
	}
	return r
}

// unturnRect is r, in a box of size on the page, in the frame of mode.
func unturnRect(r geom.Rect, on size, mode style.WritingMode) geom.Rect {
	switch mode {
	case style.VerticalRL, style.SidewaysRL:
		return geom.Rect{X: r.Y, Y: on.w - r.X - r.W, W: r.H, H: r.W}
	case style.VerticalLR:
		return geom.Rect{X: r.Y, Y: r.X, W: r.H, H: r.W}
	case style.SidewaysLR:
		return geom.Rect{X: on.h - r.Y - r.H, Y: r.X, W: r.H, H: r.W}
	case style.HorizontalTB:
	}
	return r
}

// across reports two frames whose lines run on other axes.
func across(a, b style.WritingMode) bool { return a.Vertical() != b.Vertical() }

func (s size) across() size { return size{s.h, s.w} }

func (s spaces) across() spaces { return spaces{s.h, s.w} }

// rectOf and fromRect convert between a box's space and a rect's edges.
func rectOf(r rect) geom.Rect { return geom.Rect{X: r.l, Y: r.t, W: r.r - r.l, H: r.b - r.t} }

func fromRect(r geom.Rect) rect { return rect{l: r.X, t: r.Y, r: r.X + r.W, b: r.Y + r.H} }

// reframe is r, in a box of size in in the frame from, in the frame to.
func reframe(r rect, in size, from, to style.WritingMode) rect {
	on := in
	if from.Vertical() {
		on = in.across()
	}
	page := turnRect(rectOf(r), in, from)
	return fromRect(unturnRect(page, on, to))
}

// reversedIn is which axes of the frame to scroll from their end, given those of the
// frame from.
func reversedIn(rev [2]bool, from, to style.WritingMode) [2]bool {
	page := rev
	switch from {
	case style.VerticalRL, style.SidewaysRL:
		page = [2]bool{!rev[1], rev[0]}
	case style.VerticalLR:
		page = [2]bool{rev[1], rev[0]}
	case style.SidewaysLR:
		page = [2]bool{rev[1], !rev[0]}
	case style.HorizontalTB:
	}
	switch to {
	case style.VerticalRL, style.SidewaysRL:
		return [2]bool{page[1], !page[0]}
	case style.VerticalLR:
		return [2]bool{page[1], page[0]}
	case style.SidewaysLR:
		return [2]bool{!page[1], page[0]}
	case style.HorizontalTB:
	}
	return page
}

// inner is the style n lays its own content out with, in its own frame, when it isn't
// the one its parent sees: n stands in another frame than its parent's.
func (t *tree) frame(n *node, physical *style.Computed, outer style.WritingMode) {
	n.st = t.turned(physical, outer)
	n.inner, n.across = nil, false
	if mode := physical.Mode; mode != outer && (mode.Vertical() || outer.Vertical()) {
		if in := t.turned(physical, mode); in != n.st {
			n.inner = in
		}
		n.across = across(mode, outer)
	}
}

// turnedLayout lays out n, whose content is in another frame than its parent's: in its
// own, with its own style, the space it is given and the size it takes turned across
// when the two frames' lines run on other axes.
func (t *tree) turnedLayout(n *node, in input) output {
	outer := n.st
	n.st, n.inside = n.inner, true
	defer func() { n.st, n.inside = outer, false }()
	from, to := frameOf(n), frameOf(n.parent)
	if !n.across {
		out := t.computeChildIn(n, in, nil)
		out.overflow = t.reframeOverflow(n, out, from, to)
		return out
	}
	tin := in
	tin.known, tin.parent, tin.avail = in.known.across(), in.parent.across(), in.avail.across()
	tin.knownIsDefinite = bools{in.knownIsDefinite.h, in.knownIsDefinite.w}
	switch in.axis {
	case axisHorizontal:
		tin.axis = axisVertical
	case axisVertical:
		tin.axis = axisHorizontal
	case axisBoth:
	}
	tin.collapsible, tin.clamp = [2]bool{}, clampIn{}
	if !tin.avail.w.definite() {
		// An orthogonal flow's lines are as long as the initial containing block, as
		// Chrome's when nothing gives them a length.
		tin.avail.w = def(t.icbAlong(from))
	}
	out := t.computeChildIn(n, tin, nil)
	turnedOut := out
	turnedOut.size = out.size.across()
	turnedOut.overflow = t.reframeOverflow(n, out, from, to)
	turnedOut.firstBaseline, turnedOut.lastBaseline = none, none
	turnedOut.topMargin = marginFrom(resolveOrZero(outer.Margin.Top, in.parent.w))
	turnedOut.bottomMargin = marginFrom(resolveOrZero(outer.Margin.Bottom, in.parent.w))
	turnedOut.collapsesThrough, turnedOut.lines, turnedOut.clamped = false, 0, false
	turnedOut.oof = nil
	for _, c := range out.oof {
		turnedOut.oof = append(turnedOut.oof, turnCandidate(c, out.size, from, to))
	}
	if a := out.area; a != nil {
		r := reframe(fromRect(geom.Rect{X: a.offset.x, Y: a.offset.y, W: a.size.w, H: a.size.h}), out.size, from, to)
		turnedOut.area = &oofArea{size: size{r.r - r.l, r.b - r.t}, offset: point{r.l, r.t}}
	}
	return turnedOut
}

// icbAlong is the length of the initial containing block along the lines of mode.
func (t *tree) icbAlong(mode style.WritingMode) float32 {
	if mode.Vertical() {
		return t.icb.h
	}
	return t.icb.w
}

// reframeOverflow is the scrollable overflow of out, n's layout in the frame from, in
// the frame to, n's style being its own one.
func (t *tree) reframeOverflow(n *node, out output, from, to style.WritingMode) overflowRect {
	o := out.overflow
	if from == to || o.rect == (rect{}) && o.reversed == [2]bool{} {
		return o
	}
	g := n.ownGutter()
	port := out.size.sub(n.borders().sum()).sub(size{g.x, g.y})
	return overflowRect{rect: reframe(o.rect, port, from, to), reversed: reversedIn(o.reversed, from, to)}
}

// turnCandidate is c, an out-of-flow box placed in a box of size in of the frame from,
// in the frame to.
func turnCandidate(c oofCandidate, in size, from, to style.WritingMode) oofCandidate {
	r := reframe(rect{l: c.x.area.start, r: c.x.area.end, t: c.y.area.start, b: c.y.area.end}, in, from, to)
	x, y := c.x, c.y
	if across(from, to) {
		x, y = y, x
	}
	x.area, y.area = span{r.l, r.r}, span{r.t, r.b}
	c.x, c.y = x, y
	return c
}

// turnPlaced is p, a node's layout in the frame from, in the frame to: its place left as
// it is, which only its parent's frame gives.
func turnPlaced(p placed, from, to style.WritingMode) placed {
	if from == to {
		return p
	}
	q := p
	q.border = frameEdges(pageEdges(p.border, from), to)
	q.padding = frameEdges(pageEdges(p.padding, from), to)
	q.margin = frameEdges(pageEdges(p.margin, from), to)
	if across(from, to) {
		q.size, q.scrollbar = p.size.across(), p.scrollbar.across()
	}
	port := p.size.sub(p.border.sum()).sub(p.scrollbar)
	q.overflow = overflowRect{rect: reframe(p.overflow.rect, port, from, to),
		reversed: reversedIn(p.overflow.reversed, from, to)}
	return q
}

// turnBroken turns b, a box columns cut written at at in the frame of mode, inside a box
// of size in of that frame, onto the page: its place, the union of its parts, and them.
func turnBroken(b *box.Box, at geom.Vec2, in size, mode style.WritingMode) {
	if !mode.Vertical() || !b.Broken {
		return
	}
	u := size{b.Size.X, b.Size.Y}
	r := turnRect(geom.Rect{X: at.X, Y: at.Y, W: u.w, H: u.h}, in, mode)
	b.Pos, b.Size = geom.Vec2{X: r.X, Y: r.Y}, geom.Vec2{X: r.W, Y: r.H}
	for i, p := range b.Fragments {
		b.Fragments[i] = turnRect(p, u, mode)
	}
}

// turnLines turns the lines of c, written in the frame of mode inside its owner's box of
// size in, onto the page: each run's fragments, the rectangles of its runs and spans, its
// marker and its first line's background. Each fragment's Baseline is then the x its
// baseline stands at. In vertical-lr the lines stack from the left while the glyphs'
// tops still face right: what stands on each line is mirrored across it first.
func turnLines(c *inlineContent, in size, mode style.WritingMode, runRects []geom.Rect,
	spanRects map[*box.Box]geom.Rect, spanLines map[*box.Box][]geom.Rect) {
	mirror := func(r geom.Rect) geom.Rect {
		if mode != style.VerticalLR {
			return r
		}
		mid := r.Y + r.H/2
		for _, l := range c.lineBoxes {
			if mid >= l.start && mid <= l.end {
				r.Y = l.start + l.end - r.Y - r.H
				break
			}
		}
		return r
	}
	turn := func(r geom.Rect) geom.Rect { return turnRect(mirror(r), in, mode) }
	for _, r := range c.runs {
		if r.atomic != nil {
			continue
		}
		for j := range r.box.Lines {
			f := &r.box.Lines[j]
			y := f.Baseline
			if mode == style.VerticalLR {
				y = 2*f.Rect.Y + f.Rect.H - y
			}
			f.Baseline = turnRect(geom.Rect{Y: y}, in, mode).X
			f.Rect = turnRect(f.Rect, in, mode)
		}
	}
	for i, r := range runRects {
		runRects[i] = turnRect(r, in, mode)
	}
	for b, lines := range spanLines {
		if len(lines) == 0 {
			continue
		}
		for i, r := range lines {
			lines[i] = turn(r)
		}
		u := lines[0]
		for _, r := range lines[1:] {
			u = unionRect(u, r, true)
		}
		spanRects[b] = u
	}
	for b, r := range spanRects {
		if _, ok := spanLines[b]; !ok {
			spanRects[b] = turn(r)
		}
	}
	for _, b := range []*box.Box{c.marker, c.back} {
		if b != nil {
			r := turn(geom.Rect{X: b.Pos.X, Y: b.Pos.Y, W: b.Size.X, H: b.Size.Y})
			b.Pos, b.Size = geom.Vec2{X: r.X, Y: r.Y}, geom.Vec2{X: r.W, Y: r.H}
		}
	}
}

// advance is the length of text s of style st along its line: in vertical lines each
// cluster standing upright takes its vertical advance (box.Upright).
func advance(fonts box.Fonts, st *style.Computed, s string) float32 {
	if !st.Stands() {
		return fonts.Advance(st.Font, s)
	}
	var w float32
	style.Oriented(s, st.Orientation, func(part string, upright bool) {
		if upright {
			a, _ := box.Upright(fonts, st.Font, part)
			w += a
			return
		}
		w += fonts.Advance(st.Font, part)
	})
	return w
}
