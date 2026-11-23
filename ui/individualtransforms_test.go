package ui_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

// TestAMovedChildGrowsItsScrollerAndATransformChangeLaysItOutAgain has
// the overflow follow translate as it is set, changed and taken away.
func TestAMovedChildGrowsItsScrollerAndATransformChangeLaysItOutAgain(t *testing.T) {
	sheet := `body { margin: 0; } .s { width: 100px; height: 100px; overflow: hidden; }
.s > div { width: 50px; height: 50px; } .far { translate: 100px 80px; } .turned { rotate: 90deg; scale: 4 1 }`
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"s.css": {Data: []byte(sheet)}}, "s.css"))
	child := ui.Div(ui.Class("far"))
	s := ui.Div(ui.Class("s"), child)
	d.Body().Append(s)
	size := geom.V2(400, 300)
	d.Layout(size)
	assert.InDelta(t, 150, ui.ScrollSize(d, s).X, 0.01)
	assert.InDelta(t, 130, ui.ScrollSize(d, s).Y, 0.01)
	child.RemoveClass("far")
	child.AddClass("turned")
	d.Layout(size)
	assert.InDelta(t, 50, ui.ScrollSize(d, s).X, 0.01, "nothing past the right edge")
	assert.InDelta(t, 125, ui.ScrollSize(d, s).Y, 0.01, "turned upright, four times as tall, about its centre")
	child.RemoveClass("turned")
	d.Layout(size)
	assert.InDelta(t, 50, ui.ScrollSize(d, s).Y, 0.01)
}
