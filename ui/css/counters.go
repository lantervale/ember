package css

import (
	"slices"
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// listItem is the counter every list item counts on.
const listItem = "list-item"

// counter is one counter in scope: made on an element depth deep, it
// reaches that element's descendants and the siblings after it with
// theirs, unless it is nested in a counter of its name its parent has:
// then, as in Chrome and CSS Lists' inheritance of counters, the
// element's descendants alone.
type counter struct {
	name   string
	depth  int
	value  int
	nested bool
}

// list numbers the markers of the items of an ol, a ul or a menu, apart
// from the list-item counter as Chrome does: from its start, a reversed
// ol's from its number of items, and an item's value attribute numbers
// its marker alone. An item outside any list is numbered among those of
// its parent, owner. step is what an item adds: -1 in a reversed ol;
// last is the number before the next item's.
type list struct {
	owner *dom.Node
	depth int
	step  int
	last  int
}

// counting walks the tree in document order with the counters in scope
// and the depth of the quotes open, as CSS Lists and Generated Content
// have them.
type counting struct {
	r     *restyler
	stack []counter
	lists []list
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
		c.item(n, data, st, c.number(n, st, depth))
	}
	if isList(n.Tag()) {
		start, step := listStart(n, true)
		c.lists = append(c.lists, list{owner: n, depth: depth, step: step, last: start})
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
	c.leave(depth)
	for len(c.lists) > 0 && c.lists[len(c.lists)-1].depth >= depth {
		c.lists = c.lists[:len(c.lists)-1]
	}
}

// generated counts a ::before or ::after box, a child of its element, and
// gives it its text.
func (c *counting) generated(origin, g *dom.Node, depth int) {
	gd := dom.Data(g)
	st := gd.Style
	c.apply(nil, st, depth)
	if st.ListItem {
		ordinal := 0
		if i := c.find(listItem); i >= 0 {
			ordinal = c.stack[i].value
		}
		c.item(g, gd, st, ordinal)
	}
	if c.r.fill(g, st, c.text(st)) {
		flagLayout(origin)
	}
	c.leave(depth)
}

// leave ends the scope of the counters made inside a box depth deep, and
// of those the box made inside its parent's own.
func (c *counting) leave(depth int) {
	for k := len(c.stack); k > 0; k-- {
		if top := c.stack[k-1]; top.depth < depth || top.depth == depth && !top.nested {
			return
		}
		c.stack = c.stack[:k-1]
	}
}

// item gives a list item its number and its ::marker its text: its
// content's, or the marker string of its list-style-type.
func (c *counting) item(n *dom.Node, data *dom.Pipeline, st *style.Computed, ordinal int) {
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

// number is the number of a list item's marker, n depth deep: a
// counter-set of list-item, else its value attribute, else the number
// before it in its list and its step, which a counter-increment of
// list-item replaces, together with its counter-reset.
func (c *counting) number(n *dom.Node, st *style.Computed, depth int) int {
	l := c.listOf(n, depth)
	if v, ok := lastOf(st.CounterSet, listItem); ok {
		l.last = v
		return v
	}
	if v, ok := intAttr(n, "value"); ok && n.Tag() == dom.Li {
		l.last = v
		return v
	}
	step := l.step
	if namesListItem(st.CounterIncrement) {
		step, _ = lastOf(st.CounterReset, listItem)
		for _, r := range st.CounterIncrement {
			if r.Name == listItem {
				step += r.Value
			}
		}
	}
	l.last += step
	return l.last
}

// listOf is the list numbering an item n depth deep: the innermost list
// round it, else its parent's own.
func (c *counting) listOf(n *dom.Node, depth int) *list {
	for i := len(c.lists) - 1; i >= 0; i-- {
		if isList(c.lists[i].owner.Tag()) {
			return &c.lists[i]
		}
	}
	if k := len(c.lists); k > 0 && c.lists[k-1].owner == n.Parent() {
		return &c.lists[k-1]
	}
	c.lists = append(c.lists, list{owner: n.Parent(), depth: depth - 1, step: 1})
	return &c.lists[len(c.lists)-1]
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
// box: a list resets list-item, and a list item adds to it its list's
// step, unless the box's counter properties name list-item themselves.
// An item's value attribute numbers its marker, not the counter.
func (c *counting) apply(n *dom.Node, st *style.Computed, depth int) {
	own := n != nil && (namesListItem(st.CounterReset) || namesListItem(st.CounterIncrement) ||
		namesListItem(st.CounterSet))
	if n != nil && !own && isList(n.Tag()) {
		start, _ := listStart(n, false)
		c.reset(listItem, start, depth)
	}
	for _, r := range st.CounterReset {
		c.reset(r.Name, r.Value, depth)
	}
	if n != nil && !own && st.ListItem {
		c.stack[c.touch(listItem, depth)].value += c.listOf(n, depth).step
	}
	for _, r := range st.CounterIncrement {
		c.stack[c.touch(r.Name, depth)].value += r.Value
	}
	for _, r := range st.CounterSet {
		c.stack[c.touch(r.Name, depth)].value = r.Value
	}
}

// reset makes a counter on a box depth deep: a new one inside those of
// its name in scope, or, where a sibling before it made the innermost,
// that one again.
func (c *counting) reset(name string, value, depth int) {
	i := c.find(name)
	if i >= 0 && c.stack[i].depth == depth {
		c.stack[i].value = value
		return
	}
	c.stack = append(c.stack, counter{name: name, depth: depth, value: value, nested: i >= 0})
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

// namesListItem reports whether list names the list-item counter.
func namesListItem(list []style.Counter) bool {
	return slices.ContainsFunc(list, func(c style.Counter) bool { return c.Name == listItem })
}

// lastOf is the value the last of list naming name gives it.
func lastOf(list []style.Counter, name string) (int, bool) {
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].Name == name {
			return list[i].Value, true
		}
	}
	return 0, false
}

func isList(tag dom.Tag) bool { return tag == dom.Ol || tag == dom.Ul || tag == dom.Menu }

// listStart is what a list numbers its items from, and what each adds:
// one under its start attribute, counting up, or one over it counting
// down for a reversed ol. A reversed ol with no start numbers its
// markers from its number of items, and its list-item counter, as
// Chrome's when it first counts, from 0.
func listStart(list *dom.Node, markers bool) (int, int) {
	start, given := intAttr(list, "start")
	if _, reversed := list.Attr("reversed"); reversed && list.Tag() == dom.Ol {
		if !given && markers {
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
