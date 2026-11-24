package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/ui"
)

const dragSheet = `textarea, input { font: 16px sans-serif; width: 200px; }`

// fieldAt is the middle of the line of field n where its character i
// begins.
func (l *live) fieldAt(t *testing.T, n *ui.Node, i int) geom.Vec2 {
	t.Helper()
	from, to := l.doc.SelectionRange(n)
	l.doc.SetSelectionRange(n, i, i+1)
	r := ui.FieldSelectionRects(l.doc, n)
	l.doc.SetSelectionRange(n, from, to)
	require.NotEmpty(t, r)
	return geom.V2(r[0].X+0.5, r[0].Y+r[0].H/2)
}

// selectQuick selects "quick", characters 4 to 9 of p's text.
func (l *live) selectQuick(p *ui.Node) {
	l.doc.Select(ui.Position{Node: textOf(p), Offset: 4}, ui.Position{Node: textOf(p), Offset: 9})
	l.frames(1)
}

// drag presses at from, holds still for hold frames, then moves through
// path a frame apiece.
func (l *live) dragFrom(from geom.Vec2, hold int, path ...geom.Vec2) {
	l.down(from)
	l.frames(hold)
	for _, p := range path {
		l.moveTo(p)
	}
}

func TestAPressInsideTheSelectionDragsItsTextIntoAField(t *testing.T) {
	ui.SetPlatform(t, textedit.Others)
	p := ui.P(ui.Text("The quick brown fox"))
	area := ui.Textarea(ui.Text("one two"))
	l := selectableDoc(t, dragSheet, p, area)
	l.selectQuick(p)

	at := l.at(t, p, 6)
	l.dragFrom(at, 0, at.Add(geom.V2(2, 2)))
	assert.Equal(t, "quick", l.doc.Selection().Text, "the press leaves the selection")
	assert.Nil(t, l.doc.TextDragImage(), "under three points on both axes it is no drag")

	drop := l.fieldAt(t, area, 4)
	l.moveTo(at.Add(geom.V2(3, 0)))
	require.NotNil(t, l.doc.TextDragImage(), "three points along one axis drag the text")
	assert.Equal(t, "quick", l.doc.TextDragImage().Text())
	l.moveTo(drop)
	ghost := l.doc.TextDragImage().Box()
	l.up()

	assert.Equal(t, "one quicktwo", l.doc.Value(area), "copied in where the pointer was")
	from, to := l.doc.SelectionRange(area)
	assert.Equal(t, [2]int{4, 9}, [2]int{from, to}, "and selected there")
	assert.True(t, l.doc.Focused(area))
	assert.True(t, l.doc.Selection().Empty(), "the page's selection went into the field")
	assert.Equal(t, "The quick brown fox", textOf(p).Text(), "the page keeps its text")
	assert.Nil(t, l.doc.TextDragImage())
	assert.Greater(t, ghost.W, float32(0))
}

func TestATextDragLetGoOutsideAFieldDoesNothing(t *testing.T) {
	ui.SetPlatform(t, textedit.Others)
	p := ui.P(ui.Text("The quick brown fox"))
	other := ui.P(ui.Text("jumps over the dog"))
	l := selectableDoc(t, dragSheet, p, other)
	l.selectQuick(p)
	l.dragFrom(l.at(t, p, 6), 0, l.at(t, p, 8), l.at(t, other, 3))
	l.up()
	assert.Equal(t, "quick", l.doc.Selection().Text)
	assert.Nil(t, l.doc.TextDragImage())
}

func TestAPressInsideTheSelectionLetGoPutsACaretThere(t *testing.T) {
	ui.SetPlatform(t, textedit.Others)
	p := ui.P(ui.Text("The quick brown fox"))
	l := selectableDoc(t, dragSheet, p)
	l.selectQuick(p)
	l.down(l.at(t, p, 6))
	assert.Equal(t, "quick", l.doc.Selection().Text)
	l.up()
	assert.True(t, l.doc.Selection().Empty(), "a release where it fell clears the selection")

	l.down(l.at(t, p, 6))
	assert.Equal(t, "quick", l.doc.Selection().Text, "the next press of the run takes the word")
	l.up()
}

