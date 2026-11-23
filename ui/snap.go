package ui

import (
	"math"
	"slices"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/lantervale/ember/ui/layout"
	"github.com/lantervale/ember/ui/paint"
)

// A snap resolves where a scroll lands as Chrome's snap strategies do
// (decisions 0163 and 0169): snapEnd the snap position nearest where it
// was sent (scrollTo, a thumb let go, a layout); snapAhead the nearest
// to that among those ahead of where it began (scrollBy, a notch of the
// wheel); snapNext the first ahead (an arrow); snapPage the nearest to
// a page ahead among those from 0.4 of a page to a page away, else as
// snapAhead (a page key, a track). The three that go ahead stop at a
// scroll-snap-stop: always on the way.
type snapKind uint8

const (
	snapEnd snapKind = iota
	snapAhead
	snapNext
	snapPage
)

// proximity is the share of its scrollport within which a proximity snap
// container snaps, as Chrome's.
const proximity = 1.0 / 3

// wheelSettle is how many frames without the wheel end its gesture, which
// then snaps.
const wheelSettle = 2

// scrollerState is what d keeps of a scroll container between frames:
// the areas it is snapped to on each axis and the positions it last
// laid them out at, the way its last relative scroll went on each axis
// (scroll-state(scrolled)), a trackpad's gesture not yet snapped and a
// glide under way.
type scrollerState struct {
	targets  [2]*Node
	laid     []float32
	scrolled [2]int8
	wheeling bool
	wheelAt  geom.Vec2
	quiet    int
	glide    *glide
}

func (d *Document) scrollerOf(n *Node) *scrollerState {
	if d.scrollers == nil {
		d.scrollers = map[*Node]*scrollerState{}
	}
	s := d.scrollers[n]
	if s == nil {
		s = &scrollerState{}
		d.scrollers[n] = s
	}
	return s
}

// snapStyle is the style b scrolls by: its own, the root element's for
// the viewport.
func snapStyle(b *box.Box) *style.Computed {
	if b.Kind == box.Viewport && b.Node != nil {
		if st := dom.Data(b.Node).Style; st != nil {
			return st
		}
	}
	return b.Style
}

// span is a range of scroll offsets on one axis, both ends in.
type span struct{ lo, hi float32 }

func (r span) has(v float32) bool {
	const eps = 0.01
	return v >= r.lo-eps && v <= r.hi+eps
}

// snapPoint is where a scroll lands for an area to line up on one axis;
// seen is the scrolls on the other axis at which the area meets the
// snapport, where it is a candidate.
type snapPoint struct {
	at     float32
	node   *Node
	always bool
	seen   span
}

// snapSpan is the scrolls over which an area bigger than the snapport
// covers it, any of them a snap position.
type snapSpan struct {
	lo, hi float32
	node   *Node
	seen   span
}

// snapLine is one axis of a snap container: its scroll range, its
// scrollport's length and its scroll-padding at either end, and the
// positions its areas snap it to.
type snapLine struct {
	lo, hi     float32
	length     float32
	start, end float32
	mandatory  bool
	points     []snapPoint
	covers     []snapSpan
}

// snapport is the length of l's scrollport its scroll-padding leaves.
func (l *snapLine) snapport() float32 { return l.length - l.start - l.end }

// snapGrid is a snap container: its axes, x then y, nil for one it
// doesn't snap, its scroll range on both, and the areas aligned on
// both axes, by where they snap it.
type snapGrid struct {
	lines  [2]*snapLine
	lo, hi geom.Vec2
	both   []snapBoth
}

type snapBoth struct {
	node *Node
	at   geom.Vec2
}

