package layout

import (
	"math"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lantervale/ember/ui/internal/dom"

	"github.com/rivo/uniseg"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/shape"
	"github.com/lantervale/ember/ui/internal/bidi"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// inlineContent is a block container's inline formatting context: its text, the inline
// elements around it, and its atomic inlines (inline-block, inline-flex, inline-grid),
// laid out in lines.
type inlineContent struct {
	owner *node
	runs  []*textRun
	spans []*inlineSpan
	// pieces is the content cut at break opportunities, text measured once.
	pieces []piece
	// metrics caches each style's font metrics for this layout.
	metrics map[*style.Computed]box.Metrics
	// marker is a summary's disclosure triangle, or the ::marker of a list item whose
	// marker stands inside it, which mark draws; nil elsewhere.
	marker *box.Box
	mark   *markerContent
	// outside is set where mark is the outside marker of a list item with nothing else
	// to show: its piece takes no room on the one line it makes, the marker's line height
	// tall, as Chrome's, and markerLine and markerBaseline keep where that line went, from
	// its start to its end past the floats, for listMarker to stand the marker beside it.
	outside        bool
	markerLine     geom.Rect
	markerBaseline float32
	// oofs are the out-of-flow elements inside the content, by their node.
	oofs map[*node]*inlineOOF
	// anchors are where in the runs each sits: before run i.
	anchors []anchorAt
	// unit is the layout unit, per point, that each run's text on a line rounds up to.
	unit float32
	// inWord is set while the runs gathered so far end inside a word, for capitalize.
	inWord bool
	// stamp counts the lines whose boxes were aligned, and pending holds the boxes of the
	// one being aligned that wait for their parent's size or the line's.
	stamp   int
	pending []pendingAlign
	// moved is set once a vertical-align has shifted a box of the content: until then
	// every run and span stands on its line's baseline.
	moved bool
	// broken are the lines last broken at brokenAt, and unbroken the lines with
	// no width to fit, kept while no piece's width changes, as a node is
	// measured and laid out at its width and at none several times.
	broken, unbroken []line
	brokenAt         float32
	// flow is the formatting context whose floats the lines go round, set while the
	// owner is laid out among them, with where its content box starts in its border box;
	// flowed marks broken as lines broken round floats.
	flow              *blockContext
	flowTop, flowLeft float32
	flowed            bool
	// clearedAfter is how far below the last of those lines a br ending them clears the
	// floats, which the owner's content holds, as Chrome's.
	clearedAfter float32
	// lineFloats are the floats after content in the lines, in order, and floatsFrom
	// the context they are placed from while pending.
	lineFloats []*lineFloat
	floatsFrom *floatsFrom
	// unindented is set where the first line is not its block container's: an anonymous
	// block after other content.
	unindented bool
	// reorders is set when the pieces' bidi levels may order a line otherwise than its
	// text, and base is the level of its first paragraph.
	reorders bool
	base     bidi.Level
	// isolated is, for each span that isolates its content (unicode-bidi isolate,
	// isolate-override or plaintext), the level it stands at in the text round it: its
	// isolate initiator's.
	isolated map[int]bidi.Level
	// line is the style of the block's first line where a ::first-line styles it, nil
	// elsewhere; firstPieces are the pieces as measured in the first line's styles, and
	// firstOn is set while they, and each run's and span's first-line style, stand in
	// for their own (useFirst). firstInWord is inWord for the first line's text.
	line        *style.Computed
	firstPieces []piece
	firstOn     bool
	firstInWord bool
	// allFirst keeps the first line's styles on every line, as the content is measured.
	allFirst bool
	// back is the box painting the first line's background, nil without one.
	back *box.Box
	// height is the owner's content height its atomic inlines' percentages are of, none
	// where it depends on the content.
	height float32
	// space is the advance of a space in the owner's font.
	space float32
	// lineBoxes are the tops and bottoms of the lines last written.
	lineBoxes []span
}

// inlineOOF is an absolutely positioned (or fixed) element inside inline content: its
// static position, the line spot it would take, and, when the span holding it is
// positioned, the containing block that span makes. Both are in the owner's border box.
// leafChildren reads it through oofOf.
type inlineOOF struct {
	static point
	// areaAt and areaSize are the containing block when positioned.
	areaAt     point
	areaSize   size
	positioned bool
	span       int
}

// anchorAt is an out-of-flow element before run i, in the box parent: the owner's, top,
// or an inline element's.
type anchorAt struct {
	node   *node
	run    int
	parent *box.Box
	top    bool
}

// oofOf is k's place in c's lines when k is out of flow inside it, nil otherwise.
func (c *inlineContent) oofOf(k *node) *inlineOOF {
	if c == nil {
		return nil
	}
	return c.oofs[k]
}

// textRun is one text node, its whitespace processed, or one atomic inline.
type textRun struct {
	box    *box.Box
	parent *box.Box
	st     *style.Computed
	text   string
	atomic *atomicInline
	// span is the innermost span holding the run, -1 for none.
	span int
	// shift is how far below the baseline of the line last given to lineBox the run's
	// own stands, vertical-align's.
	shift float32
	// skip is how many characters of the node's text stand before the run's: its first
	// letter, which a ::first-letter box holds.
	skip int32
	// lineSt and lineText are the run's style and text on its block's first line.
	lineSt   *style.Computed
	lineText string
}

// atomicInline is an inline-level box laid out on its own and placed in a line as one piece.
type atomicInline struct {
	node          *node
	margin        rect
	width, height float32
	baseline      float32
	overflow      overflowRect
}

// inlineSpan is an inline element: its box covers the fragments of the runs inside it, and
// its margin, border and padding take room at its start and end, as edge pieces.
type inlineSpan struct {
	box    *box.Box
	parent *box.Box
	runs   []int
	st     *style.Computed
	// first and end are the runs [first, end) inside it; open and close its edge pieces.
	first, end  int
	open, close int
	// margin, border and padding are resolved against the containing block's width.
	margin, border, padding rect
	// up is the span holding this one, -1 for none.
	up int
	// shift is how far below the baseline of the line last given to lineBox the span's
	// own stands; line is that line's stamp, and extent what the span and all it holds
	// there take around its baseline.
	shift  float32
	line   int
	extent extent
	// holds is the stamp of the last line the span holds an atomic inline on.
	holds int
	// noStart and noEnd are set on a part of an element blocks cut that is not its
	// first or not its last: it has no edge there.
	noStart, noEnd bool
	// lineSt is the span's style on its block's first line, and lineMargin, lineBorder and
	// linePadding its edges there.
	lineSt                              *style.Computed
	lineMargin, lineBorder, linePadding rect
}

type pieceKind uint8

const (
	word pieceKind = iota
	gap
	breakHere
	newline
	atomic
	// edge is a span's start (margin, border and padding on its left) or end; it sticks to
	// the content beside it.
	edge
	// emergency is a break between two graphemes of a word whose run has overflow-wrap:
	// taken only when the word fits no line whole.
	emergency
	// marker is a summary's disclosure triangle and the space after it, first on its first
	// line and stuck to what follows.
	marker
	// anchor is where an out-of-flow element sits, zero wide; span is its index in anchors.
	anchor
	// tab is a tab of text that keeps it, as wide as it takes to reach the next tab stop.
	tab
	// softHyphen is a soft hyphen a line may break after, of no width but at the end of
	// a line, where it shows its hyphen, hyphen wide.
	softHyphen
)

// piece is a word, a space, a zero-width break opportunity, a forced break, an atomic
// inline, inside run, or a span's edge (span is its index in spans, and run a run beside it).
type piece struct {
	kind  pieceKind
	run   int
	text  string
	width float32
	span  int
	// at is where a word's or a gap's text starts in its run's, in characters.
	at    int32
	start bool
	// level is its bidi embedding level: odd ones are shown right to left.
	level bidi.Level
	// stop is a tab's distance between tab stops, none under tab-size: 0, and least
	// the room it takes at the least.
	stop, least float32
	// unspaced marks a gap whose first space starts the content's text, and so takes no
	// word-spacing.
	unspaced bool
	// hyphen is a soft hyphen's width at a line's end; shows marks the one that ends
	// the line it is shown in.
	hyphen float32
	shows  bool
	// han is what text-spacing-trim and text-autospace do to a word, in its
	// width; nil for nothing.
	han *hanSpacing
	// whole numbers the text Chrome shapes as one (shapingRuns) that the piece is in,
	// 0 where it cuts it.
	whole int32
	// cutStart and cutEnd are what a word's width gains shaped again without the text
	// before it, where a line starts at it, and without the rest of its word after it,
	// where a line breaks inside the word after it (cuts).
	cutStart, cutEnd float32
	// ligated marks a space a ligature draws that starts the word before it, with text
	// of its run after it: no line breaks after it on a line the word starts
	// (ligatureOpens).
	ligated bool
}

const (
	zeroWidthSpace = "\u200b"
	softHyphenChar = '\u00ad'
	// lineSeparator and paragraphSeparator end their line whatever the white space, as
	// UAX #14 breaks after them, where Chrome does not (decision 0242); a paragraph separator
	// also ends its bidi paragraph.
	lineSeparator      = '\u2028'
	paragraphSeparator = '\u2029'
)

// unshown drops from s the characters that take no room and show nothing: zero-width
// spaces, soft hyphens and the carriage returns of text that keeps them.
func unshown(s string) string {
	if !strings.ContainsAny(s, zeroWidthSpace+string(softHyphenChar)+"\r") {
		return s
	}
	return strings.Map(func(r rune) rune {
		if r == softHyphenChar || string(r) == zeroWidthSpace || r == '\r' {
			return -1
		}
		return r
	}, s)
}

// blanks turns the whitespace that collapses like a space into one.
var blanks = strings.NewReplacer("\t", " ", "\r", " ", "\f", " ") //nolint:gochecknoglobals // a Replacer built once

// collapsedBlanks is text with the whitespace ws collapses turned into spaces: newlines too,
// unless ws is pre-line.
func collapsedBlanks(text string, ws style.WhiteSpace) string {
	if !collapses(ws) {
		return text
	}
	text = blanks.Replace(text)
	if !ws.KeepsBreaks() {
		text = strings.ReplaceAll(text, "\n", " ")
	}
	return text
}

func collapses(ws style.WhiteSpace) bool { return !ws.KeepsSpaces() }

func wraps(ws style.WhiteSpace) bool { return ws.Wraps() }

func isAtomicDisplay(d style.Display) bool {
	return d == style.InlineBlock || d == style.InlineFlex || d == style.InlineGrid
}

// inlineOf gathers the inline content of items for owner, nil when it is only collapsible
// whitespace; listItem is the item whose marker stands inside it, first on its first
// line, nil for none.
func (t *tree) inlineOf(owner *node, items []item, listItem *node) *inlineContent {
	c := &inlineContent{owner: owner, unit: unitsPerPoint(t.fonts), line: firstLineOf(owner), height: none}
	t.gather(c, owner.box, items)
	if n := owner.box.Node; n != nil && n.Tag() == dom.Summary && owner.st.ListItem && markerOf(owner.st) != "" {
		c.marker = &box.Box{Kind: box.Marker, Node: n, Style: owner.st}
	} else if listItem != nil {
		if m, ok := t.markerOf(listItem); ok {
			c.mark, c.marker = &m, &box.Box{Kind: box.Marker, Node: listItem.box.Node, Style: m.st}
		}
	}
	visible := c.marker != nil || framedCut(items) || c.framed()
	for _, r := range c.runs {
		if r.atomic != nil || !collapses(r.st.WhiteSpace) || !isCollapsibleSpace(r.text) {
			visible = true
		}
	}
	if !visible && listItem == nil && outsideMarked(owner) {
		if m, ok := t.markerOf(owner); ok {
			c.mark, c.marker, c.outside = &m, &box.Box{Kind: box.Marker, Node: owner.box.Node, Style: m.st}, true
			visible = true
		}
	}
	if !visible {
		return nil
	}
	for _, r := range c.runs {
		if r.atomic != nil {
			owner.kids = append(owner.kids, r.atomic.node)
		}
	}
	c.cut(t.fonts)
	return c
}

// shows reports runs gathered that show something: an atomic inline or text that is not
// collapsible space alone.
func (c *inlineContent) shows() bool {
	for _, r := range c.runs {
		if r.atomic != nil || r.text != "" && (!collapses(r.st.WhiteSpace) || !isCollapsibleSpace(r.text)) {
			return true
		}
	}
	return false
}

// framed reports an inline element gathered with a margin, a border or a padding on its
// start or end: a line box holds it, text or none, as CSS 2 keeps a line for it.
func (c *inlineContent) framed() bool {
	for _, s := range c.spans {
		if !s.noStart && edgeRoom(s.st, true) || !s.noEnd && edgeRoom(s.st, false) {
			return true
		}
	}
	return false
}

// edgeRoom reports a margin, a padding or a border that is not 0 on the start of an
// inline of style st, or its end: its right first in right-to-left.
func edgeRoom(st *style.Computed, start bool) bool {
	margin, padding, border := st.Margin.Left, st.Padding.Left, st.Border.Left
	if start == (st.Dir == style.RTL) {
		margin, padding, border = st.Margin.Right, st.Padding.Right, st.Border.Right
	}
	return border != 0 || margin.Px != 0 || margin.Percent != 0 || padding.Px != 0 || padding.Percent != 0
}

// framedCut reports a part of an inline element blocks cut among items, or inside one,
// with a margin, a border or a padding on its edge: Chrome keeps a line for it, text or
// none.
func framedCut(items []item) bool {
	for _, it := range items {
		if it.cut == nil {
			continue
		}
		st := styleOf(it.element)
		if it.first && edgeRoom(st, true) || it.last && edgeRoom(st, false) || framedCut(it.inside) {
			return true
		}
	}
	return false
}

func (t *tree) gather(c *inlineContent, parent *box.Box, items []item) {
	for at := 0; at < len(items); at++ {
		it := items[at]
		if it.text != nil {
			t.gatherText(c, parent, it.text)
			continue
		}
		st := styleOf(it.element)
		if st.Display == style.DisplayNone {
			continue
		}
		if it.cut == nil && partOf(it.element) != style.NoTablePart {
			at = t.gatherTable(c, parent, items, at) - 1
			continue
		}
		if st.Float != style.FloatNone {
			// A float inside an inline element: its block container takes it from the
			// owner's kids and places it before this content, among the lines when
			// content stands before it.
			n := t.build(it.element, c.owner)
			c.owner.kids = append(c.owner.kids, n)
			n.inLines = nil
			if c.shows() {
				n.inLines = &lineFloat{node: n, run: len(c.runs)}
				c.lineFloats = append(c.lineFloats, n.inLines)
			}
			continue
		}
		if outOfFlow(st.Position) {
			n := t.build(it.element, c.owner)
			c.owner.kids = append(c.owner.kids, n)
			if c.oofs == nil {
				c.oofs = map[*node]*inlineOOF{}
			}
			c.oofs[n] = &inlineOOF{span: -1}
			c.anchors = append(c.anchors, anchorAt{node: n, run: len(c.runs), parent: parent,
				top: parent == c.owner.box})
			continue
		}
		if it.element.Tag() == dom.Br {
			t.lineBreak(c, parent, it.element, st)
			continue
		}
		// an inline element of another writing mode than its line is laid out as an inline-block
		if isAtomicDisplay(st.Display) ||
			st.Display == style.Inline && (pictured(it.element) || st.Mode != frameOf(c.owner)) {
			n := t.build(it.element, c.owner)
			n.atomic = true
			c.runs = append(c.runs,
				&textRun{box: n.box, parent: parent, st: st, lineSt: st, span: -1, atomic: &atomicInline{node: n}})
			continue
		}
		b := &box.Box{Kind: box.Element, Node: it.element, Style: st}
		inside := it.inside
		if it.cut != nil {
			it.cut.parts = append(it.cut.parts, b)
		} else {
			t.place(it.element, b)
			inside = t.flatten(c.owner, it.element.Flow())
		}
		first, inner := len(c.runs), len(c.spans)
		t.gather(c, b, inside)
		mode := frameOf(c.owner)
		s := &inlineSpan{box: b, parent: parent, st: t.turned(st, mode), first: first, end: len(c.runs),
			noStart: it.cut != nil && !it.first, noEnd: it.cut != nil && !it.last}
		s.lineSt = t.turned(c.lineStyle(it.element, st), mode)
		for i := first; i < len(c.runs); i++ {
			s.runs = append(s.runs, i)
		}
		c.hold(s, inner)
	}
}

// gatherTable adds the anonymous inline table round the table parts from items[at], and
// is the index after them and the white space right after them, which makes no box as
// in Chrome.
func (t *tree) gatherTable(c *inlineContent, parent *box.Box, items []item, at int) int {
	parts, next := tableParts(items, at)
	n := t.anonymousInlineTable(c.owner, parent.Style, parts)
	c.runs = append(c.runs,
		&textRun{box: n.box, parent: parent, st: n.st, lineSt: n.st, span: -1, atomic: &atomicInline{node: n}})
	for next < len(items) && items[next].text != nil && droppedSpace(items[next].text) {
		next++
	}
	return next
}

// droppedSpace reports a text of collapsible white space alone.
func droppedSpace(text *dom.Node) bool {
	p := text.Parent()
	return isCollapsibleSpace(text.Text()) && (p == nil || collapses(styleOf(p).WhiteSpace))
}

// firstLineOf is the style of the first line of owner's block, nil where no
// ::first-line styles it.
func firstLineOf(owner *node) *style.Computed {
	e := owner.box.Node
	if e == nil && owner.parent != nil {
		e = owner.parent.box.Node
	}
	if e == nil || e.IsText() {
		return nil
	}
	return dom.Data(e).FirstLine
}

// lineBack is the node of the box behind the first line of owner's block, nil for none.
func lineBack(owner *node) *dom.Node {
	e := owner.box.Node
	if e == nil && owner.parent != nil {
		e = owner.parent.box.Node
	}
	if e == nil || e.IsText() {
		return nil
	}
	return dom.Data(e).LineBack
}

// lineStyle is the style on the first line of what e holds, of style st: the block's
// first line's for its own text, e's there inside it.
func (c *inlineContent) lineStyle(e *dom.Node, st *style.Computed) *style.Computed {
	if c.line == nil || e == nil {
		return st
	}
	if b := c.owner.box.Node; e == b || b == nil && c.owner.parent != nil && e == c.owner.parent.box.Node {
		return c.line
	}
	if s := dom.Data(e).FirstLine; s != nil {
		return s
	}
	return st
}

// letterNode is the ::first-letter node that takes text's first letter, nil for none.
func letterNode(text *dom.Node) *dom.Node {
	g := dom.Data(text).Letter
	if g == nil || dom.Data(g).Style == nil || dom.Data(g).Letter != text {
		return nil
	}
	return g
}

// letterOf is the ::first-letter node that takes the first letter of text, which shows
// shown, and where the letter ends in shown, nil when none does.
func letterOf(text *dom.Node, shown string) (*dom.Node, int) {
	g := letterNode(text)
	if g == nil {
		return nil, 0
	}
	end := dom.Data(text).LetterEnd
	if kids := g.Children(); end > len(shown) || len(kids) != 1 || !strings.HasSuffix(shown[:end], kids[0].Text()) {
		return nil, 0
	}
	return g, end
}

// gatherText adds the run of text node n, after the ::first-letter box taking its
// first letter.
func (t *tree) gatherText(c *inlineContent, parent *box.Box, n *dom.Node) {
	st := c.owner.st
	if p := n.Parent(); p != nil && p != c.owner.box.Node {
		st = styleOf(p)
	}
	text, skip := Translated(n, t.translate), int32(0)
	if g, end := letterOf(n, text); g != nil {
		t.gather(c, parent, []item{{element: g}})
		skip = int32(utf8.RuneCountInString(text[:end])) //nolint:gosec // a text node's length
		// a node of its letter alone keeps a run of no text, which selection finds it by
		text = text[end:]
	}
	b := &box.Box{Kind: box.TextRun, Node: n, Style: st}
	r := &textRun{box: b, parent: parent, st: st, span: -1, skip: skip, lineSt: st,
		text: shownText(text, st, n.Parent(), &c.inWord)}
	if c.line != nil {
		r.lineSt = c.lineStyle(n.Parent(), st)
		r.lineText = shownText(text, r.lineSt, n.Parent(), &c.firstInWord)
	}
	c.runs = append(c.runs, r)
}

// hold adds span s, the spans gathered since inner and the runs in it that no span holds
// yet being its own.
func (c *inlineContent) hold(s *inlineSpan, inner int) {
	k := len(c.spans)
	s.up = -1
	for _, in := range c.spans[inner:] {
		if in.up < 0 {
			in.up = k
		}
	}
	for _, r := range c.runs[s.first:s.end] {
		if r.span < 0 {
			r.span = k
		}
	}
	c.spans = append(c.spans, s)
}

// lineBreak is a br: an inline box holding a line break that no white-space collapses.
func (t *tree) lineBreak(c *inlineContent, parent *box.Box, e *dom.Node, st *style.Computed) {
	kept, ok := t.breakStyles[st]
	if !ok {
		pre := *st
		pre.WhiteSpace = style.Pre
		kept = &pre
		if t.breakStyles == nil {
			t.breakStyles = map[*style.Computed]*style.Computed{}
		}
		t.breakStyles[st] = kept
	}
	b := &box.Box{Kind: box.Element, Node: e, Style: st}
	t.place(e, b)
	first := len(c.runs)
	c.runs = append(c.runs,
		&textRun{box: &box.Box{Kind: box.TextRun, Style: kept}, parent: b, st: kept, lineSt: kept, span: -1, text: "\n",
			lineText: "\n"})
	c.hold(&inlineSpan{box: b, parent: parent, st: st, lineSt: st, first: first, end: first + 1, runs: []int{first}},
		len(c.spans))
}

// cut splits the runs into pieces: whitespace collapsed as each run's white-space says,
// across runs too, so one space separates two runs at most; each span's start and end are
// edge pieces around its runs.
func (c *inlineContent) cut(fonts box.Fonts) {
	c.pieces, c.broken, c.unbroken = c.pieces[:0], nil, nil
	c.space = advance(fonts, c.owner.st, " ")
	afterSpace := true
	soft := c.softBreaks()
	levels := c.levels()
	c.reorders = levels != nil
	switch {
	case c.outside:
		c.pieces = append(c.pieces, piece{kind: marker})
	case c.mark != nil:
		c.pieces = append(c.pieces, piece{kind: marker, width: c.mark.insideWidth(fonts)})
	case c.marker != nil:
		c.pieces = append(c.pieces, piece{kind: marker, width: markerWidth(fonts, c.owner.st)})
	}
	for i, r := range c.runs {
		c.openSpans(i)
		c.anchorsAt(i)
		var lv []bidi.Level
		if levels != nil {
			lv = levels[i]
		}
		if r.atomic != nil {
			p := piece{kind: atomic, run: i}
			if lv != nil {
				p.level = lv[0]
			}
			c.pieces = append(c.pieces, p)
			afterSpace = false
			c.closeSpans(i)
			continue
		}
		if r.st.Combine && r.st.Mode.Vertical() {
			// text-combine-upright: the run is one piece, set across one em of the line
			c.pieces = append(c.pieces, piece{kind: word, run: i, text: collapsedBlanks(r.text, r.st.WhiteSpace),
				width: r.st.Font.Size})
			afterSpace = false
		} else {
			afterSpace = c.cutRun(fonts, i, r, soft[i], afterSpace, lv)
		}
		c.closeSpans(i)
	}
	c.joinRuns(fonts)
	c.spaceHan(fonts)
	c.openSpans(len(c.runs))
	c.anchorsAt(len(c.runs))
	if c.reorders {
		c.levelEdges()
	}
	if c.line != nil && !c.firstOn {
		c.cutFirst(fonts)
	}
}

// joinRuns measures again the words and spaces whose shaping reads the text of other
// runs. Where Chrome shapes their runs as one (shapingRuns), each is shaped as one with
// the text beside it, a ligature or a kerned pair across an element's edge the first
// element's, and in a face that shapes a space with its words (spaceTies) every word
// and space is. Elsewhere a word that starts or ends a run beside another run's letters
// is shaped between them: an Arabic word an element cuts joins across it, as Chrome
// shapes it with its paragraph's text as context.
func (c *inlineContent) joinRuns(fonts box.Fonts) {
	c.shapingRuns()
	wf, whole := fonts.(box.WholeFonts)
	cf, joins := fonts.(box.ContextFonts)
	var ties map[*style.Computed]bool
	for k := range c.pieces {
		p := &c.pieces[k]
		if p.kind != word && p.kind != gap {
			continue
		}
		r := c.runs[p.run]
		text := unshown(p.text)
		if whole && c.amid(fonts, k, &ties) {
			lead, before := beside(c.pieces, k, -1, p.whole)
			trail, after := beside(c.pieces, k, 1, p.whole)
			w := wf.AdvanceAmid(r.st.Font, text, lead, trail, before, after)
			p.width = w + r.st.SpacingAfter(c.textBefore(k), text)
			if p.unspaced {
				p.width -= r.st.WordGap()
			}
			if lead != "" {
				p.cutStart = wf.AdvanceAmid(r.st.Font, text, "", trail, before+lead, after) - w
			}
			c.markLigated(fonts, k, lead, trail, before, after)
			continue
		}
		if p.kind != word || !joins {
			continue
		}
		end := p.at + int32(utf8.RuneCountInString(p.text)) //nolint:gosec // a text node's length
		if p.at > 0 && int(end) < utf8.RuneCountInString(r.text) {
			continue
		}
		before, after := c.around(p.run, p.at, end)
		if !shape.JoinsAcross(before, after) {
			continue
		}
		// The word's own run is in its width already: a grapheme of a word cut by
		// break-all is measured within it.
		rs := []rune(r.text)
		from, to := min(int(p.at), len(rs)), min(int(end), len(rs))
		own := cf.AdvanceIn(r.st.Font, text, string(rs[max(0, from-contextRunes):from]),
			string(rs[to:min(len(rs), to+contextRunes)]))
		p.width += cf.AdvanceIn(r.st.Font, text, before, after) - own
	}
}

// markLigated marks space k, shaped between lead and trail, ligated where a ligature
// that starts the word before it draws it and the text after it is of its run: Chrome
// breaks no line inside a ligature that opens it, but where an element's text ends.
func (c *inlineContent) markLigated(fonts box.Fonts, k int, lead, trail, before, after string) {
	p := &c.pieces[k]
	cf, ok := fonts.(box.CaretFonts)
	if !ok || p.kind != gap || lead == "" {
		return
	}
	next := k + 1
	for next < len(c.pieces) && c.pieces[next].kind != word && c.pieces[next].kind != gap {
		next++
	}
	if next == len(c.pieces) || c.pieces[next].run != p.run {
		return
	}
	font := c.runs[p.run].st.Font
	if _, starts := cf.Carets(font, unshown(p.text), lead, trail, before, after); starts[0] {
		return
	}
	prev := k - 1
	for prev >= 0 && c.pieces[prev].kind != word && c.pieces[prev].kind != gap {
		prev--
	}
	if prev < 0 || c.pieces[prev].kind != word || c.pieces[prev].whole != p.whole {
		return
	}
	w := c.pieces[prev]
	wLead, wBefore := beside(c.pieces, prev, -1, w.whole)
	wTrail, wAfter := beside(c.pieces, prev, 1, w.whole)
	_, starts := cf.Carets(c.runs[w.run].st.Font, unshown(w.text), wLead, wTrail, wBefore, wAfter)
	p.ligated = !slices.Contains(starts[1:len(starts)-1], true)
}

// shapingRuns numbers the pieces Chrome shapes as one text in whole: the words and
// spaces of runs of one font description and one bidi level, across lines, between the
// edges of elements that take no room and keep the baseline (CSS Text's boundary
// shaping), with the break opportunities and anchors among them. A piece that cuts such
// a text, an atomic inline, a newline, a tab, takes 0, as text of a vertical line whose
// characters may stand does: Chrome shapes it element by element.
func (c *inlineContent) shapingRuns() {
	var n int32
	var last *style.Computed
	var level bidi.Level
	var lang string
	open, run := false, -1
	for k := range c.pieces {
		p := &c.pieces[k]
		p.whole = 0
		switch p.kind {
		case word, gap:
			st := c.runs[p.run].st
			if st.Stands() {
				open = false
				continue
			}
			was := lang
			if p.run != run {
				run, lang = p.run, languageOf(c.runs[p.run].box.Node)
			}
			if !open || p.level != level || !sameShaping(last, st) || lang != was {
				n++
			}
			open, last, level = true, st, p.level
			p.whole = n
		case edge:
			if c.cutsShaping(p) {
				open = false
			} else if open {
				p.whole = n
			}
		case breakHere, emergency, anchor, softHyphen:
			if open {
				p.whole = n
			}
		case newline, atomic, marker, tab:
			open = false
		}
	}
}

// sameShaping reports whether Chrome shapes text of styles a and b as one: of one font,
// letter-spacing, word-spacing, text-rendering and -webkit-font-smoothing.
func sameShaping(a, b *style.Computed) bool {
	return a == b || a.Font.Equal(b.Font) && a.LetterSpacing == b.LetterSpacing && a.WordSpacing == b.WordSpacing &&
		a.TextRendering == b.TextRendering && a.PixelText == b.PixelText && a.ExactText == b.ExactText &&
		a.SubpixelText == b.SubpixelText
}

// languageOf is the language of text node n, which Chrome's font description holds: the
// lang attribute of the nearest element holding it that has one, else the Document's.
func languageOf(n *dom.Node) string {
	if n == nil {
		return ""
	}
	for p := n.Parent(); p != nil; p = p.Parent() {
		if v, ok := p.Attr("lang"); ok {
			return v
		}
		if p.Parent() == nil {
			return dom.Data(p).Language
		}
	}
	return ""
}

// cutsShaping reports whether edge p of a span cuts the text Chrome shapes as one: it
// takes room, by a margin, a border or padding on its side, or the span stands off the
// baseline.
func (c *inlineContent) cutsShaping(p *piece) bool {
	sp := c.spans[p.span]
	st := sp.st
	if st.VerticalAlign != style.AlignBaselineText {
		return true
	}
	if p.start && sp.noStart || !p.start && sp.noEnd {
		return false
	}
	if p.start == (st.Dir == style.RTL) {
		return room(st.Margin.Right) || room(st.Padding.Right) || st.Border.Right != 0
	}
	return room(st.Margin.Left) || room(st.Padding.Left) || st.Border.Left != 0
}

// room reports whether a margin or padding of l takes room, or may.
func room(l style.Length) bool {
	return l.At(0) != 0 || l.At(1000) != 0
}

// amid reports whether piece k is shaped as one with the text beside it: another run's
// in the text Chrome shapes as one with it, or its own where its face shapes a space with
// the words beside it.
func (c *inlineContent) amid(fonts box.Fonts, k int, ties *map[*style.Computed]bool) bool {
	p := c.pieces[k]
	if p.whole == 0 {
		return false
	}
	st := c.runs[p.run].st
	tied, ok := (*ties)[st]
	if !ok {
		t, can := fonts.(spaceTies)
		tied = can && t.SpaceTies(st.Font)
		if *ties == nil {
			*ties = map[*style.Computed]bool{}
		}
		(*ties)[st] = tied
	}
	for _, dir := range [2]int{-1, 1} {
		for j := k + dir; j >= 0 && j < len(c.pieces) && c.pieces[j].whole == p.whole; j += dir {
			if q := c.pieces[j]; q.kind == word || q.kind == gap {
				if tied || q.run != p.run {
					return true
				}
				break
			}
		}
	}
	return false
}

// beside is the text of the pieces before piece k (dir -1) or after it (dir 1), as much
// as shaping reads: that of the text numbered whole, which shapes as one with it, and
// the rest round it, an atomic inline standing as an object replacement character, up
// to a newline.
func beside(ps []piece, k, dir int, whole int32) (string, string) {
	return besideBy(ps, k, dir, dir, whole)
}

// besideShown is beside for a line's pieces as shown, k one of a fragment shown right
// to left when rtl: the text before it is then on its right.
func besideShown(ps []piece, k, dir int, whole int32, rtl bool) (string, string) {
	if rtl {
		return besideBy(ps, k, -dir, dir, whole)
	}
	return beside(ps, k, dir, whole)
}

// besideBy is beside walking the pieces from k by step: the text before k where dir is
// -1, after it where 1.
func besideBy(ps []piece, k, step, dir int, whole int32) (string, string) {
	var shaped, round []rune
	in := true
	for j := k + step; j >= 0 && j < len(ps) && len(round) < contextRunes; j += step {
		q := ps[j]
		if q.kind == newline {
			break
		}
		var rs []rune
		switch q.kind {
		case word, gap:
			rs = []rune(unshown(q.text))
		case atomic:
			rs = []rune{'\uFFFC'}
		case breakHere, emergency, anchor, softHyphen, edge, marker, tab, newline:
		}
		if q.whole != whole {
			in = false
		}
		if dir < 0 {
			slices.Reverse(rs)
		}
		for _, r := range rs {
			if in && len(shaped) < contextRunes {
				shaped = append(shaped, r)
				continue
			}
			in = false
			if len(round) < contextRunes {
				round = append(round, r)
			}
		}
	}
	if dir < 0 {
		slices.Reverse(shaped)
		slices.Reverse(round)
	}
	return string(shaped), string(round)
}

// contextRunes is how many characters of the text on either side of a
// run its shaping reads.
const contextRunes = 5

// around is the text of the content before the characters from from to
// to of run i and after them, as much as shaping reads: an atomic inline
// stands as an object replacement character, which joins nothing.
func (c *inlineContent) around(i int, from, to int32) (string, string) {
	var before, after []rune
	for k := i; k >= 0 && len(before) < contextRunes; k-- {
		rs := []rune(c.runs[k].text)
		if c.runs[k].atomic != nil {
			rs = []rune{'\uFFFC'}
		}
		if k == i {
			rs = rs[:min(int(from), len(rs))]
		}
		for j := len(rs) - 1; j >= 0 && len(before) < contextRunes; j-- {
			before = append(before, rs[j])
		}
	}
	for k := i; k < len(c.runs) && len(after) < contextRunes; k++ {
		rs := []rune(c.runs[k].text)
		if c.runs[k].atomic != nil {
			rs = []rune{'\uFFFC'}
		}
		if k == i {
			rs = rs[min(int(to), len(rs)):]
		}
		after = append(after, rs[:min(len(rs), contextRunes-len(after))]...)
	}
	slices.Reverse(before)
	return string(before), string(after)
}

// cutFirst cuts the runs again as the first line has them, into firstPieces: the same
// pieces in its styles. Where its text cuts otherwise, the first line takes the runs'
// own styles.
func (c *inlineContent) cutFirst(fonts box.Fonts) {
	own := c.pieces
	c.pieces = nil
	c.swapStyles()
	c.firstOn = true
	c.cut(fonts)
	c.firstOn = false
	c.swapStyles()
	c.pieces, c.firstPieces = own, c.pieces
	same := len(own) == len(c.firstPieces)
	for i := 0; same && i < len(own); i++ {
		same = own[i].kind == c.firstPieces[i].kind && own[i].run == c.firstPieces[i].run
	}
	if !same {
		c.firstPieces = nil
	}
}

// useFirst has the content stand as its first line (on) or as the others, and reports
// how it stood: the pieces measured in the first line's styles and each run's and
// span's style there in place of their own.
func (c *inlineContent) useFirst(on bool) bool {
	was := c.firstOn
	if on == was || c.firstPieces == nil {
		return was
	}
	c.firstOn = on
	c.pieces, c.firstPieces = c.firstPieces, c.pieces
	c.swapStyles()
	return was
}

func (c *inlineContent) swapStyles() {
	for _, r := range c.runs {
		r.st, r.lineSt = r.lineSt, r.st
		if r.atomic == nil {
			r.text, r.lineText = r.lineText, r.text
		}
	}
	for _, sp := range c.spans {
		sp.st, sp.lineSt = sp.lineSt, sp.st
		sp.margin, sp.lineMargin = sp.lineMargin, sp.margin
		sp.border, sp.lineBorder = sp.lineBorder, sp.border
		sp.padding, sp.linePadding = sp.linePadding, sp.padding
	}
}

// hasFirst reports whether the content's first line is its block's styled by a
// ::first-line.
func (c *inlineContent) hasFirst() bool { return c.firstPieces != nil && !c.unindented }

// rootStyle is the style of the line's own box: the owner's, or its first line's.
func (c *inlineContent) rootStyle() *style.Computed {
	if c.firstOn {
		return c.line
	}
	return c.owner.st
}

// unspace takes word-spacing off gap p when its first space starts the content's text.
func (c *inlineContent) unspace(p *piece, st *style.Computed) {
	if p.kind == gap && p.text[0] == ' ' && st.WordGap() != 0 && c.first() {
		p.width -= st.WordGap()
		p.unspaced = true
	}
}

// first reports whether the next piece starts the text of content whose block collapses
// its spaces: Chrome adds no word-spacing to a space an element inside keeps there, where
// a block keeping its own spaces widens it as any other.
func (c *inlineContent) first() bool {
	if !collapses(c.owner.st.WhiteSpace) {
		return false
	}
	for _, p := range c.pieces {
		switch p.kind {
		case word, gap, newline, atomic, marker, tab:
			return false
		case breakHere, edge, emergency, anchor, softHyphen:
		}
	}
	return true
}

// cutRun cuts run i's text into pieces, afterSpace saying whether a collapsible space
// came last before it, and reports the same after it. levels are its characters' bidi
// levels, nil for none: a word is cut where they change, its parts glued.
func (c *inlineContent) cutRun(fonts box.Fonts, i int, r *textRun, soft map[int]bool, afterSpace bool,
	levels []bidi.Level) bool {
	ws := r.st.WhiteSpace
	defer c.tieSpaces(fonts, r.st, len(c.pieces))
	var b strings.Builder
	// n counts the characters read, and from is where the text in b starts; level is the
	// level of the character read last.
	var n, from int32
	var level bidi.Level
	emit := func(kind pieceKind) {
		if b.Len() == 0 && kind == word {
			return
		}
		s := b.String()
		b.Reset()
		if kind == word && wraps(ws) && splitsWords(r.st) {
			c.graphemes(fonts, r.st, i, s, from, level)
			return
		}
		p := piece{kind: kind, run: i, text: s, at: from, level: level}
		if kind == word || kind == gap {
			p.width = advance(fonts, r.st, unshown(s)) + r.st.SpacingAfter(c.textBefore(len(c.pieces)), unshown(s))
		}
		c.unspace(&p, r.st)
		c.pieces = append(c.pieces, p)
	}
	for at, ch := range collapsedBlanks(r.text, ws) {
		if levels != nil && levels[n] != level {
			emit(word)
			level = levels[n]
		}
		if b.Len() == 0 {
			from = n
		}
		n++
		if c.softBreakAt(soft, r, at, ch, b.Len() > 0) {
			if b.Len() > 0 {
				emit(word)
				from = n - 1
				c.pieces = append(c.pieces, piece{kind: breakHere, run: i, level: level})
			} else {
				c.breakBeforeEdges(i, piece{kind: breakHere, run: i, level: level})
			}
		}
		switch {
		case ch == '\n' || ch == '\t' || ch == lineSeparator || ch == paragraphSeparator:
			emit(word)
			afterSpace = c.control(fonts, r.st, i, n-1, ch, level)
		case ch == ' ':
			emit(word)
			if collapses(ws) && afterSpace {
				continue
			}
			from = n - 1
			b.WriteRune(' ')
			emit(gap)
			afterSpace = true
		case string(ch) == zeroWidthSpace || ch == softHyphenChar && r.st.Hyphens != style.HyphensNone:
			emit(word)
			// a space after it follows no space: the spaces round it never collapse
			c.pieces, afterSpace = append(c.pieces, breakAt(fonts, r.st, ch, i, n-1, level)), false
		default:
			b.WriteRune(ch)
			afterSpace = false
		}
	}
	emit(word)
	return afterSpace
}

// softBreakAt reports whether a line breaks before character ch at byte at of run r's
// text where it is no space: inside a word, when inWord, or after the word before the run.
func (c *inlineContent) softBreakAt(soft map[int]bool, r *textRun, at int, ch rune, inWord bool) bool {
	if ch == ' ' || ch == '\n' || string(ch) == zeroWidthSpace ||
		!soft[at] && (at != 0 || !c.breaksAllBefore(collapsedBlanks(r.text, r.st.WhiteSpace))) {
		return false
	}
	return inWord || at == 0 && c.wordBefore() >= 0
}

// breakBeforeEdges adds break p, where run i's text starts, before the start edges of the
// elements run i opens: they go with its text to the line it begins, as Chrome's do.
func (c *inlineContent) breakBeforeEdges(i int, p piece) {
	k := len(c.pieces)
	for k > 0 && c.edgeAt(k-1, true) && c.spans[c.pieces[k-1].span].first == i {
		k--
	}
	c.pieces = slices.Insert(c.pieces, k, p)
	for _, s := range c.spans {
		if s.first == i && s.open >= k {
			s.open++
		}
	}
}

// breakAt is the piece a zero-width space or a soft hyphen ch of run i, of style st,
// at character at, makes.
func breakAt(fonts box.Fonts, st *style.Computed, ch rune, i int, at int32, level bidi.Level) piece {
	if ch == softHyphenChar {
		return piece{kind: softHyphen, run: i, at: at, level: level, hyphen: hyphenWidth(fonts, st)}
	}
	return piece{kind: breakHere, run: i, text: zeroWidthSpace, level: level}
}

// spaceTies is fonts that know whether a face shapes a space with the words beside it
// (fonts.Set.SpaceTies).
type spaceTies interface {
	SpaceTies(f style.Font) bool
}

// tieSpaces gives the pieces of a run from first on the kerning across its spaces where
// st's face kerns or joins a space with its words, as Chrome shapes the run whole: a
// word takes its kerning with the space after it, a space its kerning with the word
// after it.
func (c *inlineContent) tieSpaces(fonts box.Fonts, st *style.Computed, first int) {
	if _, whole := fonts.(box.WholeFonts); whole || st.Stands() {
		return
	}
	if t, ok := fonts.(spaceTies); !ok || !t.SpaceTies(st.Font) {
		return
	}
	space := fonts.Advance(st.Font, " ")
	ps := c.pieces
	for k := first; k < len(ps); k++ {
		if ps[k].kind != gap || ps[k].text != " " {
			continue
		}
		if k > first && ps[k-1].kind == word && ps[k-1].run == ps[k].run {
			w := ps[k-1].text
			ps[k-1].width += fonts.Advance(st.Font, w+" ") - fonts.Advance(st.Font, w) - space
		}
		if k+1 < len(ps) && ps[k+1].kind == word && ps[k+1].run == ps[k].run {
			w := ps[k+1].text
			ps[k].width += fonts.Advance(st.Font, " "+w) - fonts.Advance(st.Font, w) - space
		}
	}
}

// splitsWords reports whether st's text may break inside a word: by overflow-wrap,
// word-break or line-break.
func splitsWords(st *style.Computed) bool {
	return st.WrapsAnywhere() || st.WordBreak == style.BreakAll || st.LineBreak == style.LineBreakAnywhere
}

// control adds the tab ch of run i at character at, or the forced break of a newline or a
// line or paragraph separator, and reports whether a collapsible space after it is
// dropped: after a break.
func (c *inlineContent) control(fonts box.Fonts, st *style.Computed, i int, at int32, ch rune,
	level bidi.Level) bool {
	if ch != '\t' {
		c.pieces = append(c.pieces, piece{kind: newline, run: i, level: level})
		return true
	}
	space := advance(fonts, st, " ")
	c.pieces = append(c.pieces, piece{kind: tab, run: i, at: at, level: level, stop: st.TabStop(space), least: space / 2})
	return false
}

// anchorsAt adds the anchors of the out-of-flow elements before run i, and notes the
// inline element each sits in.
func (c *inlineContent) anchorsAt(i int) {
	for k, a := range c.anchors {
		if a.run != i {
			continue
		}
		o := c.oofs[a.node]
		o.span = -1
		for si, sp := range c.spans {
			if sp.box == a.parent {
				o.span = si
			}
		}
		c.pieces = append(c.pieces, piece{kind: anchor, run: min(i, max(len(c.runs)-1, 0)), span: k})
	}
}

// graphemes adds word s of run i, at bidi level level, as one piece per grapheme, each
// measured as the growth of the word's advance so that they add up to it, a pair's kerning
// its first's where fonts places carets. Between two, a line breaks where word-break:
// break-all or line-break: anywhere let it, else in an emergency where overflow-wrap
// does; two graphemes with neither, or that a ligature draws, stay one piece. A line
// that breaks between two shapes their ends again, as Chrome does (cutStart, cutEnd).
func (c *inlineContent) graphemes(fonts box.Fonts, st *style.Computed, i int, s string, from int32,
	level bidi.Level) {
	var xs []float32
	var starts []bool
	cf, carets := fonts.(box.CaretFonts)
	if carets = carets && !st.Stands() && unshown(s) == s; carets {
		xs, starts = cf.Carets(st.Font, s, "", "", "", "")
	}
	first := len(c.pieces)
	spaced := spacedEnds(c.textBefore(first), s)
	rest, state := s, -1
	var g, last string
	prev := float32(0)
	n := 0
	for at := 0; len(rest) > 0; {
		g, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		at += len(g)
		start := n
		n += utf8.RuneCountInString(g)
		var w float32
		if carets {
			w = xs[n] - xs[start] + st.SpacingOf(g, spaced[at])
		} else {
			adv := advance(fonts, st, unshown(s[:at]))
			w = adv - prev + st.SpacingOf(unshown(g), spaced[at])
			prev = adv
		}
		switch {
		case at == len(g):
		case carets && !starts[start]:
			c.pieces[len(c.pieces)-1].text += g
			c.pieces[len(c.pieces)-1].width += w
			last = g
			from += int32(utf8.RuneCountInString(g)) //nolint:gosec // a text node's length
			continue
		case st.LineBreak == style.LineBreakAnywhere || st.WordBreak == style.BreakAll && breaksAll(last, g, st.Strictness):
			c.pieces = append(c.pieces, piece{kind: breakHere, run: i, level: level})
		case st.WrapsAnywhere():
			c.pieces = append(c.pieces, piece{kind: emergency, run: i, level: level})
		default:
			p := &c.pieces[len(c.pieces)-1]
			p.text += g
			p.width += w
			last = g
			from += int32(utf8.RuneCountInString(g)) //nolint:gosec // a text node's length
			continue
		}
		last = g
		c.pieces = append(c.pieces, piece{kind: word, run: i, text: g, width: w, at: from, level: level})
		from += int32(utf8.RuneCountInString(g)) //nolint:gosec // a text node's length
	}
	if wf, ok := fonts.(box.WholeFonts); ok && carets {
		c.cutGraphemes(wf, st, s, first)
	}
}

// textBefore is the text of the pieces before the k-th back to the last
// with a character of a script, whose script the digits and punctuation
// after it take: what letter-spacing follows reads it
// (style.Text.SpacingAfter).
func (c *inlineContent) textBefore(k int) string {
	for k--; k >= 0; k-- {
		if t := c.pieces[k].text; strings.ContainsFunc(t, func(r rune) bool { _, ok := shape.ScriptOf(r); return ok }) {
			return t
		}
	}
	return ""
}

// spacedEnds are the ends, in bytes, of the grapheme clusters of s after
// before that letter-spacing follows.
func spacedEnds(before, s string) map[int]bool {
	out := map[int]bool{}
	shape.EachSpaced(before, s, func(end int) { out[end] = true })
	return out
}

// cutGraphemes gives the pieces of word s from first on what their widths gain where a
// line starts at one, shaped without the text before it, or ends inside the word after
// one, shaped without the rest of the word: that text stays its context, which a joining
// script's letters join across, as Chrome reshapes them.
func (c *inlineContent) cutGraphemes(wf box.WholeFonts, st *style.Computed, s string, first int) {
	before := c.textBefore(first)
	at := 0
	for k := first; k < len(c.pieces); k++ {
		p := &c.pieces[k]
		if p.kind != word {
			continue
		}
		end := at + len(p.text)
		if at > 0 || end < len(s) {
			w := p.width - st.SpacingAfter(before+s[:at], p.text)
			if at > 0 {
				p.cutStart = wf.AdvanceAmid(st.Font, p.text, "", s[end:], s[:at], "") - w
			}
			if end < len(s) {
				p.cutEnd = wf.AdvanceAmid(st.Font, p.text, s[:at], "", "", s[end:]) - w
			}
		}
		at = end
	}
}

// breaksAll reports whether word-break: break-all breaks between the graphemes a and b,
// as Chrome breaks it: between two printable ASCII characters by its table, elsewhere by
// UAX #14 with letters and digits read as ideographs, before a hyphen after a letter or a
// digit, but never before a letter strictness s keeps off a line's start.
func breaksAll(a, b string, s style.Strictness) bool {
	if len(a) > 0 && len(b) > 0 && printableASCII(a[len(a)-1]) && printableASCII(b[0]) {
		return tableBreak(&asciiBreaksAll, a[len(a)-1], b[0])
	}
	ra, _ := utf8.DecodeLastRuneInString(a)
	rb, _ := utf8.DecodeRuneInString(b)
	switch {
	case !letterLike(ra) && !letterLike(rb), startsNoLine(rb, s):
		return false
	case letterLike(ra) && rb == '-':
		return true
	}
	first := asIdeographs(a, s)
	seg, _, _, _ := uniseg.FirstLineSegmentInString(first+asIdeographs(b, s), -1)
	return len(seg) == len(first)
}

// asIdeographs is s with its letters and digits turned into an ideograph, which UAX #14
// breaks before and after, but those strictness keeps off a line's start.
func asIdeographs(s string, strictness style.Strictness) string {
	return strings.Map(func(r rune) rune {
		if letterLike(r) && !startsNoLine(r, strictness) {
			return '\u4e00'
		}
		return r
	}, s)
}

func letterLike(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) }

