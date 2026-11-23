package layout

import (
	"maps"
	"slices"
	"sort"

	"github.com/lantervale/ember/ui/internal/style"
)

// subgridAxis is what a grid hands a subgrid among its items in an axis the subgrid takes
// its tracks in: how many it spans, the names the grid gives their lines, the grid's gap, and
// the tracks themselves from the subgrid's margin box start once the grid has sized them.
type subgridAxis struct {
	span   int
	names  map[string][]int
	gap    float32
	tracks []span
}

// gridFrame is the frame a grid lays its tracks out in: its writing mode's, mirrored under
// direction rtl.
type gridFrame struct {
	mode style.WritingMode
	rtl  bool
}

func frameOfGrid(n *node) gridFrame {
	st := n.box.Style
	if st == nil {
		st = n.st
	}
	return gridFrame{st.Mode, st.Dir == style.RTL}
}

// page is the page's axis a of the frame runs along, and whether it runs up or leftwards.
func (f gridFrame) page(a axis) (axis, bool) {
	if a == horizontal {
		if !f.mode.Vertical() {
			return horizontal, f.rtl
		}
		return vertical, f.rtl != f.mode.Upward()
	}
	if !f.mode.Vertical() {
		return vertical, false
	}
	return horizontal, f.mode.BlockFromRight()
}

// from is the axis of the frame along the axis a of the frame o, and whether the two run
// opposite ways.
func (f gridFrame) from(o gridFrame, a axis) (axis, bool) {
	pa, back := o.page(a)
	for _, b := range [2]axis{horizontal, vertical} {
		if pb, bb := f.page(b); pb == pa {
			return b, bb != back
		}
	}
	return a, false
}

// edges are the margin, border and padding of a box of physical style st on each side of
// the frame f.
func (f gridFrame) edges(st *style.Computed, basis float32) rect {
	e := frameEdges(edges(st.Margin, basis).add(edges(st.Padding, basis)).add(borders(st)), f.mode)
	if f.rtl {
		e.l, e.r = e.r, e.l
	}
	return e
}

// toOwn is the axis of n's own grid along its grid's axis a, and whether it runs the other
// way: its grid's frame and its own differ.
func (n *node) toOwn(a axis) (axis, bool) {
	return frameOfGrid(n).from(frameOfGrid(n.parent), a)
}

// subgridAxes reports the axes, its grid's, n takes its grid's tracks in: a grid that is an
// in-flow item of a grid, no query container, with subgrid as the template of its own axis
// along it.
func (n *node) subgridAxes() [2]bool {
	st := n.st
	switch {
	case !st.Subgrid.Width && !st.Subgrid.Height, n.parent == nil, n.parent.context() != contextGrid,
		st.Display != style.GridDisplay && st.Display != style.InlineGrid, outOfFlow(st.Position),
		st.ContainerType != style.ContainerNormal, n.context() != contextGrid && (len(n.kids) > 0 || n.inline != nil):
		return [2]bool{}
	}
	own := [2]bool{st.Subgrid.Width, st.Subgrid.Height}
	var out [2]bool
	for a := range out {
		k, _ := n.toOwn(axis(a))
		out[a] = own[k]
	}
	return out
}

// handed is what n's grid handed it in each axis of its own it is a subgrid in, nil in the
// others.
func (n *node) handed() [2]*subgridAxis {
	var out [2]*subgridAxis
	for a, on := range n.subgridAxes() {
		if !on {
			continue
		}
		k, _ := n.toOwn(axis(a))
		out[k] = n.subgrid[a]
		if out[k] == nil {
			out[k] = &subgridAxis{span: 1}
		}
	}
	return out
}

// usedGap is the gap between n's tracks in a: its own, else its grid's where it is a subgrid.
func usedGap(n *node, a axis, basis float32) float32 {
	_, _, gap := templateOf(n.st, a)
	if gap.Kind == style.Auto {
		if sub := n.handed()[a]; sub != nil {
			return sub.gap
		}
	}
	return resolveOrZero(gap, basis)
}

