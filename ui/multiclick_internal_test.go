package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/world"
)

// clicker clicks in a text field, at positions along its text.
type clicker struct {
	t     *testing.T
	d     *Document
	w     *world.World
	in    *input.State
	field *Node
	m     *control.Text
}

func clicking(t *testing.T, field *Node) clicker {
	t.Helper()
	d, w, in := typed(t, field)
	return clicker{t, d, w, in, field, textOf(d.machine(field))}
}

// at is the point nearest position i's gap, off to its side by dx points, on the
// field's line.
func (c clicker) at(i int, dx float32, line int) geom.Vec2 {
	c.t.Helper()
	b := c.d.tree.Of[c.field]
	r, content := c.field.Box(), b.ContentRect()
	if a, ok := c.m.Layout.(*area); ok {
		row := a.Row(line)
		return geom.V2(r.X+content.X+row.X+c.m.Layout.X(i)+dx, r.Y+content.Y+row.Y+row.H/2)
	}
	return geom.V2(r.X+content.X+c.m.Layout.X(i)+dx, r.Y+r.H/2)
}

func (c clicker) press(p geom.Vec2) {
	c.in.SetMouse(p)
	c.w.RunFrames(1)
	c.in.SetMouseButton(input.MouseLeft, true)
	c.w.RunFrames(1)
}

func (c clicker) release() {
	c.in.SetMouseButton(input.MouseLeft, false)
	c.w.RunFrames(1)
}

func (c clicker) click(p geom.Vec2, times int) {
	for range times {
		c.press(p)
		c.release()
	}
}

func TestADoubleClickSelectsTheWordAtTheCaretAsChromeDoes(t *testing.T) {
	// What Chrome on macOS selects for each of these, read from selectionStart and
	// selectionEnd after a real double click.
	const text = "hello brave   world, ok"
	cases := map[string]struct {
		i    int
		dx   float32
		want string
	}{
		"inside a word":                              {8, 1, "brave"},
		"the left half of a word's last letter":      {4, 1, "hello"},
		"the right half of it, nearer the gap after": {5, -1, " "},
		"a run of spaces":                            {12, 1, "   "},
		"a mark":                                     {19, 1, ","},
		"past the end of the text":                   {23, 40, "ok"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := clicking(t, Input(Attr("value", text), Style("width: 300px")))
			c.click(c.at(tc.i, tc.dx, 0), 2)
			assert.Equal(t, tc.want, c.m.Editor.Selected())
		})
	}
}

func TestATripleClickSelectsAnInputsTextAndATextareasLineWithItsBreak(t *testing.T) {
	c := clicking(t, Input(Attr("value", "hello brave world"), Style("width: 300px")))
	c.click(c.at(8, 1, 0), 3)
	assert.Equal(t, "hello brave world", c.m.Editor.Selected())
	c.click(c.at(8, 1, 0), 1)
	assert.Equal(t, "hello brave world", c.m.Editor.Selected(), "a fourth click goes on as the third")

	c = clicking(t, Textarea(Attr("rows", "4"), Style("width: 300px"), Text("first line\nsecond one here\nthird")))
	c.click(c.at(19, 1, 1), 2)
	assert.Equal(t, "one", c.m.Editor.Selected())
	c.click(c.at(19, 1, 1), 1)
	assert.Equal(t, "second one here\n", c.m.Editor.Selected(), "the third click of the run")
	c.w.RunFrames(40)
	c.click(c.at(30, 1, 2), 3)
	assert.Equal(t, "third", c.m.Editor.Selected(), "the last line has no break to take")
}

func TestADragAfterADoubleClickSelectsWordByWord(t *testing.T) {
	c := clicking(t, Input(Attr("value", "hello brave world, ok"), Style("width: 300px")))
	c.click(c.at(8, 1, 0), 1)
	c.press(c.at(8, 1, 0))
	require.Equal(t, "brave", c.m.Editor.Selected(), "selected on the press, before the release")
	c.in.SetMouse(c.at(14, 1, 0))
	c.w.RunFrames(1)
	assert.Equal(t, "brave world", c.m.Editor.Selected(), "the word under the pointer joins whole")
	assert.Equal(t, 17, c.m.Editor.Caret(), "the caret at the dragged end")
	c.in.SetMouse(c.at(2, 1, 0))
	c.w.RunFrames(1)
	assert.Equal(t, "hello brave", c.m.Editor.Selected(), "back past the first word, it stays selected")
	assert.Equal(t, 0, c.m.Editor.Caret())
	c.release()
	assert.Equal(t, "hello brave", c.m.Editor.Selected())
}

func TestClicksApartInTimeOrPlaceAreSingleClicks(t *testing.T) {
	c := clicking(t, Input(Attr("value", "hello brave world"), Style("width: 300px")))
	c.click(c.at(8, 1, 0), 1)
	c.w.RunFrames(40)
	c.click(c.at(8, 1, 0), 1)
	assert.Empty(t, c.m.Editor.Selected(), "more than a double click's time apart")
	c.w.RunFrames(40)
	c.click(c.at(2, 1, 0), 1)
	c.click(c.at(14, 1, 0), 1)
	assert.Empty(t, c.m.Editor.Selected(), "the second click is on another word")
	assert.Equal(t, 14, c.m.Editor.Caret())
}

func TestADoubleClickSelectsAWholePassword(t *testing.T) {
	c := clicking(t, Input(Type("password"), Attr("value", "open sesame"), Style("width: 300px")))
	c.click(c.at(2, 1, 0), 2)
	assert.Equal(t, "open sesame", c.m.Editor.Selected(), "its words are not to be found by clicking")
}
