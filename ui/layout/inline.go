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
	// marker is a summary's disclosure triangle, nil elsewhere.
	marker *box.Box
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
	// unindented is set where the first line is not its block container's: an anonymous
	// block after other content.
	unindented bool
	// reorders is set when the pieces' bidi levels may order a line otherwise than its
	// text, and base is the level of its first paragraph.
	reorders bool
	base     bidi.Level
	// line is the style of the block's first line where a ::first-line styles it, nil
	// elsewhere; firstPieces are the pieces as measured in the first line's styles, and
	// firstOn is set while they, and each run's and span's first-line style, stand in
	// for their own (useFirst). firstInWord is inWord for the first line's text.
	line        *style.Computed
	firstPieces []piece
	firstOn     bool
	firstInWord bool
	// back is the box painting the first line's background, nil without one.
	back *box.Box
	// height is the owner's content height its atomic inlines' percentages are of, none
	// where it depends on the content.
	height float32
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

type anchorAt struct {
	node *node
	run  int
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
	// lineSt is the span's style on its block's first line.
	lineSt *style.Computed
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
	// tab is a tab of text that keeps it, as wide as it takes to reach the next tab stop,
	// or a carriage return there, of no width.
	tab
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
	// stop is a tab's distance between tab stops, none for a carriage return, and least
	// the room it takes at the least.
	stop, least float32
	// unspaced marks a gap whose first space starts the content's text, and so takes no
	// word-spacing.
	unspaced bool
}

const zeroWidthSpace = "\u200b"

// blanks turns the whitespace that collapses like a space into one.
var blanks = strings.NewReplacer("\t", " ", "\r", " ", "\f", " ") //nolint:gochecknoglobals // a Replacer built once

// collapsedBlanks is text with the whitespace ws collapses turned into spaces: newlines too,
// unless ws is pre-line.
func collapsedBlanks(text string, ws style.WhiteSpace) string {
	if !collapses(ws) {
		return text
	}
	text = blanks.Replace(text)
	if ws != style.PreLine {
		text = strings.ReplaceAll(text, "\n", " ")
	}
	return text
}

func collapses(ws style.WhiteSpace) bool {
	return ws == style.Normal || ws == style.NoWrapSpace || ws == style.PreLine
}

func wraps(ws style.WhiteSpace) bool { return ws != style.NoWrapSpace && ws != style.Pre }

func isAtomicDisplay(d style.Display) bool {
	return d == style.InlineBlock || d == style.InlineFlex || d == style.InlineGrid
}

