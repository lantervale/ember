package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/lantervale/ember/world"
)

const selectable = `body { margin: 0; user-select: text; font-size: 16px; line-height: 20px; } p { margin: 0 0 10px; }`

func selectableDoc(t *testing.T, css string, nodes ...*ui.Node) *live {
	t.Helper()
	return liveDocWith(t, []ui.DocumentOption{ui.CSS(selectable + css)}, nodes...)
}

func textOf(n *ui.Node) *ui.Node { return n.Children()[0] }

// at is the point in the middle of the line of n's text where its
// character i begins.
func (l *live) at(t *testing.T, n *ui.Node, i int) geom.Vec2 {
	t.Helper()
	was := l.doc.Selection()
	l.doc.Select(ui.Position{Node: textOf(n), Offset: i}, ui.Position{Node: textOf(n), Offset: i + 1})
	rects := l.doc.SelectionRects()
	l.doc.Select(was.Anchor, was.Focus)
	require.NotEmpty(t, rects)
	return geom.V2(rects[0].X+0.5, rects[0].Y+rects[0].H/2)
}

func (l *live) down(at geom.Vec2) {
	in := l.win.Input()
	in.SetMouse(at)
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, true)
	l.frames(1)
}

func (l *live) moveTo(at geom.Vec2) {
	l.win.Input().SetMouse(at)
	l.frames(1)
}

func (l *live) up() {
	l.win.Input().SetMouseButton(input.MouseLeft, false)
	l.frames(1)
}

func (l *live) dragged(from, to geom.Vec2) {
	l.down(from)
	l.moveTo(to)
	l.up()
}

func (l *live) held(mod, k input.Key) {
	in := l.win.Input()
	in.SetKey(mod, true)
	l.frames(1)
	in.SetKey(k, true)
	l.frames(1)
	in.SetKey(k, false)
	in.SetKey(mod, false)
	l.frames(1)
}

func TestADragSelectsTheTextBetweenItsEnds(t *testing.T) {
	first, second := ui.P(ui.Text("The quick brown fox")), ui.P(ui.Text("jumps over the dog"))
	l := selectableDoc(t, "", first, second)

	l.dragged(l.at(t, first, 4), l.at(t, first, 9))
	s := l.doc.Selection()
	assert.Equal(t, "quick", s.Text)
	assert.Equal(t, ui.Position{Node: textOf(first), Offset: 4}, s.Anchor)
	assert.Equal(t, ui.Position{Node: textOf(first), Offset: 9}, s.Focus)

	l.dragged(l.at(t, second, 5), l.at(t, first, 10))
	s = l.doc.Selection()
	assert.Equal(t, "brown fox\n\njumps", s.Text, "across two paragraphs, as Chrome's toString")
	assert.Equal(t, textOf(second), s.Anchor.Node, "the anchor stays where the press fell")
	assert.Equal(t, textOf(first), s.Focus.Node)
	assert.False(t, s.Empty())
}

func TestNothingIsSelectedOnAGameScreenByDefault(t *testing.T) {
	p := ui.P(ui.Text("The quick brown fox"))
	label := kit.Label("A kit label")
	l := liveDocWith(t, []ui.DocumentOption{kit.Theme()}, p, label)
	box := p.Box()
	l.dragged(geom.V2(box.X+2, box.Y+box.H/2), geom.V2(box.X+80, box.Y+box.H/2))
	assert.True(t, l.doc.Selection().Empty())
	box = label.Box()
	l.dragged(geom.V2(box.X+2, box.Y+box.H/2), geom.V2(box.X+40, box.Y+box.H/2))
	assert.True(t, l.doc.Selection().Empty(), "kit takes no selection either")
	assert.False(t, l.doc.SelectAll(), "nothing to select")
	l.held(input.KeyLeftSuper, input.KeyA)
	assert.True(t, l.doc.Selection().Empty())
}

func TestBrowserSheetsLetAPageBeSelected(t *testing.T) {
	p := ui.P(ui.Text("The quick brown fox"))
	l := liveDocWith(t, []ui.DocumentOption{ui.BrowserSheets()}, p)
	l.dragged(l.at(t, p, 0), l.at(t, p, 3))
	assert.Equal(t, "The", l.doc.Selection().Text)
}

