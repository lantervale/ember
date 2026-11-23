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
	// barsOn is the viewport's style barsView with its styled bars.
	barsView, barsOn *style.Computed
	bars             *style.StyledBars
	// anchors are the anchors of the elements placed by them, where the
	// last layout placed them.
	anchors *anchoring
	// turns are the styles the last layout turned into vertical frames.
	turns map[turnKey]*style.Computed
}

// Retain lays out reusing r's nodes, and keeps this layout's in r.
func Retain(r *Retained) Option { return func(t *tree) { t.keep = r } }

// layoutIn lays root out; a retained layout in which a scroll container's
// gutter came or went lays out once more, the elements whose gutter changed
// built again with it, so the boxes that shrink to fit around them measure
// it, as Chrome's layout takes the scrollbar in at once. So does a layout
// that moved the anchors of an element positioned by them, that element
// built again with them where they now stand, up to maxAnchorPasses
// times for anchors placed by anchors.
func layoutIn(root *dom.Node, avail spaces, fonts box.Fonts, rounding bool, opts ...Option) *box.Tree {
	var probe tree
	for _, o := range opts {
		o(&probe)
	}
	anchors := &anchoring{of: map[*dom.Node]*anchorsOf{}}
	if k := probe.keep; k != nil {
		if k.anchors == nil {
			k.anchors = anchors
		}
		anchors = k.anchors
	}
	var out *box.Tree
	var written, dropped []*dom.Node
	for pass := 0; ; pass++ {
		next, regutter, t := layoutOnce(root, avail, fonts, rounding, anchors, opts...)
		written, dropped = append(written, next.Written...), append(dropped, next.Dropped...)
		out = next
		again := regutter
		if pass > 0 {
			again = nil
		}
		if len(anchors.of) > 0 && pass < maxAnchorPasses {
			again = append(again, t.refreshAnchors(out, anchors)...)
		}
		if len(again) == 0 {
			break
		}
		for _, e := range again {
			dom.Data(e).Dirty |= dom.DirtyLayout
			for p := e.Parent(); p != nil; p = p.Parent() {
				dom.Data(p).Dirty |= dom.DirtyDescendant
			}
		}
	}
	out.Anchored = out.Anchored[:0]
	for e := range anchors.of {
		out.Anchored = append(out.Anchored, e)
	}
	if len(written)+len(dropped) == len(out.Written)+len(out.Dropped) {
		return out
	}
	out.Written, out.Dropped = nil, nil
	for _, k := range written {
		if _, ok := out.Of[k]; ok && !slices.Contains(out.Written, k) {
			out.Written = append(out.Written, k)
		}
	}
	for _, k := range dropped {
		if _, ok := out.Of[k]; !ok && !slices.Contains(out.Dropped, k) {
			out.Dropped = append(out.Dropped, k)
		}
	}
	return out
}