// snapLines are b's snap axes and areas.
func snapLines(b *box.Box) snapGrid {
	var g snapGrid
	st := snapStyle(b)
	if st == nil || st.Snap.Type == 0 {
		return g
	}
	bars := paint.BarsOf(b, origin(b))
	port := bars.View
	port.W -= b.Scrollbar.X
	port.H -= b.Scrollbar.Y
	rtl := b.Style != nil && b.Style.Dir == style.RTL
	if rtl {
		port.X += b.Scrollbar.X
	}
	g.lo, g.hi = bars.Start, bars.Clamp(geom.V2(math.MaxFloat32, math.MaxFloat32))
	pad := st.Snap.Padding
	axes := [2]*snapLine{
		{lo: g.lo.X, hi: g.hi.X, length: port.W, start: pad.Left.At(port.W), end: pad.Right.At(port.W),
			mandatory: st.Snap.Mandatory},
		{lo: g.lo.Y, hi: g.hi.Y, length: port.H, start: pad.Top.At(port.H), end: pad.Bottom.At(port.H),
			mandatory: st.Snap.Mandatory},
	}
	if st.Snap.Type&style.SnapX != 0 {
		g.lines[0] = axes[0]
	}
	if st.Snap.Type&style.SnapY != 0 {
		g.lines[1] = axes[1]
	}
	// the areas' edges in the content's space, as far as the scroll reaches them
	x := bars.Scroll.X - port.X
	y := bars.Scroll.Y - port.Y
	for _, a := range snapAreas(b) {
		ast := a.Style.Snap
		r := a.Rect()
		m := ast.Margin
		alignX := ast.AlignX
		if rtl {
			alignX = mirrorAlign(alignX)
		}
		x0, x1 := r.X-m.Left.At(0)+x, r.X+r.W+m.Right.At(0)+x
		y0, y1 := r.Y-m.Top.At(0)+y, r.Y+r.H+m.Bottom.At(0)+y
		atX, okX := g.lines[0].add(a.Node, alignX, x0, x1, ast.Always, axes[1].seen(y0, y1))
		atY, okY := g.lines[1].add(a.Node, ast.AlignY, y0, y1, ast.Always, axes[0].seen(x0, x1))
		if okX && okY {
			g.both = append(g.both, snapBoth{a.Node, geom.V2(atX, atY)})
		}
	}
	return g
}

// seen is the scrolls along l at which an area from lo to hi meets its
// snapport, touching it included, within its scroll range.
func (l *snapLine) seen(lo, hi float32) span {
	return span{max(lo-(l.length-l.end), l.lo), min(hi-l.start, l.hi)}
}

// mirrorAlign is a's start and end swapped, as an inline alignment reads
// right to left.
func mirrorAlign(a style.SnapAlign) style.SnapAlign {
	switch a {
	case style.SnapStart:
		return style.SnapEnd
	case style.SnapEnd:
		return style.SnapStart
	case style.SnapNone, style.SnapCenter:
	}
	return a
}

// add adds the snap position of an area from lo to hi on l aligned by
// align, seen on the other axis over seen, and the scrolls over which
// it covers the snapport when bigger, whose ends are snap positions too.
// It reports the position, false where l is nil or the area isn't
// aligned on it.
func (l *snapLine) add(n *Node, align style.SnapAlign, lo, hi float32, always bool, seen span) (float32, bool) {
	if l == nil {
		return 0, false
	}
	inner := l.snapport()
	var at float32
	switch align {
	case style.SnapNone:
		return 0, false
	case style.SnapStart:
		at = lo - l.start
	case style.SnapEnd:
		at = hi - (l.length - l.end)
	case style.SnapCenter:
		at = (lo+hi)/2 - (l.start + inner/2)
	}
	at = clamp(at, l.lo, l.hi)
	l.points = append(l.points, snapPoint{at: at, node: n, always: always, seen: seen})
	if hi-lo > inner {
		c := snapSpan{lo: clamp(lo-l.start, l.lo, l.hi), hi: clamp(hi-l.start-inner, l.lo, l.hi), node: n, seen: seen}
		l.covers = append(l.covers, c)
		l.points = append(l.points, snapPoint{at: c.lo, node: n, seen: seen}, snapPoint{at: c.hi, node: n, seen: seen})
	}
	return at, true
}

func axisOf(v geom.Vec2, axis int) float32 {
	if axis == 0 {
		return v.X
	}
	return v.Y
}

func setAxis(v *geom.Vec2, axis int, f float32) {
	if axis == 0 {
		v.X = f
	} else {
		v.Y = f
	}
}

func clamp(v, lo, hi float32) float32 { return min(max(v, lo), hi) }

// snapAreas are the boxes b is the snap container of: those with a
// scroll-snap-align whose nearest scroll container is b, in tree order.
func snapAreas(b *box.Box) []*box.Box {
	var out []*box.Box
	var seen []*Node
	var walk func(*box.Box)
	walk = func(p *box.Box) {
		for _, c := range p.Children {
			if c.Kind == box.Element && c.Node != nil && c.Style != nil {
				if a := c.Style.Snap; (a.AlignX != style.SnapNone || a.AlignY != style.SnapNone) && !slices.Contains(seen, c.Node) {
					out = append(out, c)
					seen = append(seen, c.Node)
				}
				if o := c.Style.Overflow; o.Width != style.Visible && o.Width != style.Clip ||
					o.Height != style.Visible && o.Height != style.Clip {
					continue
				}
			}
			walk(c)
		}
	}
	walk(b)
	return out
}