func TestADoubleClickTakesTheWordAndATripleTheBlock(t *testing.T) {
	p := ui.P(ui.Text("The quick "), ui.Strong(ui.Text("brown")), ui.Text(" fox"))
	next := ui.P(ui.Text("jumps over"))
	l := selectableDoc(t, "", p, next)
	at := l.at(t, p, 6)

	l.down(at)
	l.up()
	assert.True(t, l.doc.Selection().Empty(), "a click selects nothing")
	l.down(at)
	assert.Equal(t, "quick", l.doc.Selection().Text)
	l.moveTo(l.at(t, next, 7))
	assert.Equal(t, "quick brown fox\n\njumps over", l.doc.Selection().Text, "the drag goes on by words")
	l.moveTo(l.at(t, p, 1))
	assert.Equal(t, "The quick", l.doc.Selection().Text, "backwards the first word stays")
	l.up()

	l.frames(30)
	for range 3 {
		l.down(at)
		l.up()
	}
	assert.Equal(t, "The quick brown fox", l.doc.Selection().Text, "the block, through its inline elements")
}

func TestShiftClickMovesTheFocusAndAClickElsewhereClears(t *testing.T) {
	p := ui.P(ui.Text("The quick brown fox"))
	l := selectableDoc(t, "", p)
	l.dragged(l.at(t, p, 4), l.at(t, p, 9))
	in := l.win.Input()
	in.SetKey(input.KeyLeftShift, true)
	l.down(l.at(t, p, 15))
	l.up()
	in.SetKey(input.KeyLeftShift, false)
	l.frames(1)
	assert.Equal(t, "quick brown", l.doc.Selection().Text)

	l.down(l.at(t, p, 1))
	l.up()
	assert.True(t, l.doc.Selection().Empty(), "a click in the text clears")
}

func TestAPressOnAButtonALinkOrUnselectableTextKeepsTheSelection(t *testing.T) {
	p := ui.P(ui.Text("The quick brown fox"))
	clicked := 0
	button := ui.Button(ui.Text("Press"), ui.OnClick(func() { clicked++ }))
	link := ui.A(ui.Attr("href", "#x"), ui.Text("a link"))
	none := ui.P(ui.Style("user-select: none"), ui.Text("not for selecting"))
	l := selectableDoc(t, "", p, button, ui.P(link), none)
	l.dragged(l.at(t, p, 4), l.at(t, p, 9))

	l.clickAt(button)
	assert.Equal(t, 1, clicked)
	assert.Equal(t, "quick", l.doc.Selection().Text, "a button keeps it, as Chrome")
	box := link.Box()
	l.dragged(geom.V2(box.X+2, box.Y+box.H/2), geom.V2(box.X+30, box.Y+box.H/2))
	assert.Equal(t, "quick", l.doc.Selection().Text, "a drag from a link selects nothing")
	box = none.Box()
	l.dragged(geom.V2(box.X+2, box.Y+box.H/2), geom.V2(box.X+60, box.Y+box.H/2))
	assert.Equal(t, "quick", l.doc.Selection().Text, "user-select: none neither starts nor clears")
}

func TestUserSelectNoneIsSkippedAndAllIsTakenWhole(t *testing.T) {
	p := ui.P(ui.Text("keep "), ui.Span(ui.Style("user-select: none"), ui.Text("skip")), ui.Text(" this"))
	all := ui.P(ui.Text("an "), ui.Code(ui.Style("user-select: all"), ui.Text("install --all")), ui.Text(" line"))
	l := selectableDoc(t, "", p, all)

	require.True(t, l.doc.SelectContents(p))
	assert.Equal(t, "keep  this", l.doc.Selection().Text)
	assert.Len(t, l.doc.SelectionRects(), 2, "the skipped text is not drawn selected")

	code := all.Children()[1]
	l.down(l.at(t, code, 3))
	assert.Equal(t, "install --all", l.doc.Selection().Text, "a press takes it whole")
	l.up()
	l.dragged(l.at(t, all, 1), l.at(t, code, 2))
	assert.Equal(t, "n install --all", l.doc.Selection().Text, "a drag into it takes all of it")
}

