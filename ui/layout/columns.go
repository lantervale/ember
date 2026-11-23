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
// shows in, applied when the boxes are placed (keepFlow) and written. Blocks spanning
// the columns (column-span: all) cut the flow in parts, each laid out on its own one
// after the other in the flow: those between them as rows of columns, each balanced
// alone, and the spanners at the container's width, as one column of their own.

// columnEpsilon is the slack in comparing places in the flow, under Chrome's 1/64 px.
const columnEpsilon = 0.01

// columnSet is how a multi-column container's flow is cut: starts[k] is where column k
// starts in the flow, down the container's border box, and places[k] where it shows.
type columnSet struct {
	starts []float32
	places []columnPlace
	// rows are the rows of columns between the spanners, from the first column's top.
	rows []columnRow
	// top is where the first column starts in the border box; width and gap the used
	// column width and gap, height the height of the columns and spanners down to the
	// end of the last.
	top, width, gap, height float32
}

// columnPlace is where a column shows in the container's border box: x from the content
// box's left (the first column's right edge is the content box's under rtl), top and bottom down it. A spanner shows as a column at x 0.
type columnPlace struct{ x, top, bottom float32 }

// columnRow is a row of count columns from the first, top down from the first column's
// top and height high.
type columnRow struct {
	first, count int
	top, height  float32
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
	return point{c.places[k].x, c.places[k].top - c.starts[k]}
}

// flowFrag is a node's place in the multi-column flow it is laid out in.
type flowFrag struct {
	cols *columnSet
	// y is the border box's top in the flow, col the column its border box's top shows
	// in, and ox and oy the origin from where that column's part shows, all less relative
	// offsets.
	y, ox, oy float32
	col       int
	// pieces are the box's parts when columns cut it, from its origin.
	pieces []geom.Rect
}

