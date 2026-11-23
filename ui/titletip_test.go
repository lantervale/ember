package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
)

// rest puts the pointer at p and lets frames go by, a sixtieth of a second
// each.
func (l *live) rest(p geom.Vec2, frames int) {
	l.win.Input().SetMouse(p)
	l.frames(frames)
}

func titledRow() (*ui.Node, *ui.Node, *ui.Node) {
	link := ui.A(ui.Attr("href", "#"), ui.Attr("title", "https://example.test/a\nsecond line"), ui.Text("a"),
		ui.Style("display: inline-block; width: 80px; height: 30px"))
	quiet := ui.Span(ui.Attr("title", ""), ui.Text("q"), ui.Style("display: inline-block; width: 40px; height: 30px"))
	row := ui.Div(ui.Attr("title", "the row"), ui.Style("height: 30px"), link, quiet)
	return row, link, quiet
}

func TestATitleShowsAfterHalfASecondBelowThePointer(t *testing.T) {
	row, link, _ := titledRow()
	l := liveDoc(t, row)
	at := link.Box().Center()
	l.rest(at, 20)
	assert.Nil(t, l.doc.TitleTip(), "not before half a second")
	l.frames(15)
	tip := l.doc.TitleTip()
	require.NotNil(t, tip)
	assert.Equal(t, "https://example.test/a\nsecond line", tip.Text())
	box := tip.Box()
	assert.Equal(t, at.Add(geom.V2(10, 15)), geom.V2(box.X, box.Y), "10 right of the pointer and 15 below it")
	assert.Greater(t, box.H, float32(25), "its lines are kept")

	l.rest(at.Add(geom.V2(3, 0)), 2)
	assert.NotNil(t, l.doc.TitleTip(), "moving over the same element keeps it")
	l.frames(600)
	assert.Nil(t, l.doc.TitleTip(), "it hides after ten seconds")
}

func TestAnEmptyTitleHidesItsAncestorsAndAPressDismisses(t *testing.T) {
	row, link, quiet := titledRow()
	l := liveDoc(t, row)
	l.rest(quiet.Box().Center(), 40)
	assert.Nil(t, l.doc.TitleTip(), "an empty title shows none, not its row's")

	l.rest(link.Box().Center(), 40)
	require.NotNil(t, l.doc.TitleTip())
	l.win.Input().SetMouseButton(input.MouseLeft, true)
	l.frames(1)
	l.win.Input().SetMouseButton(input.MouseLeft, false)
	l.frames(40)
	assert.Nil(t, l.doc.TitleTip(), "a press hides it until another title")

	l.rest(geom.V2(row.Box().X+row.Box().W-10, row.Box().Y+5), 40)
	require.NotNil(t, l.doc.TitleTip())
	assert.Equal(t, "the row", l.doc.TitleTip().Text())
}

func TestATitleNearTheFootShowsAboveThePointerInsideTheWindow(t *testing.T) {
	foot := ui.Div(ui.Attr("title", "a rather long title that needs some room to show itself"),
		ui.Style("position: fixed; right: 0; bottom: 0; width: 40px; height: 20px"))
	l := liveDoc(t, foot)
	at := foot.Box().Center()
	l.rest(at, 40)
	tip := l.doc.TitleTip()
	require.NotNil(t, tip)
	box := tip.Box()
	assert.LessOrEqual(t, box.X+box.W, float32(640-4), "held inside the right edge")
	assert.InDelta(t, at.Y, box.Y+box.H, 0.5, "above the pointer")
}