func TestTheSelectionCopiesAsChromeWritesIt(t *testing.T) {
	l := selectableDoc(t, `.up { text-transform: uppercase; } .gone { display: none; } .ghost { visibility: hidden; }
		li::before { content: "> "; }`,
		ui.H2(ui.Text("Title")),
		ui.P(ui.Text("  one\n   two  "), ui.Br(), ui.Text("three")),
		ui.Div(ui.Text("a "), ui.Span(ui.Class("up"), ui.Text("shout")), ui.Span(ui.Class("gone"), ui.Text("gone")),
			ui.Span(ui.Class("ghost"), ui.Text("ghost")), ui.Text(" b")),
		ui.Ul(ui.Li(ui.Text("first")), ui.Li(ui.Text("second"))),
		ui.Table(ui.Tbody(
			ui.Tr(ui.Td(ui.Text("a1")), ui.Td(ui.Text("b1"))), ui.Tr(ui.Td(ui.Text("a2")), ui.Td(ui.Text("b2"))))),
		ui.Div(ui.Text("x "), ui.Input(ui.Attr("value", "typed")), ui.Text(" y "), ui.Button(ui.Text("btn")), ui.Text(" z")),
		ui.Pre(ui.Text("pre  two\n  lines")),
	)
	require.True(t, l.doc.SelectAll())
	assert.Equal(t, "Title\none two\nthree\n\na SHOUT b\nfirst\nsecond\na1\tb1\na2\tb2\nx  y btn z\npre  two\n  lines",
		l.doc.Selection().Text)
}

func TestSelectAllAndCopyAreTheKeysWhenNoFieldHasTheFocus(t *testing.T) {
	ui.SetPlatform(t, textedit.Mac)
	p := ui.P(ui.Text("The quick\u00a0brown fox"))
	field := ui.Input(ui.Attr("value", "typed here"))
	l := selectableDoc(t, "", p, field)

	l.held(input.KeyLeftSuper, input.KeyA)
	assert.Equal(t, "The quick\u00a0brown fox", l.doc.Selection().Text)
	l.held(input.KeyLeftSuper, input.KeyC)
	assert.Equal(t, "The quick brown fox", l.win.Clipboard(), "a no-break space is a plain one on the clipboard")

	l.key(input.KeyEscape)
	assert.False(t, l.doc.Selection().Empty(), "Escape keeps the selection, as Chrome")

	l.clickAt(field)
	assert.True(t, l.doc.Selection().Empty(), "a field taking the focus takes the selection")
	l.held(input.KeyLeftSuper, input.KeyA)
	assert.True(t, l.doc.Selection().Empty(), "select all is the field's")
	l.held(input.KeyLeftSuper, input.KeyC)
	assert.Equal(t, "typed here", l.win.Clipboard(), "and so is the copy")
}

func TestShiftAndTheMovingKeysMoveTheFocus(t *testing.T) {
	ui.SetPlatform(t, textedit.Mac)
	p := ui.P(ui.Style("width: 150px"), ui.Text("The quick brown fox jumps over the lazy dog"))
	l := selectableDoc(t, "", p)
	text := textOf(p)
	l.doc.Select(ui.Position{Node: text, Offset: 4}, ui.Position{Node: text, Offset: 5})

	l.held(input.KeyLeftShift, input.KeyRight)
	assert.Equal(t, "qu", l.doc.Selection().Text)
	l.held(input.KeyLeftShift, input.KeyLeft)
	assert.Equal(t, "q", l.doc.Selection().Text)
	l.held(input.KeyLeftShift, input.KeyEnd)
	end := l.doc.Selection()
	assert.Equal(t, 4, end.Anchor.Offset)
	assert.Contains(t, end.Text, "quick")
	l.held(input.KeyLeftShift, input.KeyDown)
	assert.Greater(t, len(l.doc.Selection().Text), len(end.Text), "a line down")
	l.held(input.KeyLeftShift, input.KeyHome)
	l.held(input.KeyLeftShift, input.KeyUp)
	l.held(input.KeyLeftShift, input.KeyUp)
	assert.Equal(t, "The ", l.doc.Selection().Text, "above the first line is the text's start")

	l.doc.ClearSelection()
	l.held(input.KeyLeftShift, input.KeyRight)
	assert.True(t, l.doc.Selection().Empty(), "nothing selected, nothing to move")
}