// hyphenText is the hyphen st's text shows where a line breaks at a soft hyphen:
// hyphenate-character's, else U+2010 where its face draws it and a hyphen-minus
// where not, as Chrome's.
func hyphenText(fonts box.Fonts, st *style.Computed) string {
	if st.HyphenChosen {
		return st.HyphenChar
	}
	if f, ok := fonts.(box.FallbackFonts); ok {
		if _, other := f.FallbackMetrics(st.Font, "\u2010"); other {
			return "-"
		}
	}
	return "\u2010"
}

// hyphenWidth is the room hyphenText takes.
func hyphenWidth(fonts box.Fonts, st *style.Computed) float32 {
	h := hyphenText(fonts, st)
	return advance(fonts, st, h) + st.Spacing(h)
}

// markerOf is the disclosure triangle list-style gives a ::marker, as
// Chrome's text for it: "" for none.
func markerOf(st *style.Computed) string {
	switch st.ListStyle {
	case style.ListDisclosureClosed:
		return "▸"
	case style.ListDisclosureOpen:
		return "▾"
	case style.ListDisc, style.ListNone, style.ListCircle, style.ListSquare, style.ListCounter, style.ListString:
	}
	return ""
}

// markerWidth is a summary's ::marker, the triangle and a space in its font.
func markerWidth(fonts box.Fonts, st *style.Computed) float32 {
	return fonts.Advance(st.Font, markerOf(st)+" ")
}

