package layout

import (
	"cmp"
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
	// leads are how far text-box-trim raises what each column shows, past the room
	// above the line it starts with.
	leads []float32
	// top is where the first column starts in the border box; width and gap the used
	// column width and gap, height the height of the columns and spanners down to the
	// end of the last.
	top, width, gap, height float32
	// lanes are where columns cut the boxes side by side in the flow, each its own way.
	lanes map[*node]laneCut
}

// laneCut is how columns cut a box side by side with others: from column first to the
// last of them all, at its own starts, and, stretched to the others, where it ends in
// that last column (none when it isn't).
type laneCut struct {
	first, last int
	starts      []float32
	end         float32
}

// lane is the columnSet the content of n, a box side by side with others, shows by, and
// where n ends when stretched to them.
func (c *columnSet) lane(n *node) (*columnSet, float32) {
	l, ok := c.lanes[n]
	if !ok {
		return c, none
	}
	d := *c
	d.starts = slices.Clone(c.starts)
	for k := l.first + 1; k <= l.last && k < len(d.starts); k++ {
		if j := k - l.first; j < len(l.starts) {
			d.starts[k] = l.starts[j]
		} else {
			d.starts[k] = float32(math.Inf(1))
		}
	}
	return &d, l.end
}

// columnPlace is where a column shows in the container's border box: x from the content
// box's left (the first column's right edge is the content box's under rtl), top and
// bottom down it. A spanner shows as a column at x 0, span set.
type columnPlace struct {
	x, top, bottom float32
	span           bool
}

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
	return point{c.places[k].x, c.places[k].top - c.starts[k] - c.lead(k)}
}

