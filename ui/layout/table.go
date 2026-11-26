package layout

import (
	"cmp"
	"slices"
	"strconv"

	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// tableRole marks the nodes of a table: the table, its row groups, its rows, its columns
// and their groups; its cells are ordinary block containers.
type tableRole uint8

const (
	roleNone tableRole = iota
	roleTable
	roleGroup
	roleRow
	roleCaption
	roleColumn
	roleColumnGroup
)

// isTable reports an element laid out as a table: display table or inline-table.
func isTable(e *dom.Node) bool {
	if e.IsText() {
		return false
	}
	st := styleOf(e)
	return st.Table && (st.Display == style.Block || st.Display == style.InlineBlock)
}

// partOf is the part of a table e is shown as, none for a text, an element not shown, a
// control and a replaced element.
func partOf(e *dom.Node) style.TablePart {
	if e.IsText() {
		return style.NoTablePart
	}
	st := styleOf(e)
	if st.Display == style.DisplayNone || controlOf(e) != noControl {
		return style.NoTablePart
	}
	switch e.Tag() { //nolint:exhaustive // the replaced elements, never a table's parts
	case dom.Img, dom.Svg, dom.Canvas, dom.Iframe:
		return style.NoTablePart
	}
	if pictured(e) {
		return style.NoTablePart
	}
	return st.TablePart
}

func shown(e *dom.Node) bool { return !e.IsText() && styleOf(e).Display != style.DisplayNone }

// flat lists children with those of display: contents in their place, noted on owner.
func (t *tree) flat(owner *node, children []*dom.Node) []*dom.Node {
	items := t.flatten(owner, children)
	out := make([]*dom.Node, 0, len(items))
	for _, it := range items {
		if it.text != nil {
			out = append(out, it.text)
		} else {
			out = append(out, it.element)
		}
	}
	return out
}

// loose gathers the children of a table part that are none of its own, for the
// anonymous part CSS wraps them in (CSS 2 §17.2.1).
type loose struct {
	run []*dom.Node
}

// add puts c in the run, unless it is not shown or white space before anything else.
func (l *loose) add(c *dom.Node) {
	if c.IsText() && len(l.run) == 0 && isCollapsibleSpace(c.Text()) || !c.IsText() && !shown(c) {
		return
	}
	l.run = append(l.run, c)
}

// flush hands the run to wrap, if there is one.
func (l *loose) flush(wrap func([]*dom.Node)) {
	if len(l.run) > 0 {
		wrap(l.run)
		l.run = nil
	}
}

// buildTable gives a table its captions, its row groups and rows, and each row its
// cells, from children. What is no part of a table where it stands is wrapped in the
// anonymous row or cell it misses. The columns and their groups go first, under the rows
// that are drawn over them, then the row groups as Chrome draws them: the first header
// group on top, the bodies and loose rows, then the first footer group.
func (t *tree) buildTable(n *node, children []*dom.Node) {
	n.role = roleTable
	var head, foot *node
	var body, columns []*node
	var rest loose
	wrap := func(run []*dom.Node) {
		body = append(body, t.anonymousRow(n, run))
		n.anonymous = true
	}
	for _, c := range t.flat(n, children) {
		switch partOf(c) {
		case style.TableCaption:
			rest.flush(wrap)
			cn := t.build(c, n)
			cn.role = roleCaption
			n.kids = append(n.kids, cn)
		case style.TableHeaderGroup:
			rest.flush(wrap)
			if g := t.buildGroup(c, n); head == nil {
				head = g
			} else {
				body = append(body, g)
			}
		case style.TableFooterGroup:
			rest.flush(wrap)
			if g := t.buildGroup(c, n); foot == nil {
				foot = g
			} else {
				body = append(body, g)
			}
		case style.TableRowGroup:
			rest.flush(wrap)
			body = append(body, t.buildGroup(c, n))
		case style.TableRow:
			rest.flush(wrap)
			body = append(body, t.buildRow(c, n))
		case style.TableColumn:
			rest.flush(wrap)
			columns = append(columns, t.leafBox(c, n, roleColumn))
		case style.TableColumnGroup:
			rest.flush(wrap)
			g := t.leafBox(c, n, roleColumnGroup)
			for _, k := range t.flat(g, c.Flow()) {
				if partOf(k) == style.TableColumn {
					g.kids = append(g.kids, t.leafBox(k, g, roleColumn))
				}
			}
			columns = append(columns, g)
		case style.NoTablePart, style.TableCell:
			rest.add(c)
		}
	}
	rest.flush(wrap)
	n.kids = append(n.kids, columns...)
	if head != nil {
		n.kids = append(n.kids, head)
	}
	n.kids = append(n.kids, body...)
	if foot != nil {
		n.kids = append(n.kids, foot)
	}
	if !n.st.BorderCollapse {
		n.separate()
	}
}

// buildGroup is the node of the row group c with its rows.
func (t *tree) buildGroup(c *dom.Node, parent *node) *node {
	if k := t.kept(c, parent); k != nil {
		t.adopt(k, parent)
		return k
	}
	g := t.leafBox(c, parent, roleGroup)
	var rest loose
	wrap := func(run []*dom.Node) {
		g.kids = append(g.kids, t.anonymousRow(g, run))
		g.anonymous = true
	}
	for _, r := range t.flat(g, c.Flow()) {
		if partOf(r) == style.TableRow {
			rest.flush(wrap)
			g.kids = append(g.kids, t.buildRow(r, g))
			continue
		}
		rest.add(r)
	}
	rest.flush(wrap)
	return g
}

// buildRow is the node of the row r with its cells.
func (t *tree) buildRow(r *dom.Node, parent *node) *node {
	if k := t.kept(r, parent); k != nil {
		t.adopt(k, parent)
		return k
	}
	rn := t.leafBox(r, parent, roleRow)
	t.buildCells(rn, t.flat(rn, r.Flow()))
	return rn
}

// buildCells gives the row rn its cells from children, those that are none wrapped in
// anonymous cells.
func (t *tree) buildCells(rn *node, children []*dom.Node) {
	var rest loose
	wrap := func(run []*dom.Node) {
		cn := t.anonymous(rn)
		cn.cell = true
		t.buildChildren(cn, run)
		rn.kids = append(rn.kids, cn)
		rn.anonymous = true
	}
	for _, c := range children {
		if partOf(c) == style.TableCell {
			rest.flush(wrap)
			cn := t.build(c, rn)
			cn.cell = true
			rn.kids = append(rn.kids, cn)
			continue
		}
		rest.add(c)
	}
	rest.flush(wrap)
}

// anonymousRow is the row a table or a row group wraps round children that are none.
func (t *tree) anonymousRow(parent *node, children []*dom.Node) *node {
	rn := t.anonymous(parent)
	rn.role = roleRow
	t.buildCells(rn, children)
	return rn
}

// anonymousTable is the table a container wraps round the table parts among its
// children that stand in none: border-spacing and all the rest at their initial values.
func (t *tree) anonymousTable(parent *node, parts []*dom.Node) *node {
	st := t.anonymousTables[parent.st]
	if st == nil {
		s := style.Initial()
		s.Display, s.Table = style.Block, true
		s.Text = parent.st.Text
		st = &s
		if t.anonymousTables == nil {
			t.anonymousTables = map[*style.Computed]*style.Computed{}
		}
		t.anonymousTables[parent.st] = st
	}
	n := &node{box: &box.Box{Kind: box.AnonymousBlock, Style: st}, st: st, parent: parent, fresh: true}
	t.buildTable(n, parts)
	return n
}

// leafBox is the node of a row group or row, whose children the table places itself.
func (t *tree) leafBox(e *dom.Node, parent *node, role tableRole) *node {
	n := t.element(e, parent)
	n.role = role
	return n
}

// rows lists the table's rows in order, through its row groups.
func (n *node) rows() []*node {
	var rows []*node
	for _, k := range n.kids {
		switch k.role {
		case roleGroup:
			rows = append(rows, k.kids...)
		case roleRow:
			rows = append(rows, k)
		case roleNone, roleTable, roleCaption, roleColumn, roleColumnGroup:
		}
	}
	return rows
}

// captions are the table's captions, in their order.
func (n *node) captions() []*node {
	var out []*node
	for _, k := range n.kids {
		if k.role == roleCaption {
			out = append(out, k)
		}
	}
	return out
}

// captionMin is the width the table's captions need at least, margins included.
func (t *tree) captionMin(n *node) float32 {
	w := float32(0)
	for _, c := range n.captions() {
		m := edges(c.st.Margin, 0)
		w = max(w,
			t.measureChildSize(c, noSize, noSize, spaces{minContentSpace, maxContentSpace},
				inherentSize, horizontal, [2]bool{})+m.horizontal())
	}
	return w
}

// captionHeights are the heights the captions above the table and those below take at
// its width w, margins included.
func (t *tree) captionHeights(n *node, w float32) (float32, float32) {
	var above, below float32
	for _, c := range n.captions() {
		m := edges(c.st.Margin, w)
		h := t.measureChildSize(c, size{w - m.horizontal(), none}, size{w, none},
			spaces{def(w - m.horizontal()), maxContentSpace}, inherentSize, vertical, [2]bool{}) + m.vertical()
		if c.st.CaptionSide == style.CaptionBottom {
			below += h
		} else {
			above += h
		}
	}
	return above, below
}

// placeCaptions lays the captions out at the table's width w: those above from its top,
// those below from y, under the grid.
func (t *tree) placeCaptions(n *node, w, y float32) {
	top := float32(0)
	for _, c := range n.captions() {
		m := edges(c.st.Margin, w)
		out := t.performChildLayout(c, size{w - m.horizontal(), none}, size{w, none},
			spaces{def(w - m.horizontal()), maxContentSpace}, inherentSize)
		at := &top
		if c.st.CaptionSide == style.CaptionBottom {
			at = &y
		}
		c.unrounded = placed{
			location: point{m.l, *at + m.t},
			size:     out.size,
			overflow: out.overflow,
			padding:  edges(c.st.Padding, w),
			border:   c.borders(),
			margin:   m,
		}
		*at += out.size.h + m.vertical()
	}
}

// table lays out a table by CSS's automatic table layout, as Chrome does for the plain
// case: columns from their cells' min- and max-content widths, extra width shared, rows
// as tall as their tallest cell, border-spacing around and between the cells.
//
//nolint:funlen // Chrome's automatic table layout in one pass, as its doc walks it
func (t *tree) table(n *node, in input) output {
	st := n.st
	padding := edges(st.Padding, in.parent.w)
	border := borders(st)
	inset := padding.add(border)
	rows := n.rows()
	if fixedLayout(st, in) || len(n.tracks()) > 0 || st.BorderCollapse ||
		slices.ContainsFunc(rows, func(r *node) bool { return t.rowWidths(r).general || r.collapsedRow() }) {
		return t.spanned(n, in, rows)
	}
	sx, sy := st.BorderSpacing[0], st.BorderSpacing[1]
	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r.kids))
	}
	colMin, colMax := make([]float32, cols), make([]float32, cols)
	fixed := make([]bool, cols)
	percent := make([]float32, cols)
	for _, r := range rows {
		tr := t.rowWidths(r)
		for i := range r.kids {
			colMin[i], colMax[i] = max(colMin[i], tr.lo[i]), max(colMax[i], tr.hi[i])
			fixed[i] = fixed[i] || tr.fixed[i]
			percent[i] = max(percent[i], tr.percent[i])
		}
	}
	percent = heldPercents(percent)
	sumMin, sumMax := float32(0), float32(0)
	for i := range cols {
		sumMin += colMin[i]
		sumMax += colMax[i]
	}
	spacingW := sx * float32(cols+1)
	if cols == 0 {
		spacingW = 0
	}
	extra := spacingW + inset.horizontal()

	w := in.known.w
	if !some(w) {
		w = resolve(st.Size.Width, in.parent.w)
		if some(w) && contentBox(st) {
			w += inset.horizontal()
		}
	}
	lo, hi := tableLimits(st, in, inset)
	w = mClamp(or(w, autoTableWidth(in.avail.w, sumMin, percentMax(colMax, percent, sumMax), extra)), lo.w, hi.w)
	w = max(w, sumMin+extra, t.captionMin(n))
	if in.run == computeSize && in.axis == axisHorizontal {
		return sizeOutput(size{w, 0})
	}

	widths := percentWidths(colMin, colMax, fixed, percent, sumMin, sumMax, w-extra)
	content := w - inset.horizontal()
	heights := make([]float32, len(rows))
	for ri, r := range rows {
		heights[ri] = t.rowHeight(r, widths, content)
	}
	above, below := t.captionHeights(n, w)
	hgt := tableHeight(st, in, heights, sy, inset.vertical()+above+below, lo.h, hi.h)
	s := size{w, hgt}
	if in.run == computeSize {
		return sizeOutput(s)
	}

	rowW := float32(0)
	for _, cw := range widths {
		rowW += cw
	}
	if cols > 1 {
		rowW += sx * float32(cols-1)
	}
	var candidates []oofCandidate
	y := above + inset.t + sy
	ri := 0
	first := none
	placeRow := func(r *node, origin point) {
		h := heights[ri]
		if ri == 0 {
			first = contentBottom(r, y+h, content)
		}
		ri++
		r.unrounded = placed{location: point{inset.l + sx - origin.x, y - origin.y}, size: size{rowW, h}}
		tr := r.row
		if tr.placedAt(widths, content, h) {
			y += h + sy
			return
		}
		from := len(candidates)
		x := float32(0)
		for i, c := range r.kids {
			c.cut = false
			out := t.performChildLayout(c, size{widths[i], h}, size{content, none},
				spaces{def(widths[i]), def(h)}, inherentSize)
			c.unrounded = placed{
				location: point{x, 0},
				size:     out.size,
				overflow: out.overflow,
				padding:  edges(c.st.Padding, content),
				border:   c.borders(),
			}
			candidates = append(candidates, translateCandidates(out.oof, point{inset.l + sx + x, y})...)
			x += widths[i] + sx
		}
		tr.place(widths, content, h, len(candidates) == from)
		y += h + sy
	}
	for _, k := range n.kids {
		switch k.role {
		case roleRow:
			placeRow(k, point{})
		case roleGroup:
			top := y
			for _, r := range k.kids {
				placeRow(r, point{inset.l + sx, top})
			}
			bottom := y - sy
			if len(k.kids) == 0 {
				bottom = top
			}
			k.unrounded = placed{location: point{inset.l + sx, top}, size: size{rowW, max(bottom-top, 0)}}
		case roleNone, roleTable, roleCaption, roleColumn, roleColumnGroup:
		}
	}
	t.placeCaptions(n, w, hgt-below)
	n.captionRoom = [2]float32{above, below}
	padded := rect{r: s.w - border.horizontal(), b: s.h - border.vertical()}
	return output{
		size:          s,
		overflow:      finalizeOverflow(padded, s.sub(border.sum()), [2]bool{}, false),
		firstBaseline: first,
		lastBaseline:  none,
		oof:           candidates,
		area:          &oofArea{size: s.sub(border.sum()), offset: point{border.l, border.t}},
	}
}