// inlineOf gathers the inline content of items for owner, nil when it is only collapsible
// whitespace.
func (t *tree) inlineOf(owner *node, items []item) *inlineContent {
	c := &inlineContent{owner: owner, unit: unitsPerPoint(t.fonts), line: firstLineOf(owner), height: none}
	t.gather(c, owner.box, items)
	if n := owner.box.Node; n != nil && n.Tag() == dom.Summary && owner.st.ListItem && markerOf(owner.st) != "" {
		c.marker = &box.Box{Kind: box.Marker, Node: n, Style: owner.st}
	}
	visible := c.marker != nil || framedCut(items)
	for _, r := range c.runs {
		if r.atomic != nil || !collapses(r.st.WhiteSpace) || !isCollapsibleSpace(r.text) {
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

// framedCut reports a part of an inline element blocks cut among items, or inside one,
// with a margin, a border or a padding on its edge: Chrome keeps a line for it, text or
// none.
func framedCut(items []item) bool {
	room := func(margin, padding style.Length, border float32) bool {
		return border != 0 || margin.Px != 0 || margin.Percent != 0 || padding.Px != 0 || padding.Percent != 0
	}
	for _, it := range items {
		if it.cut == nil {
			continue
		}
		st := styleOf(it.element)
		if it.first && room(st.Margin.Left, st.Padding.Left, st.Border.Left) ||
			it.last && room(st.Margin.Right, st.Padding.Right, st.Border.Right) || framedCut(it.inside) {
			return true
		}
	}
	return false
}

func (t *tree) gather(c *inlineContent, parent *box.Box, items []item) {
	for _, it := range items {
		if it.text != nil {
			t.gatherText(c, parent, it.text)
			continue
		}
		st := styleOf(it.element)
		if st.Display == style.DisplayNone {
			continue
		}
		if st.Float != style.FloatNone {
			// A float inside an inline element: its block container takes it from the
			// owner's kids and places it before this content.
			c.owner.kids = append(c.owner.kids, t.build(it.element, c.owner))
			continue
		}
		if outOfFlow(st.Position) {
			if parent == c.owner.box {
				continue
			}
			n := t.build(it.element, c.owner)
			c.owner.kids = append(c.owner.kids, n)
			if c.oofs == nil {
				c.oofs = map[*node]*inlineOOF{}
			}
			c.oofs[n] = &inlineOOF{span: -1}
			c.anchors = append(c.anchors, anchorAt{node: n, run: len(c.runs)})
			continue
		}
		if it.element.Tag() == dom.Br {
			t.lineBreak(c, parent, it.element, st)
			continue
		}
		if isAtomicDisplay(st.Display) || st.Display == style.Inline && pictured(it.element) {
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
		s := &inlineSpan{box: b, parent: parent, st: st, first: first, end: len(c.runs),
			noStart: it.cut != nil && !it.first, noEnd: it.cut != nil && !it.last}
		s.lineSt = c.lineStyle(it.element, st)
		for i := first; i < len(c.runs); i++ {
			s.runs = append(s.runs, i)
		}
		c.hold(s, inner)
	}
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

// letterOf is the ::first-letter node that takes text's first letter, and where the
// letter ends in its text, nil when none does.
func letterOf(text *dom.Node) (*dom.Node, int) {
	g := dom.Data(text).Letter
	if g == nil || dom.Data(g).Style == nil || dom.Data(g).Letter != text {
		return nil, 0
	}
	end := dom.Data(text).LetterEnd
	if end > len(text.Text()) {
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
	if g, end := letterOf(n); g != nil && text == n.Text() {
		t.gather(c, parent, []item{{element: g}})
		skip = int32(utf8.RuneCountInString(text[:end])) //nolint:gosec // a text node's length
		if text = text[end:]; text == "" {
			return
		}
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
	afterSpace := true
	soft := c.softBreaks()
	levels := c.levels()
	c.reorders = levels != nil
	if c.marker != nil {
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
		afterSpace = c.cutRun(fonts, i, r, soft[i], afterSpace, lv)
		c.closeSpans(i)
	}
	c.openSpans(len(c.runs))
	c.anchorsAt(len(c.runs))
	if c.reorders {
		c.levelEdges()
	}
	if c.line != nil && !c.firstOn {
		c.cutFirst(fonts)
	}
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
		case breakHere, edge, emergency, anchor:
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
			p.width = fonts.Advance(r.st.Font, strings.ReplaceAll(s, zeroWidthSpace, "")) + r.st.Spacing(s)
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
		if soft[at] && ch != ' ' && ch != '\n' && string(ch) != zeroWidthSpace && (b.Len() > 0 || at == 0 && c.afterWord()) {
			if b.Len() > 0 {
				emit(word)
				from = n - 1
			}
			c.pieces = append(c.pieces, piece{kind: breakHere, run: i, level: level})
		}
		switch {
		case ch == '\n' || ch == '\t' || ch == '\r':
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
		case string(ch) == zeroWidthSpace:
			emit(word)
			c.pieces = append(c.pieces, piece{kind: breakHere, run: i, level: level})
		default:
			b.WriteRune(ch)
			afterSpace = false
		}
	}
	emit(word)
	return afterSpace
}

// splitsWords reports whether st's text may break inside a word: by overflow-wrap,
// word-break or line-break.
func splitsWords(st *style.Computed) bool {
	return st.WrapsAnywhere() || st.WordBreak == style.BreakAll || st.LineBreak == style.LineBreakAnywhere
}

// control adds the newline, the tab or the carriage return ch of run i at character at,
// and reports whether a collapsible space after it is dropped: after a newline.
func (c *inlineContent) control(fonts box.Fonts, st *style.Computed, i int, at int32, ch rune,
	level bidi.Level) bool {
	p := piece{kind: tab, run: i, at: at, level: level}
	switch ch {
	case '\n':
		p = piece{kind: newline, run: i, level: level}
	case '\t':
		space := fonts.Advance(st.Font, " ")
		p.stop, p.least = st.TabStop(space), space/2
	}
	c.pieces = append(c.pieces, p)
	return ch == '\n'
}

// anchorsAt adds the anchors of the out-of-flow elements before run i, and notes the
// innermost open span each sits in.
func (c *inlineContent) anchorsAt(i int) {
	for k, a := range c.anchors {
		if a.run != i {
			continue
		}
		o := c.oofs[a.node]
		o.span = -1
		for si, sp := range c.spans {
			if sp.first <= i && i < sp.end || sp.first == i && sp.end == i {
				if o.span < 0 || c.spans[o.span].first < sp.first || c.spans[o.span].end > sp.end {
					o.span = si
				}
			}
		}
		c.pieces = append(c.pieces, piece{kind: anchor, run: min(i, max(len(c.runs)-1, 0)), span: k})
	}
}

// graphemes adds word s of run i, at bidi level level, as one piece per grapheme, each
// measured as the growth of the word's advance so that they add up to it. Between two, a line
// breaks where word-break: break-all or line-break: anywhere let it, else in an emergency where
// overflow-wrap does; two graphemes with neither stay one piece.
func (c *inlineContent) graphemes(fonts box.Fonts, st *style.Computed, i int, s string, from int32,
	level bidi.Level) {
	rest, state := s, -1
	var g, last string
	prev := float32(0)
	for at := 0; len(rest) > 0; {
		g, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		at += len(g)
		adv := fonts.Advance(st.Font, strings.ReplaceAll(s[:at], zeroWidthSpace, ""))
		w := adv - prev + st.Spacing(g)
		prev = adv
		switch {
		case at == len(g):
		case st.LineBreak == style.LineBreakAnywhere || st.WordBreak == style.BreakAll && breaksAll(last, g):
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
}

// breaksAll reports whether word-break: break-all breaks between the graphemes a and b:
// after a letter or a digit, before another or a hyphen, as Chrome breaks it.
func breaksAll(a, b string) bool {
	ra, _ := utf8.DecodeLastRuneInString(a)
	rb, _ := utf8.DecodeRuneInString(b)
	return letterLike(ra) && (letterLike(rb) || rb == '-' || rb == '\u2010')
}

func letterLike(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) }

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

// afterWord reports whether the last piece, edges aside, is a word.
func (c *inlineContent) afterWord() bool {
	for k := len(c.pieces) - 1; k >= 0; k-- {
		switch c.pieces[k].kind {
		case edge:
			continue
		case word:
			return true
		case gap, breakHere, newline, atomic, emergency, marker, anchor, tab:
		}
		return false
	}
	return false
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
// cut's own. As in Chrome, a path such as a/b doesn't break after its slash.
func (c *inlineContent) softBreaks() []map[int]bool {
	var all strings.Builder
	starts := make([]int, len(c.runs))
	for i, r := range c.runs {
		starts[i] = all.Len()
		if r.atomic != nil {
			all.WriteString("\uFFFC")
		} else {
			all.WriteString(r.text)
		}
	}
	soft := make([]map[int]bool, len(c.runs))
	text := all.String()
	rest, pos, state := text, 0, -1
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
		if r, _ := utf8.DecodeLastRuneInString(text[:pos]); r == '/' {
			if next, _ := utf8.DecodeRuneInString(rest); unicode.IsLetter(next) || unicode.IsDigit(next) {
				continue
			}
		}
		if c.runs[run].atomic != nil || c.runs[run].st.WordBreak == style.KeepAll && keeps(text[:pos], rest) {
			continue
		}
		if soft[run] == nil {
			soft[run] = map[int]bool{}
		}
		soft[run][pos-starts[run]] = true
	}
	return soft
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
		sp.margin = edges(sp.st.Margin, parent.w)
		sp.padding = edges(sp.st.Padding, parent.w)
		b := sp.st.Border
		sp.border = rect{l: b.Left, r: b.Right, t: b.Top, b: b.Bottom}
		left, right := sp.noStart, sp.noEnd
		if sp.st.Dir == style.RTL {
			left, right = right, left
		}
		if left {
			sp.margin.l, sp.border.l, sp.padding.l = 0, 0, 0
		}
		if right {
			sp.margin.r, sp.border.r, sp.padding.r = 0, 0, 0
		}
	}
	for i := range c.pieces {
		if p := &c.pieces[i]; p.kind == edge {
			start, end := startSides(c.spans[p.span])
			if p.start {
				c.resize(i, start)
			} else {
				c.resize(i, end)
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
		if k, sp := resolveSizingKeyword(st.Size.Width, inner.value(), parent.w); k == keywordMeasure {
			inner = sp
		}
		a.width = t.measureChildSize(a.node, noSize, parent,
			spaces{inner, maxContentSpace}, inherentSize, horizontal, [2]bool{})
		out := t.performChildLayout(a.node, size{a.width, none}, parent,
			spaces{def(a.width), maxContentSpace}, inherentSize)
		a.height = out.size.h
		a.baseline = out.size.h + a.margin.b
		a.overflow = out.overflow
		a.node.pendingOOF = out.oof
		switch {
		case isScrollContainer(st) && a.node.control != inputControl && a.node.control != selectControl:
		case some(out.lastBaseline):
			a.baseline = out.lastBaseline
		case some(out.firstBaseline):
			a.baseline = out.firstBaseline
		}
		c.resize(i, a.width+a.margin.horizontal())
	}
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
	t.sizeAtomics(c, maxContentSpace, parent)
	for _, l := range c.lines(unbounded) {
		maxW = max(maxW, l.x+l.width)
	}
	t.sizeAtomics(c, minContentSpace, parent)
	c.compress()
	if !c.anyWraps() {
		for _, l := range c.lines(unbounded) {
			minW = max(minW, l.x+l.width)
		}
		return minW, maxW
	}
	cur := newPen(c.unit, c.indent(unbounded))
	cur.origin = 0
	hung := false
	for i, p := range c.pieces {
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
			minW = max(minW, p.width)
			cur = newPen(c.unit, 0)
		case gap:
			switch {
			case !collapses(c.runs[p.run].st.WhiteSpace):
				cur = newPen(c.unit, 0)
			case c.edgeAt(i+1, false):
				// the span's end stays with the word before the space
				hung = true
			default:
				cur = newPen(c.unit, 0)
			}
		case breakHere, newline, anchor, tab:
			cur = newPen(c.unit, 0)
		}
	}
	return minW, maxW
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
// takes the pieces glued after it.
func (c *inlineContent) unitEnd(i int) int {
	end := i + 1
	if k := c.pieces[i].kind; k == word || k == edge || k == marker || k == anchor {
		for end < len(c.pieces) && glued(c.pieces[end].kind) {
			end++
		}
	}
	return end
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
	case word, atomic, newline, emergency, marker, tab:
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
// content and after its last, whatever span edges stand between, are breaks of no width.
func (c *inlineContent) shown(l line) []piece {
	pieces := c.pieces[l.from:l.to]
	out := pieces
	drop := func(i int) {
		if pieces[i].kind != gap {
			return
		}
		if &out[0] == &pieces[0] {
			out = slices.Clone(pieces)
		}
		out[i] = piece{kind: breakHere, run: pieces[i].run}
	}
	for i := 0; i < len(pieces) && c.passes(pieces[i]); i++ {
		drop(i)
	}
	for i := len(pieces) - 1; i >= 0 && c.passes(pieces[i]); i-- {
		drop(i)
	}
	return out
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
	case word, atomic, edge, marker, anchor:
		return true
	case gap, breakHere, newline, emergency, tab:
	}
	return false
}

// measureUnit is the pen past pieces i to end from at, with the gaps pending since then,
// and the pen past them alone.
func (c *inlineContent) measureUnit(at pen, pending, i, end int) (pen, pen) {
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
		next.step(c.pieces[k], 0)
		alone.step(c.pieces[k], 0)
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
}

func (b *breaker) finish(next int) {
	b.cur.width = b.kept.end()
	b.out = append(b.out, b.cur)
	b.open(next, 0, b.cur.avail, 0)
}

// open starts the breaker's next line at piece i, in room avail wide starting at x.
func (b *breaker) open(i int, x, avail, push float32) {
	b.cur = line{from: i, to: i, x: x, avail: avail, push: push, first: len(b.out) == 0 && b.c.hasFirst()}
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
			b.cur.forced = true
			b.finish(i + 1)
			return i + 1, true
		}
		if b.between(i) {
			i++
			continue
		}
		end, hung := c.unitEnd(i), false
		if b.pending >= 0 && (p.kind == edge || p.kind == anchor) {
			end, hung = c.hangEnd(b.pending, i, end)
		}
		next, alone := c.measureUnit(b.at, b.pending, i, end)
		if b.empty && alone.end() > width && c.wrapsAt(i) {
			if cut := c.fitEnd(i, end, width); cut < end {
				end = cut
				next, alone = c.measureUnit(b.at, b.pending, i, end)
			}
		}
		kept := next
		if hung {
			kept = b.kept
			for k := i; k < end; k++ {
				kept.step(c.pieces[k], 0)
			}
		}
		if !b.blank && !hung && b.wrapsBefore(i) && (next.end() > width || c.joined(next, end).end() > width) {
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
	case newline, word, atomic, edge, emergency, marker, anchor:
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
	trails := b.trails()
	if !b.empty && !trails || len(b.out) == 0 && b.cur.to > b.cur.from {
		b.cur.width = b.kept.end()
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
			c.useFirst(false)
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
			i = after
			y += lh
			bottom = y
			break
		}
	}
	return b.close()
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
	out := b.breakAll(width)
	if width >= unbounded {
		c.unbroken = out
	} else {
		c.broken, c.brokenAt = out, width
	}
	return out
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
	return k == word || k == edge || k == emergency || k == marker || k == anchor
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
	c.metrics[st] = m
	return m
}

// lineHeight is one run's line height and the baseline's place in it (half-leading above
// the ascent).
func (t *tree) lineHeight(c *inlineContent, st *style.Computed) (float32, float32) {
	return lineHeightOf(t.metricsOf(c, st), st)
}

// LineHeight is the line height a face with metrics m takes in a run of style st, and the
// baseline's place in it.
func LineHeight(m box.Metrics, st *style.Computed) (float32, float32) { return lineHeightOf(m, st) }

// lineHeightOf is the line height a face with metrics m takes in a run of style st, and
// the baseline's place in it.
func lineHeightOf(m box.Metrics, st *style.Computed) (float32, float32) {
	h := m.Ascent + m.Descent + m.LineGap
	switch {
	case st.LineHeight.Normal:
	case st.LineHeight.Factor != 0:
		h = st.LineHeight.Factor * st.Font.Size
	default:
		h = st.LineHeight.Px
	}
	// Chrome puts the floor of half the leading above the glyphs, the rest below.
	return h, m.Ascent + float32(math.Floor(float64(h-m.Ascent-m.Descent)/2))
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
		} else if m, ok := t.fallbackMetrics(p, r.st); ok {
			fh, fb := lineHeightOf(m, r.st)
			*into = into.with(extent{fb, fh - fb}, 0)
		}
	}
	return lo, hi
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
		return parent.Font.Size/5 + 1, true
	case style.AlignSuper:
		return -(parent.Font.Size/3 + 1), true
	case style.AlignLength:
		h, _ := t.lineHeight(c, st)
		return -st.VerticalShift.At(h), true
	case style.AlignMiddle:
		return (e.above-e.below)/2 - t.metricsOf(c, parent).XHeight/2, true
	case style.AlignTextTop, style.AlignTextBottom, style.AlignTop, style.AlignBottom:
		return 0, false
	}
	return 0, true
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
		case style.AlignBaselineText, style.AlignSub, style.AlignSuper, style.AlignMiddle, style.AlignLength:
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
			style.AlignTextTop, style.AlignTextBottom:
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
// fit them, and ignores them under any other line-height.
func (t *tree) fallbackMetrics(p piece, st *style.Computed) (box.Metrics, bool) {
	ff, ok := t.fonts.(box.FallbackFonts)
	if !ok || p.kind != word || !st.LineHeight.Normal || isASCII(p.text) {
		return box.Metrics{}, false
	}
	return ff.FallbackMetrics(st.Font, p.text)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// measure is the inline content's size and baselines, Taffy's measure function for text.
// Lines break at the available width; content wider than that (a word that breaks only
// with overflow-wrap, or not at all) widens the size it reports, not the lines.
func (t *tree) measure(c *inlineContent, known size, avail spaces, parent size) (size, float32, float32) {
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
	if c.flow != nil {
		c.broken, c.brokenAt = t.flowLines(c, lineW), lineW
	}
	var h float32
	first, last := none, none
	for i, l := range c.lines(lineW) {
		lh, lb := t.lineBox(c, l)
		h += l.push
		if i == 0 {
			first = h + lb
		}
		last = h + lb
		h += lh
	}
	if some(known.h) {
		h = known.h
	}
	return size{w, h}, first, last
}

// placeText lays the text out at the owner's content width and places its atomic inlines.
func (t *tree) placeText(n *node, s size, inset rect) {
	c := n.inline
	content := s.w - inset.horizontal()
	t.sizeAtomics(c, def(content), size{content, none})
	ls := c.lines(content)
	y := t.textTop(n, ls, inset.t, s.h-inset.vertical())
	var spanLines map[*box.Box][]geom.Rect
	if len(c.oofs) > 0 {
		spanLines = map[*box.Box][]geom.Rect{}
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
				map[*box.Box]geom.Rect{}, make([]bool, len(c.spans)), spanLines)
		}
		at := l.pen(c.unit, start)
		for _, p := range shown {
			x := at.step(p, perGap)
			if p.kind == anchor {
				c.oofs[c.anchors[p.span].node].static = point{x, y}
			}
			if p.kind == atomic {
				a := c.runs[p.run].atomic
				l := &a.node.unrounded
				l.location = point{x + a.margin.l, y + lb + c.runs[p.run].shift - a.baseline}
				l.margin = a.margin
				l.size = size{a.width, a.height}
				st := a.node.st
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
		c.containingSpan(o, spanLines)
	}
}

// containingSpan makes the nearest positioned span holding o its containing block: the
// box from the start of its first fragment's padding box to the end of its last one's,
// as CSS defines it for an inline element.
func (c *inlineContent) containingSpan(o *inlineOOF, lines map[*box.Box][]geom.Rect) {
	o.positioned = false
	for k := o.span; k >= 0; k = c.spans[k].up {
		sp := c.spans[k]
		if sp.st.Position == style.Static {
			continue
		}
		frags := lines[sp.box]
		if len(frags) == 0 {
			return
		}
		first, last := frags[0], frags[len(frags)-1]
		x0 := first.X + sp.border.l
		x1 := last.X + last.W - sp.border.r
		y0 := first.Y + sp.border.t
		y1 := last.Y + last.H - sp.border.b
		o.areaAt = point{x0, y0}
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
	if cs := n.columns; cs != nil {
		content = cs.width
	}
	t.sizeAtomics(c, def(content), size{content, none})
	runRects, spanRects, spanLines := t.writeLines(n, c, inset, content)
	origin := func(b *box.Box) geom.Vec2 {
		if r, ok := spanRects[b]; ok {
			return geom.Vec2{X: r.X, Y: r.Y}
		}
		return geom.Vec2{}
	}
	placeSpans(c, spanRects, spanLines, origin, size{content, none})
	t.attachInline(n, c, runRects, origin)
}

// placeSpans gives each span its box from the rectangles its lines took.
func placeSpans(c *inlineContent, spanRects map[*box.Box]geom.Rect, spanLines map[*box.Box][]geom.Rect,
	origin func(*box.Box) geom.Vec2, cb size) {
	for _, s := range c.spans {
		r := spanRects[s.box]
		o := origin(s.parent)
		s.box.Pos = geom.Vec2{X: r.X - o.X, Y: r.Y - o.Y}.Add(inlineOffset(s.st, cb))
		s.box.Size = geom.Vec2{X: r.W, Y: r.H}
		s.box.Margin = box.Edges{Top: s.margin.t, Right: s.margin.r, Bottom: s.margin.b, Left: s.margin.l}
		s.box.Border = box.Edges{Top: s.border.t, Right: s.border.r, Bottom: s.border.b, Left: s.border.l}
		s.box.Padding = box.Edges{Top: s.padding.t, Right: s.padding.r, Bottom: s.padding.b, Left: s.padding.l}
		s.box.Fragments = s.box.Fragments[:0]
		for _, f := range spanLines[s.box] {
			s.box.Fragments = append(s.box.Fragments, geom.Rect{X: f.X - r.X, Y: f.Y - r.Y, W: f.W, H: f.H})
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
	if c.marker != nil {
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
		for j := range r.box.Lines {
			r.box.Lines[j].Rect.X -= rr.X
			r.box.Lines[j].Rect.Y -= rr.Y
			r.box.Lines[j].Baseline -= rr.Y
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
// the runs and spans took.
func (t *tree) writeLines(n *node, c *inlineContent, inset rect, content float32) ([]geom.Rect,
	map[*box.Box]geom.Rect, map[*box.Box][]geom.Rect) {
	f := n.final
	ellipsis := n.st.Ellipsis && n.st.Overflow.Width != style.Visible
	runRects := make([]geom.Rect, len(c.runs))
	seen := make([]bool, len(c.runs))
	spanRects := make(map[*box.Box]geom.Rect)
	spanSeen := make([]bool, len(c.spans))
	spanLines := make(map[*box.Box][]geom.Rect)
	c.back = nil
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
		pieces, order := c.ordered(l)
		if c.firstOn {
			t.backOf(c, n, geom.Rect{X: x, Y: y, W: l.width, H: lh}, lb)
		}
		t.spanRects(c, l, pieces, order, x, perGap, y, lb, spanRects, spanSeen, spanLines)
		cut, cutStart := false, false
		if ellipsis && l.width > l.avail && len(c.runs) > 0 {
			pieces, x, cut, cutStart = t.ellipsizeLine(n, c, pieces, x, inset.l+l.x, l.avail)
		}
		t.writeRuns(c, pieces, geom.Rect{X: x, Y: y, H: lh}, lb, l.indent, perGap, [2]bool{cutStart, cut}, runRects,
			seen)
		c.useFirst(was)
		y = flowY + lh
	}
	return runRects, spanRects, spanLines
}

// writeRuns writes the fragments of a line's pieces into their runs' boxes, the line
// at at (its width aside) with its baseline lb, an ellipsis before or after it as cut
// says, and grows each run's rectangle in rects.
func (t *tree) writeRuns(c *inlineContent, pieces []piece, at geom.Rect, lb, indent, perGap float32, cut [2]bool,
	rects []geom.Rect, seen []bool) {
	y, lh := at.Y, at.H
	cutStart := cut[0]
	walk := line{indent: indent}.pen(c.unit, at.X)
	for i := 0; i < len(pieces); {
		run := pieces[i].run
		if k := pieces[i].kind; k == atomic || k == edge || k == marker || k == anchor || k == tab ||
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
		text, start, from, rtl, i = runText(&walk, pieces, i, perGap)
		end := walk.end()
		if i < len(pieces) && isText(pieces[i].kind) && pieces[i].run == run {
			end = walk.at()
		}
		r := geom.Rect{X: start, Y: y, W: max(end-start, 0), H: lh}
		frag := t.fragment(c.runs[run].st, text, r, y+lb+c.runs[run].shift, cutStart, cut[1] && i == len(pieces))
		frag.Start, frag.RTL, frag.Unspaced = from+c.runs[run].skip, rtl, unspaced
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
	frag := box.Fragment{Text: strings.ReplaceAll(text, zeroWidthSpace, ""), Rect: r, Baseline: baseline}
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
	c.marker.Pos = geom.Vec2{X: r.X, Y: r.Y}
	c.marker.Size = geom.Vec2{X: r.W, Y: r.H}
	c.marker.Lines = []box.Fragment{{Text: markerOf(c.owner.st), Rect: geom.Rect{W: r.W, H: r.H}, Baseline: lb}}
}

// ellipsizeLine cuts a line too wide for content and puts the ellipsis at its end, or at
// its start, then flush with the content's end.
func (t *tree) ellipsizeLine(n *node, c *inlineContent, pieces []piece, x, left, content float32) ([]piece, float32,
	bool, bool) {
	if !n.st.EllipsisStart {
		kept, cut := t.ellipsize(c, pieces, content)
		return kept, x, cut, false
	}
	pieces = t.ellipsizeStart(c, pieces, content)
	w := newPen(c.unit, 0)
	for _, p := range pieces {
		w.step(p, 0)
	}
	return pieces, left + content - w.end(), false, true
}

// spanRects adds line l's part of each span to its border box: from its start edge (or the
// line's start) to its end edge (or the line's end), over its content area plus vertical
// padding and border. The pieces are the line's before any ellipsis, as Chrome's boxes are.
func (t *tree) spanRects(c *inlineContent, l line, pieces []piece, order []int, x, perGap, y,
	lb float32, rects map[*box.Box]geom.Rect, seen []bool, lines map[*box.Box][]geom.Rect) {
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
		first, last := -1, -1
		for v := range pieces {
			if i := l.from + logical(order, v); i >= lo && i <= hi {
				if first < 0 {
					first = v
				}
				last = v
			}
		}
		left, right := at[first], at[last+1]
		if left == right && c.edgeAt(lo, true) && c.opensOnSpace(hi) {
			// a start of no width, the space after it dropped at the line's end, is no box
			continue
		}
		if p := pieces[first]; p.kind == edge && p.span == k && p.start {
			left += sp.margin.l
		}
		if p := pieces[last]; p.kind == edge && p.span == k && !p.start {
			right -= sp.margin.r
		}
		m := t.metricsOf(c, sp.st)
		top := y + lb + sp.shift - m.Ascent - sp.padding.t - sp.border.t
		bottom := y + lb + sp.shift + m.Descent + sp.padding.b + sp.border.b
		r := geom.Rect{X: left, Y: top, W: max(right-left, 0), H: bottom - top}
		if l.phantom {
			r = geom.Rect{X: left, Y: y}
		}
		rects[sp.box] = unionRect(rects[sp.box], r, seen[k])
		seen[k] = true
		lines[sp.box] = append(lines[sp.box], r)
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
// ellipsis a word of the first run kept. The line then ends at the box's end, as Chrome's
// right-to-left cut leaves its room before the ellipsis.
func (t *tree) ellipsizeStart(c *inlineContent, pieces []piece, width float32) []piece {
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
	for len(out) > 0 && out[0].kind == gap {
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
		return t.fonts.Advance(st.Font, s) + st.Spacing(s)
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
		return t.fonts.Advance(st.Font, s) + st.Spacing(s)
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