// lead is how far text-box-trim raises what column k shows.
func (c *columnSet) lead(k int) float32 {
	if k < len(c.leads) {
		return c.leads[k]
	}
	return 0
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

// cut gives f a piece in each column from first to last of a box of size w by h at at in
// the flow, each from the origin it returns: origin moved to the pieces' top-left. ends,
// when not nil, are where the box's content ends in the flow in each column but the last.
func (f *flowFrag) cut(at point, w, h float32, first, last int, origin point, ends []float32) point {
	cs := f.cols
	for k := first; k <= last; k++ {
		if cs.places[k].span && k > first && k < last {
			// a block round a spanner deeper down shows on either side of it
			continue
		}
		shift := cs.shift(k)
		top := fmax(at.y, cs.starts[k]+cs.lead(k)) + shift.y
		bottom := at.y + h + shift.y
		switch {
		case k == last:
		case ends != nil && some(ends[k-first]):
			bottom = fmin(ends[k-first]+shift.y, cs.places[k].bottom)
		case cs.places[k+1].span:
			// what a spanner cuts ends where its row's flow does
			bottom = fmin(cs.starts[k+1]+shift.y, cs.places[k].bottom)
		default:
			// A box that goes on in the next column fills this one to its end.
			bottom = cs.places[k].bottom
		}
		f.pieces = append(f.pieces, geom.Rect{X: at.x + shift.x, Y: top, W: w, H: fmax(bottom-top, 0)})
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
	return origin
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
	gutter := gutterEdges(st, n.ownGutter())
	pbSize := padding.add(border).sum()
	inset := padding.add(border).add(gutter)
	adj := boxSizingAdjustment(st, pbSize)
	minSize := stretched(resolveSize(st.MinSize, in.parent).withAspectRatio(ratio).mAdd(adj), st.MinSize, st, in.parent)
	maxSize := stretched(resolveMax(st.MaxSize, in.parent).withAspectRatio(ratio).mAdd(adj), st.MaxSize, st, in.parent)
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
	frags := map[*node][]placed{}
	for i, part := range parts {
		w := width
		if part.span {
			w = content
		}
		undo := part.apply(t)
		inner := t.flowOf(n, part.kids, in, w+inset.horizontal(), def(w))
		off := flowY - inset.t
		for _, k := range part.kids {
			k.unrounded.location.y += off
		}
		oof = append(oof, translateCandidates(inner.oof, point{0, off})...)
		flowH := max(0, inner.size.h-inset.vertical())
		h := flowH
		var starts []float32
		var fill filled
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
			fill = fillColumns(units, flowY, h)
			starts = fill.starts
			cs.addLanes(fill.lanes, len(cs.starts))
		}
		part.record(frags)
		undo()
		if k := len(cs.starts); starts != nil {
			for j, start := range starts {
				x := float32(j) * cs.pitch()
				if st.Dir == style.RTL {
					x = content - width - x
				}
				cs.starts = append(cs.starts, start)
				cs.leads = append(cs.leads, fill.leads[j])
				cs.places = append(cs.places, columnPlace{x: x, top: y, bottom: y + h})
			}
			cs.rows = append(cs.rows, columnRow{first: k, count: len(starts), top: y - inset.t, height: h})
		} else {
			cs.starts = append(cs.starts, flowY)
			cs.leads = append(cs.leads, 0)
			cs.places = append(cs.places, columnPlace{top: y, bottom: y + h, span: part.span})
		}
		if !some(firstBaseline) && some(inner.firstBaseline) {
			at := inner.firstBaseline + off
			firstBaseline = at + cs.shift(cs.at(at)).y
		}
		flowY += flowH
		y += h
	}
	cs.height = y - inset.t
	placeWrappers(n, point{}, frags)

	final := size{outerW, mMax(or(styled.h, mClamp(cs.height+inset.vertical(), minSize.h, maxSize.h)), pbSize.h)}
	scroll := isScrollContainer(st)
	reach := cs.reach(padding, scroll, content, len(parts) > 1, st.Dir == style.RTL)
	areaInset := border.add(gutter)
	out := output{
		size:          final,
		overflow:      finalizeOverflow(reach, final.sub(areaInset.sum()), fromEnd(st), scroll),
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
// blocks spanning the columns side by side (span), or what is between them, with the
// blocks round a spanner deeper down as the part holds them (views).
type columnPart struct {
	kids  []*node
	span  bool
	views []wrapView
}

// wrapView is a block round a spanner as a part of the flow holds it: the children it
// holds there, with its top edges in its first part (first), its bottom ones in its last.
type wrapView struct {
	n           *node
	kids        []*node
	first, last bool
}

// flows reports a part with something in the flow, which makes a row of columns.
func (p columnPart) flows() bool {
	return slices.ContainsFunc(p.kids, func(k *node) bool {
		return k.st.Display != style.DisplayNone && !outOfFlow(k.st.Position)
	})
}

// apply has the blocks round a spanner hold what p holds of them, cut at the spanners as
// Chrome slices them: no bottom edges above a spanner, no top ones below. It returns what
// undoes it.
func (p columnPart) apply(t *tree) func() {
	if len(p.views) == 0 {
		return func() {}
	}
	type was struct {
		kids []*node
		st   *style.Computed
	}
	saved := make([]was, len(p.views))
	for i, v := range p.views {
		saved[i] = was{v.n.kids, v.n.st}
		v.n.kids, v.n.st = v.kids, t.sliced(v.n.st, v.first, v.last)
		v.n.cache.clear()
	}
	return func() {
		for i, v := range p.views {
			v.n.kids, v.n.st = saved[i].kids, saved[i].st
			v.n.cache.clear()
		}
	}
}

// sliced is st without its top margin, border and padding unless first, nor its bottom
// ones unless last.
func (t *tree) sliced(st *style.Computed, first, last bool) *style.Computed {
	if first && last {
		return st
	}
	f := *st
	if !first {
		f.Margin.Top, f.Padding.Top, f.Border.Top = style.Length{}, style.Length{}, 0
	}
	if !last {
		f.Margin.Bottom, f.Padding.Bottom, f.Border.Bottom = style.Length{}, style.Length{}, 0
	}
	return &f
}

// record notes where the blocks round a spanner show in p's flow, laid out, and moves
// what they hold there to its place in the flow.
func (p columnPart) record(frags map[*node][]placed) {
	views := map[*node]bool{}
	for _, v := range p.views {
		views[v.n] = true
	}
	var walk func(w *node, at point)
	walk = func(w *node, at point) {
		l := w.unrounded
		l.location = point{at.x + l.location.x, at.y + l.location.y}
		frags[w] = append(frags[w], l)
		for _, k := range w.kids {
			switch {
			case views[k]:
				walk(k, l.location)
			case k.st.Display != style.DisplayNone:
				k.unrounded.location.x += l.location.x
				k.unrounded.location.y += l.location.y
			}
		}
	}
	for _, k := range p.kids {
		if views[k] {
			walk(k, point{})
		}
	}
}

// placeWrappers lays each block round a spanner under p out round its parts in the flow,
// frags: from the top of its first to the bottom of its last, at its first's left, with
// its first's top edges and its last's bottom ones, what it holds placed from it.
func placeWrappers(p *node, at point, frags map[*node][]placed) {
	for _, w := range p.kids {
		fs, ok := frags[w]
		if !ok {
			continue
		}
		first, last := fs[0], fs[len(fs)-1]
		abs := first.location
		l := last
		l.location = point{abs.x - at.x, abs.y - at.y}
		l.size = size{first.size.w, last.location.y + last.size.h - abs.y}
		l.overflow = newOverflowRect(rect{r: l.size.w, b: l.size.h}, [2]bool{})
		l.padding.t, l.border.t, l.margin.t = first.padding.t, first.border.t, first.margin.t
		w.unrounded = l
		for _, k := range w.kids {
			if _, wraps := frags[k]; !wraps && k.st.Display != style.DisplayNone {
				k.unrounded.location.x -= abs.x
				k.unrounded.location.y -= abs.y
			}
		}
		placeWrappers(w, abs, frags)
	}
}

// spans reports a child of a multi-column container spanning its columns: a block-level
// box in the flow with column-span: all.
func spans(k *node) bool {
	st := k.st
	return st.ColumnSpanAll && st.Float == style.FloatNone && !outOfFlow(st.Position) &&
		(st.Display == style.Block || st.Display == style.FlexDisplay || st.Display == style.GridDisplay)
}

// wrapsSpanner reports a block holding a spanner (column-span: all) a multi-column
// container's spanner cuts, as Chrome: one that starts no formatting context, nor
// positions any box.
func wrapsSpanner(k *node) bool {
	st := k.st
	if st.Display != style.Block || k.role != roleNone || k.context() != contextBlock || blockEstablishesBFC(st) ||
		st.Position != style.Static || st.Float != style.FloatNone || st.ColumnSpanAll {
		return false
	}
	positions := slices.ContainsFunc(k.kids, func(c *node) bool {
		return outOfFlow(c.st.Position) && c.st.Display != style.DisplayNone
	})
	return !positions && slices.ContainsFunc(k.kids, func(c *node) bool { return spans(c) || wrapsSpanner(c) })
}

// partsWalk cuts a multi-column container's children at its spanners, those deeper down
// included: open are the blocks round a spanner the walk is in, outermost first, resume
// set after a spanner until the part after it starts.
type partsWalk struct {
	parts  []columnPart
	open   []*node
	resume bool
}

// columnParts cuts n's children at its spanners, every part in the flow's order; without
// a spanner the one part is all of them. A part with nothing in the flow between two
// spanners goes with the spanners after it, so their margins collapse.
func columnParts(n *node) []columnPart {
	w := &partsWalk{parts: []columnPart{{}}}
	w.walk(n.kids)
	var parts []columnPart
	for i, p := range w.parts {
		if last := len(parts) - 1; last >= 0 && parts[last].span && !p.span && !p.flows() &&
			i+1 < len(w.parts) && w.parts[i+1].span {
			parts[last].kids = append(parts[last].kids, p.kids...)
			continue
		}
		if last := len(parts) - 1; last >= 0 && parts[last].span && p.span {
			parts[last].kids = append(parts[last].kids, p.kids...)
			continue
		}
		parts = append(parts, p)
	}
	return parts
}

func (w *partsWalk) walk(kids []*node) {
	for _, k := range kids {
		switch {
		case spans(k):
			w.span(k)
		case wrapsSpanner(k):
			w.add(k)
			p := w.part()
			p.views = append(p.views, wrapView{n: k, first: true})
			w.open = append(w.open, k)
			w.walk(k.kids)
			w.view(k).last = true
			w.open = w.open[:len(w.open)-1]
		default:
			w.add(k)
		}
	}
}

// part is the part the walk adds to, the part after a spanner started with the blocks
// still open round it.
func (w *partsWalk) part() *columnPart {
	if w.resume {
		w.resume = false
		w.parts = append(w.parts, columnPart{})
		p := &w.parts[len(w.parts)-1]
		for i, o := range w.open {
			if i == 0 {
				p.kids = append(p.kids, o)
			} else {
				v := &p.views[len(p.views)-1]
				v.kids = append(v.kids, o)
			}
			p.views = append(p.views, wrapView{n: o})
		}
	}
	return &w.parts[len(w.parts)-1]
}

// view is the view of the open block o in the part the walk adds to.
func (w *partsWalk) view(o *node) *wrapView {
	p := w.part()
	for i := range p.views {
		if p.views[i].n == o {
			return &p.views[i]
		}
	}
	panic("layout: no view of an open block round a spanner")
}

func (w *partsWalk) add(k *node) {
	if len(w.open) == 0 {
		p := w.part()
		p.kids = append(p.kids, k)
		return
	}
	v := w.view(w.open[len(w.open)-1])
	v.kids = append(v.kids, k)
}

func (w *partsWalk) span(k *node) {
	if !w.resume {
		if p := &w.parts[len(w.parts)-1]; len(p.kids) == 0 {
			// nothing before it
			w.parts = w.parts[:len(w.parts)-1]
		}
	}
	w.parts = append(w.parts, columnPart{kids: []*node{k}, span: true})
	w.resume = true
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

// reach is how far the columns go from the padding box's corner, right and down, or left
// and down under rtl, the spanners as wide as the content box when there are some
// (spanned), the padding after them counted where the container scrolls.
func (c *columnSet) reach(padding rect, scroll bool, content float32, spanned, rtl bool) rect {
	reach := rect{l: padding.l, r: padding.l, b: padding.t + c.height}
	if spanned || rtl {
		reach.r += content
	}
	for _, row := range c.rows {
		w := float32(row.count)*c.pitch() - c.gap
		if rtl {
			reach.l = fmin(reach.l, padding.l+content-w)
		} else {
			reach.r = fmax(reach.r, padding.l+w)
		}
	}
	if scroll {
		reach.l -= padding.l
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
	var spanners []*node
	var column float32
	for _, p := range parts {
		if p.span {
			spanners = append(spanners, p.kids...)
			continue
		}
		undo := p.apply(t)
		column = max(column, t.flowOf(n, p.kids, in, none, measure).size.w-inset.horizontal())
		undo()
	}
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
			t.touchPositioned(k)
		}
	}
}

// touchPositioned is touchFlow for the positioned boxes n cuts with the flow.
func (t *tree) touchPositioned(n *node) {
	for _, h := range n.hoisted {
		if h.st.Position == style.Absolute {
			h.performed = t.gen
			if cuttable(h) {
				t.touchFlow(h)
				t.touchPositioned(h)
			}
		}
	}
}

// breaksInside reports a box columns may cut: a block container in the flow that holds
// blocks or lines, a flex container between its lines and inside its items, a table or
// a row group between its rows. The others go into one column whole, as Chrome keeps
// monolithic boxes; so do a float, a row, a grid container and a nested multi-column
// container, which Chrome would cut.
func breaksInside(n *node) bool { return !outOfFlow(n.st.Position) && cuttable(n) }

// cuttable is breaksInside whatever n's position: an absolutely positioned box whose
// containing block is in the flow is cut with it.
func cuttable(n *node) bool {
	st := n.st
	if n.atomic || n.replaced || n.control != noControl ||
		st.Display != style.Block && st.Display != style.FlexDisplay && st.Display != style.GridDisplay ||
		st.Float != style.FloatNone || st.BreakInsideAvoid && !forcesBreak(n) || st.Multicol() || isScrollContainer(st) {
		return false
	}
	switch c := n.context(); n.role {
	case roleTable:
		return c == contextTable
	case roleGroup, roleCaption:
		return true
	case roleNone:
		return c == contextBlock || c == contextLeaf || c == contextFlex || c == contextGrid
	case roleRow:
		return c == contextBlock
	case roleColumn, roleColumnGroup:
	}
	return false
}

// trailingSpacing is the border-spacing below a table's last row, which Chrome cuts at the
// end of the column holding the last part of a table columns cut, and balances the columns
// without: 0 for any other box, a table collapsing its
// borders, and one with a caption below.
func trailingSpacing(n *node) float32 {
	st := n.st
	if n.role != roleTable || st.BorderCollapse || slices.ContainsFunc(n.captions(), func(c *node) bool {
		return c.st.CaptionSide == style.CaptionBottom
	}) {
		return 0
	}
	return st.BorderSpacing[1]
}

// forcesBreak reports a box holding a block that forces a column break before or after
// it: break-inside: avoid gives way to it, as Chrome's.
func forcesBreak(n *node) bool {
	for _, k := range n.kids {
		st := k.st
		if st.Display == style.DisplayNone || outOfFlow(st.Position) {
			continue
		}
		if st.BreakBefore == style.BreakColumn || st.BreakAfter == style.BreakColumn || forcesBreak(k) {
			return true
		}
	}
	return false
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
// anywhere (space): a line, a monolithic box, the edges of a block with what they hold,
// or boxes side by side (lanes), which columns cut each where its own content allows.
type flowUnit struct {
	top, bottom float32
	// space is empty room, room that below the content of a box side by side with others,
	// which it may be stretched to and which ends with the lowest of them.
	space, room bool
	// forced starts a column before the unit, keep being the margin kept above it then;
	// appeal is that of a break before it, a later line's worked out as columns fill.
	forced bool
	keep   float32
	appeal int8
	// para is the block whose line the unit is (0 for none), line its index among
	// lines, with the block's orphans and widows.
	para, line, lines int
	orphans, widows   int
	lanes             []lane
	// lead and tail are what text-box-trim takes off a line's top when it starts a
	// column, and off its bottom when it ends one.
	lead, tail float32
}

// lane is one of boxes side by side in the flow, a flex or grid row's item or a row's
// cell: n, the units of its flow and its border box's bottom.
type lane struct {
	n      *node
	units  []flowUnit
	bottom float32
}

type unitWalk struct {
	t *tree
	// root is the multi-column container, whose positioned boxes columns never cut.
	root  *node
	units []flowUnit
	// top is the edge the next unit starts at, a block's border top above its first
	// content, and pending the strongest break-before or break-after since the last unit.
	top     float32
	pending style.Break
	keep    float32
	paras   int
	sorted  bool
	// sides counts the boxes side by side walked into, whose lines no column trims.
	sides int
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
	w.add(u)
}

func (w *unitWalk) add(u flowUnit) {
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
	w := &unitWalk{t: t, root: n, top: none, sorted: true}
	w.content(n, 0, s, inset)
	return settle(w.units, w.sorted)
}

// settle puts units in the flow's order and makes those that overlap one: a float and the
// lines beside it can't be cut apart.
func settle(units []flowUnit, sorted bool) []flowUnit {
	if !sorted {
		slices.SortStableFunc(units, func(a, b flowUnit) int { return cmp.Compare(a.top, b.top) })
	}
	out := units[:0]
	for _, u := range units {
		if last := len(out) - 1; last >= 0 && u.top < out[last].bottom-columnEpsilon && !u.forced {
			p := &out[last]
			p.merge(u)
			p.space = p.space && u.space
			p.lanes = nil
			continue
		}
		out = append(out, u)
	}
	return out
}

// content walks what n holds, its border box at y in the flow, and the boxes it
// positions, which columns cut with that flow.
func (w *unitWalk) content(n *node, y float32, s size, inset rect) {
	if c := n.inline; c != nil {
		w.paras++
		ls := c.lines(s.w - inset.horizontal())
		at := y + w.t.textTop(n, ls, inset.t, s.h-inset.vertical())
		for i, l := range ls {
			lh, lb := w.t.lineBox(c, l)
			at += l.push
			lead, tail := w.trims(c, lh, lb)
			w.emit(flowUnit{top: at, bottom: at + lh, para: w.paras, line: i, lines: len(ls),
				orphans: n.st.Orphans, widows: n.st.Widows, lead: lead, tail: tail})
			at += lh
		}
	} else {
		kids := slices.DeleteFunc(slices.Clone(n.kids), func(k *node) bool {
			return k.st.Display == style.DisplayNone || outOfFlow(k.st.Position) || k.role == roleColumn ||
				k.role == roleColumnGroup
		})
		if n.role == roleTable {
			// a caption at the bottom comes after the rows in the flow
			slices.SortStableFunc(kids, func(a, b *node) int {
				return cmp.Compare(a.unrounded.location.y, b.unrounded.location.y)
			})
		}
		if c := n.context(); n.role == roleRow || c == contextFlex || c == contextGrid {
			w.side(kids, y)
		} else {
			for _, k := range kids {
				w.kid(k, y)
			}
		}
	}
	if n == w.root {
		return
	}
	for _, h := range n.hoisted {
		if h.st.Position == style.Absolute && h.st.Display != style.DisplayNone {
			w.box(h, y)
		}
	}
}

// trims are what the multi-column container's text-box-trim takes off a line of c, lh
// high with its baseline lb down, where it starts a column and where it ends one: Chrome
// trims each column's first and last lines, not those of boxes side by side.
func (w *unitWalk) trims(c *inlineContent, lh, lb float32) (float32, float32) {
	trim, e := w.root.st.TextBoxTrim, w.root.st.TextBoxEdge
	if trim == style.TrimNone || w.sides > 0 {
		return 0, 0
	}
	m := w.t.metricsOf(c, c.owner.st)
	var lead, tail float32
	if trim == style.TrimStart || trim == style.TrimBoth {
		lead = lb - overEdge(m, e)
	}
	if trim == style.TrimEnd || trim == style.TrimBoth {
		tail = lh - lb - underEdge(m, e)
	}
	return lead, tail
}

// kid walks k, a child in the flow whose parent's border box is at y, with the breaks
// asked before and after it.
func (w *unitWalk) kid(k *node, y float32) {
	st := k.st
	w.ask(st.BreakBefore)
	if w.pending == style.BreakColumn {
		w.keep = fmax(k.unrounded.margin.t, 0)
	}
	w.box(k, y)
	w.ask(st.BreakAfter)
}

// walked is a child walked apart: its lane, the break it asks after it and the margin a
// forced break before it keeps.
type walked struct {
	lane
	after    style.Break
	keep, at float32
}

// side walks kids, children of a box that may lay them side by side, whose border box is
// at y in the flow: each apart, as a lane, then those whose units overlap as one unit,
// the others one after the other.
func (w *unitWalk) side(kids []*node, y float32) {
	held, pending, keep, outer, sorted := w.top, w.pending, w.keep, w.units, w.sorted
	ws := make([]walked, 0, len(kids))
	w.sides++
	for _, k := range kids {
		w.units, w.sorted, w.top, w.pending, w.keep = nil, true, none, style.BreakAuto, 0
		w.kid(k, y)
		l := k.unrounded
		top := y + l.location.y - relativeShift(k).y
		units := settle(w.units, w.sorted)
		at := top
		if n := len(units); n > 0 {
			at = units[0].top
			units[n-1].room = n > 1 && units[n-1].space
		}
		ws = append(ws, walked{lane{k, units, top + l.size.h}, w.pending, fmax(l.margin.t, 0), at})
	}
	w.sides--
	w.top, w.pending, w.keep, w.units, w.sorted = held, pending, keep, outer, sorted
	slices.SortStableFunc(ws, func(a, b walked) int { return cmp.Compare(a.at, b.at) })
	for i := 0; i < len(ws); {
		if len(ws[i].units) == 0 {
			w.ask(ws[i].after)
			i++
			continue
		}
		j, low := i+1, ws[i].units[len(ws[i].units)-1].bottom
		for ; j < len(ws) && (len(ws[j].units) == 0 || ws[j].units[0].top < low-columnEpsilon); j++ {
			if u := ws[j].units; len(u) > 0 {
				low = fmax(low, u[len(u)-1].bottom)
			}
		}
		var lanes []lane
		after := style.BreakAuto
		for _, s := range ws[i:j] {
			if len(s.units) > 0 {
				lanes = append(lanes, s.lane)
			}
			after = max(after, s.after)
		}
		if len(lanes) == 1 {
			w.splice(lanes[0].units, ws[i].keep)
		} else {
			w.emit(flowUnit{top: lanes[0].units[0].top, bottom: low, lanes: lanes})
		}
		w.ask(after)
		i = j
	}
}

// splice puts units, walked apart, in the flow: the first takes the edge above it and the
// break asked before it, a forced one keeping margin.
func (w *unitWalk) splice(units []flowUnit, margin float32) {
	u := &units[0]
	if some(w.top) {
		u.top = fmin(u.top, w.top)
		w.top = none
	}
	switch w.pending {
	case style.BreakColumn:
		if !u.forced {
			u.forced, u.keep = true, margin
		}
	case style.BreakAvoid:
		u.appeal = min(u.appeal, appealAvoid)
	case style.BreakAuto:
	}
	w.pending, w.keep = style.BreakAuto, 0
	for _, u := range units {
		w.add(u)
	}
}

// box walks k, a box whose parent's border box is at y in the flow: one unit when
// columns never cut it, else what it holds between its edges.
func (w *unitWalk) box(k *node, y float32) {
	l := k.unrounded
	top := y + l.location.y - relativeShift(k).y
	bottom := top + l.size.h
	if !cuttable(k) {
		if l.size.h > 0 {
			w.emit(flowUnit{top: top, bottom: bottom})
		}
		return
	}
	held := some(w.top)
	if !held {
		w.top = top
	}
	before := len(w.units)
	w.content(k, top, l.size, l.padding.add(l.border))
	if k.st.BreakInsideAvoid {
		w.join(before)
	}
	if len(w.units) == before {
		if !held {
			w.top = none
		}
		if l.size.h > 0 {
			w.emit(flowUnit{top: top, bottom: bottom, space: true})
		}
		return
	}
	// Positioned boxes lie beside the rest: the lowest unit ends the content.
	low := len(w.units) - 1
	for j := before; j < len(w.units); j++ {
		if w.units[j].bottom > w.units[low].bottom {
			low = j
		}
	}
	bottom -= trailingSpacing(k)
	if end := w.units[low].bottom; bottom-end > l.padding.b+l.border.b+columnEpsilon {
		w.emit(flowUnit{top: end, bottom: bottom, space: true})
	} else {
		if bottom > end+columnEpsilon {
			// the box's edges below its last line end the column there
			w.units[low].tail = 0
		}
		w.units[low].bottom = fmax(end, bottom)
	}
}

// join makes the units from the first on one, but where a break is forced: those of a
// box with break-inside: avoid.
func (w *unitWalk) join(first int) {
	out := w.units[:first]
	for _, u := range w.units[first:] {
		if last := len(out) - 1; last >= first && !u.forced {
			o := &out[last]
			o.merge(u)
			o.space = o.space && u.space
			o.lanes = nil
			continue
		}
		out = append(out, u)
	}
	w.units = out
}

// merge makes u part of v, which takes u's trims where u's edges are its own.
func (v *flowUnit) merge(u flowUnit) {
	if u.top < v.top {
		v.top, v.lead = u.top, u.lead
	}
	if u.bottom > v.bottom {
		v.bottom, v.tail = u.bottom, u.tail
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

// filled is a flow of units cut into columns: where each column starts in the flow and
// how far a trim raises what it shows (by column, none for most), the least room one
// more unit would have asked of a column, the least appeal of the breaks taken, how far
// down its column the content of the last goes, and how the boxes side by side it cut
// are cut, their columns counted from the first.
type filled struct {
	starts   []float32
	leads    map[int]float32
	shortage float32
	appeal   int8
	used     float32
	lanes    map[*node]laneCut
}

// fillColumns cuts the flow of units, which starts at top, into columns of height h: each
// column takes the units that fit, and breaks before the first that doesn't, or earlier
// where a break appeals more, as Chrome's does. Boxes side by side that don't fit are cut
// each where its own content allows, unless one has no room for its first unit where
// something comes before them in the column.
func fillColumns(units []flowUnit, top, h float32) filled {
	f := filled{starts: []float32{top}, shortage: float32(math.Inf(1)), appeal: appealPerfect}
	if h <= 0 {
		return f
	}
	// s is where the column's room starts in the flow, below what a trim raises past it
	s, first, best := top, 0, -1
	var bestAppeal int8
	column := func(at int, y float32) {
		s, first, best = f.column(&units[at], y), at, -1
	}
	take := func(u *flowUnit) {
		if !u.room {
			f.used = fmax(f.used, u.bottom-u.tail-s)
		}
	}
	for i := 0; i < len(units); {
		u := &units[i]
		start := i == first
		if u.lanes != nil && !start && u.bottom-s > h+columnEpsilon {
			var shortage float32
			if start, shortage = lanesStart(u.lanes, s, h); !start {
				f.shortage = fmin(f.shortage, shortage)
			}
		}
		switch {
		case u.forced && i > first:
			column(i, u.top-u.keep)
		case u.bottom-u.tail-s <= h+columnEpsilon:
			if i > first {
				if a := appealOf(units, i, first); best < 0 || a >= bestAppeal {
					best, bestAppeal = i, a
				}
			}
			take(u)
			i++
		case u.lanes != nil && start:
			if end := f.side(u, s, h); end != s {
				s, first, best = end, i, -1
				f.used = u.bottom - end
			} else {
				take(u)
			}
			i++
		case u.space && u.top < s+h-columnEpsilon:
			column(i, s+h)
		case i == first:
			f.shortage = fmin(f.shortage, u.bottom-u.tail-s-h)
			take(u)
			i++
		default:
			f.shortage = fmin(f.shortage, u.bottom-u.tail-s-h)
			appeal := appealOf(units, i, first)
			if best >= 0 && bestAppeal > appeal {
				i, appeal = best, bestAppeal
			}
			f.appeal = min(f.appeal, appeal)
			column(i, units[i].top)
		}
	}
	return f
}

// column starts a column at y in the flow with unit u, and returns where its room starts:
// below the room a trim takes off u's top when the column starts with u.
func (f *filled) column(u *flowUnit, y float32) float32 {
	f.used = 0
	at := len(f.starts)
	f.starts = append(f.starts, y)
	if u.lead == 0 || y != u.top {
		return y
	}
	if f.leads == nil {
		f.leads = map[int]float32{}
	}
	f.leads[at] = u.lead
	return y + u.lead
}

// lanesStart reports boxes side by side that each have room for their first unit in a
// column of height h starting at s in the flow, and break where no rule is broken, as
// Chrome pushes a row to the next column rather than leave too few lines of an item; and
// when they don't, the least room more of them would ask.
func lanesStart(lanes []lane, s, h float32) (bool, float32) {
	ok, shortage := true, float32(math.Inf(1))
	for _, l := range lanes {
		if u := l.units[0]; !u.space && u.bottom-s > h+columnEpsilon {
			ok, shortage = false, fmin(shortage, u.bottom-s-h)
			continue
		}
		if f := fillColumns(l.units, s, h); f.appeal != appealPerfect {
			ok, shortage = false, fmin(shortage, f.shortage)
		}
	}
	return ok, shortage
}

// side cuts u, boxes side by side, in columns of height h from the one starting at s in
// the flow, each where its own content allows. It returns where the flow after them starts
// in the column they end in, as far above their end as they show there; s when they all
// fit in this one.
func (f *filled) side(u *flowUnit, s, h float32) float32 {
	fills := make([]filled, len(u.lanes))
	count, low := 1, negInf
	for i, l := range u.lanes {
		fills[i] = fillColumns(l.units, s, h)
		count = max(count, len(fills[i].starts))
		low = fmax(low, l.bottom)
		f.shortage = fmin(f.shortage, fills[i].shortage)
		f.appeal = min(f.appeal, fills[i].appeal)
	}
	if count == 1 {
		return f.tail(u, s, low, h)
	}
	// how far down the last column the lanes there go, the lowest stretching the others
	var used float32
	for _, fl := range fills {
		if len(fl.starts) == count {
			used = fmax(used, fl.used)
		}
	}
	end := low - used
	base := len(f.starts) - 1
	for j := 1; j < count; j++ {
		start := end
		if j < count-1 {
			for _, fl := range fills {
				if j < len(fl.starts) {
					start = fmin(start, fl.starts[j])
				}
			}
		}
		f.starts = append(f.starts, fmax(start, f.starts[len(f.starts)-1]))
	}
	end = f.tail(u, end, low, h)
	for i, l := range u.lanes {
		st := fills[i].starts
		c := laneCut{first: base, last: base + count - 1, starts: st, end: none}
		if len(st) == count && l.bottom >= low-columnEpsilon {
			c.end = st[count-1] + used
		}
		f.lane(l.n, c, 0)
		for n, nested := range fills[i].lanes {
			f.lane(n, nested, base)
		}
	}
	return end
}

// tail is where the flow after boxes side by side in u starts, end where it does in their
// last column, of height h: the edges of their container below them, from low, go to the
// next column when that one has no room for them.
func (f *filled) tail(u *flowUnit, end, low, h float32) float32 {
	if over := u.bottom - end - h; over > columnEpsilon {
		f.shortage = fmin(f.shortage, over)
		f.starts = append(f.starts, low)
		return low
	}
	return end
}

// lane records how columns cut n, its columns from first counted off columns on.
func (f *filled) lane(n *node, c laneCut, off int) {
	if f.lanes == nil {
		f.lanes = map[*node]laneCut{}
	}
	c.first, c.last = c.first+off, c.last+off
	f.lanes[n] = c
}

// addLanes records lanes, cut from the first of the columns from first on.
func (c *columnSet) addLanes(lanes map[*node]laneCut, first int) {
	for n, l := range lanes {
		if c.lanes == nil {
			c.lanes = map[*node]laneCut{}
		}
		l.first, l.last = l.first+first, l.last+first
		c.lanes[n] = l
	}
}

// balancedHeight is the least column height that puts the flow of units, flowH high
// from top, in count columns, found as Chrome finds it: from the flow's height shared
// among the columns, and no less than the tallest unit, grown by the least shortage a
// fill reports until the columns are few enough and no break they take breaks a rule.
func balancedHeight(units []flowUnit, top, flowH float32, count int) float32 {
	h := fmax(flowH/float32(count), tallest(units))
	for range unitCount(units) + 1 {
		f := fillColumns(units, top, h)
		if len(f.starts) <= count && f.appeal == appealPerfect || math.IsInf(float64(f.shortage), 1) {
			break
		}
		h += f.shortage
	}
	return h
}

// tallest is the height of the tallest of units columns never cut, those of boxes side
// by side included.
func tallest(units []flowUnit) float32 {
	var h float32
	for _, u := range units {
		switch {
		case u.lanes != nil:
			for _, l := range u.lanes {
				h = fmax(h, tallest(l.units))
			}
		case !u.space:
			h = fmax(h, u.bottom-u.top)
		}
	}
	return h
}

// unitCount is how many units there are, those of boxes side by side included.
func unitCount(units []flowUnit) int {
	n := len(units)
	for _, u := range units {
		for _, l := range u.lanes {
			n += unitCount(l.units)
		}
	}
	return n
}

// keepColumns places what the multi-column container m holds, as keepIn does: its flow
// where the columns show it, the boxes it is the containing block of where it put them.
func keepColumns(m *node) {
	cs := m.columns
	if m.frag == nil {
		m.frag = &flowFrag{}
	}
	*m.frag = flowFrag{cols: cs, pieces: m.frag.pieces[:0]}
	if len(cs.places) > 0 {
		// its own lines show where each column is, its origin standing where it is
		m.frag.ox = -cs.places[0].x
	}
	for _, c := range m.kids {
		if !outOfFlow(c.st.Position) || c.st.Display == style.DisplayNone {
			keepFlow(c, m, cs, point{}, point{})
		}
	}
	for _, c := range m.hoisted {
		keepIn(c, m.absolute, false)
	}
}

// keepFlow places n, a box of m's flow whose parent's border box is at flow there, less
// the relative offsets above it, which add up to moved, cut by the columns cs.
func keepFlow(n, m *node, cs *columnSet, flow, moved point) {
	if n.st.Display == style.DisplayNone {
		keepIn(n, n.parent.absolute, true)
		return
	}
	cs, stretched := cs.lane(n)
	n.final = n.laid()
	n.final.scrollbar = n.final.scrollbar.add(n.gutter)
	rel := relativeShift(n)
	moved = point{moved.x + rel.x, moved.y + rel.y}
	at := point{flow.x + n.unrounded.location.x - rel.x, flow.y + n.unrounded.location.y - rel.y}
	if n.frag == nil {
		n.frag = &flowFrag{}
	}
	f := n.frag
	breaks := cuttable(n)
	h := n.unrounded.size.h
	first := cs.at(at.y)
	last := first
	if breaks {
		last = cs.last(first, at.y+h-trailingSpacing(n))
	}
	if tail := trailingSpacing(n); last > first && tail > 0 {
		// The spacing below the last row gives way at the column's end, as a margin.
		end := at.y + h - tail + cs.shift(last).y
		h -= tail - fmin(fmax(cs.places[last].bottom-end, 0), tail)
	}
	if some(stretched) && last > first {
		// stretched to the boxes beside it, it ends where the lowest of them does
		h = stretched - at.y
	}
	by := cs.shift(first)
	origin := point{at.x + by.x, at.y + by.y}
	if clip := cs.starts[first] + cs.lead(first) - at.y; cs.lead(first) > 0 && clip > 0 && last == first {
		// a box starting with the line a column's trim raises keeps its top at the column's
		origin.y += clip
		n.final.size.h -= clip
	}
	*f = flowFrag{cols: cs, y: at.y, col: first, pieces: f.pieces[:0]}
	if last > first {
		origin = f.cut(at, n.unrounded.size.w, h, first, last, origin, contentEnds(n, cs, at, first, last))
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
				keepFlow(c, m, cs, at, moved)
			}
		}
	default:
		for _, c := range n.kids {
			if !outOfFlow(c.st.Position) || c.st.Display == style.DisplayNone {
				keepIn(c, n.absolute, false)
			}
		}
	}
	keepPositioned(n, m, cs, at, moved, breaks)
}

// contentEnds are where a box, its border box at at in the flow, ends in each column from
// first to last but the last, as Chrome ends it: a table's row group below the spacing
// after the last row it shows there, a block round a spanner below the last box ending
// there; nil for any other box.
func contentEnds(n *node, cs *columnSet, at point, first, last int) []float32 {
	var after float32
	switch table := n.parent; {
	case n.role == roleGroup && table != nil && table.role == roleTable:
		if !table.st.BorderCollapse {
			after = table.st.BorderSpacing[1]
		}
	case wrapsSpanner(n):
	default:
		return nil
	}
	ends := make([]float32, last-first)
	for i := range ends {
		ends[i] = none
	}
	for _, k := range n.kids {
		if k.st.Display == style.DisplayNone || outOfFlow(k.st.Position) {
			continue
		}
		top := at.y + k.unrounded.location.y
		bottom := top + k.unrounded.size.h
		from := cs.at(top) - first
		to := cs.last(from+first, bottom) - first
		for c := max(from, 0); c < min(to, len(ends)); c++ {
			// a box cut fills the columns before its last
			ends[c] = float32(math.Inf(1))
		}
		if to >= 0 && to < len(ends) {
			ends[to] = fmax(or(ends[to], negInf), bottom+after)
		}
	}
	return ends
}

// keepPositioned places the boxes n positions: when columns cut n, an absolutely
// positioned one through the flow, at flow less the relative offsets moved, as keepFlow
// does; the others as keepIn.
func keepPositioned(n, m *node, cs *columnSet, flow, moved point, cut bool) {
	for _, c := range n.hoisted {
		if cut && c.st.Position == style.Absolute {
			keepFlow(c, m, cs, flow, moved)
		} else {
			keepIn(c, n.absolute, false)
		}
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
