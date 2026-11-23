package ui

import (
	"math"
	"slices"
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

// RightDragSelected is text selected by a drag of the right button, let
// go, in a Document where RightDragSelects: its Selection says what. It
// stands for the ContextClicked of that press. Node is the Document's
// body, so a callback on it or on the html hears it (decision 0109).
type RightDragSelected struct{ Node *Node }

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
	// rightArmed is the right button held since a press where a selection
	// may begin, in a Document where RightDragSelects: at rightAt in the
	// text and rightFrom in the Document. rightDragged is that press a
	// drag, once the pointer went dragStart from it, until the next press.
	rightArmed, rightDragged bool
	rightAt                  Position
	rightFrom                geom.Vec2
	// scrolledAt is the clock when a drag last scrolled a container, and
	// dragScrolled the containers it scrolled, snapped once it ends.
	scrolledAt   float32
	dragScrolled []*Node
	// runs are the text nodes with fragments in tree order and maps what
	// was asked of them, as layout serial laid them out.
	runs    []*Node
	breaks  []*Node
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
// not a field's, which selects its own, nor a select's. It lists the brs
// laid out too, in sel.breaks.
func (d *Document) textRuns() []*Node {
	s := &d.sel
	if s.indexed && s.serial == d.placed {
		return s.runs
	}
	s.runs, s.breaks, s.maps, s.serial, s.indexed = s.runs[:0], s.breaks[:0], nil, d.placed, true
	if d.tree == nil {
		return nil
	}
	d.html.Walk(func(n *Node) bool {
		if n.IsText() {
			if b := d.tree.Of[n]; b != nil && b.Kind == box.TextRun && (len(b.Lines) > 0 || dom.Data(n).Letter != nil) {
				s.runs = append(s.runs, n)
			}
			return false
		}
		if n.Tag() == dom.Br {
			if d.tree.Of[n] != nil {
				s.breaks = append(s.breaks, n)
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
			for i < len(m.src) && (m.src[i] == zeroWidthSpace || m.src[i] == '\u00ad' || isBlank(m.src[i]) && r != ' ') {
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
// fragment's left, and its end: from its right edge for a fragment
// shown right to left, the first character rightmost.
func (d *Document) edges(m *runMap, b *box.Box, k int) []float32 {
	f := &m.frags[k]
	if f.xs != nil {
		return f.xs
	}
	text, st := b.Lines[k].Text, b.Style
	if s := b.Lines[k].Style; s != nil {
		st = s
	}
	f.xs = make([]float32, 1, len(f.offs))
	sp := spacingOf(st)
	frag := &b.Lines[k]
	length := frag.Rect.W
	if st.Mode.Vertical() {
		length = frag.Rect.H
	}
	advance := func(i int) float32 { return d.fonts.Advance(st.Font, text[:i]) }
	switch {
	case st.Combine && st.Mode.Vertical():
		// combined text stands across one em, its characters inside it
		advance = func(int) float32 { return 0 }
	case st.Stands():
		advance = func(i int) float32 { return d.uprightAdvance(st, text[:i]) }
	case frag.Lead != "" || frag.Trail != "":
		// A ligature across the fragment's start is the text before's, one
		// across its end its own: its characters after it stand at its end.
		advance = func(i int) float32 {
			return d.fonts.AdvanceAmid(st.Font, text[:i], frag.Lead, text[i:]+frag.Trail, frag.Before, frag.After)
		}
	}
	var room float32
	last := rune(-1)
	han := b.Lines[k].Spacing
	for i, r := range text {
		if i > 0 {
			room += sp.after(last)
			for len(han) > 0 && han[0].At < i {
				room += han[0].Space
				han = han[1:]
			}
			f.xs = append(f.xs, advance(i)+room)
		}
		last = r
	}
	if len(text) > 0 {
		f.xs = append(f.xs, max(length, f.xs[len(f.xs)-1]))
	}
	if b.Lines[k].RTL {
		w := f.xs[len(f.xs)-1]
		for i, x := range f.xs {
			f.xs[i] = w - x
		}
	}
	return f.xs
}

// uprightAdvance is the length of s, text of vertical lines of style st,
// along them: each cluster standing upright its vertical advance, as
// layout measures it.
func (d *Document) uprightAdvance(st *style.Computed, s string) float32 {
	var w float32
	style.Oriented(s, st.Orientation, func(part string, upright bool) {
		if !upright {
			w += d.fonts.Advance(st.Font, part)
			return
		}
		a, _ := box.Upright(d.fonts, st.Font, part)
		w += a
	})
	return w
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

// piece is one fragment a selection may reach: fragment k of the run b
// of text node t, which shows n's text from its character shift on (a
// ::first-letter's, or n's own), page in the Document's space as its
// ancestors show it. r is it in the flow of its lines (flow), x where
// its line's frame starts along them, and q the point asked about there.
type piece struct {
	n, t  *Node
	shift int
	b     *box.Box
	k     int
	page  geom.Rect
	r     geom.Rect
	x     float32
	q     geom.Vec2
}

// flow is where v stands in the frame of the lines of writing mode m: x
// along them, y across them from where their blocks start, so that
// vertical lines are found and measured as horizontal ones.
func flow(m style.WritingMode, v geom.Vec2) geom.Vec2 {
	switch {
	case !m.Vertical():
		return v
	case m.Upward():
		return geom.V2(-v.Y, v.X)
	case m.BlockFromRight():
		return geom.V2(v.Y, -v.X)
	}
	return geom.V2(v.Y, v.X)
}

// unflow is the point of the Document's space that stands at v in the
// frame of the lines of writing mode m.
func unflow(m style.WritingMode, v geom.Vec2) geom.Vec2 {
	switch {
	case !m.Vertical():
		return v
	case m.Upward():
		return geom.V2(v.Y, -v.X)
	case m.BlockFromRight():
		return geom.V2(-v.Y, v.X)
	}
	return geom.V2(v.Y, v.X)
}

// flowRect is r in the frame of the lines of writing mode m.
func flowRect(m style.WritingMode, r geom.Rect) geom.Rect {
	a, b := flow(m, geom.V2(r.X, r.Y)), flow(m, geom.V2(r.X+r.W, r.Y+r.H))
	return geom.Rect{X: min(a.X, b.X), Y: min(a.Y, b.Y), W: abs(b.X - a.X), H: abs(b.Y - a.Y)}
}

// fragRect is fragment k of b in the Document's space, in the flow of its
// lines.
func fragRect(b *box.Box, k int) geom.Rect {
	at, f := b.Rect(), b.Lines[k].Rect
	return flowRect(b.Style.Mode, geom.Rect{X: at.X + f.X, Y: at.Y + f.Y, W: f.W, H: f.H})
}

// at is the place in n's text of the piece's character edge j.
func (p piece) at(d *Document, j int) Position {
	return Position{p.n, p.shift + int(d.mapOf(p.t).frags[p.k].offs[j])}
}

// pieces calls each for the fragments that take a selection and show,
// those inside scope when it is not nil, in the text's order, at the
// point at: a text's first letter before the rest of it.
func (d *Document) pieces(at geom.Vec2, scope *box.Box, each func(p piece)) {
	for _, n := range d.textRuns() {
		b := d.tree.Of[n]
		if !selects(b.Style) || scope != nil && !under(b, scope) {
			continue
		}
		if l, ok := d.letterOf(n); ok {
			d.runPieces(n, l.text, l.lead, l.run, at, each)
		}
		d.runPieces(n, n, 0, b, at, each)
	}
}

// runPieces calls each for the fragments of t's run b, which shows n's
// text from its character shift on.
func (d *Document) runPieces(n, t *Node, shift int, b *box.Box, at geom.Vec2, each func(p piece)) {
	pl := place(b)
	q := at
	if pl.turned {
		q = pl.xf.Inverse().Xform(at)
	}
	m := b.Style.Mode
	for k, f := range b.Lines {
		if f.Text == "" {
			continue
		}
		r := geom.Rect{X: pl.at.X + f.Rect.X, Y: pl.at.Y + f.Rect.Y, W: f.Rect.W, H: f.Rect.H}
		x := flowRect(m, r).X
		if pl.clipped {
			if r = intersect(r, pl.clip); r.W <= 0 || r.H <= 0 {
				continue
			}
		}
		each(piece{n: n, t: t, shift: shift, b: b, k: k, page: r, r: flowRect(m, r), x: x, q: flow(m, q)})
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
		return first.at(d, 0), true
	case bestY > 0 && best.q.Y >= bottom:
		return last.at(d, len(d.mapOf(last.t).frags[last.k].offs)-1), true
	}
	return d.positionIn(best, best.q.X), true
}

// positionIn is the character edge of p nearest x.
func (d *Document) positionIn(p piece, x float32) Position {
	xs := d.edges(d.mapOf(p.t), p.b, p.k)
	j := 0
	for i, e := range xs {
		if abs(p.x+e-x) < abs(p.x+xs[j]-x) {
			j = i
		}
	}
	return p.at(d, j)
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
// click takes it: across the text nodes the inline elements round p cut
// it in, as Chrome reads wo<b>rd</b> as one word.
func (d *Document) wordAt(p Position) (Position, Position) {
	return d.wordSpan(p, false)
}

// wordSpan is wordAt, with the spaces and tabs after the word up to the
// next line break when trailing is set.
func (d *Document) wordSpan(p Position, trailing bool) (Position, Position) {
	if d.mapOf(p.Node) == nil {
		return p, p
	}
	runs := d.textRuns()
	i := indexOf(runs, p.Node)
	lo, hi := i, i
	for i >= 0 && lo > 0 && d.joined(runs[lo-1], runs[lo]) {
		lo--
	}
	for i >= 0 && hi+1 < len(runs) && d.joined(runs[hi], runs[hi+1]) {
		hi++
	}
	nodes := []*Node{p.Node}
	if i >= 0 {
		nodes = runs[lo : hi+1]
	}
	var src []rune
	starts := make([]int, 0, len(nodes)+1)
	at := p.Offset
	for _, n := range nodes {
		if n == p.Node {
			at += len(src)
		}
		starts = append(starts, len(src))
		src = append(src, d.mapOf(n).src...)
	}
	starts = append(starts, len(src))
	place := func(k int, end bool) Position {
		j := 0
		for j+1 < len(nodes) && (k > starts[j+1] || !end && k == starts[j+1]) {
			j++
		}
		return Position{nodes[j], k - starts[j]}
	}
	from, state, rest := 0, -1, string(src)
	for len(rest) > 0 {
		var w string
		w, rest, state = uniseg.FirstWordInString(rest, state)
		n := utf8.RuneCountInString(w)
		if at < from+n || len(rest) == 0 {
			to := from + n
			for trailing && to < len(src) && (src[to] == ' ' || src[to] == '\t' || src[to] == ' ') {
				to++
			}
			return place(from, false), place(to, true)
		}
		from += n
	}
	return p, p
}

// joined reports whether the text runs a and b, b next after a, are one
// stretch of text: on the lines of one block, with only inline elements
// between them, no br, no picture, no control.
func (d *Document) joined(a, b *Node) bool {
	if container(d.tree.Of[a]) != container(d.tree.Of[b]) || !selects(d.tree.Of[b].Style) {
		return false
	}
	top := a.Parent()
	for ; top != nil && !nodeInside(b, top); top = top.Parent() {
		if !plainInline(top) {
			return false
		}
	}
	for e := b.Parent(); e != nil && e != top; e = e.Parent() {
		if !plainInline(e) {
			return false
		}
	}
	between := false
	for _, k := range top.Children() {
		switch {
		case nodeInside(a, k):
			between = true
		case nodeInside(b, k):
			return true
		case between && !emptyInline(k):
			return false
		}
	}
	return true
}

// plainInline reports an element that lays its content out on its
// parent's lines: display inline, no br, no picture, no control.
func plainInline(e *Node) bool {
	st := dom.Data(e).Style
	if st == nil || st.Display != style.Inline && st.Display != style.Contents || flexItem(e) {
		return false
	}
	switch e.Tag() { //nolint:exhaustive // the inline elements that break a stretch of text
	case dom.Br, dom.Img, dom.Svg, dom.Input, dom.Textarea, dom.Select, dom.Button, dom.Canvas, dom.Iframe:
		return false
	}
	_, pictured := dom.ContentPicture(e)
	return !pictured
}

// emptyInline reports a node between two text runs that leaves them one
// stretch: text laid out as nothing, a plain inline element holding only
// such, or what display: none hides.
func emptyInline(n *Node) bool {
	if n.IsText() {
		return true
	}
	if st := dom.Data(n).Style; st != nil && st.Display == style.DisplayNone {
		return true
	}
	if !plainInline(n) {
		return false
	}
	for _, k := range n.Children() {
		if !emptyInline(k) {
			return false
		}
	}
	return true
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
		if l, ok := d.letterOf(n); ok && from < l.end && to > l.lead {
			a, z := max(from, l.lead)-l.lead, min(to, l.end)-l.lead
			if ps := d.partsOf(l.text, l.run, nil, a, z, false); len(ps) > 0 {
				s.parts[l.run] = ps
				s.order = append(s.order, l.run)
			}
		}
		if ps := d.partsOf(n, b, next, from, to, n != end.Node); len(ps) > 0 {
			s.parts[b] = ps
			s.order = append(s.order, b)
		}
	}
	marked := false
	for _, n := range s.breaks {
		if !treeBefore(start.Node, n) || !treeBefore(n, end.Node) {
			continue
		}
		if b := d.tree.Of[n]; selects(b.Style) && d.aloneOnLine(b) {
			s.parts[b] = []paint.Selected{d.breakMark(b)}
			s.order, marked = append(s.order, b), true
		}
	}
	if marked {
		slices.SortStableFunc(s.order, func(a, b *box.Box) int {
			switch {
			case treeBefore(a.Node, b.Node):
				return -1
			case treeBefore(b.Node, a.Node):
				return 1
			}
			return 0
		})
	}
	return s.parts
}

// aloneOnLine reports whether the br b holds a line no text of its block
// is on: an empty line, which a selection going past it marks.
func (d *Document) aloneOnLine(b *box.Box) bool {
	runs := d.sel.runs
	i := sort.Search(len(runs), func(i int) bool { return treeBefore(b.Node, runs[i]) }) - 1
	if i < 0 {
		return true
	}
	prev := d.tree.Of[runs[i]]
	if container(prev) != container(b) {
		return true
	}
	at, f := prev.Rect(), prev.Lines[len(prev.Lines)-1]
	r := b.Rect()
	return at.Y+f.Rect.Y >= r.Y+r.H || r.Y >= at.Y+f.Rect.Y+f.Rect.H
}

// breakMark is the mark of the empty line the br b holds, in its box's
// space: a space wide from the line's start, as high as the line, as
// Chrome draws a selected line break.
func (d *Document) breakMark(b *box.Box) paint.Selected {
	m := d.fonts.Metrics(b.Style.Font)
	h, base := layout.LineHeight(m, b.Style)
	top := m.Ascent - base
	return paint.Selected{Rect: geom.Rect{Y: top, W: d.fonts.Advance(b.Style.Font, " "), H: h}}
}

// letter is the part of a text node a ::first-letter box shows: the
// letter's own text node and its run, and where it stands in the node's
// text, from lead, after the white space it drops, up to end.
type letter struct {
	text      *Node
	run       *box.Box
	lead, end int
}

// letterOf is the first letter of n's text, which a ::first-letter box
// shows apart from n's run: a selection of n's text takes it too.
func (d *Document) letterOf(n *Node) (letter, bool) {
	data := dom.Data(n)
	g := data.Letter
	if g == nil || dom.Data(g).Letter != n || len(g.Children()) != 1 {
		return letter{}, false
	}
	shown := d.shown(n)
	if data.LetterEnd > len(shown) {
		return letter{}, false
	}
	t := g.Children()[0]
	b := d.tree.Of[t]
	if b == nil || b.Kind != box.TextRun || len(b.Lines) == 0 || !strings.HasSuffix(shown[:data.LetterEnd], t.Text()) {
		return letter{}, false
	}
	end := utf8.RuneCountInString(shown[:data.LetterEnd])
	return letter{text: t, run: b, lead: end - utf8.RuneCountInString(t.Text()), end: end}, true
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
	mode := b.Style.Mode
	ws := b.Style.WhiteSpace
	keeps := ws.KeepsBreaks() && !mode.Vertical()
	var out []paint.Selected
	for k, f := range b.Lines {
		if k > 0 && keeps {
			out = append(out, d.emptyLines(n, b, k, from, to)...)
		}
		x0, x1, goesOn := float32(0), f.Rect.W, true
		if mode.Vertical() {
			x1 = f.Rect.H
		}
		tabs, t0, t1 := d.tabRoom(m, b, k, from, to)
		if !whole {
			offs := m.frags[k].offs
			j0, j1 := lower(offs, from), lower(offs, to)
			if j1 <= j0 && !tabs {
				continue
			}
			xs := d.edges(m, b, k)
			x0, x1 = min(xs[j0], xs[j1]), max(xs[j0], xs[j1])
			goesOn = j1 == len(offs)-1 && (on || int(offs[j1]) < to)
		}
		switch {
		case !goesOn:
		case tabs && x1 > x0:
			x1 = t1
		case tabs:
			x0, x1 = t0, t1
		case f.RTL:
			x0 -= d.tail(b, k, next)
		default:
			x1 += d.tail(b, k, next)
		}
		if x1 <= x0 {
			continue
		}
		if mode.Vertical() {
			// the part in its line's frame, where the line runs across, turned onto the page
			base := paint.LineBaseline(mode, f)
			top, bottom := min(0, base-metrics.Ascent), max(f.Rect.W, base+metrics.Descent)
			r := paint.TurnRect(paint.LineTurn(mode, f.Rect), geom.Rect{X: x0, Y: top, W: x1 - x0, H: bottom - top})
			out = append(out, paint.Selected{Line: k, Rect: r})
			continue
		}
		top := min(f.Rect.Y, f.Baseline-metrics.Ascent)
		bottom := max(f.Rect.Y+f.Rect.H, f.Baseline+metrics.Descent)
		out = append(out, paint.Selected{Line: k, Rect: geom.Rect{X: f.Rect.X + x0, Y: top, W: x1 - x0, H: bottom - top}})
	}
	return out
}

// tabRoom is the room the tabs from to to take between fragment k of b
// and the next on its line, in the fragment's space, when tabs alone
// stand between them in text that keeps them: each reaches the next tab
// stop, the first from the fragment's end.
func (d *Document) tabRoom(m *runMap, b *box.Box, k, from, to int) (bool, float32, float32) {
	st, f := b.Style, b.Lines[k]
	if m == nil || f.RTL || st.WhiteSpace.Collapse() != style.Preserve || st.Mode.Vertical() {
		return false, 0, 0
	}
	var after *box.Fragment
	for j := range b.Lines {
		if g := &b.Lines[j]; g.Start > f.Start && abs(g.Rect.Y-f.Rect.Y) < 0.5 && (after == nil || g.Start < after.Start) {
			after = g
		}
	}
	offs := m.frags[k].offs
	first := int(offs[len(offs)-1])
	if after == nil || int(after.Start) <= first {
		return false, 0, 0
	}
	n := int(after.Start) - first
	for _, r := range m.src[first:after.Start] {
		if r != '\t' {
			return false, 0, 0
		}
	}
	a, z := max(from, first)-first, min(to, int(after.Start))-first
	if z <= a {
		return false, 0, 0
	}
	stop := st.TabStop(d.fonts.Advance(st.Font, " "))
	end := after.Rect.X - f.Rect.X
	edge := func(j int) float32 { return max(f.Rect.W, end-float32(n-j)*stop) }
	return true, edge(a), edge(z)
}

// emptyLines are the marks of the empty lines between fragments k-1 and
// k of n's run b, whose white space keeps its line breaks, that the
// selection from to to goes past: a space wide from the line's start, as
// high as the line. Their Line is -1, a fragment of none.
func (d *Document) emptyLines(n *Node, b *box.Box, k, from, to int) []paint.Selected {
	m := d.mapOf(n)
	prev, next := m.frags[k-1].offs, m.frags[k].offs
	var breaks []int
	for i := int(prev[len(prev)-1]); i < int(next[0]); i++ {
		if m.src[i] == '\n' {
			breaks = append(breaks, i)
		}
	}
	if len(breaks) < 2 {
		return nil
	}
	above, below := b.Lines[k-1].Rect, b.Lines[k].Rect
	step := (below.Y - above.Y) / float32(len(breaks))
	var out []paint.Selected
	for j := 1; j < len(breaks); j++ {
		if breaks[j] >= from && breaks[j] < to {
			out = append(out, paint.Selected{Line: -1, Rect: geom.Rect{X: below.X, Y: above.Y + step*float32(j),
				W: d.fonts.Advance(b.Style.Font, " "), H: step}})
		}
	}
	return out
}

// tail is how far past its end fragment k of b is selected when the
// selection goes on: up to the fragment after it on its line, the gap a
// tab or a padding leaves, nothing where that fragment shows on the
// other side, its text of the other direction, or a space past the
// line's end, on its left for a fragment shown right to left.
func (d *Document) tail(b *box.Box, k int, next *box.Box) float32 {
	f, fr := b.Lines[k], fragRect(b, k)
	line := func(g geom.Rect) bool { return abs(g.Y-fr.Y) < 0.5 }
	var after *box.Fragment
	var ar geom.Rect
	for j := range b.Lines {
		if g := &b.Lines[j]; g.Start > f.Start && (after == nil || g.Start < after.Start) {
			if r := fragRect(b, j); line(r) {
				after, ar = g, r
			}
		}
	}
	if after == nil && next != nil && len(next.Lines) > 0 && container(next) == container(b) &&
		line(fragRect(next, 0)) {
		after, ar = &next.Lines[0], fragRect(next, 0)
	}
	if after == nil {
		return d.fonts.Advance(b.Style.Font, " ")
	}
	if f.RTL {
		return max(fr.X-(ar.X+ar.W), 0)
	}
	return max(ar.X-(fr.X+fr.W), 0)
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
	if l, ok := c.d.letterOf(n); ok {
		c.dropped(m.src, from, min(l.lead, to), b.Style.WhiteSpace, false)
		if a, z := max(from, l.lead), min(to, l.end); z > a {
			if c.space && c.last != 0 && !isBlank(c.last) {
				c.put(' ')
			}
			for _, r := range m.src[a:z] {
				c.put(r)
			}
		}
		at = l.end
	}
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
	keeps := ws.KeepsBreaks()
	for _, r := range src[from:to] {
		switch {
		case r == '\n' && keeps:
			c.put('\n')
		case r == '\t' && ws.Collapse() == style.Preserve:
			c.put('\t')
		case isBlank(r) && wrap:
			if c.last != ' ' {
				c.put(' ')
			}
		case isBlank(r):
			c.space = c.last != 0 && c.last != '\n'
		}
	}
}
