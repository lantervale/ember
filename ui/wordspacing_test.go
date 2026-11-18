package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui"
)

func TestThePagesSelectionFollowsItsWidenedSpaces(t *testing.T) {
	plain, spaced := ui.P(ui.Text("ab cd\u00a0ef")), ui.P(ui.Class("wide"), ui.Text("ab cd\u00a0ef"))
	l := selectableDoc(t, ".wide { word-spacing: 10px; }", plain, spaced)
	for i, gaps := range []float32{0, 0, 0, 1, 1, 1, 2, 2} {
		assert.InDelta(t, l.at(t, plain, i).X+10*gaps, l.at(t, spaced, i).X, 0.01, "character %d", i)
	}
	width := func(n *ui.Node, from, to int) float32 {
		l.doc.Select(ui.Position{Node: textOf(n), Offset: from}, ui.Position{Node: textOf(n), Offset: to})
		rects := l.doc.SelectionRects()
		require.Len(t, rects, 1)
		return rects[0].W
	}
	assert.InDelta(t, width(plain, 2, 3)+10, width(spaced, 2, 3), 0.01, "a selected space, as wide as it is drawn")
	assert.InDelta(t, width(plain, 0, 8)+20, width(spaced, 0, 8), 0.01)
}

func TestADragOverWordSpacedTextSelectsTheWordsUnderIt(t *testing.T) {
	p := ui.P(ui.Text("ab cd ef"))
	l := selectableDoc(t, "p { word-spacing: 40px; }", p)
	l.dragged(l.at(t, p, 3), l.at(t, p, 5))
	assert.Equal(t, "cd", l.doc.Selection().Text)
}

func TestWordSpacingFarBelowZeroDrawsAndSelectsWithoutANegativeBox(t *testing.T) {
	p := ui.P(ui.Text("a b c"))
	field := ui.Input(ui.Attr("value", "a b c"))
	l := selectableDoc(t, "p, input { word-spacing: -100px; } p { width: max-content; }", p, field)
	assert.GreaterOrEqual(t, p.Box().W, float32(0))
	l.doc.Select(ui.Position{Node: textOf(p), Offset: 0}, ui.Position{Node: textOf(p), Offset: 5})
	l.frames(2)
	for _, r := range l.doc.SelectionRects() {
		assert.GreaterOrEqual(t, r.W, float32(0))
	}
	l.down(l.at(t, p, 2))
	l.up()
	l.doc.Focus(field)
	l.frames(2)
	assert.Empty(t, l.doc.Warnings())
}
