package layout

import (
	"math"
	"slices"
	"strings"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// Layout lays the styled tree under root out in a viewport, measuring text with fonts.
func Layout(root *dom.Node, viewport geom.Vec2, fonts box.Fonts, opts ...Option) *box.Tree {
	return layoutIn(root, spaces{def(viewport.X), def(viewport.Y)}, fonts, false, opts...)
}

// Option configures a Layout.
type Option func(*tree)

// Images gives layout the natural sizes of replaced elements' pictures, such as an img's
// src; without it an img is empty.
func Images(i box.Images) Option { return func(t *tree) { t.images = i } }

// Retained keeps a layout's nodes for the next layout: an element whose subtree did not
// change since (no DirtyLayout or DirtyDescendant, the same style and children) keeps its
// nodes, caches and boxes, so only the path to what changed is laid out again; an element
// on that path, unchanged itself, is built again in its own node and box. The tree's Of is
// one map updated in place, its Dropped what each layout took out of it.
type Retained struct {
	nodes map[*dom.Node]*node
	of    map[*dom.Node]*box.Box
	gen   uint32
	unit  float32
	// escaped are the boxes the last layout marked Escapes, cleared before
	// the next marks its own, as kept boxes aren't written again.
	escaped []*box.Box
	// replaced and gone are the sweep's lists, kept for their room.
	replaced []keyed
	gone     []*node
	// viewport is the box of a viewport that scrolls, and viewportGutter
	// the room its overflow: auto bars took in the last layout.
	viewport       *box.Box
	viewportGutter size
}

// Retain lays out reusing r's nodes, and keeps this layout's in r.
func Retain(r *Retained) Option { return func(t *tree) { t.keep = r } }

// layoutIn lays root out; a retained layout in which a scroll container's
// gutter came or went lays out once more, the elements whose gutter changed
// built again with it, so the boxes that shrink to fit around them measure
// it, as Chrome's layout takes the scrollbar in at once.
func layoutIn(root *dom.Node, avail spaces, fonts box.Fonts, rounding bool, opts ...Option) *box.Tree {
	out, regutter := layoutOnce(root, avail, fonts, rounding, opts...)
	if len(regutter) == 0 {
		return out
	}
	for _, e := range regutter {
		dom.Data(e).Dirty |= dom.DirtyLayout
		for p := e.Parent(); p != nil; p = p.Parent() {
			dom.Data(p).Dirty |= dom.DirtyDescendant
		}
	}
	again, _ := layoutOnce(root, avail, fonts, rounding, opts...)
	written, dropped := again.Written, again.Dropped
	again.Written, again.Dropped = nil, nil
	for _, k := range slices.Concat(out.Written, written) {
		if _, ok := again.Of[k]; ok {
			again.Written = append(again.Written, k)
		}
	}
	for _, k := range slices.Concat(out.Dropped, dropped) {
		if _, ok := again.Of[k]; !ok {
			again.Dropped = append(again.Dropped, k)
		}
	}
	return again
}

// layoutOnce lays root out once, and lists the elements whose scrollbar
// gutter changed in a retained layout.
func layoutOnce(root *dom.Node, avail spaces, fonts box.Fonts, rounding bool, opts ...Option) (*box.Tree, []*dom.Node) {
	t := &tree{fonts: fonts, out: &box.Tree{Of: map[*dom.Node]*box.Box{}}}
	for _, o := range opts {
		o(t)
	}
	if k := t.keep; k != nil {
		if k.unit == unitsPerPoint(fonts) && k.nodes != nil {
			t.prevNodes, t.nodes, t.out.Of = k.nodes, k.nodes, k.of
		} else {
			t.nodes, t.lost = map[*dom.Node]*node{}, k.of
		}
		t.gen = k.gen + 1
		t.replaced, t.gone = k.replaced[:0], k.gone[:0]
		k.nodes, k.of, k.gen, k.unit = t.nodes, t.out.Of, t.gen, unitsPerPoint(fonts)
	}
	r := t.build(root, nil)
	defer t.sweep()
	if r == nil {
		return t.out, nil
	}
	port := viewportOf(root)
	if !avail.values().both() {
		port = nil
	}
	var gutter, content size
	if port != nil {
		gutter, content = t.viewport(r, avail.values(), port)
	} else {
		t.root(r, avail)
	}
	if rounding {
		roundLayout(r, point{})
	} else {
		keepLayout(r, point{})
	}
	t.out.Root = r.box
	t.write(r, nil)
	r.box.Parent = nil
	if port != nil {
		t.out.Root = t.viewportBox(r.box, port, avail.values(), gutter, content)
	}
	t.out.Live, t.out.Applied = t.out.Live[:0], t.out.Applied[:0]
	collectLive(t.out.Root, t.out)
	t.markEscapes(r)
	clean(root)
	return t.out, t.regutter
}

// clean clears n's layout flags and those below it, down the paths DirtyDescendant marks.
func clean(n *dom.Node) {
	d := dom.Data(n)
	below := d.Dirty&dom.DirtyDescendant != 0
	d.Dirty &^= dom.DirtyLayout | dom.DirtyDescendant
	if below {
		for _, c := range n.Children() {
			clean(c)
		}
	}
	if d.Before != nil {
		clean(d.Before)
	}
	if d.After != nil {
		clean(d.After)
	}
}

type tree struct {
	fonts  box.Fonts
	out    *box.Tree
	images box.Images
	// keep, when set, holds the nodes kept between layouts: prevNodes are the last
	// layout's, nodes this one's, the same map unless the unit changed, and gen counts
	// the layouts.
	keep      *Retained
	prevNodes map[*dom.Node]*node
	nodes     map[*dom.Node]*node
	gen       uint32
	// replaced are the keys rebuilt nodes held besides their elements, dropped unless
	// written again; gone the nodes rebuilt nodes held, dropped unless reached again;
	// lost the last layout's Of when this one started anew.
	replaced []keyed
	gone     []*node
	lost     map[*dom.Node]*box.Box
	// anonymousStyles is the style of the anonymous blocks of each parent style.
	anonymousStyles map[*style.Computed]*style.Computed
	// anonymousTables is the style of the anonymous tables of each parent style.
	anonymousTables map[*style.Computed]*style.Computed
	// flowStyles are the styles multi-column containers lay their content out with, by
	// their own.
	flowStyles map[*style.Computed]*style.Computed
	// breakStyles are the styles of the line breaks br elements hold, by their own.
	breakStyles map[*style.Computed]*style.Computed
	// regutter are the elements whose scrollbar gutter changed in this
	// retained layout.
	regutter []*dom.Node
	// lookups, when set, counts the cache lookups of a layout, for tests.
	lookups *int
	// translate, when set, is what each text node says, from its own text.
	translate func(message string) string
}

type context uint8

const (
	contextLeaf context = iota
	contextNone
	contextBlock
	contextFlex
	contextGrid
	contextTable
)

// node is one box as the algorithms see it: an element, or an anonymous block around
// inline content.
type node struct {
	box  *box.Box
	st   *style.Computed
	kids []*node
	// inline is the text the node lays out in lines, when its content is inline.
	inline *inlineContent

	cache     cache
	unrounded placed
	final     placed
	absolute  point
	hoisted   []*node
	parent    *node
	grid      *gridInfo
	// subgrid is what its grid hands it in each axis it is a subgrid in.
	subgrid [2]*subgridAxis
	// row is a table row's cells' widths and heights, kept with the row;
	// captionRoom a table's captions' room above and below its grid.
	row         *tableRow
	captionRoom [2]float32
	// cut marks a cell a collapsed row or column narrowed, its content clipped.
	cut bool
	// atomic marks an atomic inline, placed by its owner's lines; pendingOOF is what its
	// layout surfaced for an ancestor to place.
	atomic     bool
	pendingOOF []oofCandidate
	control    controlKind
	role       tableRole
	// cell marks a table's cell, an element or an anonymous one; lowered is how far its
	// content stands below its content box's top to have its first line on its row's
	// baseline.
	cell    bool
	lowered float32
	// shared is the borders of a table that collapses them, and collapsed what that
	// leaves the table or one of its cells for borders, nil where borders are apart.
	shared    *borderGrid
	collapsed *rect
	// replaced is an img, natural its picture's size (zero without one).
	replaced bool
	natural  size
	// shape is what a picture that doesn't ask for a size of pixels asks
	// for, sized being whether the img has one: natural is then the size
	// it takes with no other, but with proportions alone (fills), where
	// it takes the width it is given.
	shape box.Intrinsic
	sized bool
	// intrinsic is set once a layout of the node read its picture's size or
	// proportions: with a width and a height of its own it never does.
	intrinsic bool
	// gutter is the scrollbar room layout chose beyond the style's, overflow auto's once the
	// content overflows.
	gutter size
	// children are the element's children it was built from; fresh marks a node built in
	// this layout, performed the layout in which its own layout last ran, and reached the
	// last layout that built or kept it.
	children  []*dom.Node
	fresh     bool
	performed uint32
	reached   uint32
	// contents are the display: contents elements it flattened, with their boxes;
	// anonymous is set when it wraps inline content in anonymous blocks among its kids.
	contents  []keyed
	anonymous bool
	// cuts are the inline elements among its content that blocks cut in parts.
	cuts []*cutInline
	// readsParent is set when the node's style resolves against its parent's size, which
	// its cache then keys on.
	readsParent bool
	// holdsFloats is set when a float is among its kids or under one of them: laid out
	// in its parent's formatting context, it places floats there each time.
	holdsFloats bool
	// columns is how a multi-column container's last layout cut its flow, and frag the
	// node's place in the flow of the multi-column container it is in, nil outside one.
	columns *columnSet
	frag    *flowFrag
	// clamp is what a line clamp left of the node's lines in its last layout, and
	// discarded is set when the node follows the last line a line-clamp shows.
	clamp     clampOut
	discarded bool
}

// clampOut is what a line clamp leaves of a node's lines: how many it shows, 0 for all,
// whether the last of them ends in an ellipsis, and whether those after are not drawn.
type clampOut struct {
	shown    int
	ellipsis bool
	hides    bool
}

func (n *node) context() context {
	switch {
	case n.st.Display == style.DisplayNone:
		return contextNone
	case n.role == roleTable && len(n.kids) > 0:
		return contextTable
	case len(n.kids) == 0 || n.inline != nil:
		return contextLeaf
	}
	switch n.st.Display {
	case style.FlexDisplay, style.InlineFlex:
		return contextFlex
	case style.GridDisplay, style.InlineGrid:
		return contextGrid
	case style.Inline, style.Block, style.InlineBlock, style.DisplayNone, style.Contents:
	}
	return contextBlock
}

// isBlockLeaf reports a childless node with block display, which Taffy treats as a block.
func (n *node) isBlockLeaf() bool {
	return len(n.kids) == 0 && isBlockDisplay(n.st.Display)
}

func isBlockDisplay(d style.Display) bool {
	return d == style.Block || d == style.InlineBlock || d == style.Inline
}

func styleOf(n *dom.Node) *style.Computed {
	if s := dom.Data(n).Style; s != nil {
		return s
	}
	s := style.Initial()
	return &s
}

// build makes the node of an element and its subtree, nil for display: contents (whose
// children the parent takes).
func (t *tree) build(e *dom.Node, parent *node) *node {
	if n := t.kept(e); n != nil {
		t.adopt(n, parent)
		return n
	}
	n := t.element(e, parent)
	if k := controlOf(e); k != noControl {
		t.buildControl(n, k)
		return n
	}
	if isTable(e) {
		t.buildTable(n, e.Flow())
		return n
	}
	if pictured(e) {
		t.buildImage(n, e)
		return n
	}
	if e.Tag() == dom.Canvas {
		buildCanvas(n, e)
		return n
	}
	if e.Tag() == dom.Iframe {
		buildFrame(n)
		return n
	}
	t.buildChildren(n, e.Flow())
	return n
}

// kept is e's node from the last layout when nothing under e changed since.
func (t *tree) kept(e *dom.Node) *node {
	n := t.prevNodes[e]
	if n == nil {
		return nil
	}
	d := dom.Data(e)
	if d.Dirty&(dom.DirtyLayout|dom.DirtyDescendant) != 0 || n.st != d.Style || !e.SameFlow(n.children) {
		return nil
	}
	return n
}

// element is a node for e to build its subtree in: e's node from the last layout, taken
// again with its box and emptied, when e itself is unchanged and only something under it
// did, else a new one.
func (t *tree) element(e *dom.Node, parent *node) *node {
	st := styleOf(e)
	old := t.prevNodes[e]
	if n := old; n != nil && n.st == st && dom.Data(e).Dirty&dom.DirtyLayout == 0 &&
		e.SameFlow(n.children) {
		t.retire(n, false)
		b := n.box
		*b = box.Box{Kind: box.Element, Node: e, Style: st, Children: b.Children[:0]}
		*n = node{box: b, st: st, parent: parent, fresh: true, readsParent: n.readsParent,
			kids: n.kids[:0], children: n.children, gutter: n.gutter, reached: t.gen}
		t.place(e, b)
		t.nodes[e] = n
		return n
	}
	b := &box.Box{Kind: box.Element, Node: e, Style: st}
	t.place(e, b)
	n := &node{box: b, st: st, parent: parent, fresh: true, readsParent: readsParent(st), reached: t.gen}
	if t.nodes != nil {
		if old != nil {
			t.retire(old, true)
			n.gutter = old.gutter
		}
		t.nodes[e] = n
		n.children = slices.Clone(e.Flow())
	}
	return n
}

// place gives k the box b in the tree's Of.
func (t *tree) place(k *dom.Node, b *box.Box) {
	t.out.Of[k] = b
	t.out.Written = append(t.out.Written, k)
}

// keyed is a key of the tree's Of with the box a node wrote for it.
type keyed struct {
	key *dom.Node
	box *box.Box
}

// retire notes what old, the last layout's node of an element built again, held besides
// its element for sweep: its text runs, spans and display: contents boxes, its atomic
// inlines, and all its anonymous blocks held, as they are built anew each time; its kids
// too when
// they may be gone, which they can't be from an element of the same children holding no
// inline content or display: contents.
func (t *tree) retire(old *node, kids bool) {
	t.retireInline(old)
	if old.anonymous {
		for _, k := range old.kids {
			switch {
			case k.box.Kind == box.AnonymousBlock:
				t.retireInline(k)
				t.gone = append(t.gone, k.kids...)
			case k.st.Float != style.FloatNone:
				// maybe a float from inside an inline element, no child of the element's
				t.gone = append(t.gone, k)
			}
		}
	}
	t.replaced = append(t.replaced, old.contents...)
	if kids || old.inline != nil || len(old.contents) > 0 {
		t.gone = append(t.gone, old.kids...)
	}
}

// retireInline notes old's text runs and spans, and its atomic inlines.
func (t *tree) retireInline(old *node) {
	c := old.inline
	if c == nil {
		return
	}
	for _, r := range c.runs {
		if r.atomic != nil {
			t.gone = append(t.gone, r.atomic.node)
		} else if r.box.Node != nil {
			t.replaced = append(t.replaced, keyed{r.box.Node, r.box})
		}
	}
	for _, s := range c.spans {
		t.replaced = append(t.replaced, keyed{s.box.Node, s.box})
	}
	if c.back != nil {
		t.replaced = append(t.replaced, keyed{c.back.Node, c.back})
	}
}

// sweep takes out of the tree's Of and nodes, once the layout is written, what it no
// longer reaches: a retired key nothing wrote again, a retired node's element no build or
// adoption reached, with all it held. With a new unit, every key of the last layout's
// not written again is lost.
func (t *tree) sweep() {
	keep := t.keep
	if keep == nil {
		return
	}
	defer func() { keep.replaced, keep.gone = t.replaced[:0], t.gone[:0] }()
	for _, k := range t.replaced {
		if t.out.Of[k.key] == k.box {
			t.drop(k.key)
		}
	}
	for _, n := range t.gone {
		t.dropUnreached(n)
	}
	for k := range t.lost {
		if _, ok := t.out.Of[k]; !ok {
			t.out.Dropped = append(t.out.Dropped, k)
		}
	}
}

func (t *tree) drop(k *dom.Node) {
	delete(t.out.Of, k)
	delete(t.nodes, k)
	t.out.Dropped = append(t.out.Dropped, k)
}

// dropUnreached drops n's element unless this layout reached it, and then everything n
// held.
func (t *tree) dropUnreached(n *node) {
	if e := n.box.Node; e != nil && n.box.Kind == box.Element {
		if now := t.nodes[e]; now != nil && now.reached == t.gen {
			return
		}
		if t.out.Of[e] == n.box {
			t.drop(e)
		} else if t.nodes[e] == n {
			delete(t.nodes, e)
		}
	}
	if c := n.inline; c != nil {
		for _, r := range c.runs {
			if r.atomic != nil {
				t.dropUnreached(r.atomic.node)
			} else if r.box.Node != nil && t.out.Of[r.box.Node] == r.box {
				t.drop(r.box.Node)
			}
		}
		for _, s := range c.spans {
			if t.out.Of[s.box.Node] == s.box {
				t.drop(s.box.Node)
			}
		}
	}
	for _, k := range n.contents {
		if t.out.Of[k.key] == k.box {
			t.drop(k.key)
		}
	}
	for _, k := range n.kids {
		t.dropUnreached(k)
	}
}

// adopt takes the kept node n, with its subtree, into this layout under parent.
func (t *tree) adopt(n *node, parent *node) {
	n.parent, n.reached = parent, t.gen
	if e := n.box.Node; dom.Data(e).Placed != n.box {
		t.out.Written = append(t.out.Written, e)
	}
}

// buildChildren gives n its in-flow children: elements, and the text between them either
// as n's own inline content or wrapped in anonymous blocks.
func (t *tree) buildChildren(n *node, children []*dom.Node) {
	container := n.st.Display
	inFlexOrGrid := container == style.FlexDisplay || container == style.InlineFlex ||
		container == style.GridDisplay || container == style.InlineGrid
	items := t.lift(n, t.flatten(n, children))
	if !inFlexOrGrid && !hasBlockChild(items) {
		t.buildInline(n, items)
		return
	}
	var run []item
	// inFlow is set once the container holds content in the flow: the anonymous blocks
	// after it don't start with the container's first line.
	inFlow := false
	flush := func() {
		if onlyCollapsibleSpace(run) {
			run = nil
			return
		}
		anon := t.anonymous(n)
		content := t.inlineOf(anon, run)
		n.liftFloats(anon)
		if content != nil {
			content.unindented = inFlow
			anon.inline = content
			n.kids = append(n.kids, anon)
			n.anonymous = true
			inFlow = true
		} else {
			emptyParts(run, len(n.kids))
		}
		run = nil
	}
	for i := 0; i < len(items); i++ {
		it := items[i]
		if it.text != nil {
			// A -webkit-box drops the white space between its children, inline ones too.
			if !n.st.WebkitBox || !inFlexOrGrid || !onlyCollapsibleSpace(items[i:i+1]) {
				run = append(run, it)
			}
			continue
		}
		st := styleOf(it.element)
		if !inFlexOrGrid && it.cut == nil && partOf(it.element) != style.NoTablePart {
			parts, next := tableParts(items, i)
			flush()
			n.kids = append(n.kids, t.anonymousTable(n, parts))
			n.anonymous, inFlow = true, true
			i = next - 1
			continue
		}
		if !inFlexOrGrid && st.Float != style.FloatNone && st.Display != style.DisplayNone {
			// A float among inline content stands before it, the text going on round it.
			n.kids = append(n.kids, t.build(it.element, n))
			continue
		}
		if (!inFlexOrGrid || n.st.WebkitBox) && !elementBlockLevel(it.element) {
			run = append(run, it)
			continue
		}
		flush()
		n.kids = append(n.kids, t.build(it.element, n))
		inFlow = inFlow || st.Display != style.DisplayNone && !outOfFlow(st.Position)
	}
	flush()
	for _, k := range n.kids {
		if k.st.Float != style.FloatNone || k.holdsFloats {
			n.holdsFloats = true
		}
	}
}

// tableParts lists the table parts that follow one another from items[i], with the white
// space between them, and the index after the last: what one anonymous table wraps.
func tableParts(items []item, i int) ([]*dom.Node, int) {
	var parts []*dom.Node
	kept, end := 0, i
	for j := i; j < len(items); j++ {
		it := items[j]
		if it.text != nil && isCollapsibleSpace(it.text.Text()) {
			parts = append(parts, it.text)
			continue
		}
		if it.element == nil || it.cut != nil || partOf(it.element) == style.NoTablePart {
			break
		}
		parts = append(parts, it.element)
		kept, end = len(parts), j+1
	}
	return parts[:kept], end
}

// buildInline gives n inline content alone: its items as its own lines, nodes for those
// not shown, and, as its lines are gathered, for those out of the flow.
func (t *tree) buildInline(n *node, items []item) {
	for _, it := range items {
		if it.element != nil && styleOf(it.element).Display == style.DisplayNone {
			n.kids = append(n.kids, t.build(it.element, n))
		}
	}
	n.inline = t.inlineOf(n, items)
}

// liftFloats moves the floats inside an anonymous block's inline elements to n, where
// they stand before the block.
func (n *node) liftFloats(anon *node) {
	kept := anon.kids[:0]
	for _, k := range anon.kids {
		if k.st.Float == style.FloatNone {
			kept = append(kept, k)
			continue
		}
		k.parent = n
		n.kids = append(n.kids, k)
		n.anonymous = true
	}
	anon.kids = kept
}

// onlyCollapsibleSpace reports a run of text nodes that white-space collapses away.
func onlyCollapsibleSpace(run []item) bool {
	for _, it := range run {
		if it.text == nil {
			return false
		}
		st := styleOf(it.text)
		if p := it.text.Parent(); p != nil {
			st = styleOf(p)
		}
		if !collapses(st.WhiteSpace) || !isCollapsibleSpace(it.text.Text()) {
			return false
		}
	}
	return true
}

type item struct {
	element *dom.Node
	text    *dom.Node
	// cut is set on a part of an inline element blocks cut: inside are the children of
	// the element the part holds, first and last say which of its parts it is.
	cut         *cutInline
	inside      []item
	first, last bool
}

// cutInline is an inline element holding blocks in the flow. They cut it in parts, the
// lines before, between and after them, and stand among its container's children. Its
// box says where the element is, round its parts and its blocks as Chrome's
// getBoundingClientRect; the parts draw it, each a box of its own.
type cutInline struct {
	box    *box.Box
	parts  []*box.Box
	blocks []*dom.Node
	// empty are its parts that hold no line, each by how many of its container's kids
	// stand before it: a place of no size the element's box reaches, as Chrome's.
	empty []int
}

// emptyParts notes a run of no line, after the first before kids of its container, on
// the cut elements whose parts it holds.
func emptyParts(run []item, before int) {
	for _, it := range run {
		if it.cut != nil {
			it.cut.empty = append(it.cut.empty, before)
			emptyParts(it.inside, before)
		}
	}
}

// emptyPart is where a part of no line stands in n, after the first before of its
// kids: under the last of them in the flow, at the start of n's content.
func (n *node) emptyPart(before int) geom.Rect {
	at := geom.Rect{X: n.final.border.l + n.final.padding.l, Y: n.final.border.t + n.final.padding.t}
	for _, k := range n.kids[:before] {
		if k.st.Display != style.DisplayNone && k.st.Float == style.FloatNone && !outOfFlow(k.st.Position) {
			at.Y = k.box.Pos.Y + k.box.Size.Y
		}
	}
	return at
}

// lift is items with the blocks inside their inline elements lifted among them, each
// such element cut in parts round its blocks and noted on owner.
func (t *tree) lift(owner *node, items []item) []item {
	if !slices.ContainsFunc(items, func(it item) bool { return it.element != nil && cutByBlocks(it.element) }) {
		return items
	}
	var out []item
	for _, it := range items {
		if it.element == nil || !cutByBlocks(it.element) {
			out = append(out, it)
			continue
		}
		b := &box.Box{Kind: box.Element, Node: it.element, Style: styleOf(it.element)}
		t.place(it.element, b)
		cut := &cutInline{box: b}
		owner.cuts = append(owner.cuts, cut)
		owner.contents = append(owner.contents, keyed{it.element, b})
		part := item{element: it.element, cut: cut, first: true}
		for _, in := range t.lift(owner, t.flatten(owner, it.element.Flow())) {
			if in.cut != nil || in.element == nil || !inFlowBlock(in.element) {
				part.inside = append(part.inside, in)
				continue
			}
			cut.blocks = append(cut.blocks, in.element)
			out = append(out, part, in)
			part = item{element: it.element, cut: cut}
		}
		part.last = true
		out = append(out, part)
	}
	return out
}

// placeCuts puts the box of each inline element blocks cut in n round its parts and its
// blocks, once n's children are written, the blocks moved with the inline elements
// position: relative moves. The box of each element counts its blocks where its own
// offset alone moves them, as Chrome's.
func (n *node) placeCuts(of map[*dom.Node]*box.Box) {
	cb := size{n.final.size.w - n.final.padding.horizontal() - n.final.border.horizontal(), none}
	movedBy := func(e, upTo *dom.Node) geom.Vec2 {
		var o geom.Vec2
		for p := e.Parent(); p != nil && p != upTo && p != n.box.Node; p = p.Parent() {
			if isInline(p) {
				o = o.Add(inlineOffset(styleOf(p), cb))
			}
		}
		return o
	}
	moved := map[*box.Box]bool{}
	for _, c := range n.cuts {
		for _, e := range c.blocks {
			if b := of[e]; b != nil && b.Parent == n.box && !moved[b] {
				moved[b] = true
				b.Pos = b.Pos.Add(movedBy(e, nil))
				b.Dim = dimOf(e, n.box.Node)
			}
		}
	}
	for _, c := range n.cuts {
		var r geom.Rect
		found := false
		for _, p := range c.parts {
			at, ok := p.Pos, false
			for a := p.Parent; a != nil && !ok; a = a.Parent {
				if ok = a == n.box; !ok {
					at = at.Add(a.Pos)
				}
			}
			if ok {
				r = unionRect(r, geom.Rect{X: at.X, Y: at.Y, W: p.Size.X, H: p.Size.Y}, found)
				found = true
			}
		}
		for _, before := range c.empty {
			at := n.emptyPart(before)
			o := inlineOffset(c.box.Style, cb)
			at.X, at.Y = at.X+o.X, at.Y+o.Y
			r = unionRect(r, at, found)
			found = true
		}
		for _, e := range c.blocks {
			if b := of[e]; b != nil && b.Parent == n.box {
				at := b.Pos.Sub(movedBy(e, c.box.Node))
				r = unionRect(r, geom.Rect{X: at.X, Y: at.Y, W: b.Size.X, H: b.Size.Y}, found)
				found = true
			}
		}
		c.box.Parent = n.box
		c.box.Pos, c.box.Size = geom.Vec2{X: r.X, Y: r.Y}, geom.Vec2{X: r.W, Y: r.H}
	}
}

// dimOf is the Dim of e's box, its inline ancestors' up to upTo fading it.
func dimOf(e, upTo *dom.Node) float32 {
	shown := float32(1)
	for p := e.Parent(); p != nil && p != upTo; p = p.Parent() {
		if isInline(p) {
			shown *= styleOf(p).Opacity
		}
	}
	return 1 - shown
}

// inFlowBlock reports an element that is a block in the flow of its container.
func inFlowBlock(e *dom.Node) bool {
	st := styleOf(e)
	return st.Display != style.DisplayNone && !outOfFlow(st.Position) && st.Float == style.FloatNone &&
		elementBlockLevel(e)
}

// isInline reports an element whose box is an inline one, its content in its
// container's lines.
func isInline(e *dom.Node) bool {
	st := styleOf(e)
	return st.Display == style.Inline && !outOfFlow(st.Position) && st.Float == style.FloatNone && !isTable(e)
}

// cutByBlocks reports an inline element holding a block in the flow, through the inline
// elements it holds.
func cutByBlocks(e *dom.Node) bool { return isInline(e) && holdsBlock(e) }

// holdsBlock reports a block in the flow among e's children, those of its display:
// contents children and those of the inline elements it holds.
func holdsBlock(e *dom.Node) bool {
	for _, c := range e.Flow() {
		switch {
		case c.IsText():
		case styleOf(c).Display == style.Contents:
			if holdsBlock(c) {
				return true
			}
		case inFlowBlock(c) || cutByBlocks(c):
			return true
		}
	}
	return false
}

// flatten lists children with display: contents replaced by theirs, noted on owner.
func (t *tree) flatten(owner *node, children []*dom.Node) []item {
	var items []item
	for _, c := range children {
		if c.IsText() {
			items = append(items, item{text: c})
			continue
		}
		if styleOf(c).Display == style.Contents {
			b := &box.Box{Kind: box.Element, Node: c, Style: styleOf(c)}
			t.place(c, b)
			owner.contents = append(owner.contents, keyed{c, b})
			items = append(items, t.flatten(owner, c.Flow())...)
			continue
		}
		items = append(items, item{element: c})
	}
	return items
}

func isBlockLevel(st *style.Computed) bool {
	switch st.Display {
	case style.Inline, style.InlineBlock, style.InlineFlex, style.InlineGrid:
		return outOfFlow(st.Position)
	case style.Block, style.FlexDisplay, style.GridDisplay, style.DisplayNone, style.Contents:
	}
	return true
}

// elementBlockLevel is isBlockLevel for an element, which a table always is.
func elementBlockLevel(e *dom.Node) bool { return isBlockLevel(styleOf(e)) }

// hasBlockChild reports a child that makes the container lay blocks out: one in the flow
// that is block-level, as a float is, or an inline element with a float inside.
func hasBlockChild(items []item) bool {
	for _, it := range items {
		if it.text != nil && floatedLetter(it.text) {
			return true
		}
		if it.element != nil {
			st := styleOf(it.element)
			if st.Display == style.DisplayNone || outOfFlow(st.Position) {
				continue
			}
			if elementBlockLevel(it.element) || it.cut != nil ||
				!isAtomicDisplay(st.Display) && floatInside(it.element) {
				return true
			}
		}
	}
	return false
}

// floatedLetter reports a text node whose first letter a floated ::first-letter takes.
func floatedLetter(text *dom.Node) bool {
	g, _ := letterOf(text)
	return g != nil && styleOf(g).Float != style.FloatNone
}

// floatInside reports a float under the inline element e, through the inline elements
// it holds.
func floatInside(e *dom.Node) bool {
	for _, c := range e.Flow() {
		if c.IsText() {
			if floatedLetter(c) {
				return true
			}
			continue
		}
		st := styleOf(c)
		switch {
		case st.Display == style.DisplayNone || outOfFlow(st.Position):
		case st.Float != style.FloatNone:
			return true
		case !isAtomicDisplay(st.Display) && floatInside(c):
			return true
		}
	}
	return false
}

// anonymous is the block wrapping inline content in a flex, grid or block container: it
// inherits the text properties and nothing else.
func (t *tree) anonymous(parent *node) *node {
	st := t.anonymousStyles[parent.st]
	if st == nil {
		s := style.Initial()
		s.Display = style.Block
		s.Text = parent.st.Text
		st = &s
		if t.anonymousStyles == nil {
			t.anonymousStyles = map[*style.Computed]*style.Computed{}
		}
		t.anonymousStyles[parent.st] = st
	}
	b := &box.Box{Kind: box.AnonymousBlock, Style: st}
	return &node{box: b, st: st, parent: parent, fresh: true}
}

// write fills the box tree from the laid-out nodes.
func (t *tree) write(n *node, parent *node) {
	t.writeIn(n, parent, true)
}

// writeIn writes n's box and, when n is new or its layout ran in this one, rebuilds its
// children and text; a kept node laid out from its cache keeps them. placed says n's
// parent placed it this time: an atomic inline keeps its place in a kept span otherwise.
func (t *tree) writeIn(n *node, parent *node, placed bool) {
	b := n.box
	f := n.final
	rewrite := t.keep == nil || n.fresh || n.performed == t.gen
	n.fresh = false
	b.Size = geom.Vec2{X: f.size.w, Y: f.size.h}
	if n.st.Display != style.DisplayNone && (placed || !n.atomic) {
		origin := point{}
		if parent != nil {
			origin = parent.absolute
		}
		at := n.absolute
		b.Pos = geom.Vec2{X: at.x - origin.x, Y: at.y - origin.y}
		b.Sticky = geom.Vec2{}
	}
	b.Margin = box.Edges{Top: f.margin.t, Right: f.margin.r, Bottom: f.margin.b, Left: f.margin.l}
	b.Border = box.Edges{Top: f.border.t, Right: f.border.r, Bottom: f.border.b, Left: f.border.l}
	b.Padding = box.Edges{Top: f.padding.t, Right: f.padding.r, Bottom: f.padding.b, Left: f.padding.l}
	b.Content = geom.Vec2{X: f.overflow.rect.r - f.overflow.rect.l, Y: f.overflow.rect.b - f.overflow.rect.t}
	b.ScrollStart = geom.Vec2{X: fmin(f.overflow.rect.l, 0), Y: fmin(f.overflow.rect.t, 0)}
	b.Scrollbar = geom.Vec2{X: f.scrollbar.w, Y: f.scrollbar.h}
	b.Wrap = box.Edges{Top: n.captionRoom[0], Bottom: n.captionRoom[1]}
	b.Cut = n.cut
	b.Discarded = n.discarded
	if n.frag != nil || n.columns != nil || b.Broken || b.Columns != nil {
		n.writeColumns()
	}
	if !rewrite {
		for _, c := range n.kids {
			t.writeIn(c, n, false)
		}
		n.placeCuts(t.out.Of)
		return
	}
	b.Children = b.Children[:0]
	if n.inline != nil {
		t.writeText(n)
	}
	for _, c := range n.kids {
		if c.st.Display != style.DisplayNone {
			b.Append(c.box)
		}
		t.writeIn(c, n, true)
	}
	t.atomsIntoSpans(n)
	t.listMarker(n)
	n.placeCuts(t.out.Of)
}

// markerGap is how far a bullet stands from its item's content, past its own offset, as
// Chrome's.
const markerGap = 7

// rtlMarkerPull is how much nearer its content Chrome draws the bullet or the picture of
// a right-to-left item than the mirror of a left-to-right one's.
const rtlMarkerPull = 3.5

// listMarker gives a list item its ::marker box, outside the item beside its first line:
// the marker string css worked out, its ::marker's content or its counter style's,
// ending where the item's content starts, a bullet or the list-style-image Chrome's
// distance before it, the picture on the line's baseline. An item without a line has
// none. The box's style is the ::marker's where a rule styles it, the item's otherwise.
func (t *tree) listMarker(n *node) {
	e := n.box.Node
	if e == nil || n.box.Kind != box.Element || !n.st.ListItem || e.Tag() == dom.Summary ||
		n.st.Display == style.DisplayNone {
		return
	}
	line, at, ok := firstLine(n.box, geom.Vec2{})
	if !ok {
		return
	}
	data := dom.Data(e)
	st := n.st
	if data.Marker != nil {
		st = data.Marker
	}
	m := &box.Box{Kind: box.Marker, Node: e, Style: st}
	left := n.final.border.l + n.final.padding.l
	rtl := n.st.Dir == style.RTL
	right := n.final.size.w - n.final.border.r - n.final.padding.r
	ascent := float32(math.Round(float64(t.fonts.Metrics(st.Font).Ascent)))
	baseline := at.Y + line.Baseline
	text := ""
	if data.HasMarkerText {
		inWord := false
		text = shownText(data.MarkerText, st, nil, &inWord)
	}
	picture, pictured := t.markerPicture(n.st, data)
	switch {
	case pictured:
		m.Pos = geom.Vec2{X: left - picture.X - markerGap, Y: baseline - picture.Y}
		if rtl {
			m.Pos.X = right + markerGap - rtlMarkerPull
		}
		m.Size, m.Picture = picture, n.st.ListImage
	case text != "":
		w := t.fonts.Advance(st.Font, text) + st.Spacing(text)
		m.Pos = geom.Vec2{X: left - w, Y: at.Y + line.Rect.Y}
		if rtl {
			m.Pos.X, text = right, visualText(text, 1)
		}
		m.Size = geom.Vec2{X: w, Y: line.Rect.H}
		m.Lines = []box.Fragment{{Text: text, Rect: geom.Rect{W: w, H: line.Rect.H}, Baseline: baseline - m.Pos.Y}}
	case data.HasMarkerText:
		return
	case st.ListStyle == style.ListDisc || st.ListStyle == style.ListCircle || st.ListStyle == style.ListSquare:
		offset := float32(math.Floor(float64(ascent * 2 / 3)))
		side := float32(math.Floor(float64((offset + 1) / 2)))
		top := float32(math.Floor(float64(3 * (ascent - offset) / 2)))
		m.Pos = geom.Vec2{X: left - offset - markerGap, Y: baseline - ascent + top}
		if rtl {
			m.Pos.X = right + offset + markerGap - rtlMarkerPull - side
		}
		m.Size = geom.Vec2{X: side, Y: side}
	default:
		return
	}
	n.box.Append(m)
}

// markerPicture is the size of the list-style-image st gives a list item's marker, false
// when it has none, or none found, or the ::marker's content stands in its place.
func (t *tree) markerPicture(st *style.Computed, data *dom.Pipeline) (geom.Vec2, bool) {
	if st.ListImage == "" || t.images == nil || data.Marker != nil && data.Marker.Content.Kind != style.ContentNormal {
		return geom.Vec2{}, false
	}
	s, ok := t.images.ImageSize(st.ListImage)
	return s, ok && s.X > 0 && s.Y > 0
}

// firstLine is the first line fragment under b, with its box's place in b.
func firstLine(b *box.Box, at geom.Vec2) (box.Fragment, geom.Vec2, bool) {
	if b.Kind == box.TextRun && len(b.Lines) > 0 {
		return b.Lines[0], at, true
	}
	for _, c := range b.Children {
		if c.Kind == box.Marker {
			continue
		}
		if f, p, ok := firstLine(c, at.Add(c.Pos)); ok {
			return f, p, true
		}
	}
	return box.Fragment{}, geom.Vec2{}, false
}

// keepLayout places every box without rounding, for fixtures that turn it off.
func keepLayout(n *node, cumulative point) { keepIn(n, cumulative, false) }

// keepIn is keepLayout under a display: none ancestor when hidden, where out-of-flow
// boxes take their parent's place too, as no containing block places them.
func keepIn(n *node, cumulative point, hidden bool) {
	n.final = n.laid()
	n.final.scrollbar = n.final.scrollbar.add(n.gutter)
	n.absolute = point{cumulative.x + n.unrounded.location.x, cumulative.y + n.unrounded.location.y}
	if n.frag != nil {
		n.frag = nil
	}
	hidden = hidden || n.st.Display == style.DisplayNone
	if n.columns != nil && !hidden {
		keepColumns(n)
		return
	}
	for _, c := range n.kids {
		if hidden || !outOfFlow(c.st.Position) || c.st.Display == style.DisplayNone {
			keepIn(c, n.absolute, hidden)
		}
	}
	for _, c := range n.hoisted {
		keepIn(c, n.absolute, hidden)
	}
}

func isCollapsibleSpace(s string) bool { return strings.TrimLeft(s, " \t\n\r\f") == "" }

// markEscapes sets Escapes on each box a containing block placed from
// above and the boxes between it and that block, the last layout's marks
// cleared first when the tree is kept.
func (t *tree) markEscapes(root *node) {
	var marked []*box.Box
	if k := t.keep; k != nil {
		for _, b := range k.escaped {
			b.Escapes = false
		}
		marked = k.escaped[:0]
	}
	var walk func(n *node)
	walk = func(n *node) {
		for _, h := range n.hoisted {
			for b := h.box; b != nil && b != n.box; b = b.Parent {
				b.Escapes = true
				marked = append(marked, b)
			}
		}
		for _, k := range n.kids {
			walk(k)
		}
	}
	walk(root)
	if t.keep != nil {
		t.keep.escaped = marked
	}
}