// contentBottom is the baseline of row r, whose cells stand on none and end at bottom:
// its lowest cell's content box's bottom, as Chrome's; none without cells.
func contentBottom(r *node, bottom, content float32) float32 {
	b := none
	for _, c := range r.kids {
		b = fmax(or(b, 0), bottom-edges(c.st.Padding, content).b-c.borders().b)
	}
	return b
}

// tableRow is what a table keeps of a row across its passes, and across layouts while the
// row is kept: its cells' min- and max-content widths, its height at the column widths
// it was measured at, and the column widths and height its cells were last placed at.
type tableRow struct {
	lo, hi []float32
	fixed  []bool
	// percent is each cell's width where it is a percentage of the table's, else 0.
	percent []float32
	// colspan and rowspan are each cell's, spans whether any is more than one.
	colspan, rowspan []int
	spans            bool
	// general is set when the row needs the table's general layout: a cell that spans,
	// or one on the row's baseline.
	general bool

	tall    bool
	widths  []float32
	content float32
	height  float32

	placed        bool
	placedWidths  []float32
	placedContent float32
	placedHeight  float32
}

// rowWidths is r's table row, its cells measured once.
func (t *tree) rowWidths(r *node) *tableRow {
	if r.row != nil {
		return r.row
	}
	tr := &tableRow{lo: make([]float32, len(r.kids)), hi: make([]float32, len(r.kids)), fixed: make([]bool, len(r.kids)),
		percent: make([]float32, len(r.kids)), colspan: make([]int, len(r.kids)), rowspan: make([]int, len(r.kids))}
	for i, c := range r.kids {
		tr.colspan[i], tr.rowspan[i] = cellSpan(c, "colspan", 1, 1000), cellSpan(c, "rowspan", 0, 65534)
		tr.spans = tr.spans || tr.colspan[i] != 1 || tr.rowspan[i] != 1
		tr.general = tr.general || tr.spans || onBaseline(c)
		lo := t.measureChildSize(c, noSize, noSize,
			spaces{minContentSpace, maxContentSpace}, inherentSize, horizontal, [2]bool{})
		hi := t.measureChildSize(c, noSize, noSize, maxContentSpaces, inherentSize, horizontal, [2]bool{})
		if w := resolve(c.st.Size.Width, none); some(w) {
			if contentBox(c.st) {
				w += edges(c.st.Padding, 0).add(c.borders()).horizontal()
			}
			lo, hi = max(lo, w), max(lo, w)
			tr.fixed[i] = true
		} else if wl := c.st.Size.Width; wl.Kind == style.Fixed && wl.Px == 0 && wl.Percent > 0 {
			tr.percent[i] = wl.Percent
		}
		tr.lo[i], tr.hi[i] = lo, hi
	}
	r.row = tr
	return tr
}

