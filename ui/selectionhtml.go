package ui

import (
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/lantervale/ember/ui/layout"
)

// SelectionHTML is the selection's markup, what a copy puts on the
// clipboard beside its text where the system's takes HTML, written as
// Chrome's copy writes it (decision 0224): the nodes it holds, an element
// held in part with what it holds of it, each element with its
// attributes and, in its style attribute, what its rules and its own
// style declare; the text and the elements at the top with the text
// styles they show in, so that a paste looks as the page did; round it
// the element whose structure or look the paste needs (the list of a
// whole item, the table of a row, a pre, a heading, the bold or the link
// it stands in); a <br class="Apple-interchange-newline"> where it
// begins or ends with a paragraph break. It is empty while nothing is
// selected.
func (d *Document) SelectionHTML() string {
	if d.sel.anchor == d.sel.focus || d.sel.anchor.Node == nil || d.tree == nil {
		return ""
	}
	start, end := d.ordered(d.sel.anchor, d.sel.focus)
	start, end = d.forward(start), d.backward(end)
	if !before(start, end) {
		return ""
	}
	s := &serializer{d: d, start: start, end: end}
	first := start.Node
	if d.paragraphBreakAfter(start) {
		s.newline()
		next, ok := d.nextRun(start.Node)
		if !ok || !before(Position{Node: next}, end) {
			return s.out.String()
		}
		first = next
	}
	pastEnd := d.pastEnd(end)
	common := commonAncestor(start.Node, end.Node)
	s.highest = d.wrapAncestor(common, start, end)
	if s.highest == nil {
		s.highest = (&serializer{d: d, start: start, end: end, dry: true}).traverse(first, pastEnd)
	}
	if s.highest != nil && s.highest.Parent() != nil {
		s.wrapping = d.editingStyle(s.highest.Parent())
	}
	last := s.traverse(first, pastEnd)
	if s.highest != nil && last != nil && last != s.highest && nodeInside(last, s.highest) {
		for a := last.Parent(); a != nil; a = a.Parent() {
			st := s.style(a)
			st.remove("float")
			s.wrap(a, st)
			if a == s.highest {
				break
			}
		}
	}
	if (last == nil || last.Tag() != dom.Br) && d.paragraphBreakBefore(end) {
		s.newline()
	}
	return s.out.String()
}

// serializer writes nodes as Chrome's StyledMarkupSerializer does.
type serializer struct {
	d          *Document
	start, end Position
	// dry walks without writing, to find the highest node written.
	dry bool
	// highest is the highest node written, its siblings written with
	// wrapping, the text styles of its parent.
	highest  *Node
	wrapping block
	out      strings.Builder
}

// newline writes the break Chrome marks a paragraph's end with.
func (s *serializer) newline() { s.out.WriteString(`<br class="Apple-interchange-newline">`) }

// traverse writes the nodes from first up to pastEnd and returns the
// last one it closed, the highest when the walk leaves an element it
// never opened.
func (s *serializer) traverse(first, pastEnd *Node) *Node {
	var open []*Node
	var last *Node
	for n := first; n != nil && n != pastEnd; {
		next := nextNode(n)
		switch {
		case !n.IsText() && s.d.isBlock(n) && next == pastEnd:
			// a block the end stands at the start of
			n = next
			continue
		case !s.shown(n):
			next = nextSkipping(n)
			if pastEnd != nil && nodeInside(pastEnd, n) {
				next = pastEnd
			}
		default:
			s.startTag(n)
			if !n.IsText() && len(n.Children()) > 0 {
				open = append(open, n)
				n = next
				continue
			}
			s.endTag(n)
			last = n
		}
		if sibling(n) != nil && next != pastEnd {
			n = next
			continue
		}
		for len(open) > 0 {
			a := open[len(open)-1]
			if next != nil && next != pastEnd && nodeInside(next, a) {
				break
			}
			s.endTag(a)
			last = a
			open = open[:len(open)-1]
		}
		var nextParent *Node
		if next != nil {
			nextParent = next.Parent()
		}
		if next == pastEnd || n == nextParent {
			n = next
			continue
		}
		from := n
		if last != nil && nodeInside(n, last) {
			from = last
		}
		for p := from.Parent(); p != nil && p != nextParent; p = p.Parent() {
			if !s.shown(p) {
				continue
			}
			s.wrap(p, s.style(p))
			last = p
		}
		n = next
	}
	return last
}