// snapPick is where a snap lands on one axis, the area there and the
// scrolls on the other axis at which that area meets the snapport.
type snapPick struct {
	at   float32
	node *Node
	seen span
	ok   bool
}

// pick is where a scroll on l sent from from to to lands: target's
// position when it is one of l's areas, else an area covering the
// snapport there, else by kind a snap position, which a proximity
// container takes only within its share of the scrollport. Only areas
// that meet the snapport at cross on the other axis are candidates, and
// with along only those whose position it holds, as Chrome's search in
// the corridor of a two-axis container. It reports no pick where the
// scroll doesn't snap.
func (l *snapLine) pick(from, to float32, kind snapKind, target *Node, cross float32, along *span) snapPick {
	const eps = 0.5
	valid := func(at float32, seen span) bool { return seen.has(cross) && (along == nil || along.has(at)) }
	if i := slices.IndexFunc(l.points, func(p snapPoint) bool { return target != nil && p.node == target }); i >= 0 {
		p := l.points[i]
		return snapPick{p.at, p.node, p.seen, true}
	}
	ahead := func(at float32) bool {
		return kind == snapEnd || to > from && at > from+eps || to < from && at < from-eps || to == from
	}
	if c := slices.IndexFunc(l.covers, func(c snapSpan) bool {
		return to >= c.lo-eps && to <= c.hi+eps && valid(to, c.seen)
	}); c >= 0 && ahead(to) {
		return snapPick{to, l.covers[c].node, l.covers[c].seen, true}
	}
	base := to
	if kind == snapNext {
		base = from
	}
	preferred := l.preferred(from, kind)
	best, bestPreferred := -1, false
	for i, p := range l.points {
		if !ahead(p.at) || !valid(p.at, p.seen) {
			continue
		}
		pref := preferred(p.at)
		if bestPreferred && !pref {
			continue
		}
		if best < 0 || pref && !bestPreferred || abs32(p.at-base) < abs32(l.points[best].at-base) {
			best, bestPreferred = i, pref
		}
	}
	if best < 0 {
		if kind != snapEnd && l.mandatory {
			// none ahead: the nearest to where it began, as Chrome's relaxed search
			return l.pick(from, from, snapEnd, nil, cross, along)
		}
		return snapPick{}
	}
	if kind != snapEnd {
		best = l.stopOnTheWay(from, best, valid)
	}
	p := l.points[best]
	if !l.mandatory && abs32(p.at-base) > l.snapport()*proximity+0.01 {
		return snapPick{}
	}
	return snapPick{p.at, p.node, p.seen, true}
}

// stopOnTheWay is the point of a scroll-snap-stop: always nearest from
// on the way to the point best, else best.
func (l *snapLine) stopOnTheWay(from float32, best int, valid func(float32, span) bool) int {
	to := l.points[best].at
	for i, p := range l.points {
		if p.always && valid(p.at, p.seen) && between(p.at, from, to) && abs32(p.at-from) < abs32(l.points[best].at-from) {
			best = i
		}
	}
	return best
}

// preferred reports which positions a scroll of kind from from prefers:
// a page's, those from 0.4 of a page to a page away, as Chrome's.
func (l *snapLine) preferred(from float32, kind snapKind) func(float32) bool {
	if kind != snapPage {
		return func(float32) bool { return false }
	}
	page := float32(math.Floor(float64(l.snapport()) + 0.5))
	near, far := max(float32(math.Trunc(float64(page*0.4))), 1), max(page, 1)
	return func(at float32) bool {
		d := abs32(at - from)
		return d >= near && d <= far
	}
}

// between reports whether at lies from from to to, to included.
func between(at, from, to float32) bool {
	if from < to {
		return at > from && at <= to
	}
	return at < from && at >= to
}

func abs32(v float32) float32 { return float32(math.Abs(float64(v))) }