// layoutOnce lays root out once, and lists the elements whose scrollbar
// gutter changed in a retained layout.
func layoutOnce(root *dom.Node, avail spaces, fonts box.Fonts, rounding bool, anchors *anchoring,
	opts ...Option) (*box.Tree, []*dom.Node, *tree) {
	t := &tree{fonts: fonts, out: &box.Tree{Of: map[*dom.Node]*box.Box{}}, anchors: anchors}
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
		t.prevTurns = k.turns
		defer func() { k.turns = t.turns }()
	}
	t.icb = avail.values()
	r := t.build(root, nil)
	defer t.sweep()
	if r == nil {
		return t.out, nil, t
	}
	port := t.viewportOf(root)
	if !avail.values().both() || frameOf(r).Vertical() {
		port = nil
	}
	var gutter, content size
	switch {
	case port != nil:
		gutter, content = t.viewport(r, avail.values(), port)
	case frameOf(r).Vertical():
		// the initial containing block takes the root's writing mode
		t.root(r, avail.across())
	default:
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
	return t.out, t.regutter, t
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
	// labelStyles is the style of the label row of each listbox optgroup style.
	labelStyles map[*style.Computed]*style.Computed
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
	// implicit gives an element's implicit anchor, and anchors are where
	// the last layout placed the anchors of the elements that read them.
	implicit func(*dom.Node) *dom.Node
	anchors  *anchoring
	// turns are the styles turned into vertical frames in this layout, and prevTurns
	// those of the last, which it takes again; icb is the initial containing block.
	turns, prevTurns map[turnKey]*style.Computed
	icb              size
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
	box *box.Box
	// st is the node's style in the frame of its parent's content, and inner, when not
	// the same, in its own, which it lays its content out with (turnedLayout); across is
	// set when the two frames' lines run on other axes, and inside while its own layout
	// runs.
	st             *style.Computed
	inner          *style.Computed
	across, inside bool
	kids           []*node
	// inline is the text the node lays out in lines, when its content is inline.
	inline *inlineContent

	cache     cache
	unrounded placed
	final     placed
	absolute  point
	hoisted   []*node
	parent    *node
	// placer is the containing block that placed n, an out-of-flow box, when its frame
	// isn't the one of n's parent's content: n is laid out and written in placer's.
	placer *node
	grid   *gridInfo
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
	// slider marks a range laid out as Chrome's: a row of its ::before, its
	// slider container and its ::after.
	slider bool
	// cell marks a table's cell, an element or an anonymous one; lowered is how far its
	// content stands below its content box's top to have its first line on its row's
	// baseline.
	cell    bool
	lowered float32
	// shared is the borders of a table that collapses them, and collapsed what that
	// leaves the table or one of its cells for borders, nil where borders are apart.
	shared    *borderGrid
	collapsed *rect
	// replaced is an img, natural its picture's size (zero without one); a range's
	// natural is the least content it takes, its track's (rangeNatural).
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
	// inLines is the float's place among the lines of the content it is in, nil for a
	// float before any.
	inLines *lineFloat
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
	case n.inline != nil, len(n.kids) == 0 && !n.isGrid():
		return contextLeaf
	case n.slider:
		return contextFlex
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
	if n := t.kept(e, parent); n != nil {
		t.adopt(n, parent)
		return n
	}
	n := t.element(e, parent)
	if isRange(e) {
		n.natural = rangeNatural(n.box.Style)
	}
	if k := controlPartOf(e); k != noPart {
		t.buildParts(n, k)
		return n
	}
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
		n.turnPicture()
		return n
	}
	if e.Tag() == dom.Iframe {
		buildFrame(n)
		n.turnPicture()
		return n
	}
	if inListBox(e) {
		t.buildOptgroup(n)
		return n
	}
	t.buildChildren(n, e.Flow())
	return n
}

// kept is e's node from the last layout when nothing under e changed since.
func (t *tree) kept(e *dom.Node, parent *node) *node {
	n := t.prevNodes[e]
	if n == nil {
		return nil
	}
	d := dom.Data(e)
	if d.Dirty&(dom.DirtyLayout|dom.DirtyDescendant) != 0 || n.box.Style != d.Style ||
		n.st != t.turned(styleOf(e), outerFrame(parent, styleOf(e))) || !e.SameFlow(n.children) {
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
	if n := old; n != nil && n.box.Style == st && dom.Data(e).Dirty&dom.DirtyLayout == 0 &&
		e.SameFlow(n.children) {
		t.retire(n, false)
		b := n.box
		*b = box.Box{Kind: box.Element, Node: e, Style: st, Children: b.Children[:0]}
		*n = node{box: b, st: st, parent: parent, fresh: true, readsParent: n.readsParent,
			kids: n.kids[:0], children: n.children, gutter: n.gutter, reached: t.gen}
		t.frame(n, st, outerFrame(parent, st))
		t.place(e, b)
		t.nodes[e] = n
		return n
	}
	b := &box.Box{Kind: box.Element, Node: e, Style: st}
	t.place(e, b)
	n := &node{box: b, st: st, parent: parent, fresh: true, readsParent: readsParent(st), reached: t.gen}
	t.frame(n, st, outerFrame(parent, st))
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
	t.buildItems(n, spaced(n, n.flexOrGrid(), t.lift(n, t.flatten(n, children))))
}

// buildItems gives n its in-flow children from items.
func (t *tree) buildItems(n *node, items []item) {
	inFlexOrGrid := n.flexOrGrid()
	if !inFlexOrGrid && !hasBlockChild(items) {
		t.buildInline(n, items)
		return
	}
	var run []item
	// inFlow is set once the container holds content in the flow: the anonymous blocks
	// after it don't start with the container's first line.
	inFlow := false
	// marked is the item whose marker stands inside it until a line takes the marker:
	// the first, or one of its own before a first block.
	marked := insideMarked(n, inFlexOrGrid)
	flush := func() {
		if onlyCollapsibleSpace(run) && marked == nil {
			run = nil
			return
		}
		anon := t.anonymous(n)
		content := t.inlineOf(anon, run, marked)
		marked = nil
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
			run = append(run, it)
			continue
		}
		if it.part != nil {
			flush()
			n.kids, inFlow = append(n.kids, it.part), true
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
			run = t.floatAmong(n, run, it)
			continue
		}
		if (!inFlexOrGrid || n.st.WebkitBox || n.slider) && !elementBlockLevel(it.element) {
			run = append(run, it)
			continue
		}
		flush()
		n.kids = append(n.kids, t.build(it.element, n))
		inFlow = inFlow || st.Display != style.DisplayNone && !outOfFlow(st.Position)
	}
	flush()
	n.noteFloats()
}

