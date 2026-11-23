//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// browsed is a Document of nodes styled by sheet read as a browser's.
func browsed(sheet string, nodes ...*ui.Node) *ui.Document {
	d := ui.NewDocument(ui.BrowserSheets(), ui.CSS(sheet))
	d.Body().Append(nodes...)
	return d
}

func TestStretchSizesMatchChrome(t *testing.T) {
	const page = `.box { width: 400px; height: 300px; position: relative; border: 1px solid }
.s { width: stretch; margin: 0 10px; padding: 4px; border: 2px solid } .w { width: -webkit-fill-available }
.h { height: stretch; margin: 6px 0 } .wh { height: -webkit-fill-available } .b { box-sizing: border-box }`
	cases := map[string]*ui.Document{
		"an inline block": browsed(page, ui.Div(ui.Class("box"),
			ui.Span(ui.Class("s"), ui.Style("display: inline-block"), ui.Text("one")),
			ui.Span(ui.Class("w", "b"), ui.Style("display: inline-block; margin-left: 30px"), ui.Text("two")))),
		"a block, a table and a float": browsed(page+` .t { display: table } .f { float: left }`,
			ui.Div(ui.Class("box"), ui.Div(ui.Class("s"), ui.Text("block")), ui.Div(ui.Class("s", "t"), ui.Text("table")),
				ui.Div(ui.Class("s", "f"), ui.Text("float")))),
		"a block's height": browsed(page, ui.Div(ui.Class("box"), ui.Div(ui.Class("h"), ui.Text("tall"))),
			ui.Div(ui.Class("box"), ui.Div(ui.Class("wh", "b"), ui.Style("padding: 5px"), ui.Text("tall")))),
		"an absolutely positioned box": browsed(page+` .a { position: absolute; left: 20px; top: 30px }`,
			ui.Div(ui.Class("box"), ui.Div(ui.Class("a", "s", "h"), ui.Text("placed")))),
		"a height without a definite containing block": browsed(page,
			ui.Div(ui.Div(ui.Class("h"), ui.Text("auto")))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