func TestOnMacOSAQuickMoveInsideTheSelectionSelectsOrMovesTheCaret(t *testing.T) {
	ui.SetPlatform(t, textedit.Mac)
	p := ui.P(ui.Text("The quick brown fox"))
	l := selectableDoc(t, dragSheet, p)
	l.selectQuick(p)
	at := l.at(t, p, 6)
	l.dragFrom(at, 0, at.Add(geom.V2(1, 0)))
	assert.True(t, l.doc.Selection().Empty(), "a short move selects from the press")
	l.moveTo(l.at(t, p, 12))
	assert.Equal(t, "ick br", l.doc.Selection().Text, "and the pointer extends it")
	assert.Nil(t, l.doc.TextDragImage())
	l.up()

	l.frames(60)
	l.selectQuick(p)
	l.dragFrom(at, 0, l.at(t, p, 10), l.at(t, p, 12))
	s := l.doc.Selection()
	assert.True(t, s.Empty(), "a long one has the caret follow the pointer")
	assert.Equal(t, ui.Position{Node: textOf(p), Offset: 12}, s.Focus)
	l.frames(20)
	l.moveTo(l.at(t, p, 14))
	assert.Nil(t, l.doc.TextDragImage(), "even once the delay is past")
	assert.Equal(t, 14, l.doc.Selection().Focus.Offset)
	l.up()

	l.frames(60)
	l.selectQuick(p)
	l.dragFrom(at, 10, at.Add(geom.V2(1, 0)))
	l.up()
	assert.Equal(t, "quick", l.doc.Selection().Text, "held, then moved short of a drag, the selection stays")

	l.frames(60)
	l.dragFrom(at, 10, at.Add(geom.V2(1, 0)), at.Add(geom.V2(0, 3)))
	assert.NotNil(t, l.doc.TextDragImage(), "held still first, three points drag")
	l.up()
}

func TestAFieldsSelectionMovesIntoAnotherOrCopiesWithTheCopyKey(t *testing.T) {
	for _, c := range []struct {
		platform textedit.Platform
		key      input.Key
		left     string
	}{{textedit.Mac, input.KeyLeftAlt, "two"}, {textedit.Windows, input.KeyLeftControl, " two"}} {
		ui.SetPlatform(t, c.platform)
		area := ui.Textarea(ui.Text("one two"))
		field := ui.Input(ui.Attr("value", "field"))
		l := selectableDoc(t, dragSheet, area, field)
		l.doc.Focus(area)
		l.doc.SetSelectionRange(area, 0, 3)
		l.frames(1)
		l.dragFrom(l.fieldAt(t, area, 1), 10, l.fieldAt(t, area, 2), l.fieldAt(t, field, 2))
		l.up()
		assert.Equal(t, c.left, l.doc.Value(area), "moved out, with the space after it on macOS")
		assert.Equal(t, "fione", l.doc.Value(field)[:5])
		assert.True(t, l.doc.Focused(field))

		l.doc.Focus(area)
		l.doc.SetSelectionRange(area, len(c.left)-3, len(c.left))
		l.frames(60)
		l.win.Input().SetKey(c.key, true)
		l.dragFrom(l.fieldAt(t, area, len(c.left)-2), 10, l.fieldAt(t, area, len(c.left)-1), l.fieldAt(t, field, 0))
		l.up()
		l.win.Input().SetKey(c.key, false)
		l.frames(1)
		assert.Equal(t, c.left, l.doc.Value(area), "copied")
		assert.Equal(t, "twofioneeld", l.doc.Value(field))
	}
}

func TestAFieldsSelectionMovesWithinIt(t *testing.T) {
	ui.SetPlatform(t, textedit.Others)
	area := ui.Textarea(ui.Text("one two three"))
	l := selectableDoc(t, dragSheet, area)
	l.doc.Focus(area)
	l.doc.SetSelectionRange(area, 0, 4)
	l.frames(1)
	l.dragFrom(l.fieldAt(t, area, 1), 0, l.fieldAt(t, area, 3), l.fieldAt(t, area, 8))
	l.up()
	assert.Equal(t, "two one three", l.doc.Value(area))
	from, to := l.doc.SelectionRange(area)
	assert.Equal(t, [2]int{4, 8}, [2]int{from, to})

	l.frames(60)
	l.doc.SetSelectionRange(area, 4, 8)
	l.dragFrom(l.fieldAt(t, area, 5), 0, l.fieldAt(t, area, 7), l.fieldAt(t, area, 6))
	l.up()
	assert.Equal(t, "two one three", l.doc.Value(area), "dropped on itself, nothing moves")
}

func TestEscapeCancelsATextDrag(t *testing.T) {
	ui.SetPlatform(t, textedit.Others)
	p := ui.P(ui.Text("The quick brown fox"))
	area := ui.Textarea(ui.Text("one two"))
	l := selectableDoc(t, dragSheet, p, area)
	l.selectQuick(p)
	l.dragFrom(l.at(t, p, 6), 0, l.at(t, p, 8), l.fieldAt(t, area, 2))
	l.win.Input().SetKey(input.KeyEscape, true)
	l.frames(1)
	l.win.Input().SetKey(input.KeyEscape, false)
	assert.Nil(t, l.doc.TextDragImage())
	l.up()
	assert.Equal(t, "one two", l.doc.Value(area))
}
