package layout

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// shared is the border one edge of a table's grid ends up with where borders collapse.
type shared struct {
	style  style.BorderStyle
	width  float32
	color  color.Color
	hidden bool
	// order is the place of the box it came from among the table's boxes, as Chrome
	// merges them: the cells in the document's order, then the rows, the row groups, the
	// columns, the column groups and the table. As good, the earlier wins.
	order int
}

// side is one of an element's four borders, as an edge of the grid would take it.
type side uint8

const (
	sideTop side = iota
	sideRight
	sideBottom
	sideLeft
)

// borderOf is n's border on side s, none of a border-style none: hidden is one that
// wins its edge.
func borderOf(n *node, s side) shared {
	if n == nil {
		return shared{}
	}
	e := styleBorder(n.st, s)
	if e.style == style.BorderNone && !e.hidden {
		return shared{}
	}
	return e
}

func styleBorder(st *style.Computed, s side) shared {
	switch s {
	case sideTop:
		return shared{st.BorderStyle.Top, st.Border.Top, st.BorderColor.Top, st.BorderHidden.Top, 0}
	case sideRight:
		return shared{st.BorderStyle.Right, st.Border.Right, st.BorderColor.Right, st.BorderHidden.Right, 0}
	case sideBottom:
		return shared{st.BorderStyle.Bottom, st.Border.Bottom, st.BorderColor.Bottom, st.BorderHidden.Bottom, 0}
	case sideLeft:
	}
	return shared{st.BorderStyle.Left, st.Border.Left, st.BorderColor.Left, st.BorderHidden.Left, 0}
}

// rank orders the border styles where two borders are as wide: double over solid over
// dashed over dotted over ridge, outset, groove and inset (CSS 2.1 §17.6.2.1).
func rank(s style.BorderStyle) int {
	switch s {
	case style.BorderDouble:
		return 8
	case style.BorderSolid:
		return 7
	case style.BorderDashed:
		return 6
	case style.BorderDotted:
		return 5
	case style.BorderRidge:
		return 4
	case style.BorderOutset:
		return 3
	case style.BorderGroove:
		return 2
	case style.BorderInset:
		return 1
	case style.BorderNone, style.BorderAuto:
	}
	return 0
}

// beats reports whether e wins the edge over the border held so far: a hidden one wins
// over all, then the wider, then the style; as good, the one from the earlier box.
func (e shared) beats(held shared) bool {
	switch {
	case held.hidden:
		return false
	case e.hidden:
		return true
	case e.width != held.width:
		return e.width > held.width
	case rank(e.style) != rank(held.style):
		return rank(e.style) > rank(held.style)
	}
	return held.style != style.BorderNone && e.order < held.order
}

// paints reports an edge that draws: there, not hidden, with a width and a style.
func (e shared) paints() bool { return e.width > 0 && e.style != style.BorderNone && !e.hidden }

// compare is which of two edges meeting at a corner paints over it, as Chrome's
// CompareForPaint: 1 for e, -1 for o, 0 when neither is better. An edge that paints
// beats one that doesn't, then the wider, the style, the earlier box.
func (e shared) compare(o shared) int {
	ep, op := e.paints(), o.paints()
	switch {
	case !ep && !op:
		return 0
	case !ep:
		return -1
	case !op:
		return 1
	case e.width != o.width:
		return sign(e.width > o.width)
	case rank(e.style) != rank(o.style):
		return sign(rank(e.style) > rank(o.style))
	case e.order != o.order:
		return sign(e.order < o.order)
	}
	return 0
}

func sign(win bool) int {
	if win {
		return 1
	}
	return -1
}

func resolved(candidates ...shared) shared {
	var won shared
	for _, c := range candidates {
		if c.beats(won) {
			won = c
		}
	}
	if won.hidden {
		return shared{hidden: true}
	}
	return won
}

// borderGrid is the borders of a table where they collapse: v the edge at the start of
// each column of each row, and one more at the end of the last; h the edge above each column of
// each row, and one more row below the last.
type borderGrid struct {
	rows, cols int
	v, h       [][]shared
}

