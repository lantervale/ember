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
		return shared{st.BorderStyle.Top, st.Border.Top, st.BorderColor.Top, st.BorderHidden.Top}
	case sideRight:
		return shared{st.BorderStyle.Right, st.Border.Right, st.BorderColor.Right, st.BorderHidden.Right}
	case sideBottom:
		return shared{st.BorderStyle.Bottom, st.Border.Bottom, st.BorderColor.Bottom, st.BorderHidden.Bottom}
	case sideLeft:
	}
	return shared{st.BorderStyle.Left, st.Border.Left, st.BorderColor.Left, st.BorderHidden.Left}
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
	case style.BorderNone:
	}
	return 0
}

// beats reports whether e wins the edge over the border held so far: a hidden one wins
// over all, then the wider, then the style; as good, the one held stays, the borders
// being offered from the cell out to the table and from the left and the top.
func (e shared) beats(held shared) bool {
	switch {
	case held.hidden:
		return false
	case e.hidden:
		return true
	case e.width != held.width:
		return e.width > held.width
	}
	return rank(e.style) > rank(held.style)
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
func (l gridLine) groupSide(s side, at bool) shared {
	if l.group == nil || !at {
		return shared{}
	}
	return borderOf(l.group, s)
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
			cands := []shared{borderOf(before, end), borderOf(after, start)}
			if k == 0 {
				cands = append(cands, borderOf(rowAt[r].own, start), rowAt[r].groupSide(start, true))
			}
			if k == cols {
				cands = append(cands, borderOf(rowAt[r].own, end), rowAt[r].groupSide(end, true))
			}
			cands = append(cands, borderOf(left.own, end), borderOf(right.own, start),
				left.groupSide(end, left.last), right.groupSide(start, right.first))
			if k == 0 {
				cands = append(cands, borderOf(n, sideLeft))
			}
			if k == cols {
				cands = append(cands, borderOf(n, sideRight))
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
				borderOf(before, sideBottom), borderOf(after, sideTop),
				borderOf(above.own, sideBottom), borderOf(below.own, sideTop),
				above.groupSide(sideBottom, above.last), below.groupSide(sideTop, below.first),
			}
			if r == 0 {
				cands = append(cands, borderOf(colAt[k].own, sideTop), colAt[k].groupSide(sideTop, true), borderOf(n, sideTop))
			}
			if r == len(rows) {
				cands = append(cands, borderOf(colAt[k].own, sideBottom), colAt[k].groupSide(sideBottom, true),
					borderOf(n, sideBottom))
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

// edges lists the grid's borders as the table draws them, in its box: colX and rowY
// are where the columns and the rows start, one more for the end. Where borders cross,
// the wider way goes through and the other stops at its sides; as wide, the one across,
// and a double down too, so that their lines cross as Chrome draws them.
func (g *borderGrid) edges(colX, rowY []float32) []box.Edge {
	if g.rows == 0 || g.cols == 0 {
		return nil
	}
	out := make([]box.Edge, 0, 2*g.rows*g.cols+g.rows+g.cols)
	width := func(grid [][]shared, r, k int) float32 {
		if r < 0 || r >= len(grid) || k < 0 || k >= len(grid[r]) {
			return 0
		}
		return grid[r][k].width
	}
	across := func(r, k int) float32 { return max(width(g.h, r, k-1), width(g.h, r, k)) }
	down := func(r, k int) float32 { return max(width(g.v, r-1, k), width(g.v, r, k)) }
	reach := func(through bool, crossing float32) float32 {
		if through {
			return crossing / 2
		}
		return -crossing / 2
	}
	for r := range g.v {
		for k, e := range g.v[r] {
			if e.width <= 0 {
				continue
			}
			double := e.style == style.BorderDouble
			top := reach(down(r, k) > across(r, k) || double && e.width >= across(r, k), across(r, k))
			bottom := reach(down(r+1, k) > across(r+1, k) || double && e.width >= across(r+1, k), across(r+1, k))
			out = append(out, box.Edge{
				Rect: geom.Rect{
					X: colX[k] - e.width/2, Y: rowY[r] - top, W: e.width, H: rowY[r+1] - rowY[r] + top + bottom,
				},
				Style: e.style, Color: e.color,
			})
		}
	}
	for r := range g.h {
		for k, e := range g.h[r] {
			if e.width <= 0 {
				continue
			}
			left := reach(across(r, k) >= down(r, k), down(r, k))
			right := reach(across(r, k+1) >= down(r, k+1), down(r, k+1))
			out = append(out, box.Edge{
				Rect: geom.Rect{
					X: colX[k] - left, Y: rowY[r] - e.width/2, W: colX[k+1] - colX[k] + left + right, H: e.width,
				},
				Style: e.style, Color: e.color, Across: true,
			})
		}
	}
	return out
}
