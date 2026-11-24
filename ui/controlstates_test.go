package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
)

// The pointer hovers a range's parts it is over, the thumb hovering the
// track and the container; a press holds those it began on until the
// button is let go. A press on the track moves the thumb under the
// pointer, hovered but not pressed; a press on the thumb drags it from
// where it was taken.
func TestThePointerHoversAndPressesARangesParts(t *testing.T) {
	r := ui.Input(ui.Type("range"), ui.Value("20"), ui.Class("r"))
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(`body { margin: 0; }
.r { appearance: none; margin: 0; width: 200px; }
.r::-webkit-slider-runnable-track { height: 8px; }
.r::-webkit-slider-thumb { appearance: none; width: 20px; height: 20px; margin-top: -6px; }`)}, r)
	l.frames(2)
	parts := func() dom.State { return r.State() & (dom.PartHover | dom.PartActive) }
	// the thumb's middle at 20 of 100: 10 + 180 × 0.2 = 46 along, 10 down
	l.moveTo(geom.V2(46, 10))
	assert.Equal(t, dom.PartHover, parts(), "over the thumb")
	l.moveTo(geom.V2(150, 10))
	assert.Equal(t, dom.ContainerHover|dom.TrackHover, parts(), "over the track")

	l.down(geom.V2(150, 10))
	l.frames(1)
	assert.Equal(t, dom.PartHover|dom.ContainerActive|dom.TrackActive, parts(), "the thumb jumped under the press")
	assert.Equal(t, "78", l.doc.Value(r), "the thumb's middle under the pointer: (150 - 10) / 180")
	l.up()
	assert.Equal(t, dom.PartHover, parts(), "let go")

	l.down(geom.V2(144, 10))
	assert.Equal(t, dom.PartHover|dom.PartActive, parts(), "a press on the thumb")
	assert.Equal(t, "78", l.doc.Value(r), "taken off its middle, it stays")
	l.moveTo(geom.V2(162, 10))
	assert.Equal(t, "88", l.doc.Value(r), "and moves as far as the pointer")
	l.up()
}

// A vertical range's value runs down it, up it in rtl.
func TestAVerticalRangeIsDraggedDownIt(t *testing.T) {
	down := ui.Input(ui.Type("range"), ui.Value("0"), ui.Class("v"))
	up := ui.Input(ui.Type("range"), ui.Value("0"), ui.Class("v"), ui.Attr("dir", "rtl"))
	l := liveDocWith(t, []ui.DocumentOption{ui.CSS(`body { margin: 0; display: flex; }
.v { writing-mode: vertical-lr; margin: 0; height: 100px; }`)}, down, up)
	l.frames(2)
	l.dragged(geom.V2(8, 10), geom.V2(8, 75))
	assert.Equal(t, "75", l.doc.Value(down))
	l.dragged(geom.V2(24, 90), geom.V2(24, 25))
	assert.Equal(t, "75", l.doc.Value(up))
}