// gridLine is where a row or a column stands in its group, for the borders of the group
// that reach the edge.
type gridLine struct {
	own, group  *node
	first, last bool
}

func rowLines(n *node) []gridLine {
	var lines []gridLine
	for _, k := range n.kids {
		switch k.role {
		case roleRow:
			lines = append(lines, gridLine{own: k})
		case roleGroup:
			for i, r := range k.kids {
				lines = append(lines, gridLine{own: r, group: k, first: i == 0, last: i == len(k.kids)-1})
			}
		case roleNone, roleTable, roleCaption, roleColumn, roleColumnGroup:
		}
	}
	return lines
}

func columnLines(tracks []track, cols int) []gridLine {
	lines := make([]gridLine, cols)
	for _, tr := range tracks {
		for k := tr.first; k < tr.first+tr.span && k < cols; k++ {
			l := gridLine{own: tr.col, group: tr.group}
			if g := tr.group; g != nil {
				l.first = k == tr.first && tr.col == g.kids[0]
				l.last = k == tr.first+tr.span-1 && tr.col == g.kids[len(g.kids)-1]
			}
			lines[k] = l
		}
	}
	return lines
}

// groupSide is the border of l's group on side s where l is the line of the group at
// that side, none otherwise.
func (l gridLine) groupSide(o orders, s side, at bool) shared {
	if l.group == nil || !at {
		return shared{}
	}
	return o.of(l.group, s)
}

// orders are the places of a table's boxes, as shared.order counts them.
type orders map[*node]int

func (o orders) of(n *node, s side) shared {
	e := borderOf(n, s)
	e.order = o[n]
	return e
}

// ordersOf numbers the boxes of table n, whose cells are cells and rows rows, in the
// order Chrome merges their borders: every column shares a number, and so does every
// column group.
func ordersOf(n *node, cells []slot, rows []*node, tracks []track) orders {
	o := make(orders, len(cells)+len(rows)+len(tracks)+1)
	for _, c := range cells {
		o[c.cell] = len(o)
	}
	for _, r := range rows {
		o[r] = len(o)
	}
	at := len(o)
	for _, k := range n.kids {
		if k.role == roleGroup {
			o[k] = at
			at++
		}
	}
	for _, tr := range tracks {
		if tr.col != nil {
			o[tr.col] = at
		}
		if tr.group != nil {
			o[tr.group] = at + 1
		}
	}
	o[n] = at + 2
	return o
}

