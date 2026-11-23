package ui_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/stretchr/testify/assert"
)

func TestAMarkersCursorShowsOverTheMarker(t *testing.T) {
	item := ui.Li(ui.Text("an item"))
	inside := ui.Li(ui.Style("list-style-position: inside"), ui.Text("inside"))
	sheet := ui.CSS("li::marker { cursor: pointer; font-size: 10px } li { cursor: default }")
	l := liveDocWith(t, []ui.DocumentOption{sheet}, ui.Ul(item, inside))
	cursor := func(at geom.Vec2) platform.Cursor {
		l.moveTo(at)
		l.frames(1)
		return l.win.Cursor()
	}
	b := item.Box()
	assert.Equal(t, platform.CursorHand, cursor(geom.V2(b.X-12, b.Y+11)), "over the bullet outside")
	assert.Equal(t, platform.CursorHand, cursor(geom.V2(b.X-9.5, b.Y+11)), "just past the bullet")
	assert.Equal(t, platform.CursorArrow, cursor(geom.V2(b.X-4, b.Y+11)), "between the bullet and the text")
	assert.Equal(t, platform.CursorArrow, cursor(geom.V2(b.X+20, b.Y+b.H/2)), "over the item's text")
	b = inside.Box()
	assert.Equal(t, platform.CursorHand, cursor(geom.V2(b.X+1, b.Y+11)), "over the bullet inside")
	assert.Equal(t, platform.CursorHand, cursor(geom.V2(b.X+12, b.Y+2)), "past the bullet inside, on its line")
}