// shown reports a node layout gave a box.
func (s *serializer) shown(n *Node) bool {
	if n.IsText() {
		return s.d.tree.Of[n] != nil
	}
	st := dom.Data(n).Style
	return st != nil && st.Display != style.DisplayNone
}

func (s *serializer) wraps(n *Node) bool {
	return s.wrapping != nil && s.highest != nil && s.highest.Parent() == n.Parent()
}

func (s *serializer) startTag(n *Node) {
	if s.dry {
		return
	}
	if n.IsText() {
		s.text(n)
		return
	}
	s.openTag(&s.out, n, s.style(n))
}

func (s *serializer) endTag(n *Node) {
	if s.dry || n.IsText() || n.Tag().Void() {
		return
	}
	s.out.WriteString("</" + n.Tag().String() + ">")
}

// wrap puts e's tags round what is written so far.
func (s *serializer) wrap(e *Node, st block) {
	if s.dry {
		return
	}
	var open strings.Builder
	s.openTag(&open, e, st)
	written := s.out.String()
	s.out.Reset()
	s.out.WriteString(open.String() + written)
	if !e.Tag().Void() {
		s.out.WriteString("</" + e.Tag().String() + ">")
	}
}

func (s *serializer) openTag(out *strings.Builder, e *Node, st block) {
	out.WriteByte('<')
	out.WriteString(e.Tag().String())
	write := func(name, value string) {
		out.WriteString(" " + name + `="` + escapeHTML(value, true) + `"`)
	}
	if id := e.ID(); id != "" {
		write("id", id)
	}
	if c := e.Classes(); len(c) > 0 {
		write("class", strings.Join(c, " "))
	}
	for _, a := range e.Attrs() {
		write(a.Name, a.Value)
	}
	if len(st) > 0 {
		write(styleAttr, st.text())
	}
	out.WriteByte('>')
}

// text writes what the selection holds of n's text as it shows, cased
// as written, in a span of the wrapping styles where n stands beside
// the highest node.
func (s *serializer) text(n *Node) {
	from, to := 0, -1
	if n == s.start.Node {
		from = s.start.Offset
	}
	if n == s.end.Node {
		to = s.end.Offset
	}
	shown := s.d.renderedText(n, from, to)
	wrapped := s.wraps(n)
	if wrapped {
		st := s.wrapping.clone()
		st.set(decl{name: "display", value: "inline", important: true})
		st.set(decl{name: "float", value: cssNone})
		s.out.WriteString(`<span style="` + escapeHTML(st.text(), true) + `">`)
	}
	text := escapeHTML(shown, false)
	if b := s.d.tree.Of[n]; b == nil || b.Style == nil || !b.Style.WhiteSpace.KeepsSpaces() {
		text = keepEdgeSpaces(text, n == s.start.Node, n == s.end.Node)
	}
	s.out.WriteString(text)
	if wrapped {
		s.out.WriteString("</span>")
	}
}

// keepEdgeSpaces puts a space where the selection begins or ends in a
// span of its own, which a paste does not collapse.
func keepEdgeSpaces(text string, starts, ends bool) string {
	const kept = "<span>\u00a0</span>"
	if starts && strings.HasPrefix(text, " ") {
		text = kept + text[1:]
	}
	if ends && strings.HasSuffix(text, " ") {
		text = text[:len(text)-1] + kept
	}
	return text
}

