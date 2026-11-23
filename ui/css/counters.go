package css

import (
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// listItem is the counter every list item counts on.
const listItem = "list-item"

// counter is one counter in scope: made on an element depth deep, it
// reaches that element's descendants and the siblings after it with
// theirs. step is what a list item adds to a list-item counter: -1 in a
// reversed ol.
type counter struct {
	name  string
	depth int
	value int
	step  int
}

// counting walks the tree in document order with the counters in scope
// and the depth of the quotes open, as CSS Lists and Generated Content
// have them.
type counting struct {
	r     *restyler
	stack []counter
	// quotes is how many quotes are open.
	quotes int
}

// count works out, once the tree is styled, what stands where an element
// is in the document: each list item's number and marker, and the text
// of the generated content that reads counters or quotes. It runs only
// for a tree in which something counts, and only after a change that can
// move a counter or restyles the whole tree; what it changes is flagged
// for layout.
func (r *restyler) count(root *dom.Node) {
	data := dom.Data(root)
	if r.counts && !data.Counts {
		data.Counts, r.recount = true, true
	}
	if !data.Counts || !r.recount {
		return
	}
	c := counting{r: r}
	c.element(root, 0)
}

func (c *counting) element(n *dom.Node, depth int) {
	data := dom.Data(n)
	st := data.Style
	if st == nil || st.Display == style.DisplayNone {
		return
	}
	c.apply(n, st, depth)
	if st.ListItem {
		c.item(n, data, st)
	}
	if g := data.Before; g != nil {
		c.generated(n, g, depth+1)
	}
	for _, k := range n.Children() {
		if !k.IsText() {
			c.element(k, depth+1)
		}
	}
	if g := data.After; g != nil {
		c.generated(n, g, depth+1)
	}
	for len(c.stack) > 0 && c.stack[len(c.stack)-1].depth > depth {
		c.stack = c.stack[:len(c.stack)-1]
	}
}

// generated counts a ::before or ::after box, a child of its element, and
// gives it its text.
func (c *counting) generated(origin, g *dom.Node, depth int) {
	gd := dom.Data(g)
	st := gd.Style
	c.apply(nil, st, depth)
	if st.ListItem {
		c.item(g, gd, st)
	}
	if c.r.fill(g, st, c.text(st)) {
		flagLayout(origin)
	}
}

// item gives a list item its number and its ::marker its text: its
// content's, or the marker string of its list-style-type.
func (c *counting) item(n *dom.Node, data *dom.Pipeline, st *style.Computed) {
	ordinal := 0
	if i := c.find(listItem); i >= 0 {
		ordinal = c.stack[i].value
	}
	text, has := "", false
	switch m := data.Marker; {
	case m != nil && m.Content.Kind != style.ContentNormal:
		text, has = contentText(m.Content, c.text(m)), true
	case st.ListStyle == style.ListCounter:
		text, has = c.r.counterStyles().marker(st.ListName, ordinal), true
	case st.ListStyle == style.ListString:
		text, has = st.ListName, true
	}
	if data.Ordinal == ordinal && data.Counted && data.MarkerText == text && data.HasMarkerText == has {
		return
	}
	shown := data.MarkerText != text || data.HasMarkerText != has
	data.Ordinal, data.Counted, data.MarkerText, data.HasMarkerText = ordinal, true, text, has
	if shown {
		flagLayout(n)
	}
}

// flagLayout flags n for layout after the tree was styled.
func flagLayout(n *dom.Node) {
	dom.Data(n).Dirty |= dom.DirtyLayout
	for p := n.Parent(); p != nil; p = p.Parent() {
		dom.Data(p).Dirty |= dom.DirtyDescendant
	}
}

// apply runs a box's counter properties, in CSS's order: resets, then
// increments, then sets. n is the element, nil for a pseudo-element's
// box: a list resets list-item, and a list item adds to it, unless the
// property names that counter itself.
func (c *counting) apply(n *dom.Node, st *style.Computed, depth int) {
	if n != nil && isList(n.Tag()) && !names(st.CounterReset, listItem) {
		value, step := listStart(n)
		c.reset(listItem, value, depth)
		c.stack[c.find(listItem)].step = step
	}
	for _, r := range st.CounterReset {
		c.reset(r.Name, r.Value, depth)
	}
	if st.ListItem && n != nil && !names(st.CounterIncrement, listItem) {
		at := c.touch(listItem, depth)
		step := c.stack[at].step
		if step == 0 {
			step = 1
		}
		c.stack[at].value += step
	}
	for _, r := range st.CounterIncrement {
		c.stack[c.touch(r.Name, depth)].value += r.Value
	}
	if n != nil && n.Tag() == dom.Li && st.ListItem {
		if v, ok := intAttr(n, "value"); ok {
			c.stack[c.touch(listItem, depth)].value = v
		}
	}
	for _, r := range st.CounterSet {
		c.stack[c.touch(r.Name, depth)].value = r.Value
	}
}

// reset makes a counter on a box depth deep: a new one inside those of
// its name in scope, or, where a sibling before it made the innermost,
// that one again.
func (c *counting) reset(name string, value, depth int) {
	if i := c.find(name); i >= 0 && c.stack[i].depth == depth {
		c.stack[i].value, c.stack[i].step = value, 0
		return
	}
	c.stack = append(c.stack, counter{name: name, depth: depth, value: value})
}

// touch is the innermost counter name's place, made at 0 when none is
// in scope.
func (c *counting) touch(name string, depth int) int {
	if i := c.find(name); i >= 0 {
		return i
	}
	c.stack = append(c.stack, counter{name: name, depth: depth})
	return len(c.stack) - 1
}

func (c *counting) find(name string) int {
	for i := len(c.stack) - 1; i >= 0; i-- {
		if c.stack[i].name == name {
			return i
		}
	}
	return -1
}

func names(list []style.Counter, name string) bool {
	for _, c := range list {
		if c.Name == name {
			return true
		}
	}
	return false
}

func isList(tag dom.Tag) bool { return tag == dom.Ol || tag == dom.Ul || tag == dom.Menu }

// listStart is what a list resets list-item to, and what its items add:
// one under its start attribute, counting up, or one over it counting
// down for a reversed ol, which starts from the number of its items
// when it has no start.
func listStart(list *dom.Node) (int, int) {
	start, given := intAttr(list, "start")
	if _, reversed := list.Attr("reversed"); reversed && list.Tag() == dom.Ol {
		if !given {
			start = listItems(list)
		}
		return start + 1, -1
	}
	if !given || list.Tag() != dom.Ol {
		start = 1
	}
	return start - 1, 1
}

// listItems counts the list items of list, not those of the lists inside
// it.
func listItems(list *dom.Node) int {
	count := 0
	for _, k := range list.Children() {
		st := dom.Data(k).Style
		if k.IsText() || st == nil || st.Display == style.DisplayNone {
			continue
		}
		if st.ListItem {
			count++
		}
		if !isList(k.Tag()) {
			count += listItems(k)
		}
	}
	return count
}

func intAttr(n *dom.Node, name string) (int, bool) {
	v, ok := n.Attr(name)
	if !ok {
		return 0, false
	}
	i, err := strconv.Atoi(strings.TrimSpace(v))
	return i, err == nil
}

// text gives each counter and quote of a box's content its text, as the
// counters and the quotes stand at the box; st is the box's style, whose
// quotes it reads.
func (c *counting) text(st *style.Computed) func(style.ContentItem) string {
	return func(it style.ContentItem) string {
		switch it.Kind {
		case style.ContentCounter:
			value := 0
			if i := c.find(it.Name); i >= 0 {
				value = c.stack[i].value
			}
			return c.represent(it.Style, value)
		case style.ContentCounters:
			var parts []string
			for _, k := range c.stack {
				if k.name == it.Name {
					parts = append(parts, c.represent(it.Style, k.value))
				}
			}
			if parts == nil {
				parts = []string{c.represent(it.Style, 0)}
			}
			return strings.Join(parts, it.Text)
		case style.ContentOpenQuote, style.ContentNoOpenQuote:
			c.quotes++
			if it.Kind == style.ContentOpenQuote {
				return quote(st, c.quotes-1, 0)
			}
		case style.ContentCloseQuote, style.ContentNoCloseQuote:
			if c.quotes == 0 {
				return ""
			}
			c.quotes--
			if it.Kind == style.ContentCloseQuote {
				return quote(st, c.quotes, 1)
			}
		case style.ContentString, style.ContentImage:
		}
		return ""
	}
}

// represent writes n in the counter style name counter() names: decimal
// when it names none, nothing for none.
func (c *counting) represent(name string, n int) string {
	switch name {
	case "":
		name = kwDecimal
	case kwNone:
		return ""
	}
	return c.r.counterStyles().represent(name, n)
}

// autoQuotes are quotes: auto's marks, English's as Chrome's.
//
//nolint:gochecknoglobals // a fixed list
var autoQuotes = []string{"“", "”", "‘", "’"}

// quote is the opening (side 0) or closing (side 1) mark depth quotes
// deep: the last pair's past the pairs given.
func quote(st *style.Computed, depth, side int) string {
	marks := st.Quotes
	switch {
	case st.QuotesNone:
		return ""
	case len(marks) == 0:
		marks = autoQuotes
	}
	return marks[min(depth, len(marks)/2-1)*2+side]
}