// rowHeight is r's height with its cells at widths in a table of content width, measured
// again only when those changed.
func (t *tree) rowHeight(r *node, widths []float32, content float32) float32 {
	tr := t.rowWidths(r)
	if tr.tall && tr.content == content && slices.Equal(tr.widths, widths) {
		return tr.height
	}
	h := or(resolve(r.st.Size.Height, none), 0)
	for i, c := range r.kids {
		h = max(h,
			t.measureChildSize(c, size{widths[i], none}, size{content, none},
				spaces{def(widths[i]), maxContentSpace}, inherentSize, vertical, [2]bool{}))
	}
	tr.tall, tr.widths, tr.content, tr.height = true, append(tr.widths[:0], widths...), content, h
	return h
}

// placedAt reports whether the row's cells stand placed at widths and height already,
// with nothing out of flow among them.
func (tr *tableRow) placedAt(widths []float32, content, h float32) bool {
	return tr.placed && tr.placedContent == content && tr.placedHeight == h && slices.Equal(tr.placedWidths, widths)
}

// place notes the widths and height the row's cells were placed at, worth keeping when
// none of them surfaced anything out of flow.
func (tr *tableRow) place(widths []float32, content, h float32, keep bool) {
	tr.placed, tr.placedContent, tr.placedHeight = keep, content, h
	tr.placedWidths = append(tr.placedWidths[:0], widths...)
}

