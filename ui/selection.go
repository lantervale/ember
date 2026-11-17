package ui

import (
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/rivo/uniseg"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/lantervale/ember/ui/layout"
	"github.com/lantervale/ember/ui/paint"
)

// Position is a place in a Document's text: before the character Offset
// of the text node Node, counted in the text as the node shows it,
// translated and cased, its white space as written.
type Position struct {
	Node   *Node
	Offset int
}

// Selection is the text a Document has selected (decision 0090): from
// Anchor, where it began, to Focus, which a drag or Shift moves, before
// Anchor when it went backwards. Text is what a copy puts on the
// clipboard, as Chrome's Selection.toString gives it.
type Selection struct {
	Anchor, Focus Position
	Text          string
}

// Empty reports a selection holding no text: none, or a place alone.
func (s Selection) Empty() bool { return s.Anchor == s.Focus }

// SelectionChanged is the Document's selection set, moved or cleared, by
// the pointer, the keys, automation or the game. Node is the Document's
// body, so a callback on it or on the html hears it.
type SelectionChanged struct{ Node *Node }

// selecting is a Document's selection and what the pointer is doing to it.
type selecting struct {
	anchor, focus Position
	// held is the left button holding the focus since a press that began
	// or extended the selection, by unit: characters, words after a double
	// click, blocks after a triple, base then being what the press took.
	held bool
	unit int
	base [2]Position
	// clicks counts the run of presses, the last at clickAt on the clock,
	// at clickWhere.
	clicks     int
	clickAt    float32
	clickWhere geom.Vec2
	// scrolledAt is the clock when a drag last scrolled a container.
	scrolledAt float32
	// runs are the text nodes with fragments in tree order and maps what
	// was asked of them, as layout serial laid them out.
	runs    []*Node
	maps    map[*Node]*runMap
	serial  int
	indexed bool
	// parts are the selected rectangles of each text run, in order the
	// runs they are in, made for from and partsAt.
	parts   map[*box.Box][]paint.Selected
	order   []*box.Box
	from    [2]Position
	partsAt int
	made    bool
}

// Selection is what the Document's text has selected, as it was last
// laid out.
func (d *Document) Selection() Selection {
	s := Selection{Anchor: d.sel.anchor, Focus: d.sel.focus}
	if !s.Empty() {
		start, end := d.ordered(s.Anchor, s.Focus)
		s.Text = d.textBetween(start, end)
	}
	return s
}

// Select selects the text from anchor to focus, whatever user-select says
// of where they stand; what it holds of text that takes no selection is
// neither drawn nor copied. Equal places, or a place without a text node,
// clear the selection.
func (d *Document) Select(anchor, focus Position) {
	if anchor.Node == nil || focus.Node == nil || !anchor.Node.IsText() || !focus.Node.IsText() {
		anchor, focus = Position{}, Position{}
	}
	d.setSelection(anchor, focus)
}

// SelectAll selects all the text of the Document that takes a selection,
// as Cmd+A or Ctrl+A does, and reports whether there is any.
func (d *Document) SelectAll() bool { return d.SelectContents(d.html) }

// SelectContents selects the text inside n that takes a selection, from
// its first character to its last, and reports whether there is any.
func (d *Document) SelectContents(n *Node) bool {
	first, last, ok := d.textIn(n)
	if !ok {
		return false
	}
	d.setSelection(first, last)
	return true
}

// ClearSelection selects nothing.
func (d *Document) ClearSelection() { d.setSelection(Position{}, Position{}) }

// SelectionRects are the rectangles the selection is drawn in, in the
// Document's space and in the text's order: each line's part, as high as
// its line, an ancestor's transform left out.
func (d *Document) SelectionRects() []geom.Rect {
	var out []geom.Rect
	parts := d.selectedParts()
	for _, b := range d.sel.order {
		at := b.Rect()
		for _, p := range parts[b] {
			out = append(out, geom.Rect{X: at.X + p.Rect.X, Y: at.Y + p.Rect.Y, W: p.Rect.W, H: p.Rect.H})
		}
	}
	return out
}