// lineShift is how far a line at y down the node's border box shows from there.
func (f *flowFrag) lineShift(y float32) (float32, float32) {
	k := f.cols.at(f.y + y)
	by := f.cols.shift(k)
	return by.x - f.cols.places[f.col].x - f.ox, by.y + f.y - f.oy
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
// height says otherwise; the columns past the used count overflow at its end. Spanners
// cut the content in rows of columns, each balanced, the last filled where it says so.
//
//nolint:funlen // the container's sizes, then its rows and spanners in turn
func (t *tree) columns(n *node, in input) output {
	st := n.st
	ratio := aspectRatio(st)
	padding := edges(st.Padding, in.parent.w)
	border := n.borders()
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

	parts := columnParts(n)
	outerW := styled.w
	if !some(outerW) {
		outerW = mMax(mClamp(t.columnsWidth(n, in, inset, parts), minSize.w, maxSize.w), pbSize.w)
	}
	if in.run == computeSize && in.axis == axisHorizontal {
		return sizeOutput(size{outerW, 0})
	}
	content := max(0, outerW-inset.horizontal())
	gap := columnGap(st, content)
	count, width := usedColumns(st, content, gap)

	limit := columnsLimit(styled.h, maxSize.h, inset.vertical())
	cs := &columnSet{top: inset.t, width: width, gap: gap}
	var oof []oofCandidate
	firstBaseline := none
	flowY, y := inset.t, inset.t
	for i, part := range parts {
		w := width
		if part.span {
			w = content
		}
		inner := t.flowOf(n, part.kids, in, w+inset.horizontal(), def(w))
		off := flowY - inset.t
		for _, k := range part.kids {
			k.unrounded.location.y += off
		}
		oof = append(oof, translateCandidates(inner.oof, point{0, off})...)
		flowH := max(0, inner.size.h-inset.vertical())
		h := flowH
		var starts []float32
		if !part.span && (part.flows() || len(parts) == 1) {
			units := t.flowUnitsOf(n, part.kids, size{w + inset.horizontal(), inner.size.h}, inset)
			room := none
			if i == len(parts)-1 {
				room = mMax(mSub(limit, y-inset.t), 0)
			}
			switch {
			case !st.ColumnFillAuto || i < len(parts)-1:
				h = mMin(balancedHeight(units, flowY, flowH, count), room)
			case some(room):
				h = filledHeight(room, flowH, some(styled.h))
			}
			starts, _, _ = fillColumns(units, flowY, h)
		}
		if k := len(cs.starts); starts != nil {
			for j, start := range starts {
				x := float32(j) * cs.pitch()
				if st.Dir == style.RTL {
					x = content - width - x
				}
				cs.starts = append(cs.starts, start)
				cs.places = append(cs.places, columnPlace{x: x, top: y, bottom: y + h})
			}
			cs.rows = append(cs.rows, columnRow{first: k, count: len(starts), top: y - inset.t, height: h})
		} else {
			cs.starts = append(cs.starts, flowY)
			cs.places = append(cs.places, columnPlace{top: y, bottom: y + h})
		}
		if !some(firstBaseline) && some(inner.firstBaseline) {
			at := inner.firstBaseline + off
			firstBaseline = at + cs.shift(cs.at(at)).y
		}
		flowY += flowH
		y += h
	}
	cs.height = y - inset.t

	final := size{outerW, mMax(or(styled.h, mClamp(cs.height+inset.vertical(), minSize.h, maxSize.h)), pbSize.h)}
	scroll := isScrollContainer(st)
	reach := cs.reach(padding, scroll, content, len(parts) > 1)
	areaInset := border.add(gutter)
	out := output{
		size:          final,
		overflow:      finalizeOverflow(reach, final.sub(areaInset.sum()), [2]bool{}, scroll),
		firstBaseline: firstBaseline,
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
	out.oof = cs.positioned(oof)
	return out
}

// columnPart is a run of a multi-column container's children laid out together: the
// blocks spanning the columns side by side (span), or what is between them.
type columnPart struct {
	kids []*node
	span bool
}

// flows reports a part with something in the flow, which makes a row of columns.
func (p columnPart) flows() bool {
	return slices.ContainsFunc(p.kids, func(k *node) bool {
		return k.st.Display != style.DisplayNone && !outOfFlow(k.st.Position)
	})
}

// spans reports a child of a multi-column container spanning its columns: a block-level
// box in the flow with column-span: all.
func spans(k *node) bool {
	st := k.st
	return st.ColumnSpanAll && st.Float == style.FloatNone && !outOfFlow(st.Position) &&
		(st.Display == style.Block || st.Display == style.FlexDisplay || st.Display == style.GridDisplay)
}

// columnParts cuts n's children at its spanners, every part in the flow's order; without
// a spanner the one part is all of them. A part with no spanner and nothing in the flow
// between two spanners goes with the spanners after it, so their margins collapse.
func columnParts(n *node) []columnPart {
	var parts []columnPart
	for _, k := range n.kids {
		span := spans(k)
		last := len(parts) - 1
		switch {
		case last >= 0 && parts[last].span == span:
			parts[last].kids = append(parts[last].kids, k)
		case last >= 0 && span && !parts[last].flows() && last > 0:
			parts[last-1].kids = append(append(parts[last-1].kids, parts[last].kids...), k)
			parts = parts[:last]
		default:
			parts = append(parts, columnPart{kids: []*node{k}, span: span})
		}
	}
	if len(parts) == 0 {
		parts = append(parts, columnPart{})
	}
	return parts
}

// flowOf is flow with n holding kids alone.
func (t *tree) flowOf(n *node, kids []*node, in input, w float32, avail space) output {
	all := n.kids
	n.kids = kids
	defer func() { n.kids = all }()
	return t.flow(n, in, w, avail)
}

// flowUnitsOf is flowUnits with n holding kids alone.
func (t *tree) flowUnitsOf(n *node, kids []*node, s size, inset rect) []flowUnit {
	all := n.kids
	n.kids = kids
	defer func() { n.kids = all }()
	return t.flowUnits(n, s, inset)
}

// columnsLimit is the height a container's own height, else its max-height, leaves its
// columns: none without either.
func columnsLimit(height, maxHeight, inset float32) float32 {
	limit := mSub(height, inset)
	if !some(limit) {
		limit = mSub(maxHeight, inset)
	}
	return mMax(limit, 0)
}

// filledHeight is the height of columns filled one after the other under limit: the
// flow's when it is shorter, unless the container's own height is fixed.
func filledHeight(limit, flowH float32, fixed bool) float32 {
	if fixed {
		return limit
	}
	return fmin(limit, flowH)
}

// reach is how far the columns go right and down from the padding box's corner, the
// spanners as wide as the content box when there are some (spanned), the padding after
// them counted where the container scrolls.
func (c *columnSet) reach(padding rect, scroll bool, content float32, spanned bool) rect {
	reach := rect{r: padding.l, b: padding.t + c.height}
	if spanned {
		reach.r += content
	}
	for _, row := range c.rows {
		reach.r = fmax(reach.r, padding.l+float32(row.count)*c.pitch()-c.gap)
	}
	if scroll {
		reach.r += padding.r
		reach.b += padding.b
	}
	return reach
}

// positioned are the flow's positioned boxes where the columns put them: one with no
// inset of its own starts where the flow shows its place, its line's column, or between
// blocks the column of what comes before it.
func (c *columnSet) positioned(flow []oofCandidate) []oofCandidate {
	out := make([]oofCandidate, len(flow))
	for i, o := range flow {
		k := c.at(o.y.area.start)
		if !o.inline {
			k = c.last(0, o.y.area.start)
		}
		by := c.shift(k)
		o.x.area.start += by.x
		o.x.area.end += by.x
		o.y.area.start += by.y
		o.y.area.end += by.y
		out[i] = o
	}
	return out
}

// columnsWidth is the border box's width a multi-column container asks for when nothing
// gives it one: its columns as wide as the content asks, or as column-width when wider,
// and its spanners as wide as they ask, no wider than the room when there is some.
func (t *tree) columnsWidth(n *node, in input, inset rect, parts []columnPart) float32 {
	st := n.st
	avail := in.avail.w
	measure := maxContentSpace
	if avail.kind == minContent {
		measure = minContentSpace
	}
	var flow, spanners []*node
	for _, p := range parts {
		if p.span {
			spanners = append(spanners, p.kids...)
		} else {
			flow = append(flow, p.kids...)
		}
	}
	column := max(0, t.flowOf(n, flow, in, none, measure).size.w-inset.horizontal())
	if st.ColumnWidth.Kind != style.Auto {
		column = max(column, st.ColumnWidth.Px)
	}
	count := float32(max(st.ColumnCount, 1))
	w := count*column + (count-1)*columnGap(st, none) + inset.horizontal()
	if len(spanners) > 0 {
		w = max(w, t.flowOf(n, spanners, in, none, measure).size.w)
	}
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
// takes at least widows to the next, or as many as orphans leave.
func appealOf(units []flowUnit, i, first int) int8 {
	u := &units[i]
	if u.line == 0 {
		return u.appeal
	}
	start := 0
	if f := &units[first]; f.para == u.para {
		start = f.line
	}
	// Widows give way where taking them along would leave this column fewer lines than
	// orphans.
	widows := min(u.widows, max(u.lines-start-u.orphans, 1))
	if u.line-start >= u.orphans && u.lines-u.line >= widows {
		return appealPerfect
	}
	return appealLines
}

// fillColumns cuts the flow of units, which starts at top, into columns of height h: each
// column takes the units that fit, and breaks before the first that doesn't, or earlier
// where a break appeals more, as Chrome's does. It returns where each column starts in
// the flow, the least room one more unit would have asked of a column, and the least
// appeal of the breaks it took.
func fillColumns(units []flowUnit, top, h float32) ([]float32, float32, int8) {
	starts := []float32{top}
	shortage := float32(math.Inf(1))
	least := appealPerfect
	if h <= 0 {
		return starts, shortage, least
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
			appeal := appealOf(units, i, first)
			if best >= 0 && bestAppeal > appeal {
				i, appeal = best, bestAppeal
			}
			least = min(least, appeal)
			column(i, units[i].top)
		}
	}
	return starts, shortage, least
}

// balancedHeight is the least column height that puts the flow of units, flowH high
// from top, in count columns, found as Chrome finds it: from the flow's height shared
// among the columns, and no less than the tallest unit, grown by the least shortage a
// fill reports until the columns are few enough and no break they take breaks a rule.
func balancedHeight(units []flowUnit, top, flowH float32, count int) float32 {
	h := flowH / float32(count)
	for _, u := range units {
		if !u.space {
			h = fmax(h, u.bottom-u.top)
		}
	}
	for range len(units) + 1 {
		starts, shortage, appeal := fillColumns(units, top, h)
		if len(starts) <= count && appeal == appealPerfect || math.IsInf(float64(shortage), 1) {
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
	n.final = n.laid()
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
			shift := cs.shift(k)
			top := fmax(at.y, cs.starts[k]) + shift.y
			bottom := at.y + h + shift.y
			if k < last {
				// A box that goes on in the next column fills this one to its end.
				bottom = cs.places[k].bottom
			}
			f.pieces = append(f.pieces,
				geom.Rect{X: at.x + shift.x, Y: top, W: n.unrounded.size.w, H: fmax(bottom-top, 0)})
		}
		left := origin.x
		for _, p := range f.pieces {
			origin.x, origin.y = fmin(origin.x, p.X), fmin(origin.y, p.Y)
		}
		f.ox = origin.x - left
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
		c := b.Columns
		c.Width, c.Gap, c.Rows = cs.width, cs.gap, c.Rows[:0]
		for _, r := range cs.rows {
			c.Rows = append(c.Rows, box.ColumnRow{Count: r.count, Top: r.top, Height: r.height})
		}
	} else {
		b.Columns = nil
	}
}