// columnWidths shares a table's content width out: each column its max-content and the
// rest in proportion to it when there is room, to the columns without a width of their
// own (fixed) while there are any, the space above the min-contents in
// proportion to what each can still grow when there isn't, the min-contents at least.
func columnWidths(colMin, colMax []float32, fixed []bool, sumMin, sumMax, content float32) []float32 {
	widths := make([]float32, len(colMin))
	switch {
	case content >= sumMax:
		grows := func(int) bool { return true }
		if slices.Contains(fixed, true) && slices.Contains(fixed, false) {
			grows = func(i int) bool { return !fixed[i] }
		}
		n, growMax := 0, float32(0)
		for i := range widths {
			if grows(i) {
				n++
				growMax += colMax[i]
			}
		}
		for i := range widths {
			widths[i] = colMax[i]
			switch {
			case !grows(i):
			case growMax > 0:
				widths[i] += (content - sumMax) * colMax[i] / growMax
			default:
				widths[i] += (content - sumMax) / float32(n)
			}
		}
	case content <= sumMin || sumMax == sumMin:
		copy(widths, colMin)
	default:
		f := (content - sumMin) / (sumMax - sumMin)
		for i := range widths {
			widths[i] = colMin[i] + (colMax[i]-colMin[i])*f
		}
	}
	return widths
}

// tableMax is the width a table whose percentages nothing can satisfy asks for, as
// Chrome's.
const tableMax = 1000000

// heldPercents are the columns' percentages held to 100 in all, the first columns
// served first, as Chrome holds them.
func heldPercents(percent []float32) []float32 {
	left := float32(100)
	for i, p := range percent {
		percent[i] = min(p, left)
		left -= percent[i]
	}
	return percent
}

// percentMax is the max-content width of a table's columns, sumMax, once those with a
// percentage are counted: the width at which each holds its max-content as its share,
// and the others theirs in what the percentages leave.
func percentMax(colMax, percent []float32, sumMax float32) float32 {
	total, rest := float32(0), float32(0)
	for i, p := range percent {
		total += p
		switch {
		case p > 0:
			sumMax = max(sumMax, colMax[i]*100/p)
		default:
			rest += colMax[i]
		}
	}
	switch {
	case total == 0 || rest == 0:
		return sumMax
	case total >= 100:
		return tableMax
	}
	return max(sumMax, rest*100/(100-total))
}

// percentWidths shares a table's content width out when columns have a percentage of
// it: those take their share, their min-content at least, and the others share the rest
// as columnWidths does; where the shares leave less than the others' min-contents, the
// columns with a percentage give way down to their own, and with no other column they
// share everything by their percentages.
func percentWidths(
	colMin, colMax []float32, fixed []bool, percent []float32, sumMin, sumMax, content float32,
) []float32 {
	if !slices.ContainsFunc(percent, func(p float32) bool { return p > 0 }) {
		return columnWidths(colMin, colMax, fixed, sumMin, sumMax, content)
	}
	widths := make([]float32, len(colMin))
	var restMin, restMax, shares, sharesMin, total float32
	var restLo, restHi []float32
	var restFixed []bool
	for i, p := range percent {
		if p > 0 {
			widths[i] = max(colMin[i], p*content/100)
			shares += widths[i]
			sharesMin += colMin[i]
			total += p
			continue
		}
		restMin += colMin[i]
		restMax += colMax[i]
		restLo, restHi, restFixed = append(restLo, colMin[i]), append(restHi, colMax[i]), append(restFixed, fixed[i])
	}
	switch {
	case shares+restMin > content:
		f := float32(0)
		if shares > sharesMin {
			f = max(content-restMin-sharesMin, 0) / (shares - sharesMin)
		}
		for i, p := range percent {
			if p > 0 {
				widths[i] = colMin[i] + (widths[i]-colMin[i])*f
			} else {
				widths[i] = colMin[i]
			}
		}
	case len(restLo) == 0:
		for i, p := range percent {
			widths[i] += (content - shares) * p / total
		}
	default:
		rest := columnWidths(restLo, restHi, restFixed, restMin, restMax, content-shares)
		at := 0
		for i, p := range percent {
			if p == 0 {
				widths[i] = rest[at]
				at++
			}
		}
	}
	return widths
}

// track is a column or a column group without columns, and the table's columns it
// stands for: span of them from first. group is the column's group, nil outside one.
type track struct {
	col, group  *node
	first, span int
}

// tracks lists the table's columns in order, through its column groups.
func (n *node) tracks() []track {
	var out []track
	at := 0
	add := func(c, g *node) {
		span := cellSpan(c, "span", 1, 1000)
		out = append(out, track{col: c, group: g, first: at, span: span})
		at += span
	}
	for _, k := range n.kids {
		switch k.role {
		case roleColumn:
			add(k, nil)
		case roleColumnGroup:
			if len(k.kids) == 0 {
				add(k, nil)
			}
			for _, c := range k.kids {
				add(c, k)
			}
		case roleNone, roleTable, roleCaption, roleGroup, roleRow:
		}
	}
	return out
}

