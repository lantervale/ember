package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
)

func TestAContainerQueryIsAnsweredInTheLayoutThatSizesItsContainer(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 0; }
.side { container: side / inline-size; width: 50%; }
.card { height: 10px; }
@container side (width > 300px) { .card { height: 20px; width: 50cqw; } }
`))
	card := ui.Div(ui.Class("card"))
	d.Body().Append(ui.Div(ui.Class("side"), card))
	d.Layout(geom.V2(800, 600))
	assert.Equal(t, geom.Rect{W: 200, H: 20}, card.Box(), "400 wide: the first layout shows it")
	d.Layout(geom.V2(500, 600))
	assert.Equal(t, geom.Rect{W: 250, H: 10}, card.Box(), "250 wide: a resize answers again")
	d.Layout(geom.V2(700, 600))
	assert.Equal(t, geom.Rect{W: 175, H: 20}, card.Box())
	assert.Empty(t, d.Warnings())
	matched := d.MatchedRules(card)
	require.NotEmpty(t, matched)
	assert.Equal(t, []string{"side (width > 300px)"}, matched[0].Containers, "a tool sees where a rule stands")
}

func TestNestedContainersAreAnsweredOutsideIn(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 0; }
.c { container-type: inline-size; width: 50%; }
@container (width > 300px) { .c { width: 90%; } }
@container (width > 250px) { .leaf { height: 7px; } }
`))
	leaf := ui.Div(ui.Class("leaf"))
	inner := ui.Div(ui.Class("c"), leaf)
	outer := ui.Div(ui.Class("c"), inner)
	d.Body().Append(outer)
	d.Layout(geom.V2(800, 600))
	assert.InDelta(t, 400, outer.Box().W, 0.01)
	assert.InDelta(t, 360, inner.Box().W, 0.01, "its container is over 300")
	assert.InDelta(t, 7, leaf.Box().H, 0.01, "its container is 360, over 250")
}

func TestASizeContainersSizeIsNotItsContents(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { margin: 0; font-size: 16px; line-height: 20px; }
.ib { display: inline-block; container-type: inline-size; padding: 5px; }
.size { container-type: size; width: 200px; }
.abs { position: absolute; top: 300px; container-type: inline-size; }
.mn { display: inline-block; container-type: inline-size; min-width: 50px; }
.flow { container-type: inline-size; width: 300px; }
.fl { float: left; width: 20px; height: 50px; }
p { margin: 20px 0; }
`))
	ib := ui.Div(ui.Class("ib"), ui.Text("hello world"))
	size := ui.Div(ui.Class("size"), ui.Text("abc"))
	abs := ui.Div(ui.Class("abs"), ui.Text("one two"))
	mn := ui.Div(ui.Class("mn"), ui.Text("min"))
	flow := ui.Div(ui.Class("flow"), ui.Div(ui.Class("fl")), ui.P(ui.Text("x")))
	d.Body().Append(ui.Div(ib), size, abs, ui.Div(mn), flow)
	d.Layout(geom.V2(800, 600))
	assert.InDelta(t, 10, ib.Box().W, 0.01, "an inline-size container shrinks to fit nothing")
	assert.InDelta(t, 50, ib.Box().H, 0.01, "its lines wrap at every word")
	assert.InDelta(t, 0, size.Box().H, 0.01, "a size container is as high as nothing")
	assert.InDelta(t, 0, abs.Box().W, 0.01)
	assert.InDelta(t, 50, mn.Box().W, 0.01, "within its min-width")
	assert.InDelta(t, 60, flow.Box().H, 0.01, "a container holds its margins and its floats")
}

func TestARequeryLaysOutOnlyWhenAnAnswerMoves(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
.c { container-type: inline-size; }
@container (width > 300px) { p { color: #f00; } }
`))
	p := ui.P(ui.Text("x"))
	d.Body().Append(ui.Div(ui.Class("c"), p))
	d.Layout(geom.V2(800, 600))
	require.NotNil(t, dom.Data(p).Style)
	red := dom.Data(p).Style.Color
	before := ui.Restyled(d)
	d.Layout(geom.V2(700, 600))
	assert.Equal(t, before, ui.Restyled(d), "700 crosses nothing: nothing restyles")
	d.Layout(geom.V2(200, 600))
	assert.NotEqual(t, red, dom.Data(p).Style.Color)
}