func TestTheSelectionSaysWhenItChanges(t *testing.T) {
	p := ui.P(ui.Text("The quick brown fox"))
	l := selectableDoc(t, "", p)
	var heard []string
	ui.OnSelectionChange(func() { heard = append(heard, l.doc.Selection().Text) }).Apply(l.doc.Body())
	var r world.Reader[ui.SelectionChanged]
	count := func() int {
		n := 0
		for range 2 {
			l.w.RunFrames(1)
			for e := range r.Read(l.w) {
				assert.Same(t, l.doc.Body(), e.Node)
				n++
			}
		}
		return n
	}
	text := textOf(p)

	l.doc.Select(ui.Position{Node: text, Offset: 4}, ui.Position{Node: text, Offset: 9})
	assert.Equal(t, 1, count())
	l.doc.Select(ui.Position{Node: text, Offset: 4}, ui.Position{Node: text, Offset: 9})
	assert.Zero(t, count(), "the same selection again is no change")
	l.doc.ClearSelection()
	assert.Equal(t, 1, count())
	assert.Equal(t, []string{"quick", ""}, heard)
}

func TestTheSelectionOutlivesARestyleAndGoesWithItsNodes(t *testing.T) {
	first, second := ui.P(ui.Text("The quick brown fox")), ui.P(ui.Text("jumps over the dog"))
	l := selectableDoc(t, ".wide { letter-spacing: 2px; width: 120px; }", first, second)
	l.doc.Select(ui.Position{Node: textOf(first), Offset: 4}, ui.Position{Node: textOf(second), Offset: 5})
	l.frames(1)
	before := l.doc.SelectionRects()

	first.AddClass("wide")
	l.frames(2)
	assert.Equal(t, "quick brown fox\n\njumps", l.doc.Selection().Text, "the same text, laid out anew")
	assert.NotEqual(t, before, l.doc.SelectionRects())

	l.doc.Body().Remove(second)
	l.frames(2)
	assert.True(t, l.doc.Selection().Empty(), "an end gone, the selection goes")
	assert.Empty(t, l.doc.SelectionRects())
}

func TestTheTextCursorShowsOverSelectableText(t *testing.T) {
	p := ui.P(ui.Text("The quick brown fox"))
	styled := ui.P(ui.Style("cursor: pointer"), ui.Text("an author's cursor"))
	none := ui.P(ui.Style("user-select: none"), ui.Text("not for selecting"))
	l := selectableDoc(t, "", p, styled, none)
	cursor := func(n *ui.Node) platform.Cursor {
		l.moveTo(l.at(t, n, 2))
		l.frames(1)
		return l.win.Cursor()
	}
	assert.Equal(t, platform.CursorText, cursor(p))
	assert.Equal(t, platform.CursorHand, cursor(styled), "the author's cursor wins")
	box := none.Box()
	l.moveTo(geom.V2(box.X+10, box.Y+box.H/2))
	l.frames(1)
	assert.Equal(t, platform.CursorArrow, l.win.Cursor())
	l.moveTo(geom.V2(500, p.Box().Y+5))
	l.frames(1)
	assert.Equal(t, platform.CursorArrow, l.win.Cursor(), "beside the text, the arrow")
}

func TestADragPastAScrollersEdgeScrollsIt(t *testing.T) {
	var lines []ui.Arg
	for range 30 {
		lines = append(lines, ui.P(ui.Text("a line of text")))
	}
	scroller := ui.Div(append(lines, ui.Style("height: 100px; overflow: auto"))...)
	l := selectableDoc(t, "", scroller)
	first := scroller.Children()[0]
	l.down(l.at(t, first, 2))
	for range 30 {
		l.moveTo(geom.V2(40, 160))
	}
	scrolled := func() float32 { return scroller.Box().Y - first.Box().Y }
	scroll := scrolled()
	assert.Greater(t, scroll, float32(20), "held below the scroller, it scrolls")
	assert.Greater(t, len(l.doc.Selection().Text), len("a line of text\n\na line"), "and the selection follows")
	l.up()
	l.moveTo(geom.V2(40, 160))
	l.frames(3)
	assert.InDelta(t, scroll, scrolled(), 0.01, "the button up, it stops")
}