// width is the width the track's column asks for, its group's when it has none: in
// points, or a percentage of the table's.
func (tr track) width() (float32, float32) {
	for _, c := range []*node{tr.col, tr.group} {
		if c == nil {
			continue
		}
		lo := resolve(c.st.MinSize.Width, none)
		if w := or(resolve(c.st.Size.Width, none), lo); some(w) {
			return mClamp(w, lo, resolve(c.st.MaxSize.Width, none)), 0
		}
		if wl := c.st.Size.Width; wl.Kind == style.Fixed && wl.Px == 0 && wl.Percent > 0 {
			return none, wl.Percent
		}
	}
	return none, 0
}

// cellSpan is a cell's colspan or rowspan attribute, 1 when missing or not a number,
// held to [lo, hi].
func cellSpan(c *node, name string, lo, hi int) int {
	if c.box.Node == nil {
		return 1
	}
	v, ok := c.box.Node.Attr(name)
	if !ok {
		return 1
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 1
	}
	return min(max(n, lo), hi)
}

// fixedLayout reports a table laid out by table-layout: fixed, which needs a width of
// its own; without one it is laid out automatically.
func fixedLayout(st *style.Computed, in input) bool {
	return st.TableLayout == style.TableFixed && (some(in.known.w) || st.Size.Width.Kind != style.Auto)
}

// slot is a cell where the grid puts it: its row and column, and the rows and columns it
// spans.
type slot struct {
	cell             *node
	row, col, rs, cs int
	ci               int
}

// slots places each cell in the grid, past the slots cells above already take, its
// rowspan held to the end of its row group (0 spanning to there); cols is the grid's
// width.
func (t *tree) slots(n *node, rows []*node) ([]slot, int) {
	return gridSlots(n, rows, func(r *node, ci int) (int, int) {
		tr := t.rowWidths(r)
		return tr.colspan[ci], tr.rowspan[ci]
	})
}

// gridSlots is slots with each cell's colspan and rowspan read by spans.
func gridSlots(n *node, rows []*node, spans func(r *node, ci int) (int, int)) ([]slot, int) {
	var cells []slot
	var cols int
	end := make([]int, len(rows))
	i := 0
	for _, k := range n.kids {
		switch k.role {
		case roleRow:
			end[i] = i + 1
			i++
		case roleGroup:
			last := i + len(k.kids)
			for ; i < last; i++ {
				end[i] = last
			}
		case roleNone, roleTable, roleCaption, roleColumn, roleColumnGroup:
		}
	}
	var busy []int
	for ri, r := range rows {
		col := 0
		for ci, c := range r.kids {
			for col < len(busy) && busy[col] > 0 {
				col++
			}
			cs, rs := spans(r, ci)
			if rs == 0 || ri+rs > end[ri] {
				rs = end[ri] - ri
			}
			for len(busy) < col+cs {
				busy = append(busy, 0)
			}
			for k := col; k < col+cs; k++ {
				busy[k] = rs
			}
			cells = append(cells, slot{cell: c, row: ri, col: col, rs: rs, cs: cs, ci: ci})
			col += cs
		}
		cols = max(cols, len(busy))
		for k := range busy {
			busy[k] = max(busy[k]-1, 0)
		}
	}
	return cells, cols
}

