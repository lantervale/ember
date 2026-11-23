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

func TestStretchOnItemsAndLimitsMatchesChrome(t *testing.T) {
	const page = `.c { width: 300px; height: 100px; margin-bottom: 4px; border: 1px solid }
.s { margin: 0 10px 0 5px; padding: 3px; border: 2px solid red } table { border-spacing: 0 } td { padding: 0 }`
	c := func(style string, kids ...*ui.Node) *ui.Node {
		args := []ui.Arg{ui.Class("c"), ui.Style(style)}
		for _, k := range kids {
			args = append(args, k)
		}
		return ui.Div(args...)
	}
	s := func(style, text string) *ui.Node { return ui.Div(ui.Class("s"), ui.Style(style), ui.Text(text)) }
	plain := func(style, text string) *ui.Node { return ui.Div(ui.Style(style), ui.Text(text)) }
	long := "aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa"
	cases := map[string]*ui.Document{
		"a flex item": browsed(page,
			c("display: flex", s("width: stretch", "a"), plain("width: 50px", "b")),
			c("display: flex", s("flex-basis: stretch", "a"), plain("width: 50px", "b")),
			c("display: flex", s("height: stretch", "a"), plain("height: 50px", "b")),
			c("display: flex; align-items: start", s("height: stretch", "a"), plain("height: 50px", "b")),
			c("display: flex; flex-direction: column", s("height: stretch", "a"), plain("height: 20px", "b")),
			c("display: flex; flex-direction: column; align-items: start", s("width: stretch", "a"))),
		"a grid item": browsed(page,
			c("display: grid; grid-template-columns: 100px 150px", s("width: stretch; justify-self: start", "a"),
				s("height: stretch; align-self: start", "b")),
			c("display: grid; grid-template-columns: auto 150px", s("width: stretch", "a"), plain("", "b"))),
		"min and max sizes": browsed(page,
			c("", s("min-width: stretch; width: 10px", "a"), s("max-width: stretch; width: 900px", "b"),
				s("display: inline-block; min-width: stretch", "c")),
			c("", s("float: left; max-width: stretch", long)),
			c("height: 60px", s("min-height: stretch", "a")),
			c("height: 60px", s("max-height: stretch; height: 900px", "a"))),
		"a table cell and a column, as auto": browsed(page,
			ui.Table(ui.Class("c"), ui.Style("width: 300px"),
				ui.Tr(ui.Td(ui.Class("s"), ui.Style("width: stretch"), ui.Text("a")), ui.Td(ui.Text("b")))),
			ui.Table(ui.Class("c"), ui.Style("width: 300px"),
				ui.Colgroup(ui.Col(ui.Style("width: stretch")), ui.Col()),
				ui.Tr(ui.Td(ui.Text("a")), ui.Td(ui.Text("b")))),
			ui.Table(ui.Class("c"), ui.Style("width: 300px; height: 80px"),
				ui.Tr(ui.Td(ui.Class("s"), ui.Style("height: stretch"), ui.Text("a")), ui.Td(ui.Text("b"))))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 800))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