func (d *Document) setSelection(anchor, focus Position) {
	s := &d.sel
	if anchor == focus && s.anchor == s.focus {
		// a place alone shows nothing, so moving it changes nothing
		s.anchor, s.focus = anchor, focus
		return
	}
	if s.anchor == anchor && s.focus == focus {
		return
	}
	s.anchor, s.focus = anchor, focus
	d.replay.repaint = true
	d.events = append(d.events, SelectionChanged{Node: d.body})
}

// keepSelection drops a selection whose ends left the Document, and one
// held by the pointer once the button is up.
func (d *Document) keepSelection() {
	s := &d.sel
	if s.anchor.Node == nil {
		return
	}
	if !d.inDocument(s.anchor.Node) || !d.inDocument(s.focus.Node) {
		s.held = false
		d.ClearSelection()
	}
}

// selectionPaint is the selection as paint draws it, nil while nothing is
// selected, so a frame without one asks paint nothing.
func (d *Document) selectionPaint() paint.Selection {
	if d.sel.anchor == d.sel.focus {
		return nil
	}
	return paintedSelection{d}
}

type paintedSelection struct{ d *Document }

func (p paintedSelection) Parts(run *box.Box) []paint.Selected { return p.d.selectedParts()[run] }

func (p paintedSelection) Holds(n *dom.Node) bool {
	start, end := p.d.ordered(p.d.sel.anchor, p.d.sel.focus)
	return treeBefore(start.Node, n) && treeBefore(n, end.Node) && selects(dom.Data(n).Style)
}

// selects reports whether a selection takes what st styles.
func selects(st *style.Computed) bool {
	return st != nil && st.UserSelect != style.SelectNone && st.Visible
}

// textRuns are the text nodes layout cut into fragments, in tree order:
// not a field's, which selects its own, nor a select's.
func (d *Document) textRuns() []*Node {
	s := &d.sel
	if s.indexed && s.serial == d.placed {
		return s.runs
	}
	s.runs, s.maps, s.serial, s.indexed = s.runs[:0], nil, d.placed, true
	if d.tree == nil {
		return nil
	}
	d.html.Walk(func(n *Node) bool {
		if n.IsText() {
			if b := d.tree.Of[n]; b != nil && b.Kind == box.TextRun && len(b.Lines) > 0 {
				s.runs = append(s.runs, n)
			}
			return false
		}
		if t := n.Tag(); t == dom.Input || t == dom.Textarea || t == dom.Select {
			return false
		}
		st := dom.Data(n).Style
		return st != nil && st.Display != style.DisplayNone
	})
	return s.runs
}

// runMap is a text node's text as layout cut it: src the text it counted
// its fragments' starts in, and for each fragment, where each of its
// characters is in src and along the line.
type runMap struct {
	src   []rune
	frags []fragMap
}

// fragMap is one fragment: offs[j] is the place in the node's text of
// its character j, the last entry of its end, and xs, measured when
// first asked, how far each stands from the fragment's start.
type fragMap struct {
	offs []int32
	xs   []float32
}

func isBlank(r rune) bool { return r == ' ' || r == '\n' || r == '\t' || r == '\r' || r == '\f' }

const zeroWidthSpace = '\u200b'

// mapOf is n's map, nil when layout gave it no fragments.
func (d *Document) mapOf(n *Node) *runMap {
	d.textRuns()
	if m, ok := d.sel.maps[n]; ok {
		return m
	}
	if d.tree == nil {
		return nil
	}
	b := d.tree.Of[n]
	if b == nil || b.Kind != box.TextRun || b.Style == nil {
		return nil
	}
	m := &runMap{src: []rune(layout.Shown(n, b.Style, d.tr())), frags: make([]fragMap, len(b.Lines))}
	for k, f := range b.Lines {
		offs := make([]int32, 0, len(f.Text)+1)
		i := int(f.Start)
		for _, r := range f.Text {
			// What layout dropped stands before the character it kept:
			// the white space it collapsed and the zero-width spaces.
			for i < len(m.src) && (m.src[i] == zeroWidthSpace || isBlank(m.src[i]) && r != ' ') {
				i++
			}
			offs = append(offs, int32(min(i, len(m.src)))) //nolint:gosec // a text node's length
			i++
		}
		offs = append(offs, int32(min(i, len(m.src)))) //nolint:gosec // a text node's length
		m.frags[k].offs = offs
	}
	if d.sel.maps == nil {
		d.sel.maps = map[*Node]*runMap{}
	}
	d.sel.maps[n] = m
	return m
}