func TestAutomationSelectsALabelsText(t *testing.T) {
	note := ui.P(ui.ID("note"), ui.Text("The quick brown fox"))
	fixed := ui.P(ui.ID("fixed"), ui.Style("user-select: none"), ui.Text("not for selecting"))
	l := selectableDoc(t, "", note, fixed)
	acts := map[string][]string{}
	_, all := l.auto.Snapshot("", "")
	for _, e := range all {
		acts[e.ID] = e.Acts
	}
	assert.Equal(t, []string{"select-all"}, acts["note"])
	assert.Empty(t, acts["fixed"], "what takes no selection offers none")

	l.auto.Queue("note", "select-all", nil)
	l.frames(2)
	assert.Equal(t, "The quick brown fox", l.doc.Selection().Text)
}

func TestAPressFindsTheTextOfATurnedBoxAndOfAColumn(t *testing.T) {
	turned := ui.P(ui.Style("width: 200px; transform: translate(100px, 50px)"), ui.Text("The quick brown fox"))
	l := selectableDoc(t, "", turned)
	l.doc.Select(ui.Position{Node: textOf(turned), Offset: 4}, ui.Position{Node: textOf(turned), Offset: 9})
	r := l.doc.SelectionRects()[0]
	l.doc.ClearSelection()
	l.dragged(geom.V2(r.X+100.5, r.Y+60), geom.V2(r.X+r.W+100.5, r.Y+60))
	assert.Equal(t, "quick", l.doc.Selection().Text, "where the transform shows it")

	cols := ui.Div(ui.Style("width: 400px; column-count: 2; column-gap: 40px; white-space: pre"),
		ui.P(ui.Text("one\ntwo\nthree\nfour")), ui.P(ui.Text("five\nsix")))
	l = selectableDoc(t, "p { margin: 0 }", cols)
	l.dragged(geom.V2(2, 50), geom.V2(260, 30))
	assert.Equal(t, "three\nfour\n\nfive\nsix", l.doc.Selection().Text, "from the first column into the second")
}

func TestTypingInAFieldLeavesThePageUnselected(t *testing.T) {
	p := ui.P(ui.Text("The quick brown fox"))
	field := ui.Input(ui.Attr("value", "typed here"))
	l := selectableDoc(t, "", p, field)
	require.True(t, l.doc.SelectContents(p))
	l.doc.Focus(field)
	l.frames(2)
	assert.True(t, l.doc.Selection().Empty())
	require.True(t, l.doc.SelectContents(p))
	l.frames(1)
	assert.Equal(t, "The quick brown fox", l.doc.Selection().Text, "code may still select beside a focused field")
}

func TestADragPastTheWindowsEdgeScrollsAScrollingViewport(t *testing.T) {
	var lines []*ui.Node
	for range 40 {
		lines = append(lines, ui.P(ui.Text("a line of text")))
	}
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(selectable), ui.ScrollingViewport()}, lines...)
	l.down(l.at(t, lines[0], 2))
	for range 30 {
		l.moveTo(geom.V2(40, 400))
	}
	assert.Less(t, lines[0].Box().Y, float32(-20), "held below the window, the page scrolls")
	assert.Greater(t, len(l.doc.Selection().Text), len("a line of text\n\na line"))
	l.up()
}

func TestSelectAllWorksWithAButtonFocused(t *testing.T) {
	ui.SetPlatform(t, textedit.Mac)
	p := ui.P(ui.Text("The quick brown fox"))
	button := ui.Button(ui.Text("Press"))
	l := selectableDoc(t, "", p, button)
	l.clickAt(button)
	l.held(input.KeyLeftSuper, input.KeyA)
	assert.Equal(t, "The quick brown fox\n\nPress", l.doc.Selection().Text, "as Chrome: the focus is in no field")
}
