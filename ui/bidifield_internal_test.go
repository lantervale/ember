package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
)

func TestTheCaretOfRightToLeftTextStandsWhereItIsShown(t *testing.T) {
	for _, c := range []struct {
		name string
		text string
		rtl  bool
		want []float32
	}{
		{"latin", "abc", false, []float32{0, 10, 20, 30}},
		{"hebrew", "אבג", true, []float32{30, 20, 10, 0}},
		{"hebrew in a left-to-right field", "אבג", false, []float32{30, 20, 10, 0}},
		{"latin in a right-to-left field", "abc", true, []float32{0, 10, 20, 30}},
		{"latin after hebrew", "אב c", true, []float32{40, 30, 20, 0, 10}},
	} {
		t.Run(c.name, func(t *testing.T) {
			edges, _ := visualLine([]rune(c.text), tenPoints(len([]rune(c.text))), c.rtl)
			assert.Equal(t, c.want, edges)
		})
	}
}

// tenPoints are the edges of n characters ten points wide each.
func tenPoints(n int) []float32 {
	xs := make([]float32, n+1)
	for i := range xs {
		xs[i] = float32(10 * i)
	}
	return xs
}

func TestASelectionOverBothDirectionsLiesInAStretchPerRun(t *testing.T) {
	text := []rune("ab אבג c")
	edges, cells := visualLine(text, tenPoints(len(text)), false)
	assert.Equal(t, [][2]float32{{10, 30}, {50, 60}}, ranges(edges, cells, 1, 4), "b and its space, then א at its run's right end")
	assert.Equal(t, [][2]float32{{30, 60}}, ranges(edges, cells, 3, 6), "the hebrew run whole")
	assert.Equal(t, [][2]float32{{10, 80}}, ranges(edges, cells, 1, 8), "all of it side by side: one stretch")
	assert.Nil(t, ranges(edges, cells, 2, 2))
	assert.Equal(t, [][2]float32{{10, 30}}, ranges(tenPoints(4), nil, 1, 3), "left to right throughout: one stretch")
}

// laidArea is field's layout once d laid it out focused.
func laidArea(t *testing.T, field *Node) (*Document, *area) {
	t.Helper()
	d := NewDocument()
	d.Body().Append(field)
	size := geom.V2(400, 400)
	d.Layout(size)
	d.Focus(field)
	d.Layout(size)
	d.layText()
	a, ok := textOf(d.machine(field)).Layout.(*area)
	require.True(t, ok)
	return d, a
}

func TestARightToLeftTextareasLinesStandWhereLayoutShowsThem(t *testing.T) {
	_, a := laidArea(t, Textarea(Attr("dir", "rtl"), Style("width: 200px"), Text("אבג abc\nדה")))
	require.Equal(t, 2, a.Lines())
	assert.InDelta(t, a.Row(0).W, a.X(0), 0.01, "the first letter at the line's right")
	assert.InDelta(t, 0, a.X(4), 0.01, "the latin word at its left")
	assert.Less(t, a.X(4), a.X(7), "left to right")
	assert.Equal(t, 1, a.Line(9))
	assert.Greater(t, a.X(8), a.X(10), "the second line from the right too")
}

func TestATabsRoomIsTheGapLayoutLeftForIt(t *testing.T) {
	field := Textarea(Style("width: 200px"), Text("a\tb"))
	d, a := laidArea(t, field)
	run := d.tree.Of[field].Children[0]
	require.Len(t, run.Lines, 2)
	assert.InDelta(t, run.Lines[1].Rect.X-a.Row(0).X, a.X(2), 0.01, "b where layout put it, past the tab's stop")
	assert.Equal(t, [][2]float32{{0, a.X(3)}}, a.Ranges(0, 0, 3), "the tab's room selected with its line")
}

func TestASelectionPastATextareasLineBreakTakesASpaceAtTheLineEnd(t *testing.T) {
	_, a := laidArea(t, Textarea(Style("width: 200px"), Text("ab\ncd")))
	assert.Equal(t, [][2]float32{{a.X(1), a.X(2) + a.space}}, a.Ranges(0, 1, 4))
	assert.Equal(t, [][2]float32{{a.X(1), a.X(2)}}, a.Ranges(0, 1, 2), "not past the break: none")

	_, a = laidArea(t, Textarea(Attr("dir", "rtl"), Style("width: 200px"), Text("אב\nגד")))
	assert.Equal(t, [][2]float32{{-a.space, a.X(1)}}, a.Ranges(0, 1, 4), "on its left, right to left")
}