// edges is how far each character of fragment k of b stands from the
// fragment's start, and its end.
func (d *Document) edges(m *runMap, b *box.Box, k int) []float32 {
	f := &m.frags[k]
	if f.xs != nil {
		return f.xs
	}
	text := b.Lines[k].Text
	f.xs = make([]float32, 1, len(f.offs))
	j := 0
	for i := range text {
		if i == 0 {
			continue
		}
		j++
		f.xs = append(f.xs, d.fonts.Advance(b.Style.Font, text[:i])+b.Style.LetterSpacing*float32(j))
	}
	if len(text) > 0 {
		f.xs = append(f.xs, max(b.Lines[k].Rect.W, f.xs[len(f.xs)-1]))
	}
	return f.xs
}

// lower is the first character of a fragment at or after the place off.
func lower(offs []int32, off int) int {
	return sort.Search(len(offs)-1, func(j int) bool { return int(offs[j]) >= off })
}

// placing is where a text run's fragments lie: its box's top-left in the
// Document's space, the part of it its ancestors' overflow shows, and the
// transform its ancestors turn it by.
type placing struct {
	at      geom.Vec2
	clip    geom.Rect
	clipped bool
	xf      geom.Transform2D
	turned  bool
}

func place(b *box.Box) placing {
	r := b.Rect()
	p := placing{at: geom.V2(r.X, r.Y)}
	for a := b.Parent; a != nil; a = a.Parent {
		st := a.Style
		if st == nil || a.Kind != box.Element && a.Kind != box.Viewport {
			continue
		}
		if st.Overflow.Width != style.Visible || st.Overflow.Height != style.Visible {
			view := a.Rect()
			view.X, view.Y = view.X+a.Border.Left, view.Y+a.Border.Top
			view.W, view.H = view.W-a.Border.Horizontal(), view.H-a.Border.Vertical()
			if p.clipped {
				view = intersect(view, p.clip)
			}
			p.clip, p.clipped = view, true
		}
		if xf, ok := paint.TransformOf(a); ok {
			if p.turned {
				xf = xf.Mul(p.xf)
			}
			p.xf, p.turned = xf, true
		}
	}
	return p
}