// hand gives each subgrid among items the names of the lines it spans in a, its span
// changed or the names dropping what it last had.
func hand(items []*gridItem, a axis, lines gridLines) {
	for _, it := range items {
		k := it.node
		if !it.subgrid[a] {
			k.subgrid[a] = nil
			continue
		}
		at := it.placement(a)
		names := lines.within(at)
		if _, back := k.toOwn(a); back {
			names = reverseNames(names, at.span())
		}
		if prev := k.subgrid[a]; prev != nil && prev.span == at.span() &&
			maps.EqualFunc(prev.names, names, slices.Equal[[]int]) {
			continue
		}
		k.subgrid[a] = &subgridAxis{span: at.span(), names: names}
		k.cache.clear()
	}
}

// handTracks gives each subgrid among items the tracks it spans in a, from its first:
// where the tracks stand once aligned, else their sizes laid end to end.
func handTracks(items []*gridItem, a axis, tracks []gridTrack, gap float32, aligned bool) {
	for _, it := range items {
		if !it.subgrid[a] {
			continue
		}
		sub := it.node.subgrid[a]
		ix := it.indexes(a)
		spans := make([]span, 0, sub.span)
		at, origin := float32(0), tracks[ix.start+1].offset
		for i := ix.start + 1; i < ix.end; i++ {
			tr := &tracks[i]
			if aligned {
				at = tr.offset - origin
			}
			if tr.kind == trackTrack {
				spans = append(spans, span{at, at + tr.baseSize})
			}
			at += tr.baseSize
		}
		if _, back := it.node.toOwn(a); back {
			spans = reverseSpans(spans)
		}
		if sub.gap == gap && slices.Equal(sub.tracks, spans) {
			continue
		}
		sub.gap, sub.tracks = gap, spans
		it.node.cache.clear()
	}
}

// subgridTracks are the tracks a subgrid takes from its grid, from its content box's start:
// each where the grid has it, the first starting no sooner than the content box and the last
// ending no later (content from the margin box's start), the edges beside a gutter moved by
// half what the subgrid's gap differs from the grid's.
func subgridTracks(sub *subgridAxis, content span, gap float32) []gridTrack {
	shift := (gap - sub.gap) / 2
	last := len(sub.tracks) - 1
	fixed := func(kind trackKind, v float32) gridTrack {
		return gridTrack{kind: kind, min: style.Px(v), max: style.Px(v), fitContentLimit: inf}
	}
	out := make([]gridTrack, 0, 2*len(sub.tracks)+1)
	prev := float32(0)
	for i, tr := range sub.tracks {
		s, e := tr.start-content.start, tr.end-content.start
		if i > 0 {
			s += shift
		} else {
			s = fmax(s, 0)
		}
		if i < last {
			e -= shift
		} else {
			e = fmin(e, content.end-content.start)
		}
		e = fmax(e, s)
		out = append(out, fixed(trackGutter, s-prev), fixed(trackTrack, e-s))
		prev = e
	}
	return append(out, fixed(trackGutter, 0))
}

// clampToSubgrid moves an item placed past a subgrid's n tracks into them: an item outside on
// one side takes the edge track, one partly outside loses what lies outside.
func clampToSubgrid(l lineSpan, n int) lineSpan {
	switch {
	case l.end <= 0:
		return lineSpan{0, 1}
	case l.start >= n:
		return lineSpan{n - 1, n}
	}
	return lineSpan{max(l.start, 0), min(l.end, n)}
}

// noNegative takes a subgrid's lines before its first to its first: it has no tracks there.
func (p ozLines) noNegative() ozLines {
	if p.start.kind == placeLine {
		p.start.v = max(p.start.v, 0)
	}
	if p.end.kind == placeLine {
		p.end.v = max(p.end.v, 0)
	}
	return p
}

func edgesAlong(r rect, a axis) span {
	if a == horizontal {
		return span{r.l, r.r}
	}
	return span{r.t, r.b}
}

// reverseNames are the names of the lines of a span of n tracks counted from its other end.
func reverseNames(names map[string][]int, n int) map[string][]int {
	out := make(map[string][]int, len(names))
	for name, lines := range names {
		r := make([]int, len(lines))
		for i, l := range lines {
			r[len(lines)-1-i] = n - l
		}
		out[name] = r
	}
	return out
}

