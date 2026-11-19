package ui

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui/control"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fieldOf is field's text machine once it is shown and focused.
func fieldOf(t *testing.T, field *Node) *control.Text {
	t.Helper()
	d, _, _ := typed(t, field)
	m := textOf(d.machine(field))
	require.NotNil(t, m)
	require.NotNil(t, m.Layout)
	return m
}

func TestAnInputsCaretStandsAfterItsWidenedSpaces(t *testing.T) {
	plain := fieldOf(t, Input(Attr("value", " ab cd\u00a0ef")))
	spaced := fieldOf(t, Input(Attr("value", " ab cd\u00a0ef"), Style("word-spacing: 10px")))
	for i, gaps := range []float32{0, 1, 1, 1, 2, 2, 2, 3, 3, 3} {
		assert.InDelta(t, plain.Layout.X(i)+10*gaps, spaced.Layout.X(i), 0.01, "position %d", i)
	}
}

func TestAnInputsCaretStandsAfterItsSpacedLetters(t *testing.T) {
	plain := fieldOf(t, Input(Attr("value", "ab cd")))
	spaced := fieldOf(t,
		Input(Attr("value", "ab cd"), Style("letter-spacing: 2px; word-spacing: 1em; font-size: 20px")))
	big := fieldOf(t, Input(Attr("value", "ab cd"), Style("font-size: 20px")))
	assert.Greater(t, big.Layout.X(5), plain.Layout.X(5))
	assert.InDelta(t, big.Layout.X(2)+4, spaced.Layout.X(2), 0.01)
	assert.InDelta(t, big.Layout.X(5)+10+20, spaced.Layout.X(5), 0.01, "five letters of 2 and a space of an em")
}

func TestAClickInAWordSpacedInputPlacesTheCaretWhereItsTextIsDrawn(t *testing.T) {
	field := Input(Attr("value", "ab cd ef"), Style("word-spacing: 30px"))
	d, w, in := typed(t, field)
	m := textOf(d.machine(field))
	b := d.tree.Of[field]
	c := b.ContentRect()
	at := func(i int) geom.Vec2 {
		return geom.V2(field.Box().X+c.X+m.Layout.X(i)+0.5, field.Box().Y+c.Y+c.H/2)
	}
	for _, i := range []int{4, 2, 7} {
		in.SetMouse(at(i))
		w.RunFrames(1)
		in.SetMouseButton(input.MouseLeft, true)
		w.RunFrames(1)
		in.SetMouseButton(input.MouseLeft, false)
		w.RunFrames(1)
		assert.Equal(t, i, m.Editor.Caret(), "a press where position %d is drawn", i)
	}
	run := b.Children[0]
	require.Len(t, run.Lines, 1)
	assert.InDelta(t, run.Lines[0].Rect.W, m.Layout.X(8), 0.01, "the caret after the last letter ends the drawn text")
}

func TestAnInputsSelectionCoversItsWidenedSpaces(t *testing.T) {
	field := Input(Attr("value", "ab cd"), Style("word-spacing: 30px"))
	m := fieldOf(t, field)
	plain := fieldOf(t, Input(Attr("value", "ab cd")))
	m.Editor.Select(2, 3)
	from, to := m.Editor.Selection()
	assert.InDelta(t, plain.Layout.X(3)-plain.Layout.X(2)+30, m.Layout.X(to)-m.Layout.X(from), 0.01)
}

func TestAnInputsLayoutFollowsItsWordSpacing(t *testing.T) {
	field := Input(Attr("value", "ab cd"))
	d, w, _ := typed(t, field)
	before := textOf(d.machine(field)).Layout.X(5)
	field.SetStyle("word-spacing: 12px")
	w.RunFrames(2)
	assert.InDelta(t, before+12, textOf(d.machine(field)).Layout.X(5), 0.01)
}

func TestATextareasCaretStandsAfterItsWidenedSpacesOnEachLine(t *testing.T) {
	value := "ab cd\n ef gh"
	plain := fieldOf(t, Textarea(Attr("cols", "40"), Text(value)))
	spaced := fieldOf(t, Textarea(Attr("cols", "40"), Style("word-spacing: 10px"), Text(value)))
	require.Equal(t, 2, spaced.Layout.Lines())
	for i, gaps := range []float32{0, 0, 0, 1, 1, 1, 0, 1, 1, 1, 2, 2, 2} {
		assert.InDelta(t, plain.Layout.X(i)+10*gaps, spaced.Layout.X(i), 0.01, "position %d", i)
	}
}

func TestAWrappedTextareasDroppedSpaceIsWidenedToo(t *testing.T) {
	value := "aaaa bbbb cccc dddd eeee ffff gggg hhhh"
	narrow := "width: 120px; font-size: 16px;"
	plain := fieldOf(t, Textarea(Style(narrow), Text(value)))
	spaced := fieldOf(t, Textarea(Style(narrow+"word-spacing: 4px"), Text(value)))
	require.Greater(t, plain.Layout.Lines(), 1)
	require.Equal(t, plain.Layout.Lines(), spaced.Layout.Lines(), "4 more a space breaks no other way here")
	_, end := spaced.Layout.Span(0)
	_, plainEnd := plain.Layout.Span(0)
	require.Equal(t, plainEnd, end)
	gaps := float32(0)
	for _, r := range value[:end] {
		if r == ' ' {
			gaps++
		}
	}
	assert.InDelta(t, plain.Layout.X(end)+4*gaps, spaced.Layout.X(end), 0.01)
}
