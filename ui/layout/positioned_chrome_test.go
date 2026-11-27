//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestPositionedBoxesInLinesMatchChrome(t *testing.T) {
	const page = `.w { width: 300px; margin-bottom: 20px } .a { position: absolute; width: 10px; height: 10px;
background: #c00 } .r { position: relative; left: 20px; top: 10px; padding: 2px } .i { left: 5px; top: 3px }`
	cases := map[string]*ui.Document{
		"an inline one after the text of its block": styled(page,
			ui.Div(ui.Class("w"), ui.Text("some text"), ui.Span(ui.Class("a"))),
			ui.Div(ui.Class("w"), ui.Text("some text "), ui.Span(ui.Class("a")), ui.Text(" more text")),
			ui.Div(ui.Class("w"), ui.Text("some text that goes on and on and on and wraps round the box"),
				ui.Span(ui.Class("a")))),
		"a block one after the text of its block": styled(page,
			ui.Div(ui.Class("w"), ui.Text("some text"), ui.Div(ui.Class("a"))),
			ui.Div(ui.Class("w"), ui.Div(ui.Class("a")), ui.Text("some text"))),
		"in an inline element moved by position relative": styled(page,
			ui.Div(ui.Class("w"), ui.Text("before "), ui.Span(ui.Class("r"), ui.Text("moved"),
				ui.Span(ui.Class("a"), ui.Style("left: 0; top: 0")))),
			ui.Div(ui.Class("w"), ui.Text("before "), ui.Span(ui.Class("r"), ui.Text("moved"),
				ui.Span(ui.Class("a"), ui.Style("right: 0; bottom: 0")))),
			ui.Div(ui.Class("w"), ui.Text("before "), ui.Span(ui.Class("r"), ui.Text("moved "),
				ui.Span(ui.Class("a")), ui.Text(" after")))),
		"in inline elements moved inside each other": styled(page,
			ui.Div(ui.Class("w"), ui.Span(ui.Class("r"), ui.Text("outer "), ui.Span(ui.Class("r", "i"),
				ui.Text("inner"), ui.Span(ui.Class("a"), ui.Style("left: 2px; top: 1px"))))),
			ui.Div(ui.Class("w"), ui.Span(ui.Class("r"), ui.Text("outer "), ui.Span(ui.Style("position: relative; top: 4px"),
				ui.Text("inner "), ui.Span(ui.Class("a")), ui.Text("tail"))))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

func TestAnAbsoluteTableKeepsItsOwnSizeBetweenInsetsAsChromes(t *testing.T) {
	d := styled(`.cb { position: relative; height: 300px; } .t { position: absolute; left: 1px; right: 1px; }`,
		ui.Div(ui.Class("cb"),
			ui.Table(ui.Class("t"), ui.Style("top: 0; bottom: 0"), ui.Tr(ui.Td(ui.Text("x")))),
			ui.Div(ui.Class("t"), ui.Style("display: table; top: 100px"), ui.Text("x")),
			ui.Div(ui.Class("t"), ui.Style("display: inline-table; top: 150px; justify-self: stretch"), ui.Text("x")),
			ui.Div(ui.Class("t"), ui.Style("top: 200px"), ui.Text("a block stretches")),
		))
	r := chrome.Match(t, d, geom.V2(800, 600))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