// flexOrGrid reports a node that lays its children out as flex or grid items.
func (n *node) flexOrGrid() bool {
	d := n.st.Display
	return n.slider || d == style.FlexDisplay || d == style.InlineFlex || d == style.GridDisplay ||
		d == style.InlineGrid
}

// isGrid reports a grid container, laid out by its tracks even when it holds nothing.
func (n *node) isGrid() bool {
	return !n.slider && (n.st.Display == style.GridDisplay || n.st.Display == style.InlineGrid)
}

// noteFloats sets holdsFloats where a float is among n's kids or under one of them.
func (n *node) noteFloats() {
	for _, k := range n.kids {
		if k.st.Float != style.FloatNone || k.holdsFloats {
			n.holdsFloats = true
		}
	}
}

// floatAmong takes the float it among n's inline content, of which run stands before it:
// before the content, the text going on round it, when run is collapsible space alone,
// else in run, placed among its lines.
func (t *tree) floatAmong(n *node, run []item, it item) []item {
	if !onlyCollapsibleSpace(run) {
		return append(run, it)
	}
	k := t.build(it.element, n)
	k.inLines = nil
	n.kids = append(n.kids, k)
	return run
}

// spaced is items less, in a -webkit-box laid out as flex, the white space between its
// children: Chrome drops it between inline ones too.
func spaced(n *node, inFlexOrGrid bool, items []item) []item {
	if !n.st.WebkitBox || !inFlexOrGrid {
		return items
	}
	return slices.DeleteFunc(items, func(it item) bool {
		return it.text != nil && onlyCollapsibleSpace([]item{it})
	})
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
	n.inline = t.inlineOf(n, items, insideMarked(n, false))
}

// insideMarked is n where it is a list item whose marker stands inside it, nil
// otherwise or when it lays its children out as a flex or grid container.
func insideMarked(n *node, flexOrGrid bool) *node {
	if n.st.ListItem && n.st.ListInside && n.box.Kind == box.Element && !flexOrGrid {
		return n
	}
	return nil
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
		if h := k.st.InitialLetter.Raise; h > 0 {
			n.kids = append(n.kids, raisedRoom(anon, h))
		}
		n.kids = append(n.kids, k)
		n.anonymous = true
	}
	anon.kids = kept
}

