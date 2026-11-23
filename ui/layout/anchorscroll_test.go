package layout_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
)

// TestBoxesFollowTheirScrolledAnchor checks the places Chrome gives boxes
// whose anchor a scroller holds once it scrolled; headless Chrome's
// capture runs no frame after the scroll, which is when Chrome moves
// them, so the numbers are Chrome's as a page shows them.
func TestBoxesFollowTheirScrolledAnchor(t *testing.T) {
	const page = `body { margin: 0; position: relative; height: 560px }
.sc { margin: 30px; width: 300px; height: 150px; overflow: auto; border: 2px solid }
.tall { height: 600px; width: 600px; padding-top: 120px }
.a { width: 80px; height: 40px; margin-left: 90px; anchor-name: --a }
.p { position: absolute; position-anchor: --a; width: 50px; height: 30px }`
	el := func(class, style string, kids ...ui.Arg) *ui.Node {
		return ui.Div(append([]ui.Arg{ui.Class(strings.Fields(class)...), ui.Style(style)}, kids...)...)
	}
	size := geom.V2(800, 600)
	scrolled := func(by geom.Vec2, positioned ...*ui.Node) {
		sc := el("sc", "", el("tall", "", el("a", "")))
		d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(page)}}, "page.css"))
		d.Body().Append(append([]*ui.Node{sc}, positioned...)...)
		d.Layout(size)
		d.SetScroll(sc, by)
		d.Layout(size)
	}
	below := el("p", "top: anchor(bottom); left: anchor(right)")
	above := el("p", "position-area: top")
	fixed := el("p", "position: fixed; bottom: anchor(top); right: anchor(left)")
	scrolled(geom.V2(40, 70), below, above, fixed)
	assert.Equal(t, geom.Rect{X: 162, Y: 122, W: 50, H: 30}, below.Box())
	assert.Equal(t, geom.Rect{X: 97, Y: 52, W: 50, H: 30}, above.Box())
	assert.Equal(t, geom.Rect{X: 32, Y: 52, W: 50, H: 30}, fixed.Box())

	flips := el("p", "bottom: anchor(top); left: anchor(left); position-try-fallbacks: flip-block")
	scrolled(geom.V2(0, 100), flips)
	assert.Equal(t, geom.Rect{X: 122, Y: 92, W: 50, H: 30}, flips.Box(), "flipped below the anchor the scroll lifted")
}

func TestPositionVisibilityHidesTheBox(t *testing.T) {
	const page = `body { margin: 0 }
.sc { width: 200px; height: 100px; overflow: auto }
.tall { height: 600px }
.a { width: 80px; height: 40px; margin-top: 20px; anchor-name: --a }
.p { position: absolute; position-anchor: --a; top: anchor(bottom); left: anchor(left); width: 50px; height: 30px }`
	el := func(class, style string, kids ...ui.Arg) *ui.Node {
		return ui.Div(append([]ui.Arg{ui.Class(strings.Fields(class)...), ui.Style(style)}, kids...)...)
	}
	sc := el("sc", "", el("tall", "", el("a", "")))
	visible := el("p", "")
	always := el("p", "position-visibility: always")
	overflowing := el("p", "position-visibility: no-overflow; top: anchor(top); width: 900px")
	fits := el("p", "position-visibility: no-overflow")
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(page)}}, "page.css"))
	d.Body().Append(sc, visible, always, overflowing, fits)
	size := geom.V2(800, 600)
	d.Layout(size)
	hidden := func(n *ui.Node) bool { b, _ := dom.Data(n).Placed.(*box.Box); return b.Discarded }
	assert.False(t, hidden(visible), "its anchor in view")
	assert.True(t, hidden(overflowing), "wider than the page")
	assert.False(t, hidden(fits))

	d.SetScroll(sc, geom.V2(0, 200))
	d.Layout(size)
	assert.True(t, hidden(visible), "anchors-visible, its anchor scrolled away")
	assert.False(t, hidden(always))
}
