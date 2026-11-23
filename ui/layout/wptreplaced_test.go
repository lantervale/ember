package layout_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	_ "github.com/lantervale/ember/ui/svg"
)

// replacedSVG is the picture of WPT's css/css-grid/grid-items/replaced-element-015.html: a
// square of no natural size.
const replacedSVG = `data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'>` +
	`<rect width='100' height='100' fill='green'/></svg>`

// A grid item's bottom margin set and taken off again leaves the layout of a page that
// never had it: a picture sized against its parent's height measures afresh when only
// that height changes.
func TestAMarginTakenOffLeavesTheLayoutOfAPageWithout(t *testing.T) {
	size := geom.V2(800, 600)
	build := func(margin string) (*ui.Document, *ui.Node, *ui.Node) {
		img := ui.Img(ui.Attr("src", replacedSVG), ui.Style("height: 100%"))
		item := ui.Div(ui.Style("height: 100%; margin-bottom: "+margin), img)
		d := ui.NewDocument()
		d.Body().Append(ui.Div(ui.Style("display: inline-grid; width: 100px"), item))
		return d, item, img
	}
	fresh, _, want := build("0")
	fresh.Layout(size)

	d, item, img := build("50px")
	d.Layout(size)
	for _, m := range []string{"0", "50px", "0"} {
		item.SetStyle("height: 100%; margin-bottom: " + m)
		d.Layout(size)
	}
	assert.Equal(t, want.Box(), img.Box())
}
