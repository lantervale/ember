package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

func TestAScrollerSetPastItsEndStopsWhereItsBarsLeaveItsContentInView(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`.s { height: 80px; width: 160px; overflow: auto; }
.wide { width: 400px; height: 10px; } .tall { height: 300px; }`))
	s := ui.Div(ui.Class("s"), ui.Div(ui.Class("wide")), ui.Div(ui.Class("tall")))
	d.Body().Append(s)
	d.Layout(geom.V2(600, 800))
	d.SetScroll(s, geom.V2(1000, 1000))
	d.Layout(geom.V2(600, 800))
	assert.Equal(t, geom.V2(400-145, 310-65), d.Scroll(s))
}