// reverseSpans are tracks laid from the margin box start of a subgrid, from its other end.
func reverseSpans(tracks []span) []span {
	if len(tracks) == 0 {
		return tracks
	}
	w := tracks[len(tracks)-1].end
	out := make([]span, len(tracks))
	for i, tr := range tracks {
		out[len(tracks)-1-i] = span{w - tr.end, w - tr.start}
	}
	return out
}

// gridPlan is a grid's explicit grid in each axis and its in-flow items placed on it.
type gridPlan struct {
	reps      [2]int
	lines     [2]gridLines
	counts    [2]trackCounts
	items     []*gridItem
	occupancy *cellOccupancy
}

// planGrid places n's in-flow items: autoFit is the size auto-fill and auto-fit repeat to
// fill, strategies where it is definite say how. In an axis it is a subgrid in, n has the
// tracks it spans of its grid and no implicit ones, its items clamped to them. The subgrids
// among its items are handed the names of the lines they span.
func (t *tree) planGrid(n *node, autoFit, strategies size) *gridPlan {
	st := n.st
	sub := n.handed()
	p := &gridPlan{}
	var explicit [2]int
	for a, s := range sub {
		ax := axis(a)
		if s != nil {
			explicit[a] = s.span
			p.lines[a] = subgridLines(st, ax, s)
			continue
		}
		strategy := minRepetitionsThatDoOverflow
		if some(strategies.get(ax)) {
			strategy = maxRepetitionsThatDoNotOverflow
		}
		p.reps[a], explicit[a] = explicitGridSize(st, autoFit.get(ax), strategy, ax)
		explicit[a] = max(explicit[a], templateAreaTracks(st.Areas, ax))
		p.lines[a] = newGridLines(st, ax, p.reps[a], explicit[a])
	}
	var placements []itemPlacement
	for i, k := range n.kids {
		if k.st.Display == style.DisplayNone || outOfFlow(k.st.Position) {
			continue
		}
		at := itemPlacement{h: p.lines[horizontal].placement(k.st.Column), v: p.lines[vertical].placement(k.st.Row)}
		if sub[horizontal] != nil {
			at.h = at.h.noNegative()
		}
		if sub[vertical] != nil {
			at.v = at.v.noNegative()
		}
		placements = append(placements, at)
		it := newGridItem(k, st.AlignItems, st.JustifyItems, i)
		it.subgrid = k.subgridAxes()
		p.items = append(p.items, it)
	}
	estCols, estRows := gridSizeEstimate(explicit[horizontal], explicit[vertical], placements)
	p.occupancy = newCellOccupancy(estCols, estRows)
	byOrder := make([]int, len(p.items))
	for i := range byOrder {
		byOrder[i] = i
	}
	sort.SliceStable(byOrder, func(i, j int) bool { return p.items[byOrder[i]].st.Order < p.items[byOrder[j]].st.Order })
	ordered := make([]*gridItem, len(p.items))
	orderedPlacements := make([]itemPlacement, len(p.items))
	for i, j := range byOrder {
		ordered[i], orderedPlacements[i] = p.items[j], placements[j]
	}
	placeGridItems(p.occupancy, ordered, orderedPlacements, st.Flow)
	p.counts = [2]trackCounts{p.occupancy.columns, p.occupancy.rows}
	for a, s := range sub {
		if s == nil {
			continue
		}
		p.counts[a] = trackCounts{explicit: s.span}
		for _, it := range p.items {
			it.setPlacement(axis(a), clampToSubgrid(it.placement(axis(a)), s.span))
		}
	}
	hand(p.items, horizontal, p.lines[horizontal])
	hand(p.items, vertical, p.lines[vertical])
	return p
}

// tracks are n's tracks in a: those its grid handed it where it is a subgrid, else its own.
func (p *gridPlan) tracks(n *node, a axis, gap, basis float32) []gridTrack {
	if s := n.handed()[a]; s != nil && len(s.tracks) > 0 {
		content := edgesAlong(frameOfGrid(n).edges(n.box.Style, basis), a)
		return subgridTracks(s, span{content.start, s.tracks[len(s.tracks)-1].end - content.end}, gap)
	}
	return initializeGridTracks(p.counts[a], n.st, a, p.reps[a], func(i int) bool { return p.occupancy.occupied(a, i) })
}

