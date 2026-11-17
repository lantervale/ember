package layout

import (
	"math"
	"slices"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// Multi-column layout. A multi-column container lays its content out once, as one column
// of the used column width (the flow), then cuts that flow into columns: nothing is laid
// out again per column, as every column is as wide as the others. Each node keeps one
// layout, in the flow; where the cuts fall gives each box and each line the column it
// shows in, applied when the boxes are placed (keepFlow) and written.

// columnEpsilon is the slack in comparing places in the flow, under Chrome's 1/64 px.
const columnEpsilon = 0.01

// columnSet is how a multi-column container's flow is cut: starts[k] is where column k
// starts in the flow, down the container's border box.
type columnSet struct {
	starts []float32
	// left and top are where the first column starts in the border box; width and gap
	// the used column width and gap, height the columns' height.
	left, top          float32
	width, gap, height float32
}

func (c *columnSet) pitch() float32 { return c.width + c.gap }

// at is the column showing the flow at y.
func (c *columnSet) at(y float32) int {
	k := 0
	for k+1 < len(c.starts) && c.starts[k+1] <= y+columnEpsilon {
		k++
	}
	return k
}

// last is the last column showing something of a box that ends at bottom, from column k.
func (c *columnSet) last(k int, bottom float32) int {
	for k+1 < len(c.starts) && c.starts[k+1] < bottom-columnEpsilon {
		k++
	}
	return k
}

// shift is how far column k's part of the flow shows from where the flow has it.
func (c *columnSet) shift(k int) point {
	return point{float32(k) * c.pitch(), c.top - c.starts[k]}
}

// flowFrag is a node's place in the multi-column flow it is laid out in.
type flowFrag struct {
	cols *columnSet
	// y is the border box's top in the flow, col the column its origin shows in and oy
	// the origin's y from the container's border box, both less relative offsets.
	y, oy float32
	col   int
	// pieces are the box's parts when columns cut it, from its origin.
	pieces []geom.Rect
}

// lineShift is how far a line at y down the node's border box shows from there.
func (f *flowFrag) lineShift(y float32) (float32, float32) {
	k := f.cols.at(f.y + y)
	return float32(k-f.col) * f.cols.pitch(), f.cols.top + f.y - f.cols.starts[k] - f.oy
}

// columnar reports a node laid out as a multi-column container: a block container with
// a column-count or a column-width.
func (n *node) columnar() bool {
	if n.replaced || n.control != noControl || n.role != roleNone || n.st.Display == style.DisplayNone {
		return false
	}
	c := n.context()
	return c == contextBlock || c == contextLeaf
}

// usedColumns is the used column count and width in a content box of width avail, CSS
// Multi-column's pseudo-algorithm.
func usedColumns(st *style.Computed, avail, gap float32) (int, float32) {
	count := st.ColumnCount
	if st.ColumnWidth.Kind != style.Auto {
		fit := max(1, int(math.Floor(float64((avail+gap)/(st.ColumnWidth.Px+gap)))))
		if count == 0 || fit < count {
			count = fit
		}
	}
	count = max(count, 1)
	return count, max(0, (avail+gap)/float32(count)-gap)
}

// columnGap is the used column-gap of a multi-column container: normal is 1em.
func columnGap(st *style.Computed, avail float32) float32 {
	if st.Gap.Width.Kind == style.Auto {
		return st.Font.Size
	}
	return max(0, resolveOrZero(st.Gap.Width, or(avail, 0)))
}

// flowStyle is st for laying a multi-column container's content out as one column: the
// same box, sized by the caller alone.
func (t *tree) flowStyle(st *style.Computed) *style.Computed {
	if f := t.flowStyles[st]; f != nil {
		return f
	}
	f := *st
	f.ColumnCount, f.ColumnWidth = 0, style.Length{}
	f.Size = style.Size[style.Length]{}
	f.MinSize = style.Size[style.Length]{}
	f.MaxSize = style.Size[style.Length]{Width: style.Length{Kind: style.None}, Height: style.Length{Kind: style.None}}
	f.AspectRatio, f.AlignContent = 0, style.AlignNormal
	if t.flowStyles == nil {
		t.flowStyles = map[*style.Computed]*style.Computed{}
	}
	t.flowStyles[st] = &f
	return &f
}

// flow lays n's content out as one column in a border box of width w, or measures the
// width it asks for in avail when w is none.
func (t *tree) flow(n *node, in input, w float32, avail space) output {
	st := n.st
	n.st = t.flowStyle(st)
	defer func() { n.st = st }()
	fin := input{
		run:             performLayout,
		sizing:          inherentSize,
		axis:            axisBoth,
		known:           size{w, none},
		knownIsDefinite: bools{true, true},
		parent:          in.parent,
		avail:           spaces{avail, maxContentSpace},
	}
	if !some(w) {
		fin.run, fin.axis = computeSize, axisHorizontal
	}
	if n.context() == contextBlock {
		return t.block(n, fin, nil)
	}
	return t.leaf(n, fin, nil)
}

// columns lays out a multi-column container: its content as one column of the used
// width, cut into columns of one height, balanced unless column-fill or the container's
// height says otherwise; the columns past the used count overflow at its end.
func (t *tree) columns(n *node, in input) output {
	st := n.st
	ratio := aspectRatio(st)
	padding := edges(st.Padding, in.parent.w)
	border := borders(st)
	g := n.ownGutter()
	gutter := rect{r: g.x, b: g.y}
	pbSize := padding.add(border).sum()
	inset := padding.add(border).add(gutter)
	adj := boxSizingAdjustment(st, pbSize)
	minSize := resolveSize(st.MinSize, in.parent).withAspectRatio(ratio).mAdd(adj)
	maxSize := resolveMax(st.MaxSize, in.parent).withAspectRatio(ratio).mAdd(adj)
	clamped := noSize
	if in.sizing == inherentSize {
		clamped = resolveSize(st.Size, in.parent).withAspectRatio(ratio).mAdd(adj).mClamp(minSize, maxSize)
	}
	minMax := size{definiteMinMax(minSize.w, maxSize.w), definiteMinMax(minSize.h, maxSize.h)}
	styled := in.known.or(minMax).or(clamped).mMax(pbSize)
	if in.run == computeSize && styled.both() {
		return sizeOutput(styled)
	}

	outerW := styled.w
	if !some(outerW) {
		outerW = mMax(mClamp(t.columnsWidth(n, in, inset), minSize.w, maxSize.w), pbSize.w)
	}
	if in.run == computeSize && in.axis == axisHorizontal {
		return sizeOutput(size{outerW, 0})
	}
	content := max(0, outerW-inset.horizontal())
	gap := columnGap(st, content)
	count, width := usedColumns(st, content, gap)

	inner := t.flow(n, in, width+inset.horizontal(), def(width))
	flowH := max(0, inner.size.h-inset.vertical())
	units := t.flowUnits(n, size{width + inset.horizontal(), inner.size.h}, inset)

	limit := mSub(styled.h, inset.vertical())
	if !some(limit) {
		limit = mSub(maxSize.h, inset.vertical())
	}
	limit = mMax(limit, 0)
	cs := &columnSet{left: inset.l, top: inset.t, width: width, gap: gap}
	switch {
	case !st.ColumnFillAuto:
		cs.height = balancedHeight(units, inset.t, flowH, count)
		if some(limit) && limit < cs.height {
			cs.height = limit
		}
	case some(limit):
		cs.height = fmin(limit, flowH)
		if some(styled.h) {
			cs.height = limit
		}
	default:
		cs.height = flowH
	}
	cs.starts, _ = fillColumns(units, inset.t, cs.height)

	final := size{outerW, mMax(or(styled.h, mClamp(cs.height+inset.vertical(), minSize.h, maxSize.h)), pbSize.h)}
	scroll := isScrollContainer(st)
	reach := rect{r: padding.l + float32(len(cs.starts))*cs.pitch() - gap, b: padding.t + cs.height}
	if scroll {
		reach.r += padding.r
		reach.b += padding.b
	}
	areaInset := border.add(gutter)
	out := output{
		size:          final,
		overflow:      finalizeOverflow(reach, final.sub(areaInset.sum()), [2]bool{}, scroll),
		firstBaseline: inner.firstBaseline,
		lastBaseline:  none,
		topMargin:     marginFrom(resolveOrZero(st.Margin.Top, in.parent.w)),
		bottomMargin:  marginFrom(resolveOrZero(st.Margin.Bottom, in.parent.w)),
		area:          &oofArea{size: final.sub(areaInset.sum()), offset: point{areaInset.l, areaInset.t}},
	}
	if in.run != performLayout {
		return out
	}
	n.columns = cs
	t.touchFlow(n)
	// A positioned box with no inset of its own starts where the flow shows its place:
	// its line's column, or between blocks the column of what comes before it.
	out.oof = make([]oofCandidate, len(inner.oof))
	for i, c := range inner.oof {
		k := cs.at(c.y.area.start)
		if !c.inline {
			k = cs.last(0, c.y.area.start)
		}
		by := cs.shift(k)
		c.x.area.start += by.x
		c.x.area.end += by.x
		c.y.area.start += by.y
		c.y.area.end += by.y
		out.oof[i] = c
	}
	return out
}

// columnsWidth is the border box's width a multi-column container asks for when nothing
// gives it one: its columns as wide as the content asks, or as column-width when wider,
// no wider than the room when there is some.
func (t *tree) columnsWidth(n *node, in input, inset rect) float32 {
	st := n.st
	avail := in.avail.w
	measure := maxContentSpace
	if avail.kind == minContent {
		measure = minContentSpace
	}
	column := max(0, t.flow(n, in, none, measure).size.w-inset.horizontal())
	if st.ColumnWidth.Kind != style.Auto {
		column = max(column, st.ColumnWidth.Px)
	}
	count := float32(max(st.ColumnCount, 1))
	w := count*column + (count-1)*columnGap(st, none) + inset.horizontal()
	if avail.kind == definite {
		w = fmin(w, fmax(avail.v, column+inset.horizontal()))
	}
	return w
}

// touchFlow marks the boxes columns may cut, under n, as laid out in this layout: a
// retained layout writes their text and children again, where the columns now put them.
func (t *tree) touchFlow(n *node) {
	for _, k := range n.kids {
		k.performed = t.gen
		if breaksInside(k) {
			t.touchFlow(k)
		}
	}
}

// breaksInside reports a box columns may cut: a block container in the flow that holds
// blocks or lines. The others go into one column whole, as Chrome keeps monolithic
// boxes; so do a float, a table, a flex or grid container and a nested multi-column
// container, which Chrome would cut.
func breaksInside(n *node) bool {
	st := n.st
	if n.atomic || n.replaced || n.control != noControl || n.role != roleNone || st.Display != style.Block ||
		st.Float != style.FloatNone || st.BreakInsideAvoid || st.Multicol() || outOfFlow(st.Position) ||
		isScrollContainer(st) {
		return false
	}
	c := n.context()
	return c == contextBlock || c == contextLeaf
}

// relativeShift is how far position: relative moves n from its place in the flow.
func relativeShift(n *node) point {
	st := n.st
	if st.Position != style.Relative {
		return point{}
	}
	w := none
	switch p := n.parent; {
	case p == nil:
	case p.columns != nil:
		w = p.columns.width
	default:
		w = p.unrounded.size.w - p.unrounded.padding.horizontal() - p.unrounded.border.horizontal()
	}
	l, r := resolve(st.Inset.Left, w), resolve(st.Inset.Right, w)
	tp, b := resolve(st.Inset.Top, none), resolve(st.Inset.Bottom, none)
	return point{or(l, or(-r, 0)), or(tp, or(-b, 0))}
}

// Appeals of a break, Chrome's: one that breaks no rule, one leaving fewer lines than
// orphans or widows ask, one where break-before, break-after says avoid.
const (
	appealAvoid int8 = iota + 1
	appealLines
	appealPerfect
)

// flowUnit is a stretch of the flow columns never cut, or one of empty room they cut
// anywhere (space): a line, a monolithic box, the edges of a block with what they hold.
type flowUnit struct {
	top, bottom float32
	space       bool
	// forced starts a column before the unit, keep being the margin kept above it then;
	// appeal is that of a break before it, a later line's worked out as columns fill.
	forced bool
	keep   float32
	appeal int8
	// para is the block whose line the unit is (0 for none), line its index among
	// lines, with the block's orphans and widows.
	para, line, lines int
	orphans, widows   int
}

type unitWalk struct {
	t     *tree
	units []flowUnit
	// top is the edge the next unit starts at, a block's border top above its first
	// content, and pending the strongest break-before or break-after since the last unit.
	top     float32
	pending style.Break
	keep    float32
	paras   int
	sorted  bool
}

func (w *unitWalk) emit(u flowUnit) {
	if some(w.top) {
		u.top = fmin(u.top, w.top)
		w.top = none
	}
	u.appeal = appealPerfect
	switch w.pending {
	case style.BreakColumn:
		u.forced, u.keep = true, w.keep
	case style.BreakAvoid:
		u.appeal = appealAvoid
	case style.BreakAuto:
	}
	w.pending, w.keep = style.BreakAuto, 0
	if len(w.units) > 0 && u.top < w.units[len(w.units)-1].top {
		w.sorted = false
	}
	w.units = append(w.units, u)
}

func (w *unitWalk) ask(b style.Break) {
	if b > w.pending {
		w.pending = b
	}
}

// flowUnits are the units of n's flow, in its border box of size s with inset.
func (t *tree) flowUnits(n *node, s size, inset rect) []flowUnit {
	w := &unitWalk{t: t, top: none, sorted: true}
	w.content(n, 0, s, inset)
	if !w.sorted {
		slices.SortStableFunc(w.units, func(a, b flowUnit) int {
			switch {
			case a.top < b.top:
				return -1
			case a.top > b.top:
				return 1
			}
			return 0
		})
	}
	// What overlaps can't be cut apart: a float and the lines beside it.
	out := w.units[:0]
	for _, u := range w.units {
		if last := len(out) - 1; last >= 0 && u.top < out[last].bottom-columnEpsilon && !u.forced {
			p := &out[last]
			p.bottom = fmax(p.bottom, u.bottom)
			p.space = p.space && u.space
			continue
		}
		out = append(out, u)
	}
	return out
}

// content walks what n holds, its border box at y in the flow.
func (w *unitWalk) content(n *node, y float32, s size, inset rect) {
	if c := n.inline; c != nil {
		w.paras++
		ls := c.lines(s.w - inset.horizontal())
		at := y + w.t.textTop(n, ls, inset.t, s.h-inset.vertical())
		for i, l := range ls {
			lh, _ := w.t.lineBox(c, l)
			at += l.push
			w.emit(flowUnit{top: at, bottom: at + lh, para: w.paras, line: i, lines: len(ls),
				orphans: n.st.Orphans, widows: n.st.Widows})
			at += lh
		}
		return
	}
	for _, k := range n.kids {
		st := k.st
		if st.Display == style.DisplayNone || outOfFlow(st.Position) {
			continue
		}
		l := k.unrounded
		top := y + l.location.y - relativeShift(k).y
		bottom := top + l.size.h
		w.ask(st.BreakBefore)
		if w.pending == style.BreakColumn {
			w.keep = fmax(l.margin.t, 0)
		}
		if !breaksInside(k) {
			if l.size.h > 0 {
				w.emit(flowUnit{top: top, bottom: bottom})
			}
			w.ask(st.BreakAfter)
			continue
		}
		held := some(w.top)
		if !held {
			w.top = top
		}
		before := len(w.units)
		w.content(k, top, l.size, l.padding.add(l.border))
		switch {
		case len(w.units) == before:
			if !held {
				w.top = none
			}
			if l.size.h > 0 {
				w.emit(flowUnit{top: top, bottom: bottom, space: true})
			}
		case bottom-w.units[len(w.units)-1].bottom > l.padding.b+l.border.b+columnEpsilon:
			w.emit(flowUnit{top: w.units[len(w.units)-1].bottom, bottom: bottom, space: true})
		default:
			u := &w.units[len(w.units)-1]
			u.bottom = fmax(u.bottom, bottom)
		}
		w.ask(st.BreakAfter)
	}
}

// appealOf is the appeal of a break before unit i in a column that starts with unit
// first: between lines it leaves at least orphans lines of the block in this column and
// takes at least widows to the next.
func appealOf(units []flowUnit, i, first int) int8 {
	u := &units[i]
	if u.line == 0 {
		return u.appeal
	}
	start := 0
	if f := &units[first]; f.para == u.para {
		start = f.line
	}
	if u.line-start >= u.orphans && u.lines-u.line >= u.widows {
		return appealPerfect
	}
	return appealLines
}

// fillColumns cuts the flow of units, which starts at top, into columns of height h: each
// column takes the units that fit, and breaks before the first that doesn't, or earlier
// where a break appeals more, as Chrome's does. It returns where each column starts in
// the flow and the least room one more unit would have asked of a column.
func fillColumns(units []flowUnit, top, h float32) ([]float32, float32) {
	starts := []float32{top}
	shortage := float32(math.Inf(1))
	if h <= 0 {
		return starts, shortage
	}
	s, first, best := top, 0, -1
	var bestAppeal int8
	column := func(at int, y float32) {
		s, first, best = y, at, -1
		starts = append(starts, y)
	}
	for i := 0; i < len(units); {
		u := &units[i]
		switch {
		case u.forced && i > first:
			column(i, u.top-u.keep)
		case u.bottom-s <= h+columnEpsilon:
			if i > first {
				if a := appealOf(units, i, first); best < 0 || a >= bestAppeal {
					best, bestAppeal = i, a
				}
			}
			i++
		case u.space && u.top < s+h-columnEpsilon:
			column(i, s+h)
		case i == first:
			shortage = fmin(shortage, u.bottom-s-h)
			i++
		default:
			shortage = fmin(shortage, u.bottom-s-h)
			if best >= 0 && bestAppeal > appealOf(units, i, first) {
				i = best
			}
			column(i, units[i].top)
		}
	}
	return starts, shortage
}

// balancedHeight is the least column height that puts the flow of units, flowH high
// from top, in count columns, found as Chrome finds it: from the flow's height shared
// among the columns, and no less than the tallest unit, grown by the least shortage a
// fill reports until the columns are few enough.
func balancedHeight(units []flowUnit, top, flowH float32, count int) float32 {
	h := flowH / float32(count)
	for _, u := range units {
		if !u.space {
			h = fmax(h, u.bottom-u.top)
		}
	}
	for range len(units) + 1 {
		starts, shortage := fillColumns(units, top, h)
		if len(starts) <= count || math.IsInf(float64(shortage), 1) {
			break
		}
		h += shortage
	}
	return h
}

// keepColumns places what the multi-column container m holds, as keepIn does: its flow
// where the columns show it, the boxes it is the containing block of where it put them.
func keepColumns(m *node) {
	cs := m.columns
	if m.frag == nil {
		m.frag = &flowFrag{}
	}
	*m.frag = flowFrag{cols: cs, pieces: m.frag.pieces[:0]}
	for _, c := range m.kids {
		if !outOfFlow(c.st.Position) || c.st.Display == style.DisplayNone {
			keepFlow(c, m, point{}, point{})
		}
	}
	for _, c := range m.hoisted {
		keepIn(c, m.absolute, false)
	}
}

// keepFlow places n, a box of m's flow whose parent's border box is at flow there, less
// the relative offsets above it, which add up to moved.
func keepFlow(n, m *node, flow, moved point) {
	if n.st.Display == style.DisplayNone {
		keepIn(n, n.parent.absolute, true)
		return
	}
	cs := m.columns
	n.final = n.unrounded
	n.final.scrollbar = n.final.scrollbar.add(n.gutter)
	rel := relativeShift(n)
	moved = point{moved.x + rel.x, moved.y + rel.y}
	at := point{flow.x + n.unrounded.location.x - rel.x, flow.y + n.unrounded.location.y - rel.y}
	if n.frag == nil {
		n.frag = &flowFrag{}
	}
	f := n.frag
	breaks := breaksInside(n)
	h := n.unrounded.size.h
	first := cs.at(at.y)
	last := first
	if breaks {
		last = cs.last(first, at.y+h)
	}
	by := cs.shift(first)
	origin := point{at.x + by.x, at.y + by.y}
	*f = flowFrag{cols: cs, y: at.y, col: first, pieces: f.pieces[:0]}
	if last > first {
		for k := first; k <= last; k++ {
			by := cs.shift(k)
			top := fmax(at.y, cs.starts[k]) + by.y
			bottom := at.y + h + by.y
			if k < last {
				// A box that goes on in the next column fills this one to its end.
				bottom = cs.top + cs.height
			}
			f.pieces = append(f.pieces, geom.Rect{X: at.x + by.x, Y: top, W: n.unrounded.size.w, H: fmax(bottom-top, 0)})
		}
		for _, p := range f.pieces {
			origin.y = fmin(origin.y, p.Y)
		}
		for i := range f.pieces {
			f.pieces[i].X -= origin.x
			f.pieces[i].Y -= origin.y
		}
	}
	f.oy = origin.y
	n.absolute = point{m.absolute.x + origin.x + moved.x, m.absolute.y + origin.y + moved.y}
	switch {
	case n.columns != nil:
		keepColumns(n)
		return
	case breaks:
		for _, c := range n.kids {
			if !outOfFlow(c.st.Position) || c.st.Display == style.DisplayNone {
				keepFlow(c, m, at, moved)
			}
		}
	default:
		for _, c := range n.kids {
			if !outOfFlow(c.st.Position) || c.st.Display == style.DisplayNone {
				keepIn(c, n.absolute, false)
			}
		}
	}
	for _, c := range n.hoisted {
		keepIn(c, n.absolute, false)
	}
}

// writeColumns writes n's place among columns into its box: its parts when columns cut
// it, their union as its size, and a multi-column container's columns.
func (n *node) writeColumns() {
	b := n.box
	if f := n.frag; f != nil && len(f.pieces) > 0 {
		b.Fragments = append(b.Fragments[:0], f.pieces...)
		b.Broken = true
		var u geom.Rect
		for i, p := range f.pieces {
			u = unionRect(u, p, i > 0)
		}
		b.Size = geom.Vec2{X: u.W, Y: u.H}
	} else if b.Broken {
		b.Fragments, b.Broken = b.Fragments[:0], false
	}
	if cs := n.columns; cs != nil {
		if b.Columns == nil {
			b.Columns = &box.Columns{}
		}
		*b.Columns = box.Columns{Count: len(cs.starts), Width: cs.width, Gap: cs.gap, Height: cs.height}
	} else {
		b.Columns = nil
	}
}