// renderedText is n's text from the character from to the character to
// (-1 for its end) as layout shows it, collapsed and wrapped, its case as
// written.
func (d *Document) renderedText(n *Node, from, to int) string {
	if to < 0 {
		to = 1 << 30
	}
	c := copier{d: d, start: Position{Node: n, Offset: from}, end: Position{Node: n, Offset: to}}
	if b := d.tree.Of[n]; b != nil && b.Style != nil && b.Style.TextTransform != style.NoTransform {
		uncased := *b.Style
		uncased.TextTransform = style.NoTransform
		if m := d.mapOf(n); m != nil {
			if src := []rune(layout.Shown(n, &uncased, d.tr())); len(src) == len(m.src) {
				c.source = src
			}
		}
	}
	c.text(n)
	return c.out.String()
}

// isBlock reports an element Chrome's editing takes as a block.
func (d *Document) isBlock(e *Node) bool {
	return !e.IsText() && dom.Data(e).Style != nil && !inlineLevel(e)
}

// enclosingBlock is the block n stands in, n itself where it is one.
func (d *Document) enclosingBlock(n *Node) *Node {
	for ; n != nil; n = n.Parent() {
		if d.isBlock(n) {
			return n
		}
	}
	return nil
}

// wrapAncestor is the element round the selection that a paste needs
// for its structure or its look, nil for none: the table of a row, a
// list, table, pre or heading the selection stands in, the list of an
// item it holds whole, then the highest bold, italic, underline or
// strike-through up to the block, and the link round that.
func (d *Document) wrapAncestor(common *Node, start, end Position) *Node {
	var special *Node
	if b := d.enclosingBlock(common); b != nil {
		switch b.Tag() { //nolint:exhaustive // the blocks a paste keeps
		case dom.Tbody, dom.Tr:
			special = closest(b.Parent(), dom.Table)
		case dom.Ol, dom.Ul, dom.Pre, dom.Table, dom.H1, dom.H2, dom.H3, dom.H4, dom.H5, dom.H6:
			special = b
		}
	}
	if li := closest(start.Node, dom.Li); li != nil {
		if first, last, ok := d.textIn(li); ok && first == start && last == end {
			for a := li.Parent(); a != nil; a = a.Parent() {
				if t := a.Tag(); t == dom.Ul || t == dom.Ol {
					special = a
					break
				}
			}
		}
	}
	check := common
	if special != nil {
		check = special
	}
	block := d.enclosingBlock(check)
	for a := check; a != nil && a != block; a = a.Parent() {
		if presentational(a) {
			special = a
		}
	}
	from := common
	if special != nil {
		from = special
	}
	if a := closest(from, dom.A); a != nil {
		special = a
	}
	return special
}

// presentational reports an element of bold, italic, underline or
// strike-through.
func presentational(e *Node) bool {
	switch e.Tag() { //nolint:exhaustive // the presentational elements
	case dom.U, dom.S, dom.I, dom.Em, dom.B, dom.Strong:
		return true
	}
	return false
}

// closest is n or the nearest element above it of tag t.
func closest(n *Node, t dom.Tag) *Node {
	for ; n != nil; n = n.Parent() {
		if n.Tag() == t {
			return n
		}
	}
	return nil
}

// forward moves a place at the end of a text node's shown text to the
// start of the next one of its block, where Chrome's normalised range
// begins.
func (d *Document) forward(p Position) Position {
	if !d.atEnd(p) {
		return p
	}
	if next, ok := d.nextRun(p.Node); ok && d.enclosingBlock(next) == d.enclosingBlock(p.Node) {
		return Position{Node: next}
	}
	return p
}

// backward moves a place at the start of a text node's shown text to the
// end of the one before it in its block.
func (d *Document) backward(p Position) Position {
	if !d.atStart(p) {
		return p
	}
	if prev, ok := d.prevRun(p.Node); ok && d.enclosingBlock(prev) == d.enclosingBlock(p.Node) {
		return Position{Node: prev, Offset: len(d.mapOf(prev).src)}
	}
	return p
}

