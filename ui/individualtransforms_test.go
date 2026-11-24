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

// TestATransformHoldsFixedBoxesAndTurnsOverflowInLinesAndVerticalFrames
// places a fixed box in its transformed parent, and counts a moved
// inline-block and a box moved in a vertical-rl scroller as turned.
func TestATransformHoldsFixedBoxesAndTurnsOverflowInLinesAndVerticalFrames(t *testing.T) {
	sheet := `body { margin: 0; } .t { margin: 40px; width: 100px; height: 50px; translate: 10px; }
.f { position: fixed; top: 5px; left: 5px; width: 10px; height: 10px; }
.s { width: 100px; height: 100px; overflow: hidden; } .s > * { width: 50px; height: 50px; }
.v { writing-mode: vertical-rl; } span { display: inline-block; }`
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"s.css": {Data: []byte(sheet)}}, "s.css"))
	fixed := ui.Div(ui.Class("f"))
	inline := ui.Div(ui.Class("s"), ui.Span(ui.Style("translate: 100px 80px")))
	vertical := ui.Div(ui.Class("s", "v"), ui.Div(ui.Style("translate: -100px 20px")))
	d.Body().Append(ui.Div(ui.Class("t"), fixed), inline, vertical)
	d.Layout(geom.V2(400, 400))
	r, _ := ui.TransformedBox(d, fixed)
	assert.Equal(t, geom.Rect{X: 55, Y: 45, W: 10, H: 10}, r, "in its transformed parent's padding box, moved with it")
	assert.InDelta(t, 150, ui.ScrollSize(d, inline).X, 0.01)
	assert.InDelta(t, 130, ui.ScrollSize(d, inline).Y, 0.01)
	assert.InDelta(t, 150, ui.ScrollSize(d, vertical).X, 0.01, "from the left of the box moved left to the right edge")
	assert.InDelta(t, 70, ui.ScrollSize(d, vertical).Y, 0.01)
}
