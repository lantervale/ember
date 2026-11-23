package layout_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

// wptDoc is a page of nodes under sheet, its widths content-box ones as a browser's.
func wptDoc(sheet string, nodes ...*ui.Node) *ui.Document {
	d := ui.NewDocument(ui.BrowserSheets(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(nodes...)
	return d
}

func TestAnEmptyGridTakesItsTracks(t *testing.T) {
	grid := ui.Div(ui.Style("display: inline-grid; grid: 50px 50px / 50px 50px; padding: 5px; border: 5px solid"))
	d := wptDoc("", grid)
	d.Layout(geom.V2(800, 600))
	assert.Equal(t, geom.V2(120, 120), geom.V2(grid.Box().W, grid.Box().H))
}

// The items of WPT's css/css-grid/abspos/positioned-grid-items-025.html: in a right to
// left grid, auto lines stand for the padding edge on their own side.
func TestPositionedItemsOfARightToLeftGridTakeItsLines(t *testing.T) {
	start, end := ui.Div(ui.Style("grid-column: auto / 1")), ui.Div(ui.Style("grid-column: -1 / auto"))
	d := wptDoc(`body { margin: 0 }
.g { display: grid; grid-template-columns: 50px 100px; height: 100px; width: 150px; direction: rtl;
padding-left: 50px; padding-right: 80px; position: absolute }
.g > div { position: absolute; top: 0; right: 0; bottom: 0; left: 0 }`, ui.Div(ui.Class("g"), start, end))
	d.Layout(geom.V2(800, 600))
	assert.Equal(t, geom.Rect{X: 200, W: 80, H: 100}, start.Box())
	assert.Equal(t, geom.Rect{W: 50, H: 100}, end.Box())
}

// A positioned grid or flex container after text in a line starts the next line, as a
// positioned block does.
func TestAPositionedGridAfterTextStartsALine(t *testing.T) {
	for _, display := range []string{"block", "grid", "flex", "table"} {
		box := ui.Span(ui.Style("display: " + display + "; position: absolute; width: 10px; height: 10px"))
		d := wptDoc("body { margin: 0; font: 20px/1 monospace }", ui.P(ui.Style("margin: 0"), ui.Text("text "), box))
		d.Layout(geom.V2(800, 600))
		assert.Equal(t, float32(20), box.Box().Y, display)
	}
}