// sizingItems are the items whose contributions size the tracks of a of the grid g: its own,
// but each subgrid in a gives way to the items it holds, hoisted to the grid's lines.
func (t *tree) sizingItems(g *node, items []*gridItem, a axis, gaps [2]float32, basis float32) []*gridItem {
	var out []*gridItem
	for _, it := range items {
		if it.subgrid[a] {
			out = t.hoist(g, it, a, gaps, gaps, basis, out)
			continue
		}
		out = append(out, it)
	}
	return out
}

// hoisting is a subgrid s whose items size the tracks of the grid g: the axis of its own
// along each of g's, whether it runs the other way, its own gap, half what that differs from
// g's, and the margin, border and padding at its edges with those of the subgrids round it.
type hoisting struct {
	g, n    *node
	s       *gridItem
	ownAxis [2]axis
	back    [2]bool
	own     [2]float32
	half    [2]float32
	edge    [2]span
	// fixed is the subgrid's own tracks in an axis it doesn't share with g, and counts theirs.
	fixed  []gridTrack
	counts trackCounts
}

func newHoisting(g *node, s *gridItem, gaps, parentGaps [2]float32, basis float32) *hoisting {
	n := s.node
	h := &hoisting{g: g, n: n, s: s}
	gf, nf := frameOfGrid(g), frameOfGrid(n)
	box := gf.edges(n.box.Style, 0)
	for ax := range 2 {
		h.ownAxis[ax], h.back[ax] = nf.from(gf, axis(ax))
		h.own[ax] = parentGaps[ax]
		if _, _, gap := templateOf(n.st, h.ownAxis[ax]); gap.Kind != style.Auto {
			h.own[ax] = resolveOrZero(gap, basis)
		}
		h.half[ax] = (h.own[ax] - gaps[ax]) / 2
		h.edge[ax] = edgesAlong(box, axis(ax))
		h.edge[ax].start += s.extra[ax].start
		h.edge[ax].end += s.extra[ax].end
	}
	return h
}

// place puts the subgrid's item it at g's lines in a, and in the other axis at them where
// the subgrid shares it with g, else loose on the tracks of the subgrid owning them.
func (h *hoisting) place(t *tree, it *gridItem, a axis) {
	s := h.s
	ownLines := [2]lineSpan{it.column, it.row}
	local, sub := ownLines, it.subgrid
	for ax := range 2 {
		l := ownLines[h.ownAxis[ax]]
		if span := s.placement(axis(ax)).span(); h.back[ax] {
			l = lineSpan{span - l.end, span - l.start}
		}
		local[ax], it.subgrid[ax] = l, sub[h.ownAxis[ax]]
	}
	if mode, gm := frameOf(h.n), frameOf(h.g); mode != gm {
		it.st, it.frame = t.turned(it.node.box.Style, gm), &gm
		if across(mode, gm) {
			it.alignSelf, it.justifySelf = it.justifySelf, it.alignSelf
		}
	}
	toGrid := func(ax axis) {
		l, from := local[ax], s.placement(ax)
		extra := span{h.half[ax], h.half[ax]}
		if l.start == 0 {
			extra.start = h.edge[ax].start
		}
		if l.end == from.span() {
			extra.end = h.edge[ax].end
		}
		it.extra[ax] = extra
		it.setPlacement(ax, lineSpan{from.start + l.start, from.start + l.end})
	}
	toGrid(a)
	o := a.other()
	l := ownLines[h.ownAxis[o]]
	switch {
	case !s.subgrid[o]:
		it.owner, it.local, it.localAxis = h.n, l, h.ownAxis[o]
		it.fixed = fixedExtent(h.fixed, h.counts, l)
		it.setPlacement(o, s.placement(o))
		it.loose[o] = true
	case s.loose[o]:
		it.owner, it.local, it.localAxis = s.owner, lineSpan{s.local.start + l.start, s.local.start + l.end}, s.localAxis
		it.fixed = none
		it.setPlacement(o, s.placement(o))
		it.loose[o] = true
	default:
		toGrid(o)
	}
	it.hoisted = true
	if a == vertical {
		it.node.sharing = false
	} else if it.alignSelf == style.AlignBaseline {
		it.alignSelf = style.AlignStart
	}
}

