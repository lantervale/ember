package ui_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
)

func (l *live) pressAt(at geom.Vec2) {
	in := l.win.Input()
	in.SetMouse(at)
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, true)
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, false)
	l.frames(2)
}

func TestABoxCutByColumnsIsOneBoxClickedByItsParts(t *testing.T) {
	lines := func(n int) ui.Arg { return ui.Text(strings.TrimSpace(strings.Repeat("line\n", n))) }
	var got string
	clicked := func(name string) ui.Arg { return ui.OnClick(func() { got += name }) }
	tab := ui.Attr("tabindex", "0")
	first, second := ui.P(tab, lines(6), clicked("first")), ui.P(tab, lines(2), clicked("second"))
	cols := ui.Div(ui.Style("width: 400px; column-count: 2; column-gap: 40px"), tab, first, second, clicked("cols"))
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(
		`body { margin: 0; } p { margin: 0; white-space: pre; font-size: 10px; line-height: 20px; }`)}, cols)

	at := func(x, y float32) string {
		got = ""
		l.pressAt(geom.V2(x, y))
		return got
	}

	r, ok := l.doc.BoxOf(first)
	assert.True(t, ok)
	assert.Equal(t, geom.Rect{W: 400, H: 80}, r, "BoxOf is the box round its parts, as getBoundingClientRect")
	r, _ = l.doc.BoxOf(second)
	assert.Equal(t, geom.Rect{X: 220, Y: 40, W: 180, H: 40}, r, "a whole box is where its column shows it")

	assert.Equal(t, "first", at(90, 70), "its part in the first column")
	assert.Equal(t, "first", at(300, 30), "its part in the second column")
	assert.Equal(t, "second", at(300, 50), "inside the first's box, yet the second's place")
	assert.Equal(t, "cols", at(200, 30), "the gap is the container's")
}