// wordBefore is the last piece when it is a word, edges aside, -1 otherwise.
func (c *inlineContent) wordBefore() int {
	for k := len(c.pieces) - 1; k >= 0; k-- {
		switch c.pieces[k].kind {
		case edge:
			continue
		case word:
			return k
		case gap, breakHere, newline, atomic, emergency, marker, anchor, tab, softHyphen:
		}
		return -1
	}
	return -1
}

// breaksAllBefore reports whether a line may break before text, which starts a run after
// a word, by the word-break: break-all of that word: a break between two elements follows
// the text before it, as Chrome's.
func (c *inlineContent) breaksAllBefore(text string) bool {
	k := c.wordBefore()
	if k < 0 {
		return false
	}
	st := c.runs[c.pieces[k].run].st
	if st.WordBreak != style.BreakAll || !wraps(st.WhiteSpace) {
		return false
	}
	g, _, _, _ := uniseg.FirstGraphemeClusterInString(text, -1)
	return breaksAll(c.pieces[k].text, g, st.Strictness)
}

// openSpans adds the start edges of the spans whose runs begin at run i, outer spans first
// (c.spans lists inner spans first), and both edges of the empty ones there.
func (c *inlineContent) openSpans(i int) {
	run := min(i, len(c.runs)-1)
	for k := len(c.spans) - 1; k >= 0; k-- {
		s := c.spans[k]
		if s.first != i {
			continue
		}
		s.open = len(c.pieces)
		c.pieces = append(c.pieces, piece{kind: edge, run: run, span: k, start: true})
		if s.end == i {
			s.close = len(c.pieces)
			c.pieces = append(c.pieces, piece{kind: edge, run: run, span: k})
		}
	}
}

// closeSpans adds the end edges of the spans whose last run is i, inner spans first.
func (c *inlineContent) closeSpans(i int) {
	for k, s := range c.spans {
		if s.end == i+1 && s.first <= i {
			s.close = len(c.pieces)
			c.pieces = append(c.pieces, piece{kind: edge, run: i, span: k})
		}
	}
}