// spanned lays out a table whose cells span rows or columns, or whose layout is fixed:
// each cell in the slots the grid gives it. Automatically, the columns take their single
// cells' min- and max-content widths, then each spanning cell, narrowest spans first,
// shares out what it needs beyond its columns' in proportion to their max-content
// widths, evenly where those are none. Fixed, the first row's cells with a width size
// their columns and the others share the rest. A row is as tall as its single-row cells,
// and a cell spanning rows grows them to fit it in proportion to their heights, as
// Chrome does.
//
//nolint:funlen,gocognit,gocyclo,cyclop,nestif // Chrome's table layout with spans in one pass, as its doc walks it
func (t *tree) spanned(n *node, in input, rows []*node) output {
	st := n.st
	tracks := n.tracks()
	padding := edges(st.Padding, in.parent.w)
	sx, sy := st.BorderSpacing[0], st.BorderSpacing[1]
	if st.BorderCollapse {
		t.collapse(n, rows, tracks)
		padding, sx, sy = rect{}, 0, 0
	}
	border := n.borders()
	inset := padding.add(border)
	cells, cols := t.slots(n, rows)
	if len(tracks) > 0 {
		last := tracks[len(tracks)-1]
		cols = max(cols, last.first+last.span)
	}
	spacingW := float32(0)
	if cols > 0 {
		spacingW = sx * float32(cols+1)
	}
	extra := spacingW + inset.horizontal()
	inner := func(cs int) float32 { return sx * float32(cs-1) }

	w := in.known.w
	switch {
	case !some(w):
		w = resolve(st.Size.Width, in.parent.w)
		if some(w) && contentBox(st) {
			w += inset.horizontal()
		}
	case st.BorderCollapse && contentBox(st) && st.Size.Width.Kind != style.Auto:
		// The width given round the content took the borders and padding of the
		// style; collapsed, the table has half its outer borders instead.
		w += inset.horizontal() - edges(st.Padding, in.parent.w).add(borders(st)).horizontal()
	}
	lo, hi := tableLimits(st, in, inset)
	w = mClamp(w, lo.w, hi.w)
	colMin, colMax := make([]float32, cols), make([]float32, cols)
	fixed := make([]bool, cols)
	// single are the columns' max-content widths from their single-column cells alone,
	// which their percentages widen the table by.
	percent, single := make([]float32, cols), make([]float32, cols)
	fixedW := slices.Repeat([]float32{-1}, cols)
	if fixedLayout(st, in) {
		if len(rows) > 0 {
			firstRowWidths(cells, fixedW, w-extra, inner)
		}
		for _, tr := range tracks {
			if cw, _ := tr.width(); some(cw) {
				for k := tr.first; k < tr.first+tr.span; k++ {
					fixedW[k] = cw
				}
			}
		}
		for k, fw := range fixedW {
			colMin[k], colMax[k] = max(fw, 0), max(fw, 0)
		}
	} else {
		spannedColumns(cells, colMin, colMax, fixed, inner)
		for _, c := range cells {
			if tr := c.cell.parent.row; c.cs == 1 {
				percent[c.col], single[c.col] = max(percent[c.col], tr.percent[c.ci]), max(single[c.col], tr.hi[c.ci])
			}
		}
		for _, tr := range tracks {
			cw, pc := tr.width()
			for k := tr.first; k < tr.first+tr.span; k++ {
				if some(cw) {
					colMin[k], colMax[k], fixed[k] = max(colMin[k], cw), max(colMax[k], cw), true
					single[k] = max(single[k], cw)
				}
				percent[k] = max(percent[k], pc)
			}
		}
		spannedPercents(cells, percent, colMax)
		percent = heldPercents(percent)
	}
	sumMin, sumMax := float32(0), float32(0)
	for k := range cols {
		sumMin += colMin[k]
		sumMax += colMax[k]
	}
	w = mClamp(or(w, autoTableWidth(in.avail.w, sumMin, percentMax(single, percent, sumMax), extra)), lo.w, hi.w)
	w = max(w, sumMin+extra, t.captionMin(n))
	if in.run == computeSize && in.axis == axisHorizontal {
		return sizeOutput(size{w, 0})
	}

	var widths []float32
	if fixedLayout(st, in) {
		widths = fixedWidths(fixedW, w-extra)
	} else {
		widths = percentWidths(colMin, colMax, fixed, percent, sumMin, sumMax, w-extra)
	}
	colX := make([]float32, cols+1)
	for k := range cols {
		colX[k+1] = colX[k] + widths[k] + sx
	}
	cellW := func(c slot) float32 { return colX[c.col+c.cs] - colX[c.col] - sx }
	content := w - inset.horizontal()
	heights := make([]float32, len(rows))
	for ri, r := range rows {
		heights[ri] = or(resolve(r.st.Size.Height, none), 0)
	}
	lowered, baselines := t.baselines(cells, len(rows), cellW, content)
	measure := func(i int) float32 {
		c := cells[i]
		cw := cellW(c)
		return lowered[i] + t.measureChildSize(c.cell, size{cw, none}, size{content, none},
			spaces{def(cw), maxContentSpace}, inherentSize, vertical, [2]bool{})
	}
	for i, c := range cells {
		if c.rs == 1 {
			heights[c.row] = max(heights[c.row], measure(i))
		}
	}
	var tall []int
	for i, c := range cells {
		if c.rs != 1 {
			tall = append(tall, i)
		}
	}
	slices.SortStableFunc(tall, func(a, b int) int { return cmp.Compare(cells[a].rs, cells[b].rs) })
	for _, i := range tall {
		c := cells[i]
		rows := heights[c.row : c.row+c.rs]
		share(rows, slices.Clone(rows), measure(i)-sy*float32(c.rs-1))
	}
	colGone, rowGone := collapsedTracks(tracks, cols), collapsedRows(rows)
	shownX, colAt := shownLines(widths, colGone, nil, sx, 0)
	// An item of a flex row keeps the width its container gave it, as wide as before the
	// collapse, its rows too, as in Chrome; stretched across a flex column or in a grid
	// area, it narrows as a block does.
	kept := some(in.known.w) && inFlexRow(n)
	if !kept {
		w -= colX[cols] - shownX[cols]
	}
	above, below := t.captionHeights(n, w)
	hgt := tableHeight(st, in, heights, sy, inset.vertical()+above+below, lo.h, hi.h)
	fullY, _ := shownLines(heights, nil, nil, sy, 0)
	rowY, rowAt := shownLines(heights, rowGone, rowSections(n, len(rows)), sy, above+inset.t+sy)
	hgt -= fullY[len(rows)] - (rowY[len(rows)] - rowY[0])
	s := size{w, hgt}
	if in.run == computeSize {
		return sizeOutput(s)
	}

	cellH := func(c slot) float32 { return fullY[c.row+c.rs] - fullY[c.row] - sy }
	rowW := max(shownX[cols]-sx, 0)
	if kept {
		rowW = max(colX[cols]-sx, 0)
	}
	rowH := func(ri int) float32 { return max(rowY[ri+1]-rowAt[ri]-sy, 0) }
	at := 0
	place := func(r *node, origin point) {
		r.unrounded = placed{location: point{inset.l + sx - origin.x, rowAt[at] - origin.y}, size: size{rowW, rowH(at)}}
		at++
	}
	for _, k := range n.kids {
		switch k.role {
		case roleRow:
			place(k, point{})
		case roleGroup:
			top := rowY[at]
			for _, r := range k.kids {
				place(r, point{inset.l + sx, top})
			}
			bottom := rowY[at] - sy
			if len(k.kids) == 0 {
				bottom = top
			}
			k.unrounded = placed{location: point{inset.l + sx, top}, size: size{rowW, max(bottom-top, 0)}}
		case roleNone, roleTable, roleCaption, roleColumn, roleColumnGroup:
		}
	}
	gridTop := rowY[0]
	if i := slices.Index(rowGone, false); i > 0 {
		gridTop = rowAt[i]
	}
	gridH := max(rowY[len(rows)]-sy-gridTop, 0)
	for _, tr := range tracks {
		at := point{inset.l + sx + colAt[tr.first], gridTop}
		colH := gridH
		if tr.col.st.Collapse {
			colH = 0
		}
		tr.col.unrounded = placed{location: at, size: size{max(shownX[tr.first+tr.span]-colAt[tr.first]-sx, 0), colH}}
		if g := tr.group; g != nil {
			if tr.col == g.kids[0] {
				g.unrounded = placed{location: at, size: size{0, gridH}}
				if g.st.Collapse {
					// Chrome leaves a collapsed column group at the table's top.
					g.unrounded = placed{location: point{at.x, 0}}
				}
			}
			tr.col.unrounded.location = point{at.x - g.unrounded.location.x, 0}
			g.unrounded.size.w = at.x + tr.col.unrounded.size.w - g.unrounded.location.x
		}
	}
	var candidates []oofCandidate
	for i, c := range cells {
		cw, ch := cellW(c), cellH(c)
		c.cell.lower(lowered[i])
		out := t.performChildLayout(c.cell, size{cw, ch}, size{content, none},
			spaces{def(cw), def(ch)}, inherentSize)
		shown := size{max(shownX[c.col+c.cs]-colAt[c.col]-sx, 0), max(rowY[c.row+c.rs]-rowAt[c.row]-sy, 0)}
		// Narrowed by a collapsed row or column, the cell keeps its content laid out at
		// the size it would have, clipped to its box.
		c.cell.cut = shown != out.size
		out.size = shown
		c.cell.unrounded = placed{
			location: point{colAt[c.col], 0},
			size:     out.size,
			overflow: out.overflow,
			padding:  edges(c.cell.st.Padding, content),
			border:   c.cell.borders(),
		}
		candidates = append(candidates, translateCandidates(out.oof, point{inset.l + sx + colAt[c.col], rowAt[c.row]})...)
	}
	if n.shared != nil {
		at := func(lines []float32, origin float32) []float32 {
			out := make([]float32, len(lines))
			for i, l := range lines {
				out[i] = origin + l
			}
			return out
		}
		n.box.Collapsed = n.shared.edges(at(shownX, inset.l), rowY)
	}
	t.placeCaptions(n, w, hgt-below)
	n.captionRoom = [2]float32{above, below}
	padded := rect{r: s.w - border.horizontal(), b: s.h - border.vertical()}
	first := none
	if len(rows) > 0 {
		first = baselines[0]
		for i, c := range cells {
			if c.row == 0 && c.rs == 1 && !some(baselines[0]) {
				first = fmax(or(first, 0),
					measure(i)-edges(c.cell.st.Padding, content).b-c.cell.borders().b)
			}
		}
		first += rowY[0]
	}
	return output{
		size:          s,
		overflow:      finalizeOverflow(padded, s.sub(border.sum()), [2]bool{}, false),
		firstBaseline: first,
		lastBaseline:  none,
		oof:           candidates,
		area:          &oofArea{size: s.sub(border.sum()), offset: point{border.l, border.t}},
	}
}

