// The CSS table model: tables of any elements by their display, with the
// anonymous rows and cells they miss; the backgrounds of a table's layers,
// column groups under columns under row groups under rows under cells; and
// collapsed borders, each edge to the border that wins it; cells on their
// row's baseline.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 12px; display: flex; flex-wrap: wrap; gap: 14px; align-items: flex-start; font-size: 13px; }
.t { display: table; border-spacing: 4px; background-color: #e4ecf7; }
.r { display: table-row; } .c { display: table-cell; padding: 3px 6px; background-color: #fff; }
.g { display: table-row-group; } .cap { display: table-caption; background-color: #fde9c8; }
.it { display: inline-table; border: 1px solid #345; }
.layers { border-spacing: 5px; background-color: #eee; }
.layers td { padding: 3px 8px; }
.layers colgroup { background-color: #f6c; } .layers col.b { background-color: #fd6; }
.layers tbody { background-color: #9cf; } .layers tr.r2 { background-color: #7c7; }
.layers td.own { background-color: #fff; }
.collapse { border-collapse: collapse; }
.collapse td { border: 1px solid #345; padding: 3px 8px; }
.collapse td.wide { border: 5px solid #c33; } .collapse td.dash { border: 3px dashed #283; }
.collapse td.dot { border: 4px dotted #28c; } .collapse td.hid { border-style: hidden; }
.collapse td.fill { background-color: #fde9c8; }
.frame { border: 4px solid #345; } .frame td { border: 2px solid #9ab; }
.frame tr.mid { border: 6px solid #c93; }
.base td { vertical-align: baseline; padding: 2px 6px; background-color: #fff; border: 1px solid #9ab; }
.big { font-size: 24px; } .small { font-size: 10px; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"table-model.css": {Data: []byte(sheet)}}, "table-model.css"))
	div := func(class, text string, kids ...ui.Arg) *ui.Node {
		args := []ui.Arg{ui.Class(class)}
		if text != "" {
			args = append(args, ui.Text(text))
		}
		return ui.Div(append(args, kids...)...)
	}
	td := func(text string, class ...string) *ui.Node { return ui.Td(ui.Class(class...), ui.Text(text)) }
	tr := func(class string, cells ...*ui.Node) *ui.Node {
		r := ui.Tr(ui.Class(class))
		r.Append(cells...)
		return r
	}
	d.Body().Append(
		div("t", "", div("cap", "Divs as a table"),
			div("g", "", div("r", "", div("c", "Ada"), div("c", "12")), div("r", "", div("c", "Bo"), div("c", "7")))),
		div("t", "", div("c", "cells"), div("c", "without"), div("c", "a row")),
		div("", "", div("c", "no"), div("c", "table"), div("c", "at all")),
		div("t", "", div("r", "", ui.Text("loose text"), div("c", "a cell"), ui.Span(ui.Text("a span")))),
		div("", "before ", ui.Span(ui.Class("it"), div("r", "", div("c", "in"), div("c", "line"))), ui.Text(" after")),
		ui.Table(ui.Class("layers"),
			ui.Colgroup(ui.Col(), ui.Col(ui.Class("b"))), ui.Colgroup(ui.Col()),
			ui.Thead(tr("", td("head"), td("of"), td("three"))),
			ui.Tbody(tr("", td("body"), td("row", "own"), td("one")), tr("r2", td("body"), td("row"), td("two", "own"))),
			ui.Tfoot(tr("", td("foot"), td("of"), td("three")))),
		ui.Table(ui.Class("collapse"), ui.Tbody(
			tr("", td("one"), td("two", "wide"), td("three")),
			tr("", td("four", "dash"), td("five", "fill"), td("six", "dot")),
			tr("", td("seven"), td("eight", "hid"), td("nine")))),
		ui.Table(ui.Class("collapse", "frame"), ui.Tbody(
			tr("", td("a"), td("b"), td("c")), tr("mid", td("d"), td("e"), td("f")), tr("", td("g"), td("h"), td("i")))),
		ui.Table(ui.Class("base"), ui.Tbody(
			tr("", td("big", "big"), td("plain"), td("small", "small"),
				ui.Td(ui.Div(ui.Text("two")), ui.Div(ui.Text("lines")))))),
	)
	pages.Run(d, 640, 360)
}