// snap is where a scroll of b from from, sent to to, lands once snapped,
// noting the areas it is snapped to; to where b snaps on no axis. A
// mandatory container scrolled ahead with no snap position that way on
// one of its axes stays where it was, as Chrome's. A container snapping
// on both axes lands where the areas it snaps to on each are seen
// together, as Chrome's: each axis searched in the corridor of where
// the scroll goes on the other, then, when the two areas found aren't
// seen together, the other axis again among those seen with the nearer
// one; with none found, an area aligned on both axes nearest the
// corridor.
func (d *Document) snap(b *box.Box, from, to geom.Vec2, kind snapKind) geom.Vec2 {
	g := snapLines(b)
	if g.lines[0] == nil && g.lines[1] == nil {
		return to
	}
	if kind == snapEnd {
		to = geom.V2(clamp(to.X, g.lo.X, g.hi.X), clamp(to.Y, g.lo.Y, g.hi.Y))
	}
	base := to
	if kind == snapNext {
		base = from
	}
	pick := func(axis int, with *snapPick) snapPick {
		other := 1 - axis
		cross := clamp(axisOf(base, other), axisOf(g.lo, other), axisOf(g.hi, other))
		var along *span
		if with != nil {
			cross, along = with.at, &with.seen
		}
		return g.lines[axis].pick(axisOf(from, axis), axisOf(to, axis), kind, nil, cross, along)
	}
	// a scroll that goes ahead snaps only the axes it moves on
	var searched [2]bool
	var picks [2]snapPick
	for axis, l := range g.lines {
		if searched[axis] = l != nil && (kind == snapEnd || axisOf(from, axis) != axisOf(to, axis)); searched[axis] {
			picks[axis] = pick(axis, nil)
		}
	}
	if searched[0] && searched[1] {
		g.together(&picks, base, to, kind, pick)
	}
	out := to
	for axis, l := range g.lines {
		if !searched[axis] {
			continue
		}
		switch p := picks[axis]; {
		case p.ok:
			setAxis(&out, axis, p.at)
		case kind != snapEnd && l.mandatory:
			return from
		}
	}
	s := d.scrollerOf(b.Node)
	for axis := range g.lines {
		if searched[axis] {
			d.setTarget(s, axis, picks[axis].node)
		}
	}
	return out
}

// together has the picks of a two-axis container land where both their
// areas are seen: the axis whose pick is the nearer to base kept, the
// other searched again among the areas seen with it, or with neither
// found, an area aligned on both axes nearest the corridor of to.
func (g snapGrid) together(picks *[2]snapPick, base, to geom.Vec2, kind snapKind, pick func(int, *snapPick) snapPick) {
	x, y := picks[0], picks[1]
	switch {
	case x.ok && y.ok && !(x.seen.has(y.at) && y.seen.has(x.at)):
		if abs32(x.at-base.X) <= abs32(y.at-base.Y) {
			picks[1] = pick(1, &x)
		} else {
			picks[0] = pick(0, &y)
		}
	case x.ok && !y.ok:
		picks[1] = pick(1, &x)
	case y.ok && !x.ok:
		picks[0] = pick(0, &y)
	case !x.ok && !y.ok && kind == snapEnd:
		*picks = g.mutual(to)
	}
	g.preferBoth(picks)
}

// mutual is the snap of a two-axis container that found no area in
// either corridor: the area aligned on both axes whose position is
// nearest to the scrollport's reach of to, on y first, as Chrome's
// FindSnapPositionForMutualSnap.
func (g snapGrid) mutual(to geom.Vec2) [2]snapPick {
	var out [2]snapPick
	best := geom.V2(math.MaxFloat32, math.MaxFloat32)
	for _, a := range g.both {
		dx, dy := a.at.X-to.X, a.at.Y-to.Y
		near := geom.V2(g.lines[0].snapport(), g.lines[1].snapport()).Scale(proximity)
		if !g.lines[0].mandatory && (abs32(dx) > near.X || abs32(dy) > near.Y) {
			continue
		}
		dist := geom.V2(outside(dx, g.lines[0].snapport()), outside(dy, g.lines[1].snapport()))
		if dist.Y < best.Y || dist.Y == best.Y && dist.X < best.X {
			best = dist
			out = [2]snapPick{{at: a.at.X, node: a.node, ok: true}, {at: a.at.Y, node: a.node, ok: true}}
		}
	}
	return out
}