// softBreaks finds UAX #14's line break opportunities inside words, run by run: soft[i][at]
// is set when a line may break before byte at of run i's text. The runs are read as one text,
// atoms as U+FFFC, so the rules see across run boundaries; breaks at spaces and newlines are
// cut's own. As in Chrome, two printable ASCII characters break by its table instead, and
// no word breaks after a carriage return.
func (c *inlineContent) softBreaks() []map[int]bool {
	var all, classes strings.Builder
	starts := make([]int, len(c.runs))
	classed := false
	own, first := c.strictnesses()
	for i, r := range c.runs {
		starts[i] = all.Len()
		if r.atomic != nil {
			all.WriteString("\uFFFC")
			classes.WriteString("\uFFFC")
		} else {
			all.WriteString(r.text)
			classed = writeStrict(&classes, r.text, own[i], first[i]) || classed
		}
	}
	soft := make([]map[int]bool, len(c.runs))
	text, classedText := all.String(), all.String()
	if classed {
		classedText = classes.String()
	}
	runAt := func(pos int) int {
		k := 0
		for k+1 < len(c.runs) && starts[k+1] <= pos {
			k++
		}
		return k
	}
	mark := func(pos int) {
		k := runAt(pos)
		if soft[k] == nil {
			soft[k] = map[int]bool{}
		}
		soft[k][pos-starts[k]] = true
	}
	rest, pos, state := classedText, 0, -1
	run := 0
	for len(rest) > 0 {
		var seg string
		seg, rest, _, state = uniseg.FirstLineSegmentInString(rest, state)
		pos += len(seg)
		if len(rest) == 0 {
			break
		}
		for run+1 < len(c.runs) && starts[run+1] <= pos {
			run++
		}
		if asciiPair(text, pos) {
			continue
		}
		if c.runs[run].atomic != nil || c.runs[run].st.WordBreak == style.KeepAll && keeps(text[:pos], rest) {
			continue
		}
		if r, _ := utf8.DecodeLastRuneInString(text[:pos]); r == softHyphenChar && c.hyphensOff(starts, pos-1) ||
			r == '\r' {
			continue
		}
		mark(pos)
	}
	add := func(pos int) {
		if k := runAt(pos); c.runs[k].atomic == nil && c.runs[k].st.WordBreak != style.KeepAll {
			mark(pos)
		}
	}
	asciiBreaksIn(text, mark)
	quoteBreaks(text, add)
	c.anywhereEnds(add, starts)
	looseBreaks(text, classedText, func(pos int) style.Strictness {
		k := runAt(pos)
		if pos == starts[k] {
			return first[k]
		}
		return own[k]
	}, add, func(pos int) {
		k := runAt(pos)
		delete(soft[k], pos-starts[k])
	})
	return soft
}

// hyphensOff reports whether the text at byte at of the runs, which start at starts,
// has hyphens: none: its soft hyphens break nothing.
func (c *inlineContent) hyphensOff(starts []int, at int) bool {
	i := 0
	for i+1 < len(starts) && starts[i+1] <= at {
		i++
	}
	return c.runs[i].atomic == nil && c.runs[i].st.Hyphens == style.HyphensNone
}

// keeps reports whether word-break: keep-all keeps the break between before and after:
// one between two letters or digits, ideographs included.
func keeps(before, after string) bool {
	a, _ := utf8.DecodeLastRuneInString(before)
	b, _ := utf8.DecodeRuneInString(after)
	return letterLike(a) && letterLike(b)
}

// fills reports whether an atomic inline is a picture that takes the width it is given.
func (c *inlineContent) fills() bool {
	for i := range c.runs {
		if a := c.runs[i].atomic; a != nil && a.node.fills() {
			return true
		}
	}
	return false
}

// sizeAtomics sizes the spans' edges against the containing block's width, and lays out
// each atomic inline as a shrink-to-fit box in avail, setting its piece's width to its
// margin box.
func (t *tree) sizeAtomics(c *inlineContent, avail space, parent size) {
	for _, sp := range c.spans {
		sp.margin, sp.border, sp.padding = sp.edgesIn(sp.st, parent.w)
		if c.firstPieces != nil {
			sp.lineMargin, sp.lineBorder, sp.linePadding = sp.edgesIn(sp.lineSt, parent.w)
		}
	}
	for i := range c.pieces {
		if p := &c.pieces[i]; p.kind == edge {
			sp := c.spans[p.span]
			start, end := startSides(sp.st, sp.margin, sp.border, sp.padding)
			lineStart, lineEnd := start, end
			if c.firstPieces != nil {
				lineStart, lineEnd = startSides(sp.lineSt, sp.lineMargin, sp.lineBorder, sp.linePadding)
			}
			if p.start {
				c.resizeEdge(i, start, lineStart)
			} else {
				c.resizeEdge(i, end, lineEnd)
			}
		}
	}
	for i := range c.pieces {
		p := &c.pieces[i]
		if p.kind != atomic {
			continue
		}
		a := c.runs[p.run].atomic
		st := a.node.st
		if !some(parent.h) {
			parent.h = c.height
		}
		a.margin = edges(st.Margin, parent.w)
		inner := avail.mSub(a.margin.horizontal()).mMax(0)
		known := noSize
		switch k, sp := resolveSizingKeyword(st.Size.Width, inner.value(), parent.w); k {
		case keywordMeasure:
			inner = sp
		case keywordExact:
			known.w = sp.v
		case keywordNone:
		}
		a.width = t.measureChildSize(a.node, known, parent,
			spaces{inner, maxContentSpace}, inherentSize, horizontal, [2]bool{})
		out := t.performChildLayout(a.node, size{a.width, none}, parent,
			spaces{def(a.width), maxContentSpace}, inherentSize)
		a.height = out.size.h
		a.baseline = out.size.h + a.margin.b
		if c.owner.st.Stands() {
			// a central baseline of the line, made up across the middle of its margin box
			mh := a.height + a.margin.vertical()
			a.baseline = mh - float32(math.Floor(float64(mh*c.unit)/2))/c.unit - a.margin.t
		}
		a.overflow = out.overflow
		a.node.pendingOOF = out.oof
		switch {
		case isScrollContainer(st) && a.node.control != inputControl && a.node.control != selectControl,
			a.node.slider:
		case some(out.lastBaseline) && st.Display != style.InlineFlex && st.Display != style.InlineGrid:
			a.baseline = lineBaseline(st, out.lastBaseline, a.height)
		case out.synthetic > 0 && c.owner.st.Stands():
			// a central line synthesizes it across the middle of the item, as Chrome
			h := out.synthetic
			a.baseline = out.firstBaseline - float32(math.Floor(float64(h*c.unit)/2))/c.unit
		case some(out.firstBaseline):
			a.baseline = lineBaseline(st, out.firstBaseline, a.height)
		}
		c.resize(i, a.width+a.margin.horizontal())
	}
}

// lineBaseline is where a line's baseline b, measured from the frame's top of a box h tall
// written in st, stands from the box's over side, as the line holding it aligns it: a
// vertical-lr box's lines measure theirs from the frame's top, their over side its bottom.
func lineBaseline(st *style.Computed, b, h float32) float32 {
	if st.Mode == style.VerticalLR {
		return h - b
	}
	return b
}

// measured are the content's lines where nothing wraps, every one in the first line's
// styles while those stand (useFirst).
func (c *inlineContent) measured() []line {
	if !c.firstOn {
		return c.lines(unbounded)
	}
	c.allFirst = true
	defer func() { c.allFirst = false }()
	b := breaker{c: c}
	return b.breakAll(unbounded)
}

// compress takes the pictures whose width or max-width is a percentage down to their
// margins: such a picture asks its container for no width at its narrowest, as in CSS
// Sizing 3 (compressible replaced elements).
func (c *inlineContent) compress() {
	for i := range c.pieces {
		p := &c.pieces[i]
		if p.kind != atomic {
			continue
		}
		a := c.runs[p.run].atomic
		if st := a.node.st; a.node.replaced && (st.Size.Width.IsPercent || st.MaxSize.Width.IsPercent) {
			c.resize(i, a.margin.horizontal())
		}
	}
}

// contentWidths is the min-content width (the widest unbreakable segment, or the widest
// line where nothing wraps) and the max-content width (the widest line between forced
// breaks).
func (t *tree) contentWidths(c *inlineContent, parent size) (float32, float32) {
	var minW, maxW float32
	// A percentage of the width being found is no width yet.
	parent.w = none
	// Chrome measures every line of a block whose first line is styled in the first
	// line's styles.
	was := c.useFirst(true)
	defer c.useFirst(was)
	t.sizeAtomics(c, maxContentSpace, parent)
	for _, l := range c.measured() {
		maxW = max(maxW, l.x+l.width)
	}
	t.sizeAtomics(c, minContentSpace, parent)
	c.compress()
	if !c.anyWraps() {
		for _, l := range c.measured() {
			minW = max(minW, l.x+l.width)
		}
		return minW, maxW
	}
	return c.minContent(), maxW
}

// minContent is the widest unbreakable segment of the content, which wraps.
func (c *inlineContent) minContent() float32 {
	var minW float32
	cur := newPen(c.unit, c.indent(unbounded))
	cur.origin = 0
	hung, bare := false, true
	for i, p := range c.pieces {
		if p.kind != edge && p.kind != atomic {
			bare = false
		}
		switch p.kind {
		case word, marker:
			cur.step(p, 0)
			minW = max(minW, cur.end())
		case edge:
			cur.step(p, 0)
			minW = max(minW, cur.end())
			if hung && !c.edgeAt(i+1, false) {
				cur, hung = newPen(c.unit, 0), false
			}
		case emergency:
			if c.runs[p.run].st.MinAnywhere() {
				cur = newPen(c.unit, 0)
			}
		case atomic:
			if bare {
				minW = max(minW, cur.end()+p.width)
				bare = false
			} else {
				minW = max(minW, p.width)
			}
			cur = newPen(c.unit, 0)
		case gap:
			switch {
			case c.held(i):
				cur.step(p, 0)
				minW = max(minW, cur.end())
				cur = newPen(c.unit, 0)
			case !collapses(c.runs[p.run].st.WhiteSpace):
				cur = newPen(c.unit, 0)
			case c.edgeAt(i+1, false):
				// the span's end stays with the word before the space
				hung = true
			default:
				cur = newPen(c.unit, 0)
			}
		case softHyphen:
			if c.wrapsAt(i) {
				minW = max(minW, cur.end()+p.hyphen)
				cur = newPen(c.unit, 0)
			}
		case tab:
			if c.held(i) {
				cur.step(p, 0)
				minW = max(minW, cur.end())
			}
			cur = newPen(c.unit, 0)
		case breakHere, newline, anchor:
			cur = newPen(c.unit, 0)
		}
	}
	return minW
}

// edgeAt reports whether piece i is a span's edge, its start or its end.
func (c *inlineContent) edgeAt(i int, start bool) bool {
	return i >= 0 && i < len(c.pieces) && c.pieces[i].kind == edge && c.pieces[i].start == start
}

func (c *inlineContent) anyWraps() bool {
	for _, r := range c.runs {
		if r.atomic == nil && wraps(r.st.WhiteSpace) {
			return true
		}
	}
	return false
}

// line is the pieces [from, to) on one line, without its trailing spaces; forced is set when
// a newline ends it. x is where its room starts in the content box, past a float or the
// first line's indent, avail how wide that room is, and push how far below the line
// before it starts, to clear a float.
type line struct {
	from, to int
	width    float32
	// first is set on the block's first line where a ::first-line styles it.
	first  bool
	forced bool
	// phantom is a line holding only inline elements of no size, which has no height.
	phantom  bool
	x, avail float32
	push     float32
	// indent is the part of x text-indent is, which tab stops are not measured from.
	indent float32
	// hyphen is the soft hyphen the line breaks after, showing its hyphen, -1 for none.
	hyphen int
	// start is how the line starts, which sets the trim of an opening bracket starting
	// it, and closed is set when its last closing bracket is trimmed to fit
	// (text-spacing-trim).
	start  lineStart
	closed bool
}

// pen is a pen at x walking l, its tab stops measured from l's start.
func (l line) pen(unit, x float32) pen {
	p := newPen(unit, x)
	p.origin = x - l.indent
	return p
}

// startInset is how far right of its room's left a line indented by indent starts: the
// indent itself, none under rtl, where the indent is on the right.
func (c *inlineContent) startInset(indent float32) float32 {
	if c.owner.st.Dir == style.RTL {
		return 0
	}
	return indent
}

// indent is text-indent for lines of that width: how far the first line starts in, a
// percentage being of the width, none where none is known. Controls take none.
func (c *inlineContent) indent(width float32) float32 {
	l := c.owner.st.TextIndent
	if c.unindented || c.owner.control != noControl || l.Kind != style.Fixed {
		return 0
	}
	if width >= unbounded {
		return l.Px
	}
	return l.At(width)
}

func (c *inlineContent) wrapsAt(i int) bool {
	if c.pieces[i].run < 0 {
		// the edges of the elements of lines holding no text
		return wraps(c.owner.st.WhiteSpace)
	}
	r := c.runs[c.pieces[i].run]
	if r.atomic != nil {
		return wraps(c.owner.st.WhiteSpace)
	}
	return wraps(r.st.WhiteSpace)
}

// unitEnd is where the unit starting at piece i ends: a word, edge, marker or anchor
// takes the pieces glued after it, and the space or tab break-spaces holds after them.
func (c *inlineContent) unitEnd(i int) int {
	end := i + 1
	if k := c.pieces[i].kind; k == word || k == edge || k == marker || k == anchor {
		hyphen := false
		for end < len(c.pieces) && (glued(c.pieces[end].kind) || c.held(end) && !c.anywhere(end)) {
			// after a soft hyphen, only the ends of spans and anchors stay
			if p := c.pieces[end]; hyphen && (p.kind != edge || p.start) && p.kind != anchor {
				break
			}
			hyphen = hyphen || c.pieces[end].kind == softHyphen
			end++
			if c.held(end - 1) {
				break
			}
		}
	}
	return end
}

// held reports whether piece i is a space or a tab of text under white-space:
// break-spaces: it takes its room at a line's end too, and a line may break after it.
func (c *inlineContent) held(i int) bool {
	p := c.pieces[i]
	return (p.kind == gap || p.kind == tab) && c.runs[p.run].st.WhiteSpace == style.BreakSpaces
}

// anywhere reports whether piece i's text breaks between any two characters: line-break:
// anywhere, which breaks before a space too.
func (c *inlineContent) anywhere(i int) bool {
	return c.runs[c.pieces[i].run].st.LineBreak == style.LineBreakAnywhere
}

// hyphenEnd is the soft hyphen a line ending before piece end ends at, -1 for none:
// the last piece before end but span ends and anchors.
func (c *inlineContent) hyphenEnd(end int) int {
	for k := end - 1; k >= 0; k-- {
		switch c.pieces[k].kind {
		case edge:
			if c.pieces[k].start {
				return -1
			}
			continue
		case anchor:
			continue
		case softHyphen:
			if c.wrapsAt(k) {
				return k
			}
		case word, gap, breakHere, newline, atomic, emergency, marker, tab:
		}
		return -1
	}
	return -1
}

// hyphenAt is the room the hyphen of a line ending before piece end takes.
func (c *inlineContent) hyphenAt(end int) float32 {
	if k := c.hyphenEnd(end); k >= 0 {
		return c.pieces[k].hyphen
	}
	return 0
}

// opensOnSpace reports whether piece i is a span's start edge with a collapsible space
// next, or after the start edges that follow it: stuck to what is before it, with the
// space a line may end at after it.
func (c *inlineContent) opensOnSpace(i int) bool {
	for c.edgeAt(i, true) {
		i++
	}
	return i < len(c.pieces) && c.pieces[i].kind == gap && c.passes(c.pieces[i])
}

// passes reports whether p may stand between a line's edge and the collapsible spaces CSS
// drops there: a span's edge, an anchor, a break, or such a space.
func (c *inlineContent) passes(p piece) bool {
	switch p.kind {
	case edge, anchor, breakHere:
		return true
	case gap:
		return collapses(c.runs[p.run].st.WhiteSpace)
	case word, atomic, newline, emergency, marker, tab, softHyphen:
	}
	return false
}

// fitEnd is where the unit [i, end), first on its line and too wide for it, ends for the
// start edges that close it and open on a space to begin the next line, when that makes
// it fit in width.
func (c *inlineContent) fitEnd(i, end int, width float32) int {
	cut := end
	for cut > i+1 && c.edgeAt(cut-1, true) {
		cut--
	}
	if cut == end || !c.opensOnSpace(cut) {
		return end
	}
	at := newPen(c.unit, 0)
	for k := i; k < cut; k++ {
		at.step(c.pieces[k], 0)
	}
	if at.end() > width {
		return end
	}
	return cut
}

// hangEnd is where the unit [i, end) after the spaces pending since pending ends when its
// first edges stay with the content before those spaces, on its line whatever room it has:
// past its last end edge, or all of it when it is edges and anchors alone. Were the line to
// end there, the spaces would be dropped. hung is false for a unit that begins otherwise.
func (c *inlineContent) hangEnd(pending, i, end int) (int, bool) {
	for k := pending; k < i; k++ {
		if !c.passes(c.pieces[k]) {
			return end, false
		}
	}
	last, k := -1, i
	for ; k < end && (c.pieces[k].kind == edge || c.pieces[k].kind == anchor); k++ {
		if c.edgeAt(k, false) {
			last = k
		}
	}
	switch {
	case k == end:
		return end, true
	case last >= 0:
		return last + 1, true
	}
	return end, false
}