// collapse resolves the borders of a table whose borders collapse (CSS 2.1 §17.6.2):
// each edge of the grid takes the border that wins it among those of the cells beside
// it, their rows, row groups, columns, column groups and the table. Each cell is given
// half of its edges for borders, the table half of its outer ones, and the grid is kept
// for the table to draw.
//
//nolint:gocognit,gocyclo,cyclop,funlen // the candidates of each edge, in the order CSS 2.1 §17.6.2.1 ranks them
func (t *tree) collapse(n *node, rows []*node, tracks []track) {
	cells, cols := gridSlots(n, rows, func(r *node, ci int) (int, int) {
		return cellSpan(r.kids[ci], "colspan", 1, 1000), cellSpan(r.kids[ci], "rowspan", 0, 65534)
	})
	if len(tracks) > 0 {
		last := tracks[len(tracks)-1]
		cols = max(cols, last.first+last.span)
	}
	g := &borderGrid{rows: len(rows), cols: cols, v: make([][]shared, len(rows)), h: make([][]shared, len(rows)+1)}
	at := make([][]*node, len(rows))
	for r := range at {
		at[r] = make([]*node, cols)
	}
	for _, c := range cells {
		for r := c.row; r < c.row+c.rs; r++ {
			for k := c.col; k < c.col+c.cs; k++ {
				at[r][k] = c.cell
			}
		}
	}
	cellAt := func(r, k int) *node {
		if r < 0 || r >= len(rows) || k < 0 || k >= cols {
			return nil
		}
		return at[r][k]
	}
	rowAt, colAt := rowLines(n), columnLines(tracks, cols)
	// The grid is the table's columns from its start: the start side of a cell, a row or
	// a column is its left, its right under direction rtl. The table itself is its mirror
	// image there, its start its left either way, as are the outer borders resolved.
	start, end := sideLeft, sideRight
	rtl := n.st.Dir == style.RTL
	if rtl {
		start, end = end, start
	}
	o := ordersOf(n, cells, rows, tracks)
	line := func(lines []gridLine, i int) gridLine {
		if i < 0 || i >= len(lines) {
			return gridLine{}
		}
		return lines[i]
	}
	for r := range rows {
		g.v[r] = make([]shared, cols+1)
		for k := range cols + 1 {
			before, after := cellAt(r, k-1), cellAt(r, k)
			if before != nil && before == after {
				continue
			}
			left, right := line(colAt, k-1), line(colAt, k)
			cands := []shared{o.of(before, end), o.of(after, start)}
			if k == 0 {
				cands = append(cands, o.of(rowAt[r].own, start), rowAt[r].groupSide(o, start, true))
			}
			if k == cols {
				cands = append(cands, o.of(rowAt[r].own, end), rowAt[r].groupSide(o, end, true))
			}
			cands = append(cands, o.of(left.own, end), o.of(right.own, start),
				left.groupSide(o, end, left.last), right.groupSide(o, start, right.first))
			if k == 0 {
				cands = append(cands, o.of(n, sideLeft))
			}
			if k == cols {
				cands = append(cands, o.of(n, sideRight))
			}
			g.v[r][k] = resolved(cands...)
		}
	}
	for r := range len(rows) + 1 {
		g.h[r] = make([]shared, cols)
		for k := range cols {
			before, after := cellAt(r-1, k), cellAt(r, k)
			if before != nil && before == after {
				continue
			}
			above, below := line(rowAt, r-1), line(rowAt, r)
			cands := []shared{
				o.of(before, sideBottom), o.of(after, sideTop),
				o.of(above.own, sideBottom), o.of(below.own, sideTop),
				above.groupSide(o, sideBottom, above.last), below.groupSide(o, sideTop, below.first),
			}
			if r == 0 {
				cands = append(cands, o.of(colAt[k].own, sideTop), colAt[k].groupSide(o, sideTop, true), o.of(n, sideTop))
			}
			if r == len(rows) {
				cands = append(cands, o.of(colAt[k].own, sideBottom), colAt[k].groupSide(o, sideBottom, true),
					o.of(n, sideBottom))
			}
			g.h[r][k] = resolved(cands...)
		}
	}
	n.shared = g
	var outer rect
	for r := range rows {
		outer.l, outer.r = max(outer.l, g.v[r][0].width/2), max(outer.r, g.v[r][cols].width/2)
	}
	for k := range cols {
		outer.t, outer.b = max(outer.t, g.h[0][k].width/2), max(outer.b, g.h[len(rows)][k].width/2)
	}
	if len(rows) == 0 || cols == 0 {
		outer = borders(n.st)
	}
	n.setCollapsed(outer)
	for _, c := range cells {
		var half rect
		for r := c.row; r < c.row+c.rs; r++ {
			half.l, half.r = max(half.l, g.v[r][c.col].width/2), max(half.r, g.v[r][c.col+c.cs].width/2)
		}
		for k := c.col; k < c.col+c.cs; k++ {
			half.t, half.b = max(half.t, g.h[c.row][k].width/2), max(half.b, g.h[c.row+c.rs][k].width/2)
		}
		if rtl {
			half.l, half.r = half.r, half.l
		}
		c.cell.setCollapsed(half)
	}
}

// separate gives the cells of a table whose borders are apart their own borders again,
// those kept from a layout that collapsed them.
func (n *node) separate() {
	for _, r := range n.rows() {
		for _, c := range r.kids {
			c.own()
		}
	}
}

func (n *node) own() {
	if n.collapsed != nil {
		n.collapsed = nil
		n.forget()
	}
}

// forget drops what was measured of n with other borders: its layouts, and its row's
// widths when it is a cell.
func (n *node) forget() {
	n.cache.clear()
	if n.cell && n.parent != nil {
		n.parent.row = nil
	}
}