// raisedRoom is an empty block of anon's style h tall beside it: the room a
// drop cap raised over its first line takes above the lines.
func raisedRoom(anon *node, h float32) *node {
	s := *anon.st
	s.Size.Height = style.Px(h)
	return &node{box: &box.Box{Kind: box.AnonymousBlock, Style: &s}, st: &s, parent: anon.parent, fresh: true}
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
	// part is a control's part, built already (buildParts).
	part *node
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
		if it.text != nil && floatedLetter(it.text) || it.part != nil {
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
	g := letterNode(text)
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
// inherits the text properties and nothing else, text-box-trim being none of them.
func (t *tree) anonymous(parent *node) *node {
	st := t.anonymousStyles[parent.st]
	if st == nil {
		s := style.Initial()
		s.Display = style.Block
		s.Text = parent.st.Text
		s.TextBoxTrim = style.TrimNone
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
	outer, in := frameOf(parent), size{}
	if parent != nil {
		in = parent.final.size
	} else if outer = frameOf(n); outer.Vertical() {
		in = t.icb.across()
	}
	if n.placer != nil {
		outer = frameOf(n.placer)
	}
	f := turnPlaced(n.final, outer, style.HorizontalTB)
	rewrite := t.keep == nil || n.fresh || n.performed == t.gen
	n.fresh = false
	b.Size = geom.Vec2{X: f.size.w, Y: f.size.h}
	var framed geom.Vec2
	if n.st.Display != style.DisplayNone && (placed || !n.atomic) {
		origin := point{}
		if parent != nil {
			origin = parent.absolute
		}
		at := n.absolute
		b.Pos = geom.Vec2{X: at.x - origin.x, Y: at.y - origin.y}
		framed = b.Pos
		if p := n.placer; p != nil {
			b.Pos = placedPos(n, p)
		} else if outer.Vertical() {
			r := turnRect(geom.Rect{X: b.Pos.X, Y: b.Pos.Y, W: n.final.size.w, H: n.final.size.h}, in, outer)
			b.Pos = geom.Vec2{X: r.X, Y: r.Y}
		}
		b.Sticky = geom.Vec2{}
	}
	b.Margin = box.Edges{Top: f.margin.t, Right: f.margin.r, Bottom: f.margin.b, Left: f.margin.l}
	b.Border = box.Edges{Top: f.border.t, Right: f.border.r, Bottom: f.border.b, Left: f.border.l}
	b.Padding = box.Edges{Top: f.padding.t, Right: f.padding.r, Bottom: f.padding.b, Left: f.padding.l}
	b.Content, b.ScrollStart = scrollRange(n.st, f)
	b.Scrollbar = endBars(b.Style, f.scrollbar)
	b.Wrap = box.Edges{Top: n.captionRoom[0], Bottom: n.captionRoom[1]}
	b.Cut = n.cut
	b.Discarded = n.discarded
	if n.frag != nil || n.columns != nil || b.Broken || b.Columns != nil {
		n.writeColumns()
		turnBroken(b, framed, in, outer)
	}
	n.final = turnPlaced(n.final, outer, frameOf(n))
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

// scrollRange is the size of what a box of style st, placed as f, scrolls
// and its least offset: its scrollable overflow, but across a box that
// keeps a stable gutter, the range Chrome gives it once its content
// overflows that way: as long as the content less the padding box but
// the vertical bar that stands, whatever gutter is empty, a right-to-left
// box starting scrolled by its empty gutters (decision 0198).
// placedPos is where n, an out-of-flow box p placed in its own frame, stands in its
// parent's box on the page: its place in p turned onto the page, less the place of its
// parent there, its ancestors up to p being written already.
func placedPos(n, p *node) geom.Vec2 {
	at := geom.Rect{X: n.absolute.x - p.absolute.x, Y: n.absolute.y - p.absolute.y, W: n.final.size.w,
		H: n.final.size.h}
	if mode := frameOf(p); mode.Vertical() {
		at = turnRect(at, p.final.size, mode)
	}
	pos := geom.Vec2{X: at.X, Y: at.Y}
	for a := n.parent; a != nil && a != p; a = a.parent {
		pos = pos.Sub(a.box.Pos)
	}
	return pos
}

func scrollRange(st *style.Computed, f placed) (geom.Vec2, geom.Vec2) {
	o := f.overflow.rect
	content := geom.Vec2{X: o.r - o.l, Y: o.b - o.t}
	start := geom.Vec2{X: fmin(o.l, 0), Y: fmin(o.t, 0)}
	if !st.GutterStable || !scrollContainer(st.Overflow.Height) || st.Overflow.Width == style.Hidden {
		return content, start
	}
	if st.Mode.Vertical() {
		return verticalScrollRange(st, f, content, start)
	}
	g := gutterEdges(st, point{x: f.scrollbar.w, y: f.scrollbar.h})
	w := f.size.w - f.border.l - f.border.r
	bar := float32(0)
	if h := st.Overflow.Height; h == style.Scroll ||
		h == style.OverflowAuto && content.Y > f.size.h-f.border.t-f.border.b-f.scrollbar.h {
		bar = st.ScrollbarWidth
	}
	port := w - (f.scrollbar.w - fmin(startGutter(st), f.scrollbar.w))
	over := content.X - (w - bar)
	if over <= 0 {
		return content, start
	}
	lo, hi := float32(0), over
	if st.Dir == style.RTL {
		shift := g.l + g.r - bar
		lo, hi = -(shift + over), -shift
	}
	return geom.Vec2{X: hi - lo + port, Y: content.Y}, geom.Vec2{X: lo, Y: start.Y}
}

// verticalScrollRange is scrollRange for a box of a vertical writing mode:
// along its lines, the content less the padding box but the horizontal
// bar that stands, whatever gutter is empty; a box whose blocks start on
// the right starts scrolled to its end.
func verticalScrollRange(st *style.Computed, f placed, content, start geom.Vec2) (geom.Vec2, geom.Vec2) {
	w, h := f.size.w-f.border.l-f.border.r, f.size.h-f.border.t-f.border.b
	bars := endBars(st, f.scrollbar)
	port := geom.Vec2{X: w - bars.X, Y: h - bars.Y}
	if content.X > port.X && st.Mode.BlockFromRight() {
		// the overflow, from the right, takes in the vertical bar's room
		over := fmax(content.X-w, 0)
		content.X, start.X = over+port.X, -over
	}
	bar := float32(0)
	if o := st.Overflow.Width; o == style.Scroll || o == style.OverflowAuto && content.X > port.X {
		bar = st.ScrollbarHeight
	}
	if content.Y > port.Y {
		content.Y = fmax(content.Y-(h-bar), 0) + port.Y
	}
	return content, start
}

// markerGap is how far a bullet stands from its item's content, past its own offset, as
// Chrome's; a picture inside its item stands as far from the content after it.
const markerGap = 7

// rtlMarkerPull is how much nearer its content Chrome draws the bullet or the picture of
// a right-to-left item than the mirror of a left-to-right one's.
const rtlMarkerPull = 3.5

// markerContent is what a list item's ::marker shows, in the style it draws in: its
// text, or a picture of its size (list-style-image's, or its gradient's), or its bullet.
type markerContent struct {
	st       *style.Computed
	text     string
	picture  geom.Vec2
	source   string
	gradient *style.Gradient
	bullet   bool
	// ascent is the marker font's, rounded as Chrome places a bullet by it.
	ascent float32
}

// markerOf is what n's ::marker shows: the marker string css worked out, its ::marker's
// content or its counter style's, or the list-style-image in their place, false when it
// shows nothing. The style is the ::marker's where a rule styles it, the item's
// otherwise.
func (t *tree) markerOf(n *node) (markerContent, bool) {
	e := n.box.Node
	if e == nil || !n.st.ListItem || e.Tag() == dom.Summary || n.st.Display == style.DisplayNone {
		return markerContent{}, false
	}
	data := dom.Data(e)
	m := markerContent{st: n.st}
	if data.Marker != nil {
		m.st = data.Marker
	}
	m.ascent = float32(math.Round(float64(t.fonts.Metrics(m.st.Font).Ascent)))
	if data.HasMarkerText {
		inWord := false
		m.text = shownText(data.MarkerText, m.st, nil, &inWord)
	}
	if data.Marker == nil || data.Marker.Content.Kind == style.ContentNormal {
		switch {
		case n.st.ListGradient != nil:
			// a picture of no size of its own: Chrome's is half the font's ascent square
			side := float32(math.Floor(float64(m.ascent))) / 2
			m.picture, m.gradient = geom.Vec2{X: side, Y: side}, n.st.ListGradient
			return m, true
		case n.st.ListImage != "" && t.images != nil:
			if s, ok := t.images.ImageSize(n.st.ListImage); ok && s.X > 0 && s.Y > 0 {
				m.picture, m.source = s, n.st.ListImage
				return m, true
			}
		}
	}
	switch {
	case m.text != "":
		return m, true
	case data.HasMarkerText:
		return m, false
	case m.st.ListStyle == style.ListDisc || m.st.ListStyle == style.ListCircle || m.st.ListStyle == style.ListSquare:
		m.bullet = true
		return m, true
	}
	return m, false
}

// bullet is the side of a bullet's square and how far below the font's top it starts,
// as Chrome sizes it from the ascent.
func (m markerContent) bulletRect() (float32, float32) {
	offset := float32(math.Floor(float64(m.ascent * 2 / 3)))
	return float32(math.Floor(float64((offset + 1) / 2))), float32(math.Floor(float64(3 * (m.ascent - offset) / 2)))
}

// insideWidth is how far the marker inside its item puts the content after it, as
// Chrome's: its text; its picture and markerGap; its bullet, two more points and its
// margins, one point before and the font's size after.
func (m markerContent) insideWidth(fonts box.Fonts) float32 {
	switch {
	case m.picture.X > 0:
		return m.picture.X + markerGap
	case m.bullet:
		side, _ := m.bulletRect()
		return side + 2 - 1 + m.st.Font.Size
	}
	return fonts.Advance(m.st.Font, m.text) + m.st.Spacing(m.text)
}

// place puts the marker's box b in the rectangle r its piece takes on a line, on
// baseline lb below r's top: a bullet or a picture at its start, the right end in a
// right-to-left item, whose marker text is drawn as that direction orders it.
func (m markerContent) place(b *box.Box, r geom.Rect, lb float32) {
	b.Style = m.st
	rtl := m.st.Dir == style.RTL
	switch {
	case m.picture.X > 0:
		b.Pos, b.Size = geom.Vec2{X: r.X, Y: r.Y + lb - m.picture.Y}, m.picture
		if rtl {
			b.Pos.X = r.X + r.W - m.picture.X
		}
		b.Picture, b.Gradient = m.source, m.gradient
	case m.bullet:
		side, top := m.bulletRect()
		b.Pos, b.Size = geom.Vec2{X: r.X, Y: r.Y + lb - m.ascent + top}, geom.Vec2{X: side, Y: side}
		if rtl {
			b.Pos.X = r.X + r.W - side
		}
	default:
		text := m.text
		if rtl {
			text = visualText(text, 1)
		}
		b.Pos, b.Size = geom.Vec2{X: r.X, Y: r.Y}, geom.Vec2{X: r.W, Y: r.H}
		b.Lines = []box.Fragment{{Text: text, Rect: geom.Rect{W: r.W, H: r.H}, Baseline: lb}}
		return
	}
	// a bullet or a picture takes the pointer over all its piece of the line, as Chrome's
	b.Area = geom.Rect{X: r.X - b.Pos.X, Y: r.Y - b.Pos.Y, W: r.W, H: r.H}
}

// listMarker gives a list item its ::marker box, outside the item beside its first line,
// ending where the item's content starts, a bullet or a picture Chrome's distance before
// it, the picture on the line's baseline. An item without a line has none, and one whose
// marker stands inside it has it among its lines.
func (t *tree) listMarker(n *node) {
	if n.box.Kind != box.Element || n.st.ListInside {
		return
	}
	mark, ok := t.markerOf(n)
	if !ok {
		return
	}
	line, at, ok := firstLine(n.box, geom.Vec2{})
	if !ok {
		return
	}
	m := &box.Box{Kind: box.Marker, Node: n.box.Node, Style: mark.st}
	left := n.final.border.l + n.final.padding.l
	rtl := n.st.Dir == style.RTL
	right := n.final.size.w - n.final.border.r - n.final.padding.r
	baseline := at.Y + line.Baseline
	switch {
	case mark.picture.X > 0:
		m.Pos = geom.Vec2{X: left - mark.picture.X - markerGap, Y: baseline - mark.picture.Y}
		if rtl {
			m.Pos.X = right + markerGap - rtlMarkerPull
		}
		m.Size, m.Picture, m.Gradient = mark.picture, mark.source, mark.gradient
		m.Area = t.markerArea(mark, m, baseline, 0, mark.picture.X, rtl)
	case mark.bullet:
		side, top := mark.bulletRect()
		offset := float32(math.Floor(float64(mark.ascent * 2 / 3)))
		m.Pos = geom.Vec2{X: left - offset - markerGap, Y: baseline - mark.ascent + top}
		if rtl {
			m.Pos.X = right + offset + markerGap - rtlMarkerPull - side
		}
		m.Size = geom.Vec2{X: side, Y: side}
		m.Area = t.markerArea(mark, m, baseline, -1, side+2, rtl)
	default:
		text := mark.text
		w := t.fonts.Advance(mark.st.Font, text) + mark.st.Spacing(text)
		m.Pos = geom.Vec2{X: left - w, Y: at.Y + line.Rect.Y}
		if rtl {
			m.Pos.X, text = right, visualText(text, 1)
		}
		m.Size = geom.Vec2{X: w, Y: line.Rect.H}
		m.Lines = []box.Fragment{{Text: text, Rect: geom.Rect{W: w, H: line.Rect.H}, Baseline: baseline - m.Pos.Y}}
	}
	n.box.Append(m)
}

// markerArea is what an outside marker box m of a bullet or a picture takes the pointer
// over, in m's space, as Chrome's: w wide from x past the bullet's or the picture's left,
// where a right-to-left item's stands as the mirror of a left-to-right one's, and the
// height of the marker's line on the baseline.
func (t *tree) markerArea(mark markerContent, m *box.Box, baseline, x, w float32, rtl bool) geom.Rect {
	if rtl {
		x += rtlMarkerPull
	}
	h, above := lineHeightOf(t.fonts.Metrics(mark.st.Font), mark.st)
	return geom.Rect{X: x, Y: baseline - above - m.Pos.Y, W: w, H: h}
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
	n.final.scrollbar = n.barRoom(n.final.scrollbar)
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