// shown is line l's pieces as they are placed: the collapsible spaces before its first
// content and after its last, whatever span edges and the break ending it stand between, are breaks of no width,
// and the spaces break-spaces keeps after its last content are text, which neither
// justify widens nor the bidi algorithm takes to the paragraph's level, as Chrome's.
func (c *inlineContent) shown(l line) []piece {
	pieces := c.pieces[l.from:l.to]
	out := pieces
	set := func(i int, p piece) {
		if &out[0] == &pieces[0] {
			out = slices.Clone(pieces)
		}
		out[i] = p
	}
	drop := func(i int) {
		if pieces[i].kind == gap {
			set(i, piece{kind: breakHere, run: pieces[i].run})
		}
	}
	for i := 0; i < len(pieces) && c.passes(pieces[i]); i++ {
		drop(i)
	}
	i := len(pieces) - 1
	for ; i >= 0 && (c.passes(pieces[i]) || pieces[i].kind == newline); i-- {
		drop(i)
	}
	for ; i >= 0 && (pieces[i].kind == edge || pieces[i].kind == anchor || c.held(l.from+i)); i-- {
		if pieces[i].kind == gap {
			p := pieces[i]
			p.kind = word
			set(i, p)
		}
	}
	if l.hyphen >= l.from && l.hyphen < l.to {
		if &out[0] == &pieces[0] {
			out = slices.Clone(pieces)
		}
		p := &out[l.hyphen-l.from]
		p.width, p.shows = p.hyphen, true
	}
	place := func(k int, start lineStart, end bool) {
		if k < 0 || c.pieces[k].han == nil {
			return
		}
		h, d := c.pieces[k].han.placed(len(c.pieces[k].text), start, end, end && l.closed)
		if &out[0] == &pieces[0] {
			out = slices.Clone(pieces)
		}
		p := &out[k-l.from]
		p.han, p.width = h, p.width+d
	}
	if first, last := c.firstWord(l.from, l.to), c.lastWord(l.from, l.to); first == last {
		place(first, l.start, l.closed || c.runGoesOn(first, l.to))
	} else {
		place(first, l.start, false)
		place(last, midLine, l.closed || c.runGoesOn(last, l.to))
	}
	if first, start, last, end := c.cuts(l); start != 0 || end != 0 {
		if &out[0] == &pieces[0] {
			out = slices.Clone(pieces)
		}
		if start != 0 {
			out[first-l.from].width += start
		}
		if end != 0 {
			out[last-l.from].width += end
		}
	}
	return out
}

// cuts is line l's first word and what it gains, then its last and what it gains, shaped
// again at the line's ends as Chrome reshapes them: the first word without the text
// before it, where the line does not start the content, and the last without the rest of
// its word, where the line breaks inside one.
func (c *inlineContent) cuts(l line) (int, float32, int, float32) {
	var first, last int
	var start, end float32
	if l.from > 0 {
		for k := l.from; k < l.to; k++ {
			if p := c.pieces[k]; p.kind == word {
				first, start = k, p.cutStart
				break
			} else if !c.passes(p) {
				break
			}
		}
	}
	if k := l.to - 1; k >= l.from && l.to < len(c.pieces) && c.pieces[k].kind == word {
		if b := c.pieces[l.to].kind; b == emergency || b == breakHere {
			last, end = k, c.pieces[k].cutEnd
		}
	}
	return first, start, last, end
}

// reshaped is ls with each line as wide as its ends shaped again make it (cuts).
func (c *inlineContent) reshaped(ls []line) []line {
	for k := range ls {
		_, start, _, end := c.cuts(ls[k])
		ls[k].width += start + end
	}
	return ls
}

// joined is the pen from at past the pieces from end no line may end among: the units
// joined by the spaces of text that does not wrap.
func (c *inlineContent) joined(at pen, end int) pen {
	for end < len(c.pieces) && c.pieces[end].kind == gap && !c.wrapsAt(end) {
		from := end
		for end < len(c.pieces) && c.pieces[end].kind == gap {
			end++
		}
		if end == len(c.pieces) || !c.joins(end) {
			break
		}
		for end = c.unitEnd(end); from < end; from++ {
			at.step(c.pieces[from], 0)
		}
	}
	return at
}

// joins reports whether piece i is of a unit a line holds: not a break.
func (c *inlineContent) joins(i int) bool {
	switch c.pieces[i].kind {
	case word, atomic, edge, marker, anchor, softHyphen:
		return true
	case gap, breakHere, newline, emergency, tab:
	}
	return false
}

// measureUnit is the pen past pieces i to end from at, with the gaps pending since then,
// and the pen past them alone, the word lead names as wide as the line's start sets it.
func (c *inlineContent) measureUnit(at pen, pending, i, end int, lead leadWord) (pen, pen) {
	next := at
	if pending >= 0 {
		for k := pending; k < i; k++ {
			if kind := c.pieces[k].kind; kind == gap || kind == tab {
				next.step(c.pieces[k], 0)
			}
		}
	}
	alone := newPen(c.unit, 0)
	for k := i; k < end; k++ {
		p := lead.of(c.pieces, k)
		next.step(p, 0)
		alone.step(p, 0)
	}
	return next, alone
}

// breaker is lines' state while it breaks: the lines so far, the open one and the pen
// across it, the first gap pending before the next unit, and whether the line is empty.
// kept is the pen were the line to end here, without the spaces dropped at its end, and
// blank is set while the line holds only span edges and anchors.
type breaker struct {
	c        *inlineContent
	out      []line
	cur      line
	at, kept pen
	pending  int
	// spaced is the space the edges hung on the line's last unit came after, -1 past
	// any other unit: the line may still end at it, whatever follows the edges.
	spaced int
	empty  bool
	blank  bool
	// stop is a piece the line ends before where it may, whatever room it
	// has left, 0 for none; seen, when set, is told of every other piece
	// it may end before.
	stop int
	seen func(i int)
}

func (b *breaker) finish(next int) {
	b.cur.width = b.kept.end()
	if !b.cur.forced {
		if b.cur.hyphen = b.c.hyphenEnd(b.cur.to); b.cur.hyphen >= 0 {
			b.cur.width += b.c.pieces[b.cur.hyphen].hyphen
		}
	}
	b.cur.width, b.cur.closed = b.c.lineEnd(b.cur, !b.cur.forced)
	b.out = append(b.out, b.cur)
	b.open(next, 0, b.cur.avail, 0)
}

// open starts the breaker's next line at piece i, in room avail wide starting at x.
func (b *breaker) open(i int, x, avail, push float32) {
	b.cur = line{from: i, to: i, x: x, avail: avail, push: push, first: len(b.out) == 0 && b.c.hasFirst(), hyphen: -1,
		start: openingStart}
	if n := len(b.out); n > 0 && !b.out[n-1].forced {
		b.cur.start = wrappedStart
	}
	b.at = newPen(b.c.unit, 0)
	b.kept = b.at
	b.pending, b.spaced = -1, -1
	b.empty, b.blank = true, true
}

// indent has the open line start in by indent, text-indent.
func (b *breaker) indent(indent float32) {
	b.cur.indent = indent
	b.at.origin = -indent
	b.kept = b.at
}

// one breaks the open line from piece i, no wider than width where it may wrap, and
// returns where the next begins; ended is false when the pieces ran out first, the line
// left open.
func (b *breaker) one(i int, width float32) (int, bool) {
	c := b.c
	for i < len(c.pieces) {
		p := c.pieces[i]
		if p.kind == newline {
			b.keepPreserved(i)
			end := b.closingAfter(i)
			b.cur.forced = true
			b.finish(end)
			return end, true
		}
		if b.between(i) {
			i++
			continue
		}
		end, hung := c.unitEnd(i), false
		if b.pending >= 0 && (p.kind == edge || p.kind == anchor) {
			end, hung = c.hangEnd(b.pending, i, end)
		}
		lead := b.lead(i, end)
		next, alone := c.measureUnit(b.at, b.pending, i, end, lead)
		if b.empty && alone.end() > width && c.wrapsAt(i) {
			if cut := c.fitEnd(i, end, width); cut < end {
				end = cut
				next, alone = c.measureUnit(b.at, b.pending, i, end, lead)
			}
		}
		kept := next
		if hung {
			kept = b.kept
			for k := i; k < end; k++ {
				kept.step(lead.of(c.pieces, k), 0)
			}
		}
		if !b.blank && !hung && b.wrapsBefore(i) && !b.ligatureOpens() && b.endsBefore(i, end, next, width) {
			b.finish(i)
			return i, true
		}
		if b.empty && alone.end() > width {
			if after, ok := b.breakInside(i, end, width); ok {
				return after, true
			}
		}
		b.at, b.kept = next, kept
		b.spaced = -1
		if hung {
			b.spaced = b.pending
		}
		b.pending = -1
		b.cur.to = end
		b.empty = false
		for k := i; b.blank && k < end; k++ {
			b.blank = c.pieces[k].kind == edge || c.pieces[k].kind == anchor
		}
		i = end
	}
	return i, false
}

// closingAfter takes the newline at i and the span ends right after it onto the open
// line, as Chrome's line ends with the elements the break closes, and returns where the
// next line begins.
func (b *breaker) closingAfter(i int) int {
	c := b.c
	end := i + 1
	for end < len(c.pieces) && c.edgeAt(end, false) {
		end++
	}
	if end == i+1 {
		return end
	}
	for k := i; k < end; k++ {
		b.at.step(c.pieces[k], 0)
		b.kept.step(c.pieces[k], 0)
	}
	b.cur.to = end
	return end
}

// ligatureOpens reports whether the space pending is inside a ligature the line's first
// word starts, where Chrome breaks no line.
func (b *breaker) ligatureOpens() bool {
	c := b.c
	if b.pending < 0 || !c.pieces[b.pending].ligated {
		return false
	}
	words := 0
	for k := b.cur.from; k < b.pending; k++ {
		if c.pieces[k].kind == word {
			words++
		}
	}
	return words == 1
}

// endsBefore reports whether the line, which may end before piece i, does: at its stop,
// or where the unit [i, end), next the pen past it, leaves it wider than width; else
// seen hears of the place.
func (b *breaker) endsBefore(i, end int, next pen, width float32) bool {
	c := b.c
	if i == b.stop || next.end()+c.hyphenAt(end)+c.endFit(b.cur.from, end, next.end(), width) > width {
		return true
	}
	if joined := c.joined(next, end); joined != next && joined.end() > width {
		return true
	}
	if b.seen != nil {
		b.seen(i)
	}
	return false
}

// wrapsBefore reports whether the line may end before piece i: by the white-space of the
// space pending before it, whose text the break is in, and by the piece's own after none.
func (b *breaker) wrapsBefore(i int) bool {
	if b.pending >= 0 {
		return b.c.wrapsAt(b.pending)
	}
	if b.spaced >= 0 && b.c.wrapsAt(b.spaced) {
		return true
	}
	return b.c.wrapsAt(i)
}

// between takes piece i when it stands between the units of a line: a space, dropped
// where it collapses at the line's start and pending otherwise, or a place the line may
// break at.
func (b *breaker) between(i int) bool {
	p := b.c.pieces[i]
	if b.c.held(i) {
		return false
	}
	switch p.kind {
	case gap:
		if (b.empty || b.blank) && collapses(b.c.runs[p.run].st.WhiteSpace) {
			if b.empty {
				b.cur.from, b.cur.to = i+1, i+1
			}
			return true
		}
		if b.pending < 0 {
			b.pending = i
		}
		return true
	case tab:
		if b.pending < 0 {
			b.pending = i
		}
		return true
	case breakHere:
		return true
	case newline, word, atomic, edge, emergency, marker, anchor, softHyphen:
	}
	return false
}

// breakInside ends the line inside the unit from i to end, too wide for width alone,
// where it may break in an emergency, and returns where the next line begins.
func (b *breaker) breakInside(i, end int, width float32) (int, bool) {
	take, ok := b.c.emergencyBreak(i, end, width)
	if !ok {
		return 0, false
	}
	for k := i; k < take; k++ {
		b.at.step(b.c.pieces[k], 0)
	}
	b.kept = b.at
	b.cur.to = take
	b.empty = false
	b.finish(take + 1)
	return take + 1, true
}

// close ends the breaking: the line left open joins the lines when it holds something,
// more than the ends of the spans a forced break ended the line before in.
func (b *breaker) close() []line {
	b.keepPreserved(len(b.c.pieces))
	trails := b.trails()
	if !b.empty && !trails || len(b.out) == 0 && b.cur.to > b.cur.from {
		b.cur.width = b.kept.end()
		b.cur.width, b.cur.closed = b.c.lineEnd(b.cur, false)
		b.out = append(b.out, b.cur)
	} else if trails && b.opensSpan() {
		// An inline element whole after the break stands on a line of its own, as
		// Chrome's, a line of no height.
		b.cur.phantom = true
		b.out = append(b.out, b.cur)
	}
	if b.out == nil {
		b.out = []line{}
	}
	return b.out
}

// keepPreserved puts the spaces pending before piece end, a newline or the content's
// end, on the open line where they neither collapse nor wrap (white-space: pre), as
// Chrome's line holds them, a line of its own after a forced break.
func (b *breaker) keepPreserved(end int) {
	c := b.c
	if b.pending < 0 {
		return
	}
	for _, p := range c.pieces[b.pending:end] {
		if p.kind != gap && p.kind != tab {
			return
		}
		if ws := c.runs[p.run].st.WhiteSpace; collapses(ws) || wraps(ws) {
			return
		}
	}
	for _, p := range c.pieces[b.pending:end] {
		b.at.step(p, 0)
	}
	b.kept = b.at
	b.cur.to, b.pending = end, -1
	b.empty, b.blank = false, false
}

// trails reports an open line after a forced break that holds only span edges of no
// width: no line of its own, as after a br that ends its block.
func (b *breaker) trails() bool {
	if !b.blank || len(b.out) == 0 || !b.out[len(b.out)-1].forced {
		return false
	}
	for _, p := range b.c.pieces[b.cur.from:b.cur.to] {
		if p.kind != edge || p.width != 0 {
			return false
		}
	}
	return true
}

// opensSpan reports an open line holding the start of a span.
func (b *breaker) opensSpan() bool {
	for _, p := range b.c.pieces[b.cur.from:b.cur.to] {
		if p.kind == edge && p.start {
			return true
		}
	}
	return false
}

func (b *breaker) breakAll(width float32) []line {
	c := b.c
	indent := c.indent(width)
	b.open(0, c.startInset(indent), width-indent, 0)
	b.indent(indent)
	was := c.useFirst(b.cur.first)
	defer c.useFirst(was)
	for i := 0; i < len(c.pieces); {
		var ended bool
		if i, ended = b.one(i, b.cur.avail); ended {
			b.open(i, 0, width, 0)
			if !c.allFirst {
				c.useFirst(false)
			}
		}
	}
	return b.close()
}

// flowLines breaks the pieces into lines that go round the floats of c.flow: each line
// takes the room the floats leave at its height, and goes below those that leave it less
// than its first word.
func (t *tree) flowLines(c *inlineContent, width float32) []line {
	b := breaker{c: c}
	strut, _ := t.lineHeight(c, c.owner.st)
	indent := c.indent(width)
	y, bottom := c.flowTop, c.flowTop
	floats := c.restartFloats()
	for i := 0; i < len(c.pieces); {
		kept := len(b.out)
		if kept > 0 {
			indent = 0
		}
		h := strut
		for tries := 0; ; tries++ {
			x, room, next := c.flow.lineSlot(y, h)
			room -= indent
			b.out = b.out[:kept]
			b.open(i, x-c.flowLeft+c.startInset(indent), room, y-bottom)
			b.indent(indent)
			was := c.useFirst(b.cur.first)
			after, ended := b.one(i, room)
			c.useFirst(was)
			l := b.cur
			l.width = b.kept.end()
			if ended {
				l = b.out[kept]
			}
			last := tries >= 16
			if !last && some(next) && l.width > room+fitTolerance {
				y, h = next, strut
				continue
			}
			lh, _ := t.lineBox(c, l)
			if _, taller, _ := c.flow.lineSlot(y, lh); !last && lh > h && taller-indent < room-fitTolerance {
				h = lh
				continue
			}
			on := c.floatsOn(floats, l, after, ended)
			if len(on) > 0 && c.besideLine(on[0], l, y, room, kept == 0) {
				floats = floats[1:]
				continue
			}
			i = after
			y += lh
			if kept == 0 {
				// The lines after a trimmed first line rise by its trim, the floats beside
				// it staying where it was laid out, as Chrome's.
				top, _ := t.trims(c, []line{l})
				y -= top
			}
			for _, f := range on {
				f.at, f.placed = c.flow.place(f.box, y, f.side, f.clearing, false), true
			}
			floats = floats[len(on):]
			bottom = y
			if side := c.brClear(l); side != style.ClearNone {
				if below := c.flow.clearedThreshold(side); some(below) && below > y {
					y = below
				}
			}
			break
		}
	}
	c.clearedAfter = y - bottom
	return c.reshaped(b.close())
}

// restartFloats has the floats of c's context be those before its lines, and returns the
// floats the lines place, none when they place none.
func (c *inlineContent) restartFloats() []*lineFloat {
	from := c.floatsFrom
	if from == nil || !c.pendingFloats() {
		return nil
	}
	*c.flow.bfc.floats = *from.floats.clone()
	c.flow.floatHeight = from.floatHeight
	for _, f := range c.lineFloats {
		f.placed, f.first = false, false
	}
	return c.lineFloats
}

// floatsOn are the first of floats that stand on line l, which ends before piece after,
// ended set when a break ended it: those before after, or at it where the line takes what
// follows it, as only a forced break does not.
func (c *inlineContent) floatsOn(floats []*lineFloat, l line, after int, ended bool) []*lineFloat {
	n := 0
	for _, f := range floats {
		k := c.pieceOf(f.run)
		if k > after || k == after && ended && l.forced {
			break
		}
		n++
	}
	return floats[:n]
}

// besideLine places f on line l, at y in room wide, when it fits beside what stands
// before it there, or nothing does, and reports whether it did: the line is then broken
// again round it. A float that doesn't goes below the line, and the floats after it too.
func (c *inlineContent) besideLine(f *lineFloat, l line, y, room float32, first bool) bool {
	used := c.widthBefore(l, c.pieceOf(f.run))
	if used > 0 && used+f.box.w > room+fitTolerance {
		return false
	}
	f.at, f.placed, f.first = c.flow.place(f.box, y, f.side, f.clearing, false), true, first
	return true
}

// brClear is the clear of the br line l ends with, none where it ends otherwise, the end
// of an element the br is in included: the line after it starts below the floats it
// names, as Chrome's.
func (c *inlineContent) brClear(l line) style.Clear {
	i := l.to - 2
	if !l.forced || i < l.from || c.pieces[i].kind != newline || !c.edgeAt(i+1, false) {
		return style.ClearNone
	}
	b := c.runs[c.pieces[i].run].parent
	if b == nil || b.Node == nil || b.Node.Tag() != dom.Br || c.spans[c.pieces[i+1].span].box != b {
		return style.ClearNone
	}
	return b.Style.Clear
}