// fixedExtent is how far the tracks between lines l reach where each is of a fixed size, none
// where one isn't: an item's height in a subgrid of columns alone, known before its rows are.
func fixedExtent(tracks []gridTrack, counts trackCounts, l lineSpan) float32 {
	s, okS := counts.trackVecIndex(l.start)
	e, okE := counts.trackVecIndex(l.end)
	if !okS || !okE || e >= len(tracks) {
		return none
	}
	sum := float32(0)
	for i := s + 1; i < e; i++ {
		lo, hi := definiteValue(tracks[i].min, none), definiteValue(tracks[i].max, none)
		if !some(lo) || lo != hi {
			return none
		}
		sum += lo
	}
	return sum
}

// edgeItems are empty items on the subgrid's first and last tracks in a, holding its edges.
func (h *hoisting) edgeItems(a axis) [2]*gridItem {
	var out [2]*gridItem
	at, o := h.s.placement(a), a.other()
	for i := range out {
		e := &gridItem{st: &style.Computed{}, baseline: none, minContentCache: noSize, minimumCache: noSize,
			maxContentCache: noSize, loose: [2]bool{a == vertical, a == horizontal}}
		e.setPlacement(o, h.s.placement(o))
		if i == 0 {
			e.setPlacement(a, lineSpan{at.start, at.start + 1})
			e.extra[a].start = h.edge[a].start
		} else {
			e.setPlacement(a, lineSpan{at.end - 1, at.end})
			e.extra[a].end = h.edge[a].end
		}
		out[i] = e
	}
	return out
}

// hoist appends the items that size the tracks of a of the grid g in place of the subgrid s,
// an item at lines of g: its own at their lines there, those of its subgrids in a in turn, and
// an empty item on each of its edge tracks. Each item takes as more margin, in each axis s
// shares with g, the margin, border and padding of the subgrids whose edge it stands on, and
// beside a gutter half what the subgrid's gap differs from g's. Lines, axes and sides are g's:
// a subgrid of another writing mode or direction has its own along g's that lie with them,
// maybe the other way round, and its items are sized in g's frame.
func (t *tree) hoist(g *node, s *gridItem, a axis, gaps, parentGaps [2]float32, basis float32,
	out []*gridItem) []*gridItem {
	h := newHoisting(g, s, gaps, parentGaps, basis)
	p := t.planGrid(s.node, noSize, noSize)
	if o := h.ownAxis[a.other()]; !s.subgrid[a.other()] {
		h.fixed, h.counts = p.tracks(s.node, o, usedGap(s.node, o, basis), 0), p.counts[o]
	}
	for _, it := range p.items {
		h.place(t, it, a)
		if it.subgrid[a] {
			out = t.hoist(g, it, a, gaps, h.own, basis, out)
			continue
		}
		out = append(out, it)
	}
	e := h.edgeItems(a)
	return append(out, e[0], e[1])
}

// layOutOwners lays out each subgrid in rows among items, at the width its columns give it,
// when some of the items sizing the rows stand on columns of a subgrid's own, so that their
// width is known.
func (t *tree) layOutOwners(items, rowItems []*gridItem, columns, rows []gridTrack, inner size,
	estimate trackEstimate) {
	if !slices.ContainsFunc(rowItems, func(it *gridItem) bool { return it.owner != nil }) {
		return
	}
	for _, it := range items {
		if !it.subgrid[vertical] {
			continue
		}
		area := it.gridAreaSize(vertical, rows, columns, inner, estimate)
		w := it.knownDimensions(area).w
		if !some(w) {
			w = mSub(area.w, it.marginSums(area.w).w)
		}
		t.probeChildLayout(it.node, size{w, none}, area, spaces{spaceOr(w, maxContentSpace), maxContentSpace}, inherentSize)
	}
}