// outside is how far v lies outside 0 to size.
func outside(v, size float32) float32 {
	switch {
	case v < 0:
		return -v
	case v > size:
		return v - size
	}
	return 0
}

// preferBoth has each axis snapped to an area aligned on both axes where
// it is snapped on the other axis too, among the areas at its position:
// of areas lined up alike, Chrome's alternative.
func (g snapGrid) preferBoth(picks *[2]snapPick) {
	if !picks[0].ok || !picks[1].ok || picks[0].node == picks[1].node {
		return
	}
	for axis := range 2 {
		p, q := &picks[axis], picks[1-axis]
		at := func(a snapBoth, axis int) float32 { return axisOf(a.at, axis) }
		if slices.ContainsFunc(g.both, func(a snapBoth) bool { return a.node == p.node && abs32(at(a, 1-axis)-q.at) <= 1 }) {
			continue
		}
		for _, a := range g.both {
			if abs32(at(a, axis)-p.at) <= 0.5 && abs32(at(a, 1-axis)-q.at) <= 1 {
				p.node = a.node
				break
			}
		}
	}
}

// setTarget makes node the area s is snapped to on axis, and has the
// scroll-state() queries asked again when that changed.
func (d *Document) setTarget(s *scrollerState, axis int, node *Node) {
	if s.targets[axis] != node {
		s.targets[axis] = node
		d.scrolled = true
	}
}

// scrollBoxTo scrolls b from where it is to off, snapped by kind as it
// lands, at once or gliding when smooth.
// A relative scroll is noted for scroll-state(scrolled).
func (d *Document) scrollBoxTo(b *box.Box, off geom.Vec2, kind snapKind, smooth, relative bool) {
	from := b.Scroll
	bars := paint.BarsOf(b, origin(b))
	off = bars.Clamp(d.snap(b, from, off, kind))
	s := d.scrollerOf(b.Node)
	s.wheeling = false
	if relative {
		d.noteScrolled(s, from, off)
	}
	d.glideTo(b, s, from, off, smooth)
}

// glideTo moves b from from to to: at once unless smooth, and while the
// Document has no clock yet.
func (d *Document) glideTo(b *box.Box, s *scrollerState, from, to geom.Vec2, smooth bool) {
	s.glide = nil
	if !smooth || from == to {
		d.setScroll(b, to)
		return
	}
	s.glide = newGlide(from, to, byProgram)
	d.unsettled = true
}

// noteScrolled notes the way a relative scroll from from to to went on
// each axis it moved on.
func (d *Document) noteScrolled(s *scrollerState, from, to geom.Vec2) {
	const eps = 0.01
	for axis := range 2 {
		f, t := axisOf(from, axis), axisOf(to, axis)
		var dir int8
		switch {
		case t > f+eps:
			dir = 1
		case t < f-eps:
			dir = -1
		default:
			continue
		}
		if s.scrolled[axis] != dir {
			s.scrolled[axis] = dir
			d.scrolled = true
		}
	}
}

// stepScrolls moves the glides on to the Document's clock and snaps the
// trackpad gestures that have ended, reporting whether a glide still
// runs.
func (d *Document) stepScrolls() bool {
	if d.tree == nil {
		return false
	}
	running := false
	for n, s := range d.scrollers {
		b := d.scroller(n)
		if b == nil || !d.inDocument(n) {
			delete(d.scrollers, n)
			continue
		}
		if s.wheeling {
			if s.quiet++; s.quiet >= wheelSettle {
				s.wheeling = false
				to := d.snap(b, s.wheelAt, b.Scroll, snapAhead)
				d.glideTo(b, s, b.Scroll, paint.BarsOf(b, origin(b)).Clamp(to), true)
			}
		}
		g := s.glide
		if g == nil {
			continue
		}
		if g.start < 0 {
			g.start = d.clock
		}
		d.setScroll(b, g.at(d.clock))
		if d.clock-g.start >= g.dur {
			s.glide = nil
		} else {
			running = true
		}
	}
	return running
}

