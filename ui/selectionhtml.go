package ui

import "strings"

// SelectionHTML is the selection's markup, what a copy puts on the
// clipboard beside its text where the system's takes HTML (decision
// 0162): the nodes it holds, the elements it holds in part cut to it, as
// the DOM's Range.cloneContents makes them, inside the inline elements
// round it up to its block, so that a word in bold pastes bold and a
// link as a link. Each element keeps its own attributes, and no computed
// style is written. It is empty while nothing is selected.
func (d *Document) SelectionHTML() string {
	if d.sel.anchor == d.sel.focus || d.sel.anchor.Node == nil {
		return ""
	}
	start, end := d.ordered(d.sel.anchor, d.sel.focus)
	m := markup{d: d, start: start, end: end}
	common := commonAncestor(start.Node, end.Node)
	var round []*Node
	e := common
	if e.IsText() {
		e = e.Parent()
	}
	for ; e != nil && e != d.body && e != d.html && inlineLevel(e); e = e.Parent() {
		round = append(round, e)
	}
	for i := len(round) - 1; i >= 0; i-- {
		m.open(round[i])
	}
	if common.IsText() {
		m.node(common)
	} else {
		for _, k := range common.Children() {
			if m.done {
				break
			}
			m.node(k)
		}
	}
	for _, e := range round {
		m.close(e)
	}
	return m.out.String()
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

// markup writes the nodes from start to end as the HTML serialisation
// writes them, an element on the way in part.
type markup struct {
	d          *Document
	start, end Position
	out        strings.Builder
	began      bool
	done       bool
}

func (m *markup) node(n *Node) {
	if n.IsText() {
		m.text(n)
		return
	}
	if !m.began && !nodeInside(m.start.Node, n) {
		return
	}
	m.open(n)
	if n.Tag().Void() {
		return
	}
	for _, k := range n.Children() {
		if m.done {
			break
		}
		m.node(k)
	}
	m.close(n)
}

// text writes the part of n's text the selection holds, counted as its
// positions count, in the node's own characters where layout's shown
// text keeps their number.
func (m *markup) text(n *Node) {
	from, to := 0, -1
	if n == m.start.Node {
		m.began, from = true, m.start.Offset
	}
	if n == m.end.Node {
		m.done, to = true, m.end.Offset
	}
	if !m.began {
		return
	}
	text := []rune(n.Text())
	if r := m.d.mapOf(n); r != nil && len(r.src) != len(text) {
		text = r.src
	}
	if to < 0 || to > len(text) {
		to = len(text)
	}
	if from < to {
		m.out.WriteString(escapeHTML(string(text[from:to]), false))
	}
}

func (m *markup) open(e *Node) {
	m.out.WriteByte('<')
	m.out.WriteString(e.Tag().String())
	write := func(name, value string) {
		m.out.WriteString(" " + name + `="` + escapeHTML(value, true) + `"`)
	}
	if id := e.ID(); id != "" {
		write("id", id)
	}
	if c := e.Classes(); len(c) > 0 {
		write("class", strings.Join(c, " "))
	}
	if s := e.Style(); s != "" {
		write(styleAttr, s)
	}
	for _, a := range e.Attrs() {
		write(a.Name, a.Value)
	}
	m.out.WriteByte('>')
}

func (m *markup) close(e *Node) {
	if !e.Tag().Void() {
		m.out.WriteString("</" + e.Tag().String() + ">")
	}
}

const styleAttr = "style"

// escapeHTML escapes s as the HTML serialisation does, an attribute's
// value or text.
func escapeHTML(s string, attr bool) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '&':
			b.WriteString("&amp;")
		case r == ' ':
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