func intersect(a, b geom.Rect) geom.Rect {
	x0, y0 := max(a.X, b.X), max(a.Y, b.Y)
	x1, y1 := min(a.X+a.W, b.X+b.W), min(a.Y+a.H, b.Y+b.H)
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// away is how far v lies outside lo to hi.
func away(v, lo, hi float32) float32 {
	switch {
	case v < lo:
		return lo - v
	case v > hi:
		return v - hi
	}
	return 0
}

// piece is one fragment a selection may reach: fragment k of n's run b,
// r in the Document's space as its ancestors show it, and q the point
// asked about in that space.
type piece struct {
	n *Node
	b *box.Box
	k int
	r geom.Rect
	x float32
	q geom.Vec2
}

// pieces calls each for the fragments that take a selection and show,
// those inside scope when it is not nil, in the text's order, at the
// point at.
func (d *Document) pieces(at geom.Vec2, scope *box.Box, each func(p piece)) {
	for _, n := range d.textRuns() {
		b := d.tree.Of[n]
		if !selects(b.Style) || scope != nil && !under(b, scope) {
			continue
		}
		pl := place(b)
		q := at
		if pl.turned {
			q = pl.xf.Inverse().Xform(at)
		}
		for k, f := range b.Lines {
			if f.Text == "" {
				continue
			}
			r := geom.Rect{X: pl.at.X + f.Rect.X, Y: pl.at.Y + f.Rect.Y, W: f.Rect.W, H: f.Rect.H}
			x := r.X
			if pl.clipped {
				if r = intersect(r, pl.clip); r.W <= 0 || r.H <= 0 {
					continue
				}
			}
			each(piece{n: n, b: b, k: k, r: r, x: x, q: q})
		}
	}
}

func under(b, scope *box.Box) bool {
	for ; b != nil; b = b.Parent {
		if b == scope {
			return true
		}
	}
	return false
}

// positionNear is the place in the text nearest at, among the text
// inside scope when it is not nil: on the line at's height falls in, or
// the nearest one, the character edge nearest across it; the text's start
// above all of it and its end below, as Chrome on macOS.
func (d *Document) positionNear(at geom.Vec2, scope *box.Box) (Position, bool) {
	if d.tree == nil {
		return Position{}, false
	}
	var best, first, last piece
	var bestY, bestX, top, bottom float32
	found := false
	d.pieces(at, scope, func(p piece) {
		dy, dx := away(p.q.Y, p.r.Y, p.r.Y+p.r.H-0.01), away(p.q.X, p.r.X, p.r.X+p.r.W)
		if !found {
			first, top, bottom = p, p.r.Y, p.r.Y+p.r.H
		}
		last, top, bottom = p, min(top, p.r.Y), max(bottom, p.r.Y+p.r.H)
		if !found || dy < bestY || dy == bestY && dx < bestX {
			best, bestY, bestX = p, dy, dx
		}
		found = true
	})
	if !found {
		return Position{}, false
	}
	switch {
	case bestY > 0 && best.q.Y < top:
		return Position{first.n, int(d.mapOf(first.n).frags[first.k].offs[0])}, true
	case bestY > 0 && best.q.Y >= bottom:
		offs := d.mapOf(last.n).frags[last.k].offs
		return Position{last.n, int(offs[len(offs)-1])}, true
	}
	return d.positionIn(best, best.q.X), true
}

// positionIn is the character edge of p nearest x.
func (d *Document) positionIn(p piece, x float32) Position {
	m := d.mapOf(p.n)
	xs := d.edges(m, p.b, p.k)
	j := 0
	for i, e := range xs {
		if abs(p.x+e-x) < abs(p.x+xs[j]-x) {
			j = i
		}
	}
	return Position{p.n, int(m.frags[p.k].offs[j])}
}

// treeBefore reports whether a comes before b in tree order, an ancestor
// before what it holds.
func treeBefore(a, b *Node) bool {
	if a == b || a == nil || b == nil {
		return false
	}
	da, db := depth(a), depth(b)
	x, y := a, b
	for ; da > db; da-- {
		x = x.Parent()
	}
	for ; db > da; db-- {
		y = y.Parent()
	}
	if x == y {
		return x == a
	}
	for x.Parent() != y.Parent() {
		x, y = x.Parent(), y.Parent()
	}
	if x.Parent() == nil {
		return false
	}
	for _, c := range x.Parent().Children() {
		if c == x || c == y {
			return c == x
		}
	}
	return false
}

func depth(n *Node) int {
	k := 0
	for ; n != nil; n = n.Parent() {
		k++
	}
	return k
}

func nodeInside(n, e *Node) bool {
	for ; n != nil; n = n.Parent() {
		if n == e {
			return true
		}
	}
	return false
}

func before(a, b Position) bool {
	if a.Node == b.Node {
		return a.Offset < b.Offset
	}
	return treeBefore(a.Node, b.Node)
}

// ordered is a and b, the earlier in the text first.
func (d *Document) ordered(a, b Position) (Position, Position) {
	if before(b, a) {
		return b, a
	}
	return a, b
}

// textIn is the first and last place of the text inside e that takes a
// selection.
func (d *Document) textIn(e *Node) (Position, Position, bool) {
	var first, last *Node
	for _, n := range d.textRuns() {
		if nodeInside(n, e) && selects(d.tree.Of[n].Style) {
			if first == nil {
				first = n
			}
			last = n
		}
	}
	if first == nil {
		return Position{}, Position{}, false
	}
	return Position{first, 0}, Position{last, len(d.mapOf(last).src)}, true
}

// wordAt is the word, the white space or the mark at p, as a double
// click takes it.
func (d *Document) wordAt(p Position) (Position, Position) {
	m := d.mapOf(p.Node)
	if m == nil {
		return p, p
	}
	at, state, s := 0, -1, string(m.src)
	for len(s) > 0 {
		var w string
		w, s, state = uniseg.FirstWordInString(s, state)
		n := utf8.RuneCountInString(w)
		if p.Offset < at+n || len(s) == 0 {
			return Position{p.Node, at}, Position{p.Node, at + n}
		}
		at += n
	}
	return p, p
}

// blockAt is the text of the block p's text is in, as a triple click
// takes it.
func (d *Document) blockAt(p Position) (Position, Position) {
	e := p.Node.Parent()
	for e != nil && e.Parent() != nil && inlineLevel(e) {
		e = e.Parent()
	}
	if first, last, ok := d.textIn(e); ok {
		return first, last
	}
	return p, p
}

func inlineLevel(e *Node) bool {
	st := dom.Data(e).Style
	if st == nil {
		return true
	}
	switch e.Tag() { //nolint:exhaustive // the table's parts, blocks whatever their display
	case dom.Table, dom.Caption, dom.Thead, dom.Tbody, dom.Tfoot, dom.Tr, dom.Td, dom.Th:
		return false
	}
	switch st.Display {
	case style.Inline, style.InlineBlock, style.InlineFlex, style.InlineGrid, style.Contents:
		return !flexItem(e)
	case style.Block, style.FlexDisplay, style.GridDisplay, style.DisplayNone:
	}
	return false
}

// flexItem reports a child of a flex or grid container, a block whatever
// its display.
func flexItem(e *Node) bool {
	p := e.Parent()
	if p == nil || dom.Data(p).Style == nil {
		return false
	}
	switch dom.Data(p).Style.Display {
	case style.FlexDisplay, style.InlineFlex, style.GridDisplay, style.InlineGrid:
		return true
	case style.Inline, style.Block, style.InlineBlock, style.DisplayNone, style.Contents:
	}
	return false
}

// wholes widens start and end over the elements with user-select: all
// they stand in, taken whole or not at all.
func (d *Document) wholes(start, end Position) (Position, Position) {
	if e := allOf(start.Node); e != nil {
		if first, _, ok := d.textIn(e); ok {
			start = first
		}
	}
	if e := allOf(end.Node); e != nil {
		if _, last, ok := d.textIn(e); ok {
			end = last
		}
	}
	return start, end
}

// allOf is the outermost element with user-select: all holding n in one
// run of such elements, nil for none.
func allOf(n *Node) *Node {
	var all *Node
	for e := n.Parent(); e != nil; e = e.Parent() {
		st := dom.Data(e).Style
		if st == nil || st.UserSelect != style.SelectAll {
			break
		}
		all = e
	}
	return all
}

// point sets the selection as the pointer or a key asks, from anchor to
// focus, whole over what user-select: all holds.
func (d *Document) point(anchor, focus Position) {
	if before(focus, anchor) {
		focus, anchor = d.wholes(focus, anchor)
	} else {
		anchor, focus = d.wholes(anchor, focus)
	}
	d.setSelection(anchor, focus)
}

// selectedParts are the rectangles of each text run the selection
// covers, in the run's space as its fragments' are.
func (d *Document) selectedParts() map[*box.Box][]paint.Selected {
	s := &d.sel
	ends := [2]Position{s.anchor, s.focus}
	d.textRuns()
	if s.made && s.partsAt == d.placed && s.from == ends {
		return s.parts
	}
	s.made, s.partsAt, s.from, s.order = true, d.placed, ends, s.order[:0]
	clear(s.parts)
	if s.anchor == s.focus || d.tree == nil {
		return s.parts
	}
	if s.parts == nil {
		s.parts = map[*box.Box][]paint.Selected{}
	}
	start, end := d.ordered(s.anchor, s.focus)
	runs := s.runs
	lo := sort.Search(len(runs), func(i int) bool { return !treeBefore(runs[i], start.Node) })
	hi := sort.Search(len(runs), func(i int) bool { return treeBefore(end.Node, runs[i]) })
	for i := lo; i < hi; i++ {
		n := runs[i]
		b := d.tree.Of[n]
		if !selects(b.Style) {
			continue
		}
		from, to := 0, math.MaxInt
		if n == start.Node {
			from = start.Offset
		}
		if n == end.Node {
			to = end.Offset
		}
		var next *box.Box
		if i+1 < len(runs) {
			next = d.tree.Of[runs[i+1]]
		}
		if ps := d.partsOf(n, b, next, from, to, n != end.Node); len(ps) > 0 {
			s.parts[b] = ps
			s.order = append(s.order, b)
		}
	}
	return s.parts
}

// partsOf are the rectangles of the characters from to to of n's run b:
// on each line from the first selected character to the last, as high as
// the line and its glyphs, and where the selection goes on past the
// fragment's end, on to the next fragment of the line, or a space wide
// past the line's end, as Chrome draws a selected line break. next is
// the run after b in the text, and on whether the selection goes on
// past n.
func (d *Document) partsOf(n *Node, b, next *box.Box, from, to int, on bool) []paint.Selected {
	whole := from == 0 && to == math.MaxInt
	var m *runMap
	if !whole {
		m = d.mapOf(n)
	}
	metrics := d.fonts.Metrics(b.Style.Font)
	var out []paint.Selected
	for k, f := range b.Lines {
		x0, x1, goesOn := float32(0), f.Rect.W, true
		if !whole {
			offs := m.frags[k].offs
			j0, j1 := lower(offs, from), lower(offs, to)
			if j1 <= j0 {
				continue
			}
			xs := d.edges(m, b, k)
			x0, x1 = xs[j0], xs[j1]
			goesOn = j1 == len(offs)-1 && (on || int(offs[j1]) < to)
		}
		if goesOn {
			x1 += d.tail(b, k, next)
		}
		top := min(f.Rect.Y, f.Baseline-metrics.Ascent)
		bottom := max(f.Rect.Y+f.Rect.H, f.Baseline+metrics.Descent)
		out = append(out, paint.Selected{Line: k, Rect: geom.Rect{X: f.Rect.X + x0, Y: top, W: x1 - x0, H: bottom - top}})
	}
	return out
}

// tail is how far past its end fragment k of b is selected when the
// selection goes on: up to the next fragment on its line, or a space
// past the line's end.
func (d *Document) tail(b *box.Box, k int, next *box.Box) float32 {
	f := b.Lines[k]
	at := b.Rect()
	var after *box.Fragment
	origin := at
	switch {
	case k+1 < len(b.Lines):
		after = &b.Lines[k+1]
	case next != nil && len(next.Lines) > 0 && container(next) == container(b):
		after, origin = &next.Lines[0], next.Rect()
	}
	if after != nil && abs(origin.Y+after.Rect.Y-(at.Y+f.Rect.Y)) < 0.5 {
		return max(origin.X+after.Rect.X-(at.X+f.Rect.X+f.Rect.W), 0)
	}
	return d.fonts.Advance(b.Style.Font, " ")
}

// container is the block whose lines b's fragments are on.
func container(b *box.Box) *box.Box {
	for b = b.Parent; b != nil; b = b.Parent {
		if b.Kind != box.Element || b.Node == nil || !inlineLevel(b.Node) {
			return b
		}
	}
	return nil
}

// copier writes the text between two places as Chrome's
// Selection.toString does: the text as drawn, what user-select: none or
// visibility hides left out, a line break for a br and around each block
// and table row, two after a p, a tab between a row's cells.
type copier struct {
	d          *Document
	start, end Position
	out        strings.Builder
	// began and done are set once the walk met the start and the end.
	began, done bool
	// space is a white space layout collapsed away, written before the
	// next text of the line.
	space bool
	last  rune
}

func (d *Document) textBetween(start, end Position) string {
	if d.tree == nil || start.Node == nil {
		return ""
	}
	c := copier{d: d, start: start, end: end}
	c.node(d.html)
	return c.out.String()
}

func (c *copier) put(r rune) {
	c.out.WriteRune(r)
	c.last, c.space = r, false
}

func (c *copier) node(n *Node) {
	if c.done {
		return
	}
	if n.IsText() {
		c.text(n)
		return
	}
	st := dom.Data(n).Style
	tag := n.Tag()
	hidden := st == nil || st.Display == style.DisplayNone || tag == dom.Input || tag == dom.Textarea || tag == dom.Select
	if hidden && !nodeInside(c.start.Node, n) && !nodeInside(c.end.Node, n) {
		return
	}
	block, cell := !hidden && c.breaks(n), tag == dom.Td || tag == dom.Th
	if c.began && !hidden {
		switch {
		case tag == dom.Br:
			c.put('\n')
		case cell && c.last != '\n' && c.last != 0:
			c.put('\t')
		case block && c.last != '\n':
			c.put('\n')
		}
	}
	for _, k := range n.Children() {
		c.node(k)
	}
	if !c.began || c.done || !block || c.last == 0 {
		return
	}
	if c.last != '\n' {
		c.put('\n')
		if tag == dom.P {
			c.put('\n')
		}
	} else if tag == dom.P {
		c.put('\n')
	}
}

// breaks reports an element Chrome writes a line break before and after:
// a block in the flow, a table row, never the page's html and body nor a
// table's cell.
func (c *copier) breaks(n *Node) bool {
	switch n.Tag() { //nolint:exhaustive // the page's roots and a table's parts; the others by their style
	case dom.HTML, dom.Body, dom.Td, dom.Th, dom.Thead, dom.Tbody, dom.Tfoot:
		return false
	case dom.Tr:
		return true
	}
	st := dom.Data(n).Style
	if st.Float != style.FloatNone || st.Position == style.Absolute || st.Position == style.FixedPosition {
		return false
	}
	return !inlineLevel(n)
}

func (c *copier) text(n *Node) {
	from, to := 0, math.MaxInt
	if n == c.start.Node {
		c.began, from = true, c.start.Offset
	}
	if n == c.end.Node {
		c.done, to = true, c.end.Offset
	}
	if !c.began {
		return
	}
	b := c.d.tree.Of[n]
	if b == nil || b.Kind != box.TextRun || !selects(b.Style) {
		return
	}
	m := c.d.mapOf(n)
	at := 0
	for k, f := range b.Lines {
		offs := m.frags[k].offs
		c.dropped(m.src, max(at, from), min(int(offs[0]), to), b.Style.WhiteSpace, k > 0)
		if j0, j1 := lower(offs, from), lower(offs, to); j1 > j0 {
			runes := []rune(f.Text)
			if c.space && c.last != 0 && !isBlank(c.last) {
				c.put(' ')
			}
			for _, r := range runes[j0:j1] {
				c.put(r)
			}
		}
		at = int(offs[len(offs)-1])
	}
	c.dropped(m.src, max(at, from), min(len(m.src), to), b.Style.WhiteSpace, false)
}

// dropped is the part from to to of a node's text that no fragment
// holds: a line break where white-space keeps them, the space of a soft
// wrap, which Chrome writes, else a space that waits for the text after
// it.
func (c *copier) dropped(src []rune, from, to int, ws style.WhiteSpace, wrap bool) {
	if from >= to {
		return
	}
	keeps := ws == style.Pre || ws == style.PreWrap || ws == style.PreLine
	for _, r := range src[from:to] {
		switch {
		case r == '\n' && keeps:
			c.put('\n')
		case isBlank(r) && wrap:
			if c.last != ' ' {
				c.put(' ')
			}
		case isBlank(r):
			c.space = c.last != 0 && c.last != '\n'
		}
	}
}
