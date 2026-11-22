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

// subgridAxes reports the axes n takes its grid's tracks in: a grid that is an in-flow item of
// a grid, no query container, with subgrid as that axis's template.
func (n *node) subgridAxes() [2]bool {
	st := n.st
	switch {
	case !st.Subgrid.Width && !st.Subgrid.Height, n.parent == nil, n.parent.context() != contextGrid,
		st.Display != style.GridDisplay && st.Display != style.InlineGrid, outOfFlow(st.Position),
		st.ContainerType != style.ContainerNormal, n.context() != contextGrid && (len(n.kids) > 0 || n.inline != nil):
		return [2]bool{}
	}
	return [2]bool{st.Subgrid.Width, st.Subgrid.Height}
}

// handed is what n's grid handed it in each axis it is a subgrid in, nil in the others.
func (n *node) handed() [2]*subgridAxis {
	var out [2]*subgridAxis
	for a, on := range n.subgridAxes() {
		if !on {
			continue
		}
		out[a] = n.subgrid[a]
		if out[a] == nil {
			out[a] = &subgridAxis{span: 1}
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

// subgridEdges are the margin, border and padding of a subgrid's start and end in a.
func subgridEdges(n *node, a axis, basis float32) span {
	return edgesAlong(edges(n.st.Margin, basis).add(edges(n.st.Padding, basis)).add(n.borders()), a)
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
		content := subgridEdges(n, a, basis)
		return subgridTracks(s, span{content.start, s.tracks[len(s.tracks)-1].end - content.end}, gap)
	}
	return initializeGridTracks(p.counts[a], n.st, a, p.reps[a], func(i int) bool { return p.occupancy.occupied(a, i) })
}

// sizingItems are the items whose contributions size a grid's tracks of a: its own, but each
// subgrid in a gives way to the items it holds, hoisted to the grid's lines.
func (t *tree) sizingItems(items []*gridItem, a axis, gap, basis float32) []*gridItem {
	var out []*gridItem
	for _, it := range items {
		if it.subgrid[a] {
			out = t.hoist(it, a, gap, gap, basis, out)
			continue
		}
		out = append(out, it)
	}
	return out
}

// hoist appends the items that size the tracks of a in place of the subgrid s, an item at
// lines of the grid sizing them: its own at their lines there, those of its subgrids in a in
// turn, and an empty item on each of its edge tracks. Each item takes as more margin the
// margin, border and padding of the subgrids whose edge it stands on, and beside a gutter
// half what the subgrid's gap differs from the sizing grid's.
func (t *tree) hoist(s *gridItem, a axis, gap, parentGap, basis float32, out []*gridItem) []*gridItem {
	n := s.node
	own := parentGap
	if _, _, g := templateOf(n.st, a); g.Kind != style.Auto {
		own = resolveOrZero(g, basis)
	}
	half := (own - gap) / 2
	edge := subgridEdges(n, a, 0)
	edge.start += s.extra[a].start
	edge.end += s.extra[a].end
	at := s.placement(a)
	o := a.other()
	p := t.planGrid(n, noSize, noSize)
	for _, it := range p.items {
		local := it.placement(a)
		extra := span{half, half}
		if local.start == 0 {
			extra.start = edge.start
		}
		if local.end == at.span() {
			extra.end = edge.end
		}
		it.extra[a] = extra
		it.setPlacement(a, lineSpan{at.start + local.start, at.start + local.end})
		l := it.placement(o)
		switch {
		case !s.subgrid[o]:
			it.owner, it.local = n, l
			it.setPlacement(o, s.placement(o))
			it.loose[o] = true
		case s.loose[o]:
			it.owner, it.local = s.owner, lineSpan{s.local.start + l.start, s.local.start + l.end}
			it.setPlacement(o, s.placement(o))
			it.loose[o] = true
		default:
			it.setPlacement(o, lineSpan{s.placement(o).start + l.start, s.placement(o).start + l.end})
		}
		if it.alignSelf == style.AlignBaseline {
			it.alignSelf = style.AlignStart
		}
		if it.subgrid[a] {
			out = t.hoist(it, a, gap, own, basis, out)
			continue
		}
		out = append(out, it)
	}
	for _, e := range [2]bool{true, false} {
		edgeItem := &gridItem{st: &style.Computed{}, baseline: none, minContentCache: noSize, minimumCache: noSize,
			maxContentCache: noSize, loose: [2]bool{a == vertical, a == horizontal}}
		edgeItem.setPlacement(o, s.placement(o))
		if e {
			edgeItem.setPlacement(a, lineSpan{at.start, at.start + 1})
			edgeItem.extra[a].start = edge.start
		} else {
			edgeItem.setPlacement(a, lineSpan{at.end - 1, at.end})
			edgeItem.extra[a].end = edge.end
		}
		out = append(out, edgeItem)
	}
	return out
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