// lines breaks the pieces greedily into lines no wider than width where they may wrap.
// unbounded is the width lines take for no width to fit: every break forced.
const unbounded = float32(1e30)

func (c *inlineContent) lines(width float32) []line {
	switch {
	case width >= unbounded && c.unbroken != nil:
		return c.unbroken
	case width < unbounded && c.broken != nil && c.brokenAt == width:
		return c.broken
	case width < unbounded && c.broken != nil && c.flowed && c.brokenAt-width < 0.01 && width-c.brokenAt < 0.01:
		// lines broken round floats stand for their width, however it was reached
		return c.broken
	}
	b := breaker{c: c}
	out := c.reshaped(c.styled(width, b.breakAll(width)))
	if width >= unbounded {
		c.unbroken = out
	} else {
		c.broken, c.brokenAt = out, width
	}
	return out
}

// edgesIn are the margin, border and padding of sp in style st, resolved against basis,
// none at a side where an element blocks cut has no edge.
func (sp *inlineSpan) edgesIn(st *style.Computed, basis float32) (rect, rect, rect) {
	m, p := edges(st.Margin, basis), edges(st.Padding, basis)
	b := rect{l: st.Border.Left, r: st.Border.Right, t: st.Border.Top, b: st.Border.Bottom}
	left, right := sp.noStart, sp.noEnd
	if st.Dir == style.RTL {
		left, right = right, left
	}
	if left {
		m.l, b.l, p.l = 0, 0, 0
	}
	if right {
		m.r, b.r, p.r = 0, 0, 0
	}
	return m, b, p
}

// resizeEdge gives the edge piece i width w, and line as the other styles measure it:
// the first line's where the pieces stand in their own, their own otherwise.
func (c *inlineContent) resizeEdge(i int, w, line float32) {
	if c.firstPieces != nil && c.firstPieces[i].width != line {
		c.firstPieces[i].width = line
		c.broken, c.unbroken = nil, nil
	}
	if p := &c.pieces[i]; p.width != w {
		p.width = w
		c.broken, c.unbroken = nil, nil
	}
}

// resize gives piece i width w, as the first line measures it too, forgetting the lines
// broken at the old one.
func (c *inlineContent) resize(i int, w float32) {
	if c.firstPieces != nil {
		c.firstPieces[i].width = w
	}
	if p := &c.pieces[i]; p.width != w {
		p.width = w
		c.broken, c.unbroken = nil, nil
	}
}

func glued(k pieceKind) bool {
	return k == word || k == edge || k == emergency || k == marker || k == anchor || k == softHyphen
}

// emergencyBreak finds where to cut the unbreakable pieces [from, end) that fit no line:
// at the last emergency break that fits in width, or after the first grapheme. It returns
// the emergency piece's index and the width before it, false without such breaks.
func (c *inlineContent) emergencyBreak(from, end int, width float32) (int, bool) {
	take := -1
	acc := newPen(c.unit, 0)
	for k := from; k < end; k++ {
		if c.pieces[k].kind == emergency {
			w := acc.end()
			if w > width && take >= 0 {
				break
			}
			if take < 0 || w <= width {
				take = k
			}
		}
		acc.step(c.pieces[k], 0)
	}
	return take, take >= 0
}

// metricsOf is st's font metrics, measured once per inline context and layout.
func (t *tree) metricsOf(c *inlineContent, st *style.Computed) box.Metrics {
	if m, ok := c.metrics[st]; ok {
		return m
	}
	if c.metrics == nil {
		c.metrics = map[*style.Computed]box.Metrics{}
	}
	m := t.fonts.Metrics(st.Font)
	if st.Stands() {
		m = central(m, c.unit)
	}
	c.metrics[st] = m
	return m
}

// central is metrics m about the central baseline, which vertical lines whose text may
// stand align their inline boxes on, as Chrome's: the room above and below it each half
// of the ascent and descent, in whole device pixels, the odd one above, unit the layout
// unit.
func central(m box.Metrics, unit float32) box.Metrics {
	px := unit / 64
	h := m.Ascent + m.Descent
	m.Descent = float32(math.Floor(float64(h*px)/2)) / px
	m.Ascent = h - m.Descent
	return m
}

// alphabetic is how far below the baseline the inline boxes of st align on its glyphs'
// alphabetic baseline stands: below the central one of vertical lines whose text may
// stand, else none.
func (t *tree) alphabetic(c *inlineContent, st *style.Computed) float32 {
	if !st.Stands() {
		return 0
	}
	return t.fonts.Metrics(st.Font).Ascent - t.metricsOf(c, st).Ascent
}

// lineHeight is one run's line height and the baseline's place in it (half-leading above
// the ascent).
func (t *tree) lineHeight(c *inlineContent, st *style.Computed) (float32, float32) {
	return LineHeight(t.metricsOf(c, st), st, PixelsPerPoint(t.fonts))
}

// LineHeight is the line height a face with metrics m takes in a run of style st, and
// the baseline's place in it, on a display of px device pixels per point.
func LineHeight(m box.Metrics, st *style.Computed, px float32) (float32, float32) {
	h := m.Ascent + m.Descent + m.LineGap
	switch {
	case st.LineHeight.Normal:
	case st.LineHeight.Factor != 0:
		h = floorToUnit(st.LineHeight.Factor*st.Font.Size, 64*px)
	default:
		h = floorToUnit(st.LineHeight.Px, 64*px)
	}
	// Chrome puts half the leading above the glyphs, the rest below: halved in layout
	// units, truncated as C++ divides, then floored to device pixels.
	half := int64(math.Round(float64((h-m.Ascent-m.Descent)*64*px))) / 2
	return h, m.Ascent + float32(math.Floor(float64(half)/64))/px
}

// extent is what a box takes above and below a baseline.
type extent struct{ above, below float32 }

// with is e grown to hold o, whose baseline stands shift below e's.
func (e extent) with(o extent, shift float32) extent {
	return extent{max(e.above, o.above-shift), max(e.below, o.below+shift)}
}

// pendingAlign is a box whose vertical-align waits for another's size: text-top and
// text-bottom for its parent's content area, top and bottom for the line, or for the
// nearest box above it that is itself top or bottom aligned, as Chrome has it. The box is
// span, or the atomic inline of run when span is -1; by is the span it waits for, -1 for
// the line.
type pendingAlign struct {
	span, run, by int
	at            style.VerticalAlign
	extent        extent
}

// lineBox is the height of line l and its baseline (CSS 2 section 10.8): the strut of
// the owner, then every inline box open on the line by its line height and every atomic
// inline by its margin box, each where its vertical-align puts it from its parent's
// baseline, and at last the boxes aligned to the line's top and bottom. It leaves the
// shift of each run and span of the line in them: from its parent's baseline as the
// boxes are sized, inner ones first, then from the line's.
func (t *tree) lineBox(c *inlineContent, l line) (float32, float32) {
	if l.phantom {
		return 0, 0
	}
	was := c.useFirst(l.first)
	defer c.useFirst(was)
	h, base := t.lineHeight(c, c.rootStyle())
	root := extent{base, h - base}
	if m := c.listMarker(); m != nil && l.from == 0 {
		// an outside ::marker of its own style pushes the item's first line down to
		// hold its top, its bottom left to overflow, as Chrome places it
		_, mb := t.lineHeight(c, m)
		root.above = max(root.above, mb)
	}
	c.stamp++
	c.pending = c.pending[:0]
	lo, hi := t.openBoxes(c, l, &root)
	for k := lo; k <= hi; k++ {
		sp := c.spans[k]
		if sp.line != c.stamp {
			continue
		}
		t.alignPending(c, k, &sp.extent)
		shift, ok := t.shiftOf(c, sp.st, c.styleAt(sp.up), sp.extent)
		if !ok {
			c.pending = append(c.pending,
				pendingAlign{span: k, by: c.aligner(sp.st, sp.up), at: sp.st.VerticalAlign, extent: sp.extent})
			continue
		}
		sp.shift = shift
		c.moved = c.moved || shift != 0
		into := &root
		if sp.up >= 0 {
			into = &c.spans[sp.up].extent
		}
		*into = into.with(sp.extent, shift)
	}
	t.whereTheyStood(c, lo, hi, &root)
	t.alignPending(c, -1, &root)
	if !c.moved {
		return root.above + root.below, root.above
	}
	for k := hi; k >= lo; k-- {
		if sp := c.spans[k]; sp.line == c.stamp && sp.up >= 0 {
			sp.shift += c.spans[sp.up].shift
		}
	}
	for i := l.from; i < l.to; i++ {
		p := c.pieces[i]
		if p.kind == marker || p.kind == anchor || p.kind == edge {
			continue
		}
		r := c.runs[p.run]
		switch {
		case r.span < 0 && r.atomic == nil:
			r.shift = 0
		case r.span < 0:
		case r.atomic == nil:
			r.shift = c.spans[r.span].shift
		default:
			r.shift += c.spans[r.span].shift
		}
	}
	return root.above + root.below, root.above
}

// whereTheyStood counts in root, as Chrome does, each span of the line inside a top or
// bottom aligned one that makes a box of its own in Chrome's fragments (one holding an
// atomic inline, or with a background, a border, padding or a margin): by its line height,
// where the shifts between the two put it from the line's baseline, as if the top or
// bottom box stood on it.
func (t *tree) whereTheyStood(c *inlineContent, lo, hi int, root *extent) {
	for k := lo; k <= hi; k++ {
		sp := c.spans[k]
		if sp.line != c.stamp || (sp.holds != c.stamp && !ownBox(sp)) {
			continue
		}
		var shift float32
		j := k
		for ; j >= 0; j = c.spans[j].up {
			at := c.spans[j].st.VerticalAlign
			if at == style.AlignTop || at == style.AlignBottom {
				break
			}
			shift += c.spans[j].shift
		}
		if j < 0 || j == k || shift == 0 {
			continue
		}
		h, base := t.lineHeight(c, sp.st)
		*root = root.with(extent{base, h - base}, shift)
	}
}

// ownBox reports whether sp paints something of its own round its text.
func ownBox(sp *inlineSpan) bool {
	return sp.margin != (rect{}) || sp.border != (rect{}) || sp.padding != (rect{}) || sp.st.Background.A > 0 ||
		len(sp.st.BackgroundImages) > 0
}

// openBoxes starts line l's alignment: each span open on it takes its line height, stamped
// with the line (the first and last of them returned), and each atomic inline and fallback
// face is counted in the box holding it, root for the line's own, or left pending.
func (t *tree) openBoxes(c *inlineContent, l line, root *extent) (int, int) {
	lo, hi := len(c.spans), -1
	for i := l.from; i < l.to; i++ {
		p := c.pieces[i]
		if p.kind == marker && c.mark != nil {
			// an item's marker inside it is an inline box of its own style
			mh, mb := t.lineHeight(c, c.mark.st)
			*root = root.with(extent{max(mb, c.mark.picture.Y), mh - mb}, 0)
		}
		if p.kind == marker || p.kind == anchor {
			continue
		}
		k := p.span
		if p.kind != edge {
			k = c.runs[p.run].span
		}
		for j := k; j >= 0 && c.spans[j].line != c.stamp; j = c.spans[j].up {
			sp := c.spans[j]
			sh, sb := t.lineHeight(c, sp.st)
			sp.line, sp.extent, sp.shift = c.stamp, extent{sb, sh - sb}, 0
			lo, hi = min(lo, j), max(hi, j)
		}
		if p.kind == edge {
			continue
		}
		into := root
		if k >= 0 {
			into = &c.spans[k].extent
		}
		r := c.runs[p.run]
		if a := r.atomic; a != nil {
			if k >= 0 {
				c.spans[k].holds = c.stamp
			}
			top := a.margin.t + a.baseline
			e := extent{top, a.height + a.margin.vertical() - top}
			if shift, ok := t.shiftOf(c, r.st, c.styleAt(k), e); ok {
				r.shift = shift
				c.moved = c.moved || shift != 0
				*into = into.with(e, shift)
			} else {
				c.pending = append(c.pending,
					pendingAlign{span: -1, run: p.run, by: c.aligner(r.st, k), at: r.st.VerticalAlign, extent: e})
			}
		} else if m, ok := t.fallbackMetrics(c, p, r.st); ok {
			fh, fb := LineHeight(m, r.st, PixelsPerPoint(t.fonts))
			*into = into.with(extent{fb, fh - fb}, 0)
		}
	}
	return lo, hi
}

// listMarker is the ::marker style a rule gives the list item whose lines c holds,
// nil for content of another owner or a marker in the item's own style.
func (c *inlineContent) listMarker() *style.Computed {
	n := c.owner.box.Node
	if n == nil || !c.owner.st.ListItem || n.Tag() == dom.Summary || c.owner.st.ListStyle == style.ListNone &&
		!dom.Data(n).HasMarkerText {
		return nil
	}
	return dom.Data(n).Marker
}

// styleAt is the style of span k, the owner's for -1.
func (c *inlineContent) styleAt(k int) *style.Computed {
	if k < 0 {
		return c.rootStyle()
	}
	return c.spans[k].st
}

// shiftOf is how far below its parent's baseline vertical-align puts the baseline of a
// box of style st taking e around it, false for the values that wait for the parent's
// size or the line's.
func (t *tree) shiftOf(c *inlineContent, st, parent *style.Computed, e extent) (float32, bool) {
	switch st.VerticalAlign {
	case style.AlignBaselineText:
	case style.AlignSub:
		return scriptShift(parent.Font.Size, 5, PixelsPerPoint(t.fonts)), true
	case style.AlignSuper:
		return -scriptShift(parent.Font.Size, 3, PixelsPerPoint(t.fonts)), true
	case style.AlignLength:
		h, _ := t.lineHeight(c, st)
		return -st.VerticalShift.At(h), true
	case style.AlignMiddle:
		return (e.above-e.below)/2 - t.metricsOf(c, parent).XHeight/2, true
	case style.AlignBaselineMiddle:
		return (e.above - e.below) / 2, true
	case style.AlignTextTop, style.AlignTextBottom, style.AlignTop, style.AlignBottom:
		return 0, false
	}
	return 0, true
}

// scriptShift is how far sub (by 5) or super (by 3) moves a box from its parent's
// baseline, as Blink works it out in device pixels: a share of the font size and one
// pixel more, cut to the layout unit.
func scriptShift(size, by, px float32) float32 {
	return float32(math.Trunc(float64((size*px/by+1)*64))) / 64 / px
}

// aligner is the box that places a box of style st inside span up once sized: its parent
// for text-top and text-bottom, else the nearest span above that is top or bottom aligned.
func (c *inlineContent) aligner(st *style.Computed, up int) int {
	if st.VerticalAlign == style.AlignTextTop || st.VerticalAlign == style.AlignTextBottom {
		return up
	}
	for ; up >= 0; up = c.spans[up].up {
		if at := c.spans[up].st.VerticalAlign; at == style.AlignTop || at == style.AlignBottom {
			break
		}
	}
	return up
}

// alignPending places the boxes waiting for span by (the line for -1), which takes e so
// far: text-top and text-bottom ones on its content area, then top and bottom ones on e
// grown to hold the tallest of them, above for a bottom one and below otherwise.
func (t *tree) alignPending(c *inlineContent, by int, e *extent) {
	m := t.metricsOf(c, c.styleAt(by))
	var top, bottom float32
	for i := range c.pending {
		p := &c.pending[i]
		if p.by != by {
			continue
		}
		switch p.at {
		case style.AlignTextTop:
			c.settle(p, e, p.extent.above-m.Ascent)
		case style.AlignTextBottom:
			c.settle(p, e, m.Descent-p.extent.below)
		case style.AlignTop:
			top = max(top, p.extent.above+p.extent.below)
		case style.AlignBottom:
			bottom = max(bottom, p.extent.above+p.extent.below)
		case style.AlignBaselineText, style.AlignSub, style.AlignSuper, style.AlignMiddle, style.AlignLength,
			style.AlignBaselineMiddle:
		}
	}
	grown := *e
	switch {
	case max(top, bottom) <= grown.above+grown.below:
	case bottom > top:
		grown.above = bottom - grown.below
	default:
		grown.below = top - grown.above
	}
	for i := range c.pending {
		p := &c.pending[i]
		if p.by != by {
			continue
		}
		switch p.at {
		case style.AlignTop:
			c.settle(p, e, p.extent.above-grown.above)
		case style.AlignBottom:
			c.settle(p, e, grown.below-p.extent.below)
		case style.AlignBaselineText, style.AlignSub, style.AlignSuper, style.AlignMiddle, style.AlignLength,
			style.AlignTextTop, style.AlignTextBottom, style.AlignBaselineMiddle:
		}
	}
}

// settle puts p's baseline shift below that of the box it waited for, which takes e. A
// shifted box between the two moves p again, out of the room e made for it, as in Chrome.
func (c *inlineContent) settle(p *pendingAlign, e *extent, shift float32) {
	*e = e.with(p.extent, shift)
	c.moved = true
	if p.span >= 0 {
		c.spans[p.span].shift = shift
		return
	}
	c.runs[p.run].shift = shift
}

// fallbackMetrics is the metrics of the fallback faces a word draws with, when the fonts
// have fallbacks and its text needs them: Chrome stretches a line-height: normal line to
// fit them, and ignores them under any other line-height. They are about the central
// baseline where st's are (metricsOf).
func (t *tree) fallbackMetrics(c *inlineContent, p piece, st *style.Computed) (box.Metrics, bool) {
	ff, ok := t.fonts.(box.FallbackFonts)
	if !ok || p.kind != word || !st.LineHeight.Normal {
		return box.Metrics{}, false
	}
	m, ok := ff.FallbackMetrics(st.Font, p.text)
	if ok && st.Stands() {
		m = central(m, c.unit)
	}
	return m, ok
}