// wheelTurned scrolls b by a wheel's turn of by points. A mouse's notch
// glides from where the glide under way is going, snapped as a scrollBy
// is, as Chrome snaps each notch; a trackpad's turn moves at once, and
// its gesture snaps once it ends.
func (d *Document) wheelTurned(b *box.Box, by geom.Vec2, precise bool) {
	bars := paint.BarsOf(b, origin(b))
	if !precise {
		from := d.userAim(b)
		d.userScroll(b, bars.Clamp(d.snap(b, from, bars.Clamp(from.Add(by)), snapAhead)), byWheel)
		return
	}
	s := d.scrollerOf(b.Node)
	if !s.wheeling {
		s.wheeling, s.wheelAt = true, bars.Scroll
	}
	s.quiet, s.glide = 0, nil
	to := bars.Clamp(bars.Scroll.Add(by))
	d.noteScrolled(s, bars.Scroll, to)
	d.setScroll(b, to)
}

// resnap snaps again each snap container whose areas moved since it was
// last laid out, as Chrome after a layout: to the areas it was snapped
// to while they still are its areas, else the nearest positions to where
// it stands. Containers under a wheel gesture, a glide or a held thumb
// are left to it.
func (d *Document) resnap() {
	if d.tree == nil {
		return
	}
	moved := false
	for n, b := range d.tree.Of {
		if b == nil || b.Kind != box.Element && b.Kind != box.Viewport {
			continue
		}
		if n == d.html && d.tree.Root != nil && d.tree.Root.Kind == box.Viewport {
			b = d.tree.Root
		}
		if st := snapStyle(b); st == nil || st.Snap.Type == 0 {
			continue
		}
		g := snapLines(b)
		var laid []float32
		for _, l := range g.lines {
			if l != nil {
				for _, p := range l.points {
					laid = append(laid, p.at)
				}
				laid = append(laid, math.MaxFloat32)
			}
		}
		s := d.scrollerOf(n)
		if slices.Equal(laid, s.laid) || s.wheeling || s.glide != nil || d.dragging == n {
			continue
		}
		s.laid = laid
		off := d.relaid(b, s, g)
		if off != b.Scroll {
			d.setScroll(b, off)
			moved = true
		}
	}
	if moved {
		layout.Refresh(d.tree, d.html)
	}
}

// relaid is where b, whose areas moved, snaps again: on each axis to
// the area it was snapped to while that still is one of its areas, the
// other axes as a scrollTo where it stands.
func (d *Document) relaid(b *box.Box, s *scrollerState, g snapGrid) geom.Vec2 {
	off := b.Scroll
	var held [2]*Node
	for axis, l := range g.lines {
		if l == nil || s.targets[axis] == nil {
			continue
		}
		f := axisOf(off, axis)
		if p := l.pick(f, f, snapEnd, s.targets[axis], 0, nil); p.ok && p.node == s.targets[axis] {
			setAxis(&off, axis, p.at)
			held[axis] = p.node
		}
	}
	if (g.lines[0] == nil || held[0] != nil) && (g.lines[1] == nil || held[1] != nil) {
		return off
	}
	to := d.snap(b, off, off, snapEnd)
	for axis, l := range g.lines {
		switch {
		case l == nil:
		case held[axis] != nil:
			d.setTarget(s, axis, held[axis])
		default:
			setAxis(&off, axis, axisOf(to, axis))
		}
	}
	return off
}

// snappedOf is the axes n is snapped to by its snap container, as
// scroll-state(snapped) reads them.
func (d *Document) snappedOf(n *Node) css.SnapAxes {
	var on css.SnapAxes
	for c, s := range d.scrollers {
		if c == n || !d.inDocument(c) {
			continue
		}
		if s.targets[0] == n {
			on |= css.SnappedX
		}
		if s.targets[1] == n {
			on |= css.SnappedY
		}
	}
	return on
}

// scrolledOf is the sides n last scrolled toward by a relative scroll.
func (d *Document) scrolledOf(n *Node) css.Sides {
	s := d.scrollers[n]
	if s == nil {
		return 0
	}
	var on css.Sides
	switch s.scrolled[0] {
	case 1:
		on |= css.SideRight
	case -1:
		on |= css.SideLeft
	}
	switch s.scrolled[1] {
	case 1:
		on |= css.SideBottom
	case -1:
		on |= css.SideTop
	}
	return on
}

// overscrollContained reports whether b keeps a scroll the way w goes from chaining
// past it, by its overscroll-behavior.
func overscrollContained(b *box.Box, w geom.Vec2) bool {
	st := snapStyle(b)
	if st == nil {
		return false
	}
	return w.X != 0 && st.Snap.OverscrollX != style.OverscrollAuto ||
		w.Y != 0 && st.Snap.OverscrollY != style.OverscrollAuto
}
