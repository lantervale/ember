package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestAFloatAfterABrStandsBelowItsLineWithWhatItHolds(t *testing.T) {
	abs := absBox(nil)
	f := el(dom.Div, float(style.FloatLeft, 50, 50), abs)
	p := el(dom.P, width(200), text("HH"), el(dom.Br, nil), el(dom.Span, nil, f, text("H")))
	tr := Layout(el(dom.Div, block, p), geom.V2(400, 300), ahemFonts{})
	assert.Equal(t, onPage(tr.Of[f]), onPage(tr.Of[abs]), "at its static place, the float's content top-left")
	assert.Equal(t, float32(10), onPage(tr.Of[f]).Y, "on the second line")

	abs = absBox(nil)
	f = el(dom.Div, float(style.FloatLeft, 50, 50), abs)
	tall := func(st *style.Computed) { st.LineHeight = style.LineHeight{Px: 40} }
	p = el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	},
		el(dom.Span, tall, el(dom.Br, nil), f))
	tr = Layout(el(dom.Div, block, p), geom.V2(400, 300), ahemFonts{})
	assert.Equal(t, float32(40), onPage(tr.Of[f]).Y, "below the first line")
	assert.Equal(t, onPage(tr.Of[f]), onPage(tr.Of[abs]), "alone on its line too")

	abs = absBox(nil)
	f = el(dom.Div, float(style.FloatLeft, 50, 50), abs)
	p = el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	},
		el(dom.Span, tall, text("\n  "), el(dom.Br, nil), text("\n  "), f, text("\n")))
	tr = Layout(el(dom.Div, block, p), geom.V2(400, 300), ahemFonts{})
	assert.Equal(t, float32(40), onPage(tr.Of[f]).Y, "below the first line, the white space round it collapsed")
	assert.Equal(t, onPage(tr.Of[f]), onPage(tr.Of[abs]), "and what it holds with it")
}