// measure is the inline content's size and baselines, Taffy's measure function for text.
// Lines break at the available width; content wider than that (a word that breaks only
// with overflow-wrap, or not at all) widens the size it reports, not the lines.
func (t *tree) measure(c *inlineContent, known size, avail spaces, parent size, clamp int) (size, float32, float32,
	int) {
	minW, maxW := t.contentWidths(c, parent)
	w := known.w
	lineW := w
	if !some(w) {
		switch avail.w.kind {
		case minContent:
			lineW = minW
		case maxContent:
			lineW = maxW
		case definite:
			lineW = min(avail.w.v, maxW)
		}
		w = max(lineW, minW)
	}
	if !some(parent.w) {
		parent.w = w
	}
	t.sizeAtomics(c, def(lineW), parent)
	var cleared float32
	if c.flow != nil {
		c.broken, c.brokenAt = t.flowLines(c, lineW), lineW
		cleared = c.clearedAfter
	}
	var h float32
	first, last := none, none
	ls := c.lines(lineW)
	for i, l := range ls {
		if clamp > 0 && i == clamp {
			break
		}
		lh, lb := t.lineBox(c, l)
		if c.owner.st.Mode == style.VerticalLR {
			// the line's over side is the frame's bottom, as turnLines mirrors it
			lb = lh - lb
		}
		h += l.push
		if i == 0 {
			first = h + lb
		}
		last = h + lb
		h += lh
	}
	shown := ls
	if clamp > 0 && clamp < len(ls) {
		shown = ls[:clamp]
	} else {
		h += cleared
	}
	if top, end := t.trims(c, shown); top != 0 || end != 0 {
		h -= top + end
		first, last = first-top, last-top
	}
	if some(known.h) {
		h = known.h
	}
	return size{w, h}, first, last, len(ls)
}

// trims is how far text-box-trim moves the owner's content edges in, past the half-leading
// and the room above and below the text-box-edge its first line's and its last line's
// root inline boxes keep: off the top of the first line and the bottom of the last.
func (t *tree) trims(c *inlineContent, ls []line) (float32, float32) {
	if c == nil || len(ls) == 0 {
		return 0, 0
	}
	start, end := trimmed(c.owner)
	if start == nil && end == nil {
		return 0, 0
	}
	st := c.owner.st
	m := t.metricsOf(c, st)
	var top, bottom float32
	if start != nil {
		_, lb := t.lineBox(c, ls[0])
		top = lb - overEdge(m, start.st.TextBoxEdge)
	}
	if end != nil {
		lh, lb := t.lineBox(c, ls[len(ls)-1])
		bottom = lh - lb - underEdge(m, end.st.TextBoxEdge)
	}
	return top, bottom
}

// overEdge is how far above the baseline text-box-edge's edge over the text stands in
// a face of metrics m, and underEdge how far below it its edge under the text does.
func overEdge(m box.Metrics, e style.TextBoxEdge) float32 {
	switch {
	case e.Named && e.Over == style.EdgeCap:
		return m.CapHeight
	case e.Named && e.Over == style.EdgeEx:
		return m.XHeight
	}
	return m.Ascent
}

func underEdge(m box.Metrics, e style.TextBoxEdge) float32 {
	if e.Named && e.Under == style.EdgeAlphabetic {
		return 0
	}
	return m.Descent
}

// trimmed is the block whose text-box-trim trims the first line of n's lines, and the
// one trimming their last, nil for none: n's own, or one round it that n is the first
// (or the last) block in the flow of, no padding nor border between, as Chrome's first
// formatted line.
func trimmed(n *node) (*node, *node) {
	var start, end *node
	for k, first, last := n, true, true; k != nil && (first || last); k = k.parent {
		trim := k.st.TextBoxTrim
		if first && (trim == style.TrimStart || trim == style.TrimBoth) && start == nil {
			start = k
		}
		if last && (trim == style.TrimEnd || trim == style.TrimBoth) && end == nil {
			end = k
		}
		p := k.parent
		if p == nil || !flowsBlocks(p) || !inFlow(k) {
			break
		}
		first = first && k.st.Padding.Top.Px == 0 && k.st.Border.Top == 0 && firstInFlow(p) == k
		last = last && k.st.Padding.Bottom.Px == 0 && k.st.Border.Bottom == 0 && lastInFlow(p) == k
	}
	return start, end
}

// flowsBlocks reports whether n lays its children out as blocks in a flow.
func flowsBlocks(n *node) bool {
	d := n.st.Display
	return n.inline == nil && (d == style.Block || d == style.InlineBlock) && !n.st.Table && n.st.TablePart == 0
}

func inFlow(n *node) bool {
	return n.st.Float == style.FloatNone && !outOfFlow(n.st.Position)
}

// firstInFlow and lastInFlow are n's first and last children in its flow.
func firstInFlow(n *node) *node {
	for _, k := range n.kids {
		if inFlow(k) {
			return k
		}
	}
	return nil
}

func lastInFlow(n *node) *node {
	for i := len(n.kids) - 1; i >= 0; i-- {
		if inFlow(n.kids[i]) {
			return n.kids[i]
		}
	}
	return nil
}

// placeText lays the text out at the owner's content width and places its atomic inlines.
func (t *tree) placeText(n *node, s size, inset rect) {
	c := n.inline
	content := s.w - inset.horizontal()
	t.sizeAtomics(c, def(content), size{content, none})
	ls := c.lines(content)
	y := t.textTop(n, ls, inset.t, s.h-inset.vertical())
	var spanLines map[*box.Box][]spanLine
	if len(c.oofs) > 0 {
		spanLines = map[*box.Box][]spanLine{}
	}
	for li, l := range ls {
		lh, lb := t.lineBox(c, l)
		was := c.useFirst(l.first)
		y += l.push
		off, perGap := c.align(n.st, l, li == len(ls)-1, l.avail-l.width)
		start := inset.l + l.x + off
		shown, order := c.ordered(l)
		if spanLines != nil {
			t.spanRects(c, l, shown, order, start, perGap, y, lb,
				map[*box.Box]geom.Rect{}, make([]bool, len(c.spans)), spanLines, nil)
		}
		at := l.pen(c.unit, start)
		for _, p := range shown {
			x := at.step(p, perGap)
			if p.kind == anchor {
				k := c.anchors[p.span].node
				c.oofs[k].static = c.staticOf(k, l, p.span, point{x, y}, start, lh, inset.l, content)
			}
			if p.kind == atomic {
				a := c.runs[p.run].atomic
				l := &a.node.unrounded
				l.location = point{x + a.margin.l, y + lb + c.runs[p.run].shift - a.baseline}
				if n.st.Mode == style.VerticalLR {
					// its line's over side is the page's right, as turnLines mirrors the text:
					// its margin box mirrored across the line, its margins on their sides
					l.location.y = 2*y + lh - l.location.y - a.height - a.margin.b + a.margin.t
				}
				st := a.node.st
				moved := inlineOffset(st, size{content, none})
				for k := c.runs[p.run].span; k >= 0; k = c.spans[k].up {
					moved = moved.Add(inlineOffset(c.spans[k].st, size{content, none}))
				}
				l.location = point{l.location.x + moved.X, l.location.y + moved.Y}
				l.margin = a.margin
				l.size = size{a.width, a.height}
				l.padding = edges(st.Padding, content)
				l.border = a.node.borders()
				g := scrollbarGutter(st)
				l.scrollbar = size{g.x, g.y}
				l.overflow = a.overflow
			}
		}
		c.useFirst(was)
		y += lh
	}
	for _, o := range c.oofs {
		c.containingSpan(o, spanLines, size{content, none})
	}
}

// staticOf is the static place of out-of-flow box k anchored at at on line l, which
// starts at start and is lh tall, in content from left: there for an inline-level one;
// a block-level one starts a line, below this one when content is before it, at its
// start, the right in a right-to-left block.
func (c *inlineContent) staticOf(k *node, l line, span int, at point, start, lh, left, content float32) point {
	if k.st.StaticInline {
		return at
	}
	static := point{left, at.y}
	before := at.x > start
	if c.owner.st.Dir == style.RTL {
		static.x = left + content
		before = c.showsBefore(l, span)
	}
	if before {
		static.y += lh
	}
	return static
}

// showsBefore reports whether anything with a width stands before the anchor of
// out-of-flow box k on line l, in the content's order.
func (c *inlineContent) showsBefore(l line, k int) bool {
	for _, p := range c.pieces[l.from:l.to] {
		if p.kind == anchor && p.span == k {
			return false
		}
		if p.width > 0 {
			return true
		}
	}
	return false
}

// containingSpan makes the nearest positioned span holding o its containing block: the
// box from the start of its first fragment's padding box to the end of its last one's,
// as CSS defines it for an inline element, moved with the spans position: relative moves
// round it. o's static place moves with the spans holding it; cb is what their offsets'
// percentages are of.
func (c *inlineContent) containingSpan(o *inlineOOF, lines map[*box.Box][]spanLine, cb size) {
	o.positioned = false
	for k := o.span; k >= 0; k = c.spans[k].up {
		moved := inlineOffset(c.spans[k].st, cb)
		o.static = point{o.static.x + moved.X, o.static.y + moved.Y}
	}
	for k := o.span; k >= 0; k = c.spans[k].up {
		sp := c.spans[k]
		if sp.st.Position == style.Static {
			continue
		}
		frags := lines[sp.box]
		if len(frags) == 0 {
			return
		}
		first, last := frags[0].r, frags[len(frags)-1].r
		x0 := first.X + sp.border.l
		x1 := last.X + last.W - sp.border.r
		y0 := first.Y + sp.border.t
		y1 := last.Y + last.H - sp.border.b
		moved := geom.Vec2{}
		for j := k; j >= 0; j = c.spans[j].up {
			moved = moved.Add(inlineOffset(c.spans[j].st, cb))
		}
		o.areaAt = point{x0 + moved.X, y0 + moved.Y}
		o.areaSize = size{max(x1-x0, 0), max(y1-y0, 0)}
		o.positioned = true
		return
	}
}

// writeText writes the fragments of the laid-out text into the box tree.
func (t *tree) writeText(n *node) {
	for p := n; p != nil; p = p.parent {
		if p.st.Display == style.DisplayNone {
			return
		}
	}
	c := n.inline
	for _, s := range c.spans {
		s.box.Children = s.box.Children[:0]
	}
	for _, r := range c.runs {
		if r.atomic == nil {
			r.box.Lines = r.box.Lines[:0]
		}
	}
	f := n.final
	inset := f.padding.add(f.border)
	content := f.size.w - inset.horizontal() - f.scrollbar.w
	if !frameOf(n).Vertical() {
		ge := gutterEdges(n.st, point{x: f.scrollbar.w, y: f.scrollbar.h})
		inset.l, inset.r = inset.l+ge.l, inset.r+ge.r
	}
	if cs := n.columns; cs != nil {
		content = cs.width
	}
	t.sizeAtomics(c, def(content), size{content, none})
	runRects, spanRects, spanLines, hidden := t.writeLines(n, c, inset, content)
	cb := size{content, none}
	if mode := frameOf(n); mode.Vertical() {
		// the box's size, cut by columns the union of its parts
		turnLines(c, size{n.box.Size.Y, n.box.Size.X}, mode, runRects, spanRects, spanLines)
		cb = cb.across()
	}
	origin := func(b *box.Box) geom.Vec2 {
		if r, ok := spanRects[b]; ok {
			return geom.Vec2{X: r.X, Y: r.Y}
		}
		return geom.Vec2{}
	}
	placeSpans(c, spanRects, spanLines, hidden, origin, cb)
	t.attachInline(n, c, runRects, origin)
}

// placeSpans gives each span its box from the rectangles its lines took.
func placeSpans(c *inlineContent, spanRects map[*box.Box]geom.Rect, spanLines map[*box.Box][]spanLine,
	hidden map[*box.Box]int, origin func(*box.Box) geom.Vec2, cb size) {
	for _, s := range c.spans {
		r := spanRects[s.box]
		o := origin(s.parent)
		s.box.Pos = geom.Vec2{X: r.X - o.X, Y: r.Y - o.Y}.Add(inlineOffset(s.box.Style, cb))
		s.box.Size = geom.Vec2{X: r.W, Y: r.H}
		mode := frameOf(c.owner)
		m, b, p := pageEdges(s.margin, mode), pageEdges(s.border, mode), pageEdges(s.padding, mode)
		s.box.Margin = box.Edges{Top: m.t, Right: m.r, Bottom: m.b, Left: m.l}
		s.box.Border = box.Edges{Top: b.t, Right: b.r, Bottom: b.b, Left: b.l}
		s.box.Padding = box.Edges{Top: p.t, Right: p.r, Bottom: p.b, Left: p.l}
		s.box.Fragments, s.box.Ends, s.box.Hidden = s.box.Fragments[:0], s.box.Ends[:0], hidden[s.box]
		parts := spanLines[s.box]
		moved := false
		for i, f := range parts {
			s.box.Fragments = append(s.box.Fragments, geom.Rect{X: f.r.X - r.X, Y: f.r.Y - r.Y, W: f.r.W, H: f.r.H})
			var usual box.Ends
			if i == 0 {
				usual |= box.LeftEnd
			}
			if i == len(parts)-1 {
				usual |= box.RightEnd
			}
			moved = moved || f.ends != usual
		}
		if moved && c.reorders && !mode.Vertical() {
			for _, f := range parts {
				s.box.Ends = append(s.box.Ends, f.ends)
			}
		}
	}
}

// attachInline places each run's box over its fragments and appends the runs, spans and
// marker to their parents, in order.
func (t *tree) attachInline(n *node, c *inlineContent, runRects []geom.Rect, origin func(*box.Box) geom.Vec2) {
	children := map[*box.Box][]*box.Box{}
	var order []*box.Box
	add := func(parent, child *box.Box) {
		if _, ok := children[parent]; !ok {
			order = append(order, parent)
		}
		children[parent] = append(children[parent], child)
	}
	openAt := func(i int) {
		for k := len(c.spans) - 1; k >= 0; k-- {
			if c.spans[k].first == i {
				add(c.spans[k].parent, c.spans[k].box)
			}
		}
	}
	if c.back != nil {
		add(n.box, c.back)
	}
	if c.marker != nil && !c.outside {
		add(n.box, c.marker)
	}
	for i, r := range c.runs {
		openAt(i)
		if r.atomic != nil {
			continue
		}
		rr := runRects[i]
		o := origin(r.parent)
		r.box.Pos = geom.Vec2{X: rr.X - o.X, Y: rr.Y - o.Y}
		r.box.Size = geom.Vec2{X: rr.W, Y: rr.H}
		if r.box.Node != nil {
			t.place(r.box.Node, r.box)
		}
		across := rr.Y
		if r.box.Style.Mode.Vertical() {
			across = rr.X
		}
		for j := range r.box.Lines {
			r.box.Lines[j].Rect.X -= rr.X
			r.box.Lines[j].Rect.Y -= rr.Y
			r.box.Lines[j].Baseline -= across
		}
		if len(r.box.Lines) > 0 {
			r.box.Baseline = r.box.Lines[0].Baseline
		}
		add(r.parent, r.box)
	}
	openAt(len(c.runs))
	for _, p := range order {
		p.Append(children[p]...)
	}
}

// writeLines writes each line's fragments into its runs' boxes, and returns the rectangles
// the runs and spans took, and how many of each span's lines are past a line-clamp's last.
//
//nolint:funlen // one pass over the lines, each written whole
func (t *tree) writeLines(n *node, c *inlineContent, inset rect, content float32) ([]geom.Rect,
	map[*box.Box]geom.Rect, map[*box.Box][]spanLine, map[*box.Box]int) {
	f := n.final
	ellipsis := n.st.Ellipsis && n.st.Overflow.Width != style.Visible
	runRects := make([]geom.Rect, len(c.runs))
	seen := make([]bool, len(c.runs))
	spanRects := make(map[*box.Box]geom.Rect)
	spanSeen := make([]bool, len(c.spans))
	spanLines := make(map[*box.Box][]spanLine)
	hidden := make(map[*box.Box]int)
	c.back = nil
	c.lineBoxes = c.lineBoxes[:0]
	ls := c.lines(content)
	y := t.textTop(n, ls, inset.t, f.size.h-inset.vertical()-f.scrollbar.h)
	for li, l := range ls {
		lh, lb := t.lineBox(c, l)
		was := c.useFirst(l.first)
		y += l.push
		flowY := y
		if n.frag != nil {
			dx, dy := n.frag.lineShift(y)
			l.x += dx
			y += dy
		}
		off, perGap := c.align(n.st, l, li == len(ls)-1, l.avail-l.width)
		x := inset.l + l.x + off
		if c.outside {
			c.markerLine.X, c.markerLine.W = inset.l+l.x, l.avail
		}
		pieces, order := c.ordered(l)
		if c.firstOn {
			t.backOf(c, n, geom.Rect{X: x, Y: y, W: l.width, H: lh}, lb)
		}
		past := n.clamp.shown > 0 && li >= n.clamp.shown && n.clamp.hides
		var hides map[*box.Box]int
		if past {
			hides = hidden
		}
		t.spanRects(c, l, pieces, order, x, perGap, y, lb, spanRects, spanSeen, spanLines, hides)
		cut, cutStart := false, false
		switch {
		case past:
			pieces = nil
		case n.clamp.ellipsis && li == n.clamp.shown-1 && len(c.runs) > 0:
			// the clamp's last line makes room for its ellipsis, aligned with it
			pieces, cut = t.ellipsize(c, pieces, l.avail)
			w := newPen(c.unit, 0)
			for _, p := range pieces {
				w.step(p, 0)
			}
			ends := c.runs[pieces[len(pieces)-1].run].st
			off, perGap = c.align(n.st, l, true, l.avail-w.end()-t.fonts.Advance(ends.Font, "…"))
			x = inset.l + l.x + off
		case ellipsis && l.width > l.avail && len(c.runs) > 0:
			pieces, x, cut, cutStart = t.ellipsizeLine(n, c, pieces, x, inset.l+l.x, l.avail)
		}
		t.writeRuns(c, pieces, geom.Rect{X: x, Y: y, H: lh}, lb, l.indent, perGap, [2]bool{cutStart, cut}, runRects,
			seen, c.tailOf(l))
		c.lineBoxes = append(c.lineBoxes, span{y, y + lh})
		c.useFirst(was)
		y = flowY + lh
	}
	return runRects, spanRects, spanLines, hidden
}

