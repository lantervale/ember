package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
)

// touch puts finger 1 on the screen at at for frames frames, moving it
// through path a frame apiece, and lifts it.
func (l *live) touch(at geom.Vec2, frames int, path ...geom.Vec2) {
	in := l.win.Input()
	in.AddTouch(1, input.Touchscreen, at)
	l.frames(frames)
	for _, p := range path {
		in.MoveTouch(1, p)
		l.frames(1)
	}
	in.EndTouch(1)
	l.frames(2)
}

func TestALongPressSelectsTheWordAndItsHandlesExtendIt(t *testing.T) {
	p := ui.P(ui.Text("The quick brown fox"))
	l := selectableDoc(t, dragSheet, p)

	l.touch(l.at(t, p, 6), 20)
	assert.True(t, l.doc.Selection().Empty(), "a tap selects nothing")
	l.touch(l.at(t, p, 6), 3, l.at(t, p, 6).Add(geom.V2(20, 0)))
	assert.True(t, l.doc.Selection().Empty(), "nor a finger that moved")

	l.touch(l.at(t, p, 6), 40)
	assert.Equal(t, "quick", l.doc.Selection().Text, "half a second's press takes the word")
	h := l.doc.SelectionHandles()
	require.NotNil(t, h[0])
	require.NotNil(t, h[1])
	start, end := h[0].Box(), h[1].Box()
	word := l.doc.SelectionRects()[0]
	assert.InDelta(t, word.X, start.X+start.W, 0.5, "the start's handle points at the word's start")
	assert.InDelta(t, word.X+word.W, end.X, 0.5)
	assert.InDelta(t, word.Y+word.H, end.Y, 0.5, "below the line")

	grab := geom.V2(end.X+end.W/2, end.Y+end.H/2)
	l.touch(grab, 1, grab.Add(geom.V2(l.at(t, p, 15).X-end.X, 0)))
	assert.Equal(t, "quick brown", l.doc.Selection().Text, "the end's handle takes the end along")
	assert.NotNil(t, l.doc.SelectionHandles()[1])

	start = l.doc.SelectionHandles()[0].Box()
	grab = geom.V2(start.X+start.W/2, start.Y+start.H/2)
	l.touch(grab, 1, grab.Add(geom.V2(l.at(t, p, 0).X-(start.X+start.W), 0)))
	assert.Equal(t, "The quick brown", l.doc.Selection().Text)

	l.touch(geom.V2(5, 300), 2)
	assert.True(t, l.doc.Selection().Empty(), "a tap clears it")
	assert.Nil(t, l.doc.SelectionHandles()[0])
}

func TestALongPressInAFieldSelectsItsWord(t *testing.T) {
	area := ui.Textarea(ui.Text("one two three"))
	l := selectableDoc(t, dragSheet, area)
	l.touch(l.fieldAt(t, area, 5), 40)
	assert.True(t, l.doc.Focused(area))
	from, to := l.doc.SelectionRange(area)
	assert.Equal(t, [2]int{4, 7}, [2]int{from, to})
	assert.Nil(t, l.doc.SelectionHandles()[0], "a field shows no handles")
}

func TestTheMouseHidesTheHandles(t *testing.T) {
	p := ui.P(ui.Text("The quick brown fox"))
	l := selectableDoc(t, dragSheet, p)
	l.touch(l.at(t, p, 6), 40)
	require.NotNil(t, l.doc.SelectionHandles()[0])
	l.dragged(l.at(t, p, 0), l.at(t, p, 3))
	assert.Equal(t, "The", l.doc.Selection().Text)
	assert.Nil(t, l.doc.SelectionHandles()[0])
}