// atStart reports a place before any character its text node shows.
func (d *Document) atStart(p Position) bool { return d.renderedText(p.Node, 0, p.Offset) == "" }

// atEnd reports a place after every character its text node shows.
func (d *Document) atEnd(p Position) bool { return d.renderedText(p.Node, p.Offset, -1) == "" }

// paragraphBreakAfter reports a place at the end of its paragraph with
// another after it, where Chrome's copy begins with a newline.
func (d *Document) paragraphBreakAfter(p Position) bool {
	if !d.atEnd(p) {
		return false
	}
	next, ok := d.nextRun(p.Node)
	return ok && d.enclosingBlock(next) != d.enclosingBlock(p.Node) && !d.brBetween(p.Node, next)
}

// paragraphBreakBefore reports a place at the start of its paragraph
// with another before it, where Chrome's copy ends with a newline.
func (d *Document) paragraphBreakBefore(p Position) bool {
	if !d.atStart(p) {
		return false
	}
	prev, ok := d.prevRun(p.Node)
	return ok && d.enclosingBlock(prev) != d.enclosingBlock(p.Node) && !d.brBetween(prev, p.Node)
}

// brBetween reports a br laid out between the text nodes a and b.
func (d *Document) brBetween(a, b *Node) bool {
	for n := nextNode(a); n != nil && n != b; n = nextNode(n) {
		if n.Tag() == dom.Br {
			return true
		}
	}
	return false
}

// pastEnd is the first node after the selection's end: the text node it
// ends at the start of, lifted out of the inline elements it opens, else
// the node after the text node it ends in.
func (d *Document) pastEnd(end Position) *Node {
	if !d.atStart(end) {
		return nextSkipping(end.Node)
	}
	n := end.Node
	for n.Parent() != nil && n.Index() == 0 && !d.isBlock(n.Parent()) {
		n = n.Parent()
	}
	return n
}

// nextRun and prevRun are the text nodes laid out after and before n.
func (d *Document) nextRun(n *Node) (*Node, bool) {
	runs := d.textRuns()
	for i, r := range runs {
		if r == n && i+1 < len(runs) {
			return runs[i+1], true
		}
	}
	return nil, false
}

func (d *Document) prevRun(n *Node) (*Node, bool) {
	runs := d.textRuns()
	for i, r := range runs {
		if r == n && i > 0 {
			return runs[i-1], true
		}
	}
	return nil, false
}

// nextNode is the node after n in tree order: its first child, else the
// next of nextSkipping.
func nextNode(n *Node) *Node {
	if k := n.Children(); len(k) > 0 && !n.IsText() {
		return k[0]
	}
	return nextSkipping(n)
}

// nextSkipping is the node after n and its descendants.
func nextSkipping(n *Node) *Node {
	for ; n != nil; n = n.Parent() {
		if s := sibling(n); s != nil {
			return s
		}
	}
	return nil
}

// sibling is the node after n under its parent.
func sibling(n *Node) *Node {
	p := n.Parent()
	if p == nil {
		return nil
	}
	k := p.Children()
	if i := n.Index(); i >= 0 && i+1 < len(k) {
		return k[i+1]
	}
	return nil
}

// commonAncestor is the deepest node holding both a and b, themselves
// included.
func commonAncestor(a, b *Node) *Node {
	for n := a; n != nil; n = n.Parent() {
		if nodeInside(b, n) {
			return n
		}
	}
	return nil
}

const styleAttr = "style"

// escapeHTML escapes s as Chrome's copy does, an attribute's value or
// text.
func escapeHTML(s string, attr bool) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '&':
			b.WriteString("&amp;")
		case r == ' ' && attr:
			b.WriteString("&nbsp;")
		case r == '"' && attr:
			b.WriteString("&quot;")
		case r == '<' && !attr:
			b.WriteString("&lt;")
		case r == '>' && !attr:
			b.WriteString("&gt;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