// writeRuns writes the fragments of a line's pieces into their runs' boxes, the line
// at at (its width aside) with its baseline lb, an ellipsis before or after it as cut
// says, and grows each run's rectangle in rects.
func (t *tree) writeRuns(c *inlineContent, pieces []piece, at geom.Rect, lb, indent, perGap float32, cut [2]bool,
	rects []geom.Rect, seen []bool, tail lineTail) {
	y, lh := at.Y, at.H
	cutStart := cut[0]
	walk := line{indent: indent}.pen(c.unit, at.X)
	for i := 0; i < len(pieces); {
		run := pieces[i].run
		if k := pieces[i].kind; k == atomic || k == edge || k == marker || k == anchor || k == tab || k == newline ||
			k == gap && perGap > 0 {
			x := walk.step(pieces[i], perGap)
			if k == marker {
				c.placeMarker(geom.Rect{X: x, Y: y, W: pieces[i].width, H: lh}, lb)
			}
			i++
			continue
		}
		var text string
		var start float32
		var from int32
		var rtl bool
		unspaced := pieces[i].unspaced
		first := i
		text, start, from, rtl, i = runText(&walk, pieces, i, perGap)
		end := walk.end()
		if i < len(pieces) && isText(pieces[i].kind) && pieces[i].run == run {
			end = walk.at()
		}
		r := geom.Rect{X: start, Y: y, W: max(end-start, 0), H: lh}
		st := c.runs[run].st
		frag := t.fragment(st, text, r, y+lb+c.runs[run].shift+t.alphabetic(c, st), cutStart, cut[1] && i == len(pieces))
		frag.Start, frag.RTL, frag.Unspaced = from+c.runs[run].skip, rtl, unspaced
		if !rtl {
			frag.Spacing = fragmentSpacing(pieces[first:i])
		}
		for _, p := range pieces[first:i] {
			if p.shows {
				frag.Hyphen = hyphenText(t.fonts, c.runs[run].st)
			}
		}
		//nolint:gosec // a text node's length
		if before, after := c.around(run, from, from+int32(utf8.RuneCountInString(text))); shape.JoinsAcross(before,
			after) {
			frag.Before, frag.After = before, after
		}
		if whole := wholeOf(pieces[first:i]); whole != 0 && !frag.Ellipsis {
			start, end := first, i-1
			if rtl {
				start, end = end, start
			}
			lead, before := besideShown(pieces, start, -1, whole, rtl)
			trail, after := besideShown(pieces, end, 1, whole, rtl)
			if tail.whole == whole && trail == "" && tail.in(pieces[first:i]) {
				trail, after = tail.trail, tail.after
			}
			// Where the line cuts the text shaped as one, the text round the fragment
			// stays its context.
			if lead != "" {
				frag.Lead, frag.Before = lead, before
			}
			if trail != "" {
				frag.Trail, frag.After = trail, after
			}
		}
		if c.firstOn {
			frag.Style = c.runs[run].st
		}
		cutStart = false
		c.runs[run].box.Lines = append(c.runs[run].box.Lines, frag)
		rects[run] = unionRect(rects[run], frag.Rect, seen[run])
		seen[run] = true
	}
}

// backOf gives the first line, at r with its baseline lb, the box that paints its
// background: over its content, as high as its font's content area.
func (t *tree) backOf(c *inlineContent, n *node, r geom.Rect, lb float32) {
	g := lineBack(n)
	if g == nil || dom.Data(g).Style == nil {
		return
	}
	st := dom.Data(g).Style
	m := t.metricsOf(c, c.line)
	r.Y += lb - m.Ascent
	r.H = m.Ascent + m.Descent
	c.back = &box.Box{Kind: box.Element, Node: g, Style: st, Pos: geom.Vec2{X: r.X, Y: r.Y},
		Size: geom.Vec2{X: r.W, Y: r.H}, Fragments: []geom.Rect{{W: r.W, H: r.H}}}
	t.place(g, c.back)
}

// runText steps walk over the text one run has on a line from piece i, shown in one
// direction: its words and the spaces no justification widens. It returns the text, in
// its own order, where it starts, the character of the run it starts at, whether it is
// shown right to left, and the piece after it.
func runText(walk *pen, pieces []piece, i int, perGap float32) (string, float32, int32, bool, int) {
	run, level, start := pieces[i].run, pieces[i].level, walk.at()
	var texts []piece
	for i < len(pieces) && pieces[i].run == run && pieces[i].level == level && isText(pieces[i].kind) &&
		(pieces[i].kind != gap || perGap == 0) {
		at := walk.step(pieces[i], perGap)
		if len(texts) == 0 {
			start = at
		}
		if pieces[i].kind == word || pieces[i].kind == gap {
			texts = append(texts, pieces[i])
		}
		i++
	}
	rtl := level%2 == 1
	var text strings.Builder
	var from int32
	for k, p := range textOf(texts, rtl) {
		if k == 0 {
			from = p.at
		}
		text.WriteString(p.text)
	}
	return text.String(), start, from, rtl, i
}

// fragment is a line's piece of a run's text at r, with the ellipsis where the line was
// cut before or after it.
func (t *tree) fragment(st *style.Computed, text string, r geom.Rect, baseline float32,
	cutBefore, cutAfter bool) box.Fragment {
	frag := box.Fragment{Text: unshown(text), Rect: r, Baseline: baseline}
	if cutBefore {
		frag.Ellipsis = true
	}
	if cutAfter {
		frag.Text += "…"
		frag.Ellipsis = true
		frag.Rect.W += t.fonts.Advance(st.Font, "…")
	}
	return frag
}

// placeMarker puts the ::marker box at r, its triangle on baseline lb.
func (c *inlineContent) placeMarker(r geom.Rect, lb float32) {
	if c.outside {
		c.markerLine.Y, c.markerLine.H, c.markerBaseline = r.Y, r.H, r.Y+lb
		return
	}
	if c.mark != nil {
		c.mark.place(c.marker, r, lb)
		return
	}
	c.marker.Pos = geom.Vec2{X: r.X, Y: r.Y}
	c.marker.Size = geom.Vec2{X: r.W, Y: r.H}
	c.marker.Lines = []box.Fragment{{Text: markerOf(c.owner.st), Rect: geom.Rect{W: r.W, H: r.H}, Baseline: lb}}
}

// ellipsizeLine cuts a line too wide for content and puts the ellipsis at its end, or at
// its left, then flush with the content's right: the end of a line of direction rtl.
func (t *tree) ellipsizeLine(n *node, c *inlineContent, pieces []piece, x, left, content float32) ([]piece, float32,
	bool, bool) {
	if !n.st.EllipsisStart && n.st.Dir != style.RTL {
		kept, cut := t.ellipsize(c, pieces, content)
		return kept, x, cut, false
	}
	pieces = t.ellipsizeStart(c, pieces, content, n.st.EllipsisStart)
	w := newPen(c.unit, 0)
	for _, p := range pieces {
		w.step(p, 0)
	}
	return pieces, left + content - w.end(), false, true
}

// spanLine is a span's part on a line: its border box there, and which of the span's left
// and right edges it holds.
type spanLine struct {
	r    geom.Rect
	ends box.Ends
}

// spanRects adds line l's parts of each span to its border box: from its start edge (or the
// line's start) to its end edge (or the line's end), over its content area plus vertical
// padding and border, one part to each run of its pieces the line shows side by side once
// bidi reorders it. The pieces are the line's before any ellipsis, as Chrome's boxes are.
func (t *tree) spanRects(c *inlineContent, l line, pieces []piece, order []int, x, perGap, y,
	lb float32, rects map[*box.Box]geom.Rect, seen []bool, lines map[*box.Box][]spanLine, hidden map[*box.Box]int) {
	at := make([]float32, len(pieces)+1)
	walk := l.pen(c.unit, x)
	for i, p := range pieces {
		at[i] = walk.step(p, perGap)
	}
	at[len(pieces)] = walk.end()
	for k, sp := range c.spans {
		lo, hi := max(sp.open, l.from), min(sp.close, l.from+len(pieces)-1)
		if lo > hi {
			continue
		}
		if seen[k] && sp.box.Node != nil && sp.box.Node.Tag() == dom.Br {
			// a br's box is where its line ends, not the start of the next
			continue
		}
		in := func(v int) bool { i := l.from + logical(order, v); return i >= lo && i <= hi }
		first, last := -1, -1
		for v := range pieces {
			if in(v) {
				if first < 0 {
					first = v
				}
				last = v
			}
		}
		if at[first] == at[last+1] && c.edgeAt(lo, true) && c.opensOnSpace(hi) {
			// a start of no width, the space after it dropped at the line's end, is no box
			continue
		}
		from := -1
		for v := first; v <= last; v++ {
			if !in(v) {
				continue
			}
			if from < 0 {
				from = v
			}
			if v == last || !in(v+1) {
				t.spanPart(c, sp, k, pieces, at, from, v, l, y, lb, rects, seen, lines, hidden)
				from = -1
			}
		}
	}
}

// spanPart adds the part of span k shown by pieces first to last of line l to its box.
func (t *tree) spanPart(c *inlineContent, sp *inlineSpan, k int, pieces []piece, at []float32, first, last int,
	l line, y, lb float32, rects map[*box.Box]geom.Rect, seen []bool, lines map[*box.Box][]spanLine,
	hidden map[*box.Box]int) {
	left, right := at[first], at[last+1]
	var ends box.Ends
	if p := pieces[first]; p.kind == edge && p.span == k && p.start {
		left += sp.margin.l
		ends |= box.LeftEnd
	}
	if p := pieces[last]; p.kind == edge && p.span == k && !p.start {
		right -= sp.margin.r
		ends |= box.RightEnd
	}
	m := t.metricsOf(c, sp.st)
	over, under := m.Ascent, m.Descent
	switch e := sp.st.TextBoxEdge; sp.st.TextBoxTrim {
	case style.TrimStart:
		over = overEdge(m, e)
	case style.TrimEnd:
		under = underEdge(m, e)
	case style.TrimBoth:
		over, under = overEdge(m, e), underEdge(m, e)
	case style.TrimNone:
	}
	top := y + lb + sp.shift - over - sp.padding.t - sp.border.t
	bottom := y + lb + sp.shift + under + sp.padding.b + sp.border.b
	r := geom.Rect{X: left, Y: top, W: max(right-left, 0), H: bottom - top}
	if l.phantom {
		r = geom.Rect{X: left, Y: y}
	}
	rects[sp.box] = unionRect(rects[sp.box], r, seen[k])
	seen[k] = true
	lines[sp.box] = append(lines[sp.box], spanLine{r, ends})
	if hidden != nil {
		hidden[sp.box]++
	}
}

// ellipsize drops the line's end until it and an ellipsis fit in width.
func (t *tree) ellipsize(c *inlineContent, pieces []piece, width float32) ([]piece, bool) {
	last := pieces[len(pieces)-1].run
	room := width - t.fonts.Advance(c.runs[last].st.Font, "…")
	out := make([]piece, 0, len(pieces))
	used := float32(0)
	for _, p := range pieces {
		if used+p.width <= room {
			out = append(out, p)
			used += p.width
			continue
		}
		if p.kind == word {
			if head, w := t.fitPrefix(c.runs[p.run].st, p.text, room-used); head != "" {
				out = append(out, piece{kind: word, run: p.run, text: head, width: w})
			}
		}
		break
	}
	for len(out) > 0 && out[len(out)-1].kind == gap {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		out = []piece{{kind: word, run: last}}
	}
	return out, true
}

// ellipsizeStart drops the line's start until an ellipsis and the rest fit in width, the
// ellipsis a word of the first run kept, a space after it dropped when trim says so, as
// truncate-start's isolated text has none there. The line then ends at the box's end, as Chrome's
// right-to-left cut leaves its room before the ellipsis.
func (t *tree) ellipsizeStart(c *inlineContent, pieces []piece, width float32, trim bool) []piece {
	last := pieces[len(pieces)-1].run
	room := width - t.fonts.Advance(c.runs[last].st.Font, "…")
	used := float32(0)
	i := len(pieces)
	var out []piece
	for ; i > 0; i-- {
		p := pieces[i-1]
		if used+p.width <= room {
			used += p.width
			continue
		}
		if p.kind == word {
			if tail, w := t.fitSuffix(c.runs[p.run].st, p.text, room-used); tail != "" {
				out = append(out, piece{kind: word, run: p.run, text: tail, width: w})
			}
		}
		break
	}
	out = append(out, pieces[i:]...)
	for trim && len(out) > 0 && out[0].kind == gap {
		out = out[1:]
	}
	run := last
	for _, p := range out {
		if p.kind == word {
			run = p.run
			break
		}
	}
	e := piece{kind: word, run: run, text: "…", width: t.fonts.Advance(c.runs[run].st.Font, "…")}
	return append([]piece{e}, out...)
}

// align is where line l starts in free room and how much each of its spaces grows: justify
// spreads the room over the spaces of every line but the last and those a newline ends.
func (c *inlineContent) align(st *style.Computed, l line, last bool, free float32) (float32, float32) {
	dir := st.Dir
	if st.Bidi == style.BidiPlaintext {
		// the paragraph's own direction, read from its text
		dir = style.LTR
		if c.base%2 == 1 {
			dir = style.RTL
		}
	}
	if free < 0 && dir == style.RTL {
		// a line too long for its box is start aligned, and overflows its end
		return free, 0
	}
	start := alignOffset(style.Start.Resolve(dir), free)
	if a := st.TextAlign; a != style.Justify {
		return alignOffset(a.Resolve(dir), free), 0
	}
	if last || l.forced || free <= 0 {
		return start, 0
	}
	gaps := 0
	for _, p := range c.shown(l) {
		if p.kind == gap {
			gaps++
		}
	}
	if gaps == 0 {
		return start, 0
	}
	return 0, free / float32(gaps)
}

// fitPrefix is the longest run of whole graphemes at the start of text no wider than room,
// and its width, found by bisection over the grapheme boundaries.
func (t *tree) fitPrefix(st *style.Computed, text string, room float32) (string, float32) {
	var buf [64]int
	ends := append(buf[:0], 0)
	rest, state := text, -1
	var g string
	for at := 0; len(rest) > 0; {
		g, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		at += len(g)
		ends = append(ends, at)
	}
	width := func(i int) float32 {
		s := text[:ends[i]]
		return advance(t.fonts, st, s) + st.Spacing(s)
	}
	lo, hi := 0, len(ends)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if width(mid) <= room {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return text[:ends[lo]], width(lo)
}

// fitSuffix is the longest run of whole graphemes at the end of text no wider than room,
// and its width.
func (t *tree) fitSuffix(st *style.Computed, text string, room float32) (string, float32) {
	var buf [64]int
	starts := append(buf[:0], 0)
	rest, state := text, -1
	var g string
	for at := 0; len(rest) > 0; {
		g, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		at += len(g)
		starts = append(starts, at)
	}
	width := func(i int) float32 {
		s := text[starts[i]:]
		return advance(t.fonts, st, s) + st.Spacing(s)
	}
	lo, hi := 0, len(starts)-1
	for lo < hi {
		mid := (lo + hi) / 2
		if width(mid) <= room {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return text[starts[lo]:], width(lo)
}

func alignOffset(a style.TextAlign, free float32) float32 {
	if free <= 0 {
		return 0
	}
	switch a {
	case style.Right, style.End, style.BlockRight:
		return free
	case style.Center, style.BlockCenter:
		return free / 2
	case style.Start, style.Left, style.Justify, style.BlockLeft:
	}
	return 0
}

func unionRect(a, b geom.Rect, haveA bool) geom.Rect {
	if !haveA {
		return b
	}
	x0, y0 := min(a.X, b.X), min(a.Y, b.Y)
	x1, y1 := max(a.X+a.W, b.X+b.W), max(a.Y+a.H, b.Y+b.H)
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// atomsIntoSpans moves the boxes of atomic inlines inside spans from the owner's box to their
// span's, keeping where they are; tree.write calls it once the owner's kids are written.
func (t *tree) atomsIntoSpans(n *node) {
	c := n.inline
	if c == nil {
		return
	}
	for _, r := range c.runs {
		if r.atomic == nil || r.parent == n.box || r.box.Parent != n.box {
			continue
		}
		kids := n.box.Children
		for i, k := range kids {
			if k == r.box {
				n.box.Children = slices.Concat(kids[:i], kids[i+1:])
				break
			}
		}
		for p := r.parent; p != nil && p != n.box; p = p.Parent {
			r.box.Pos = r.box.Pos.Sub(p.Pos)
		}
		r.parent.Append(r.box)
	}
}

// lineTail is the text after a line's end that its last text is shaped as one with: that
// of the text Chrome shapes as one it is in, whole, and the text round it.
type lineTail struct {
	whole        int32
	run          int
	at           int32
	trail, after string
}

// in reports whether pieces hold the line's last text, which t follows.
func (t lineTail) in(pieces []piece) bool {
	for _, p := range pieces {
		if p.kind == word && p.run == t.run && p.at == t.at {
			return true
		}
	}
	return false
}

// tailOf is what line l's last text is drawn shaped with past the line's end: where the
// line ends at a space, the text after it, as Chrome keeps the shaping of a line's end
// there; nothing where it breaks inside a word, whose end Chrome shapes again.
func (c *inlineContent) tailOf(l line) lineTail {
	k := l.to - 1
	if k < l.from || l.to >= len(c.pieces) || c.pieces[l.to].kind != gap || c.pieces[k].whole == 0 {
		return lineTail{}
	}
	if p := c.pieces[k]; p.kind == word {
		trail, after := beside(c.pieces, k, 1, p.whole)
		return lineTail{p.whole, p.run, p.at, trail, after}
	}
	return lineTail{}
}

// wholeOf is the text Chrome shapes as one that a fragment's pieces are in, 0 for none.
func wholeOf(pieces []piece) int32 {
	for _, p := range pieces {
		if p.kind == word || p.kind == gap {
			return p.whole
		}
	}
	return 0
}
