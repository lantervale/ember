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
// (decision 0163): snapEnd the snap position nearest where it was sent
// (scrollTo, a thumb let go, a layout), snapAhead the nearest to that
// among those ahead of where it began, stopping at a scroll-snap-stop:
// always on the way (scrollBy, the wheel, the keys).
type snapKind uint8

const (
	snapEnd snapKind = iota
	snapAhead
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
// (scroll-state(scrolled)), a wheel gesture not yet snapped and a glide
// under way.
type scrollerState struct {
	targets  [2]*Node
	laid     []float32
	scrolled [2]int8
	wheeling bool
	wheelAt  geom.Vec2
	quiet    int
	glide    *glide
}

// glide is a smooth scroll on Chrome's curve: from and to, begun at
// start on the Document's clock (-1 until the next frame), lasting dur
// seconds.
type glide struct {
	from, to   geom.Vec2
	start, dur float32
}

//nolint:gochecknoglobals // Chrome's curve for a programmatic smooth scroll on the desktop
var glideCurve = css.CubicBezier(0.4, 0, 0, 1)

// glideTime is how long Chrome takes to scroll by delta smoothly: the
// square root of its longer side in frames of 1/60s, at most 1.5s.
func glideTime(delta geom.Vec2) float32 {
	longest := math.Max(math.Abs(float64(delta.X)), math.Abs(float64(delta.Y)))
	return float32(math.Min(math.Sqrt(longest), 90) / 60)
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

// snapPoint is where a scroll lands for an area to line up on one axis.
type snapPoint struct {
	at     float32
	node   *Node
	always bool
}

// snapSpan is the scrolls over which an area bigger than the snapport
// covers it, any of them a snap position.
type snapSpan struct {
	lo, hi float32
	node   *Node
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

// snapLines are b's snap axes, x then y, nil for one it doesn't snap.
func snapLines(b *box.Box) [2]*snapLine {
	var lines [2]*snapLine
	st := snapStyle(b)
	if st == nil || st.Snap.Type == 0 {
		return lines
	}
	bars := paint.BarsOf(b, origin(b))
	port := bars.View
	port.W -= b.Scrollbar.X
	port.H -= b.Scrollbar.Y
	rtl := b.Style != nil && b.Style.Dir == style.RTL
	if rtl {
		port.X += b.Scrollbar.X
	}
	hi := bars.Clamp(geom.V2(math.MaxFloat32, math.MaxFloat32))
	pad := st.Snap.Padding
	if st.Snap.Type&style.SnapX != 0 {
		lines[0] = &snapLine{lo: bars.Start.X, hi: hi.X, length: port.W, start: pad.Left.At(port.W),
			end: pad.Right.At(port.W), mandatory: st.Snap.Mandatory}
	}
	if st.Snap.Type&style.SnapY != 0 {
		lines[1] = &snapLine{lo: bars.Start.Y, hi: hi.Y, length: port.H, start: pad.Top.At(port.H),
			end: pad.Bottom.At(port.H), mandatory: st.Snap.Mandatory}
	}
	for _, a := range snapAreas(b) {
		ast := a.Style.Snap
		r := a.Rect()
		m := ast.Margin
		// the area's edges in the content's space, as far as the scroll reaches them
		x := bars.Scroll.X - port.X
		y := bars.Scroll.Y - port.Y
		alignX := ast.AlignX
		if rtl {
			alignX = mirrorAlign(alignX)
		}
		if l := lines[0]; l != nil {
			l.add(a.Node, alignX, r.X-m.Left.At(0)+x, r.X+r.W+m.Right.At(0)+x, ast.Always)
		}
		if l := lines[1]; l != nil {
			l.add(a.Node, ast.AlignY, r.Y-m.Top.At(0)+y, r.Y+r.H+m.Bottom.At(0)+y, ast.Always)
		}
	}
	return lines
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
// align, and the scrolls over which it covers the snapport when bigger,
// whose ends are snap positions too.
func (l *snapLine) add(n *Node, align style.SnapAlign, lo, hi float32, always bool) {
	inner := l.snapport()
	var at float32
	switch align {
	case style.SnapNone:
		return
	case style.SnapStart:
		at = lo - l.start
	case style.SnapEnd:
		at = hi - (l.length - l.end)
	case style.SnapCenter:
		at = (lo+hi)/2 - (l.start + inner/2)
	}
	l.points = append(l.points, snapPoint{at: clamp(at, l.lo, l.hi), node: n, always: always})
	if hi-lo > inner {
		span := snapSpan{lo: clamp(lo-l.start, l.lo, l.hi), hi: clamp(hi-l.start-inner, l.lo, l.hi), node: n}
		l.covers = append(l.covers, span)
		l.points = append(l.points, snapPoint{at: span.lo, node: n}, snapPoint{at: span.hi, node: n})
	}
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

// pick is where a scroll on l sent from from to to lands: target's
// position when it is one of l's areas, else an area covering the
// snapport there, else by kind the nearest snap position, which a
// proximity container takes only within its share of the scrollport. It
// reports false where the scroll doesn't snap.
func (l *snapLine) pick(from, to float32, kind snapKind, target *Node) (float32, *Node, bool) {
	const eps = 0.5
	if target != nil {
		for _, p := range l.points {
			if p.node == target {
				return p.at, p.node, true
			}
		}
	}
	ahead := func(at float32) bool {
		return kind == snapEnd || to > from && at > from+eps || to < from && at < from-eps || to == from
	}
	for _, c := range l.covers {
		if to >= c.lo-eps && to <= c.hi+eps && ahead(to) {
			return to, c.node, true
		}
	}
	best := -1
	for i, p := range l.points {
		if !ahead(p.at) {
			continue
		}
		if best < 0 || abs32(p.at-to) < abs32(l.points[best].at-to) {
			best = i
		}
	}
	if best < 0 {
		return 0, nil, false
	}
	if kind == snapAhead {
		for i, p := range l.points {
			if p.always && ahead(p.at) && between(p.at, from, l.points[best].at) {
				best = i
			}
		}
	}
	p := l.points[best]
	if !l.mandatory && abs32(p.at-to) > l.snapport()*proximity+0.01 {
		return 0, nil, false
	}
	return p.at, p.node, true
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
// one of its axes stays where it was, as Chrome's.
func (d *Document) snap(b *box.Box, from, to geom.Vec2, kind snapKind) geom.Vec2 {
	lines := snapLines(b)
	if lines[0] == nil && lines[1] == nil {
		return to
	}
	out := to
	var targets [2]*Node
	for axis, l := range lines {
		if l == nil {
			continue
		}
		f, t := axisOf(from, axis), axisOf(to, axis)
		if kind == snapEnd {
			t = clamp(t, l.lo, l.hi)
		}
		at, node, ok := l.pick(f, t, kind, nil)
		switch {
		case ok:
			setAxis(&out, axis, at)
			targets[axis] = node
		case kind == snapAhead && l.mandatory:
			return from
		}
	}
	s := d.scrollerOf(b.Node)
	for axis, l := range lines {
		if l != nil {
			d.setTarget(s, axis, targets[axis])
		}
	}
	return out
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
	s.glide = &glide{from: from, to: to, start: -1, dur: glideTime(to.Sub(from))}
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
// wheel gestures that have ended, reporting whether a glide still runs.
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
		t := float32(1)
		if g.dur > 0 {
			t = min((d.clock-g.start)/g.dur, 1)
		}
		d.setScroll(b, g.from.Add(g.to.Sub(g.from).Scale(glideCurve(t))))
		if t >= 1 {
			s.glide = nil
		} else {
			running = true
		}
	}
	return running
}

// wheelTurned notes a wheel's turn of b from from, snapped once the gesture
// ends; a glide under way stops where it is.
func (d *Document) wheelTurned(b *box.Box, from, to geom.Vec2) {
	s := d.scrollerOf(b.Node)
	if !s.wheeling {
		s.wheeling, s.wheelAt = true, from
	}
	s.quiet, s.glide = 0, nil
	d.noteScrolled(s, from, to)
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
		lines := snapLines(b)
		var laid []float32
		for _, l := range lines {
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
		off := b.Scroll
		for axis, l := range lines {
			if l == nil {
				continue
			}
			f := axisOf(off, axis)
			at, node, ok := l.pick(f, f, snapEnd, s.targets[axis])
			if ok {
				setAxis(&off, axis, at)
			}
			d.setTarget(s, axis, node)
		}
		if off != b.Scroll {
			d.setScroll(b, off)
			moved = true
		}
	}
	if moved {
		layout.Refresh(d.tree, d.html)
	}
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
