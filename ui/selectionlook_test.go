package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
)

func TestAnInactiveWindowRestylesTheSelectionsWindowInactiveRules(t *testing.T) {
	p := ui.P(ui.Text("The quick brown fox"))
	plain := ui.P(ui.Text("jumps over"))
	l := selectableDoc(t, `p::selection { background-color: #00ff00 }
p:first-child::selection:window-inactive { background-color: #ff0000 }`, p, plain)
	look := func(n *ui.Node) color.Color { return dom.Data(n).Style.SelectionColor }
	assert.Equal(t, color.Hex("#00ff00"), look(p), "focused: the active rule")

	l.win.SetFocused(false)
	l.frames(1)
	assert.Equal(t, color.Hex("#ff0000"), look(p), "inactive: the :window-inactive rule")
	assert.Equal(t, color.Hex("#00ff00"), look(plain), "where none matches, the active one")

	l.win.SetFocused(true)
	l.frames(1)
	assert.Equal(t, color.Hex("#00ff00"), look(p))
}

func TestADoubleClickTakesAWordInlineElementsCut(t *testing.T) {
	p := ui.P(ui.Text("a wo"), ui.B(ui.Text("rd")), ui.Text(" sp"), ui.I(ui.Text("l"), ui.Span(ui.Text("it"))),
		ui.Text("s and"), ui.Br(), ui.Text("x"))
	l := selectableDoc(t, "", p)
	double := func(n *ui.Node, i int) string {
		l.doc.Select(ui.Position{Node: n, Offset: i}, ui.Position{Node: n, Offset: i + 1})
		r := l.doc.SelectionRects()[0]
		l.doc.ClearSelection()
		at := geom.V2(r.X+0.5, r.Y+r.H/2)
		l.frames(30)
		l.down(at)
		l.up()
		l.down(at)
		l.up()
		return l.doc.Selection().Text
	}
	kids := p.Children()
	assert.Equal(t, "word", double(kids[0], 3), "pressed on its first part")
	assert.Equal(t, "word", double(textOf(kids[1]), 1), "and on its last")
	assert.Equal(t, "splits", double(kids[2], 2), "through elements nested")
	assert.Equal(t, "splits", double(textOf(kids[3].Children()[1]), 1))
	assert.Equal(t, "and", double(kids[4], 3), "a br ends a word's stretch")
}

func TestAnEmptyLinesBreakIsMarkedASpaceWide(t *testing.T) {
	p := ui.P(ui.Text("a"), ui.Br(), ui.Br(), ui.Br(), ui.Text("b"))
	pre := ui.P(ui.Style("white-space: pre"), ui.Text("x\n\n\ny"))
	l := selectableDoc(t, "", p, pre)
	space := func() float32 {
		l.doc.Select(ui.Position{Node: textOf(pre), Offset: 0}, ui.Position{Node: textOf(pre), Offset: 1})
		w := l.doc.SelectionRects()[0].W
		l.doc.Select(ui.Position{Node: textOf(pre), Offset: 0}, ui.Position{Node: textOf(pre), Offset: 2})
		return l.doc.SelectionRects()[0].W - w
	}()
	require.Greater(t, space, float32(2))

	l.doc.Select(ui.Position{Node: textOf(p), Offset: 0}, ui.Position{Node: p.Children()[4], Offset: 1})
	rects := l.doc.SelectionRects()
	require.Len(t, rects, 4, "a, two empty lines, b")
	for i, r := range rects[1:3] {
		assert.InDelta(t, 0, r.X, 0.01, "line %d: from its start", i+1)
		assert.InDelta(t, space, r.W, 0.01, "line %d: a space wide", i+1)
		assert.InDelta(t, 20, r.H, 0.01, "line %d: as high as its line", i+1)
		assert.InDelta(t, rects[0].Y+float32(i+1)*20, r.Y, 0.01, "line %d: on its line", i+1)
	}

	l.doc.Select(ui.Position{Node: textOf(p), Offset: 0}, ui.Position{Node: p.Children()[4], Offset: 0})
	assert.Len(t, l.doc.SelectionRects(), 3, "up to b's start: the last break goes on to b, marked too")

	l.doc.Select(ui.Position{Node: textOf(pre), Offset: 0}, ui.Position{Node: textOf(pre), Offset: 5})
	rects = l.doc.SelectionRects()
	require.Len(t, rects, 4, "x, two empty lines, y")
	assert.InDelta(t, space, rects[1].W, 0.01)
	assert.InDelta(t, rects[0].Y+20, rects[1].Y, 0.01)
	assert.InDelta(t, rects[0].Y+40, rects[2].Y, 0.01)
	l.doc.Select(ui.Position{Node: textOf(pre), Offset: 0}, ui.Position{Node: textOf(pre), Offset: 3})
	assert.Len(t, l.doc.SelectionRects(), 2, "x and the first empty line, the second not reached")
}

func TestADragPastAScrollersSideScrollsItSideways(t *testing.T) {
	long := ui.P(ui.Text("a line far too long for the scroller it is in, so it scrolls sideways"))
	scroller := ui.Div(ui.Style("width: 120px; overflow: auto; white-space: nowrap"), long)
	l := selectableDoc(t, "", scroller)
	l.down(l.at(t, long, 2))
	for range 30 {
		l.moveTo(geom.V2(200, long.Box().Y+10))
	}
	assert.Less(t, long.Box().X, float32(-20), "held right of the scroller, it scrolls left")
	assert.Greater(t, len(l.doc.Selection().Text), len("line far"), "and the selection follows")
	l.up()
}