// baselines is how far each cell's content is lowered to stand on its row's baseline,
// and each row's baseline from its top, none in a row with no cell on it (an inline
// table then stands on the lowest content box of its first row, as Chrome's): the lowest
// first baseline of its cells that are on it, with cellW their widths in a table of
// content width. A cell without a line has its content box's bottom for one.
func (t *tree) baselines(cells []slot, rows int, cellW func(slot) float32, content float32) ([]float32, []float32) {
	lowered, own := make([]float32, len(cells)), make([]float32, len(cells))
	rowBase := slices.Repeat([]float32{none}, rows)
	for i, c := range cells {
		if !onBaseline(c.cell) {
			c.cell.lower(0)
			continue
		}
		cw := cellW(c)
		out := t.probeChildLayout(c.cell, size{cw, none}, size{content, none},
			spaces{def(cw), maxContentSpace})
		own[i] = out.firstBaseline - c.cell.lowered
		if !some(out.firstBaseline) {
			own[i] = out.size.h - edges(c.cell.st.Padding, content).b - c.cell.borders().b
		}
		rowBase[c.row] = fmax(or(rowBase[c.row], 0), own[i])
	}
	for i, c := range cells {
		if onBaseline(c.cell) {
			lowered[i] = rowBase[c.row] - own[i]
		}
	}
	return lowered, rowBase
}

// inFlexRow reports whether n is an item of a flex row, its width the main size its
// container gives it.
func inFlexRow(n *node) bool {
	return n.parent != nil && (n.parent.st.Display == style.FlexDisplay || n.parent.st.Display == style.InlineFlex) &&
		flexDir(n.parent.st.Direction).isRow()
}

// collapsedTracks marks the columns of a table with cols of them that tracks collapse,
// nil when none does.
func collapsedTracks(tracks []track, cols int) []bool {
	var gone []bool
	for _, tr := range tracks {
		if !tr.col.st.Collapse {
			continue
		}
		if gone == nil {
			gone = make([]bool, cols)
		}
		for k := tr.first; k < tr.first+tr.span; k++ {
			gone[k] = true
		}
	}
	return gone
}

// collapsedRows marks the rows with visibility: collapse, nil when none has it.
func collapsedRows(rows []*node) []bool {
	var gone []bool
	for ri, r := range rows {
		if !r.collapsedRow() {
			continue
		}
		if gone == nil {
			gone = make([]bool, len(rows))
		}
		gone[ri] = true
	}
	return gone
}

// collapsedRow reports a row with visibility: collapse, or one, anonymous, in a row group
// with it.
func (n *node) collapsedRow() bool {
	return n.st.Collapse || n.box.Node == nil && n.parent != nil && n.parent.role == roleGroup && n.parent.st.Collapse
}

// rowSections marks the first row of each of the table's sections: a row group, or a
// run of rows outside one.
func rowSections(n *node, count int) []bool {
	firsts := make([]bool, count)
	at, loose := 0, false
	for _, k := range n.kids {
		switch k.role {
		case roleRow:
			firsts[at] = !loose
			at, loose = at+1, true
		case roleGroup:
			if len(k.kids) > 0 {
				firsts[at] = true
			}
			at, loose = at+len(k.kids), false
		case roleNone, roleTable, roleCaption, roleColumn, roleColumnGroup:
		}
	}
	return firsts
}

// shownLines are where the tracks of sizes start, spacing apart from origin, once those
// gone marks are taken out with a spacing each, as Chrome takes collapsed rows and
// columns out: lines[k] is where track k starts after the spacing before it, at[k]
// where its box stands, at the end of the shown track before it in its section, which
// firsts starts (one for all when nil). A section of tracks all gone keeps one spacing.
func shownLines(sizes []float32, gone, firsts []bool, spacing, origin float32) ([]float32, []float32) {
	lines, at := make([]float32, len(sizes)+1), make([]float32, len(sizes))
	lines[0] = origin
	shown := false
	for k, sz := range sizes {
		if firsts != nil && firsts[k] {
			shown = false
		}
		at[k], lines[k+1] = lines[k], lines[k]+sz+spacing
		switch {
		case gone == nil || !gone[k]:
			shown = true
		case shown:
			at[k], lines[k+1] = lines[k]-spacing, lines[k]
		case k+1 < len(sizes) && (firsts == nil || !firsts[k+1]):
			lines[k+1] = lines[k]
		default:
			lines[k+1] = lines[k] + spacing
		}
	}
	return lines, at
}