// setCollapsed gives a table or a cell the borders it has where borders collapse, its
// layouts with others forgotten.
func (n *node) setCollapsed(b rect) {
	if n.collapsed == nil || *n.collapsed != b {
		n.collapsed = &b
		n.forget()
	}
}

// laid is the node's place as its parent laid it out, with the borders collapsing left
// it, and no padding for a table that collapses its own.
func (n *node) laid() placed {
	u := n.unrounded
	if n.collapsed != nil {
		u.border = *n.collapsed
		if n.shared != nil {
			u.padding = rect{}
		}
	}
	return u
}

// borders is the node's border widths: its style's, or what collapsing borders left it.
func (n *node) borders() rect {
	if n.collapsed != nil {
		return *n.collapsed
	}
	return borders(n.st)
}

// edges lists the grid's borders as the table draws them, in its box and in Chrome's
// order, row by row, each edge down before the one across from the same corner: colX and
// rowY are where the columns and the rows start, one more for the end. At each corner the
// edges meeting there are weighed as Chrome's ComputeEdgeJoints weighs them: an edge
// that wins its corner reaches through the way across it, one that loses stops short of
// it, and when two win, the later drawn shows.
func (g *borderGrid) edges(colX, rowY []float32) []box.Edge {
	if g.rows == 0 || g.cols == 0 {
		return nil
	}
	v := func(r, k int) shared {
		if r < 0 || r >= g.rows || k < 0 || k > g.cols {
			return shared{}
		}
		return g.v[r][k]
	}
	h := func(r, k int) shared {
		if r < 0 || r > g.rows || k < 0 || k >= g.cols {
			return shared{}
		}
		return g.h[r][k]
	}
	out := make([]box.Edge, 0, 2*g.rows*g.cols+g.rows+g.cols)
	for r := range g.rows + 1 {
		for k := range g.cols + 1 {
			if e := v(r, k); e.paints() {
				start, startWide := joint(h(r, k-1), h(r, k), v(r-1, k), e, false, false)
				end, endWide := joint(h(r+1, k-1), h(r+1, k), e, v(r+1, k), false, true)
				top, bottom := reach(start, startWide), reach(end, endWide)
				out = append(out, box.Edge{
					Rect: geom.Rect{
						X: colX[k] - e.width/2, Y: rowY[r] - top, W: e.width, H: rowY[r+1] - rowY[r] + top + bottom,
					},
					Style: e.style, Color: e.color,
				})
			}
			if e := h(r, k); e.paints() {
				start, startWide := joint(h(r, k-1), e, v(r-1, k), v(r, k), true, false)
				end, endWide := joint(e, h(r, k+1), v(r-1, k+1), v(r, k+1), true, true)
				left, right := reach(start, startWide), reach(end, endWide)
				out = append(out, box.Edge{
					Rect: geom.Rect{
						X: colX[k] - left, Y: rowY[r] - e.width/2, W: colX[k+1] - colX[k] + left + right, H: e.width,
					},
					Style: e.style, Color: e.color, Across: true,
				})
			}
		}
	}
	return out
}

// joint weighs a corner of the grid for one of the edges meeting there: before and after
// the edges across, left and right of it, over and under those down, above and below.
// It reports whether the edge, across when across is set, wins the corner from its start
// or, when end is set, its end, and how wide the corner is along it: the width of the
// winner of the other way.
func joint(before, after, over, under shared, across, end bool) (bool, float32) {
	acrossCmp, downCmp := before.compare(after), over.compare(under)
	acrossWinner, downWinner := after, under
	if acrossCmp == 1 {
		acrossWinner = before
	}
	if downCmp == 1 {
		downWinner = over
	}
	ways := acrossWinner.compare(downWinner)
	own := acrossCmp
	if !across {
		own, ways = downCmp, -ways
	}
	if end {
		own = -own
	}
	if across {
		return ways != -1 && own != 1, downWinner.width
	}
	return ways != -1 && own != 1, acrossWinner.width
}

// reach is how far an edge goes past a corner as wide as wide: through it when it wins,
// else short of it.
func reach(wins bool, wide float32) float32 {
	if wins {
		return wide / 2
	}
	return -wide / 2
}
