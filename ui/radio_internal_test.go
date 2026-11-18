package ui

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlacingTheRadiosWalksTheTreeForNoneOfThem(t *testing.T) {
	radio := func(name string, args ...Arg) *Node {
		return Input(append([]Arg{Type("radio"), Attr("name", name)}, args...)...)
	}
	before := Button(Text("Before"))
	a, b, c := radio("size"), radio("size", Attr("checked", "")), radio("size")
	round, square := radio("shape"), radio("shape")
	d := NewDocument()
	d.Body().Append(before, a, b, c, round, square)
	for range 200 {
		d.Body().Append(P(Text("a paragraph")))
	}
	win := platform.NewWindow(platform.Size(300, 200))
	win.Add(d)
	w := ember.New().AddEntities(win)
	w.RunFrames(4)

	walks := d.grouped
	for i := range 10 {
		win.Input().SetMouse(geom.V2(float32(10+i*20), 100))
		w.RunFrames(1)
	}
	require.Equal(t, walks, d.grouped, "a frame that places the radios again reads their groups from its last walk")

	d.Focus(before)
	w.RunFrames(1)
	tab := func() {
		win.Input().SetKey(input.KeyTab, true)
		w.RunFrames(1)
		win.Input().SetKey(input.KeyTab, false)
		w.RunFrames(1)
	}
	tab()
	assert.True(t, d.Focused(b), "Tab reaches the first group at its checked radio")
	tab()
	assert.True(t, d.Focused(round), "and the next at its first, with none checked")

	late := radio("shape", Attr("checked", ""))
	d.Body().Append(late)
	d.Focus(before)
	w.RunFrames(2)
	tab()
	tab()
	assert.True(t, d.Focused(late), "a radio added later is in its group")
}