// autoTableWidth is a table's width with none given: its columns' min-content or
// max-content widths and extra, or the room there is between the two.
func autoTableWidth(a space, sumMin, sumMax, extra float32) float32 {
	switch a.kind {
	case minContent:
		return sumMin + extra
	case maxContent:
		return sumMax + extra
	case definite:
	}
	return min(max(a.v, sumMin+extra), sumMax+extra)
}

// firstRowWidths sets fixedW, by fixed table layout, from the first row's cells with a
// width, a spanning one's shared evenly between its columns.
func firstRowWidths(cells []slot, fixedW []float32, content float32, inner func(int) float32) {
	for _, c := range cells {
		if c.row != 0 {
			break
		}
		if cw := resolve(c.cell.st.Size.Width, content); some(cw) {
			for k := c.col; k < c.col+c.cs; k++ {
				fixedW[k] = (cw - inner(c.cs)) / float32(c.cs)
			}
		}
	}
}

// spannedColumns sets each column's min- and max-content widths, by automatic layout:
// from its single-column cells, then from the spanning ones, narrowest span first.
func spannedColumns(cells []slot, colMin, colMax []float32, fixed []bool, inner func(int) float32) {
	for _, c := range cells {
		if c.cs == 1 {
			tr := c.cell.parent.row
			colMin[c.col], colMax[c.col] = max(colMin[c.col], tr.lo[c.ci]), max(colMax[c.col], tr.hi[c.ci])
			fixed[c.col] = fixed[c.col] || tr.fixed[c.ci]
		}
	}
	wide := slices.DeleteFunc(slices.Clone(cells), func(c slot) bool { return c.cs == 1 })
	slices.SortStableFunc(wide, func(a, b slot) int { return cmp.Compare(a.cs, b.cs) })
	for _, c := range wide {
		tr := c.cell.parent.row
		share(colMin[c.col:c.col+c.cs], colMax[c.col:c.col+c.cs], tr.lo[c.ci]-inner(c.cs))
		share(colMax[c.col:c.col+c.cs], colMax[c.col:c.col+c.cs], tr.hi[c.ci]-inner(c.cs))
		for k := c.col; k < c.col+c.cs; k++ {
			colMax[k] = max(colMax[k], colMin[k])
		}
	}
}

// spannedPercents shares the percentage width of each cell spanning columns, narrowest
// spans first, out to its columns, as Chrome does: what it asks beyond the percentages
// they have goes to those with none, in proportion to their max-content widths, evenly
// where those are none.
func spannedPercents(cells []slot, percent, colMax []float32) {
	wide := slices.DeleteFunc(slices.Clone(cells), func(c slot) bool {
		return c.cs == 1 || c.cell.parent.row.percent[c.ci] == 0
	})
	slices.SortStableFunc(wide, func(a, b slot) int { return cmp.Compare(a.cs, b.cs) })
	for _, c := range wide {
		need, free, weight := c.cell.parent.row.percent[c.ci], 0, float32(0)
		for k := c.col; k < c.col+c.cs; k++ {
			need -= percent[k]
			if percent[k] == 0 {
				free++
				weight += colMax[k]
			}
		}
		if need <= 0 || free == 0 {
			continue
		}
		for k := c.col; k < c.col+c.cs; k++ {
			switch {
			case percent[k] > 0:
			case weight > 0:
				percent[k] = need * colMax[k] / weight
			default:
				percent[k] = need / float32(free)
			}
		}
	}
}

// tableHeight is the height of a table whose rows are heights tall, around being its
// inset and captions: the rows' with the spacing round them, or the height the table is
// given, which grows the rows where it is above theirs: a floor only, as Chrome's, which
// never shortens a table below its rows, and for a table that collapses rows one taken
// from before the collapse.
func tableHeight(st *style.Computed, in input, heights []float32, sy, around, lo, hi float32) float32 {
	hgt := around
	if len(heights) > 0 {
		hgt += sy
	}
	for _, h := range heights {
		hgt += h + sy
	}
	given := or(mClamp(or(in.known.h, resolve(st.Size.Height, in.parent.h)), lo, hi), lo)
	if some(given) && given > hgt {
		growRows(heights, given-hgt)
		return given
	}
	return hgt
}

// tableLimits are the min and max sizes of a table's border box, none
// where it has none or is measured for its content alone, as a flex basis is.
func tableLimits(st *style.Computed, in input, inset rect) (size, size) {
	if in.sizing == contentSize {
		return noSize, noSize
	}
	lo, hi := resolveSize(st.MinSize, in.parent), resolveSize(st.MaxSize, in.parent)
	if contentBox(st) {
		around := size{inset.horizontal(), inset.vertical()}
		lo, hi = lo.mAdd(around), hi.mAdd(around)
	}
	return lo, hi
}

// growRows shares extra, the height a table is given beyond its rows', between them in
// proportion to their heights.
func growRows(heights []float32, extra float32) {
	have := float32(0)
	for _, h := range heights {
		have += h
	}
	share(heights, slices.Clone(heights), have+extra)
}

// share grows cols so they hold need between them, the shortfall shared in proportion
// to weights, evenly where those are all none.
func share(cols, weights []float32, need float32) {
	have, total := float32(0), float32(0)
	for k := range cols {
		have += cols[k]
		total += weights[k]
	}
	if need <= have {
		return
	}
	extra := need - have
	for k := range cols {
		if total > 0 {
			cols[k] += extra * weights[k] / total
		} else {
			cols[k] += extra / float32(len(cols))
		}
	}
}

// fixedWidths are a fixed table's columns in its content width: those its first row
// sized at their width, the others sharing what is left evenly; with none left over,
// the sized ones grow in proportion to fill it.
func fixedWidths(fixedW []float32, content float32) []float32 {
	widths := make([]float32, len(fixedW))
	sized, free := float32(0), 0
	for _, fw := range fixedW {
		if fw >= 0 {
			sized += fw
		} else {
			free++
		}
	}
	for k, fw := range fixedW {
		switch {
		case fw < 0:
			widths[k] = max(content-sized, 0) / float32(free)
		case free == 0 && sized > 0 && content > sized:
			widths[k] = fw * content / sized
		default:
			widths[k] = fw
		}
	}
	return widths
}
