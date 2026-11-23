//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestTablesOfAnyElementsMatchChrome(t *testing.T) {
	const parts = `.t { display: table; } .it { display: inline-table; } .r { display: table-row; }
.c { display: table-cell; padding: 2px 6px; border: 1px solid #000; } .g { display: table-row-group; }
.h { display: table-header-group; } .f { display: table-footer-group; } .cap { display: table-caption; }
.sp { border-spacing: 4px 6px; } .b { display: block; } p { margin: 0; }`
	c := func(text string, args ...ui.Arg) *ui.Node {
		return ui.Div(append([]ui.Arg{ui.Class("c"), ui.Text(text)}, args...)...)
	}
	cases := map[string]*ui.Document{
		"a table of divs": styled(parts,
			ui.Div(ui.Class("t", "sp"),
				ui.Div(ui.Class("cap"), ui.Text("A caption")),
				ui.Div(ui.Class("g"),
					ui.Div(ui.Class("r"), c("one"), c("two two")),
					ui.Div(ui.Class("r"), c("three three three"), c("four"))))),
		"a footer before a header": styled(parts,
			ui.Div(ui.Class("t"),
				ui.Div(ui.Class("f"), ui.Div(ui.Class("r"), c("foot"), c("f"))),
				ui.Div(ui.Class("g"), ui.Div(ui.Class("r"), c("body"), c("b"))),
				ui.Div(ui.Class("h"), ui.Div(ui.Class("r"), c("head"), c("h"))))),
		"rows without groups, spans as cells": styled(parts,
			ui.Section(ui.Class("t"),
				ui.P(ui.Class("r"), ui.Span(ui.Class("c"), ui.Text("a span")), ui.Span(ui.Class("c"), ui.Text("another"))),
				ui.P(ui.Class("r"), ui.Span(ui.Class("c"), ui.Text("x")), ui.Span(ui.Class("c"), ui.Text("y"))))),
		"cells without a row or a table": styled(parts,
			ui.Div(ui.Style("width: 300px"),
				ui.P(ui.Text("Before.")), c("left"), ui.Text(" "), c("right, a longer one"), ui.P(ui.Text("After.")))),
		"rows without a table": styled(parts,
			ui.Div(ui.Style("width: 300px"),
				ui.Div(ui.Class("r"), c("one"), c("two")), ui.Div(ui.Class("r"), c("three, longer"), c("four")))),
		"cells in a table without a row": styled(parts, ui.Div(ui.Class("t", "sp"), c("one"), c("two"), c("three"))),
		"what is no cell in a row": styled(parts,
			ui.Div(ui.Class("t", "sp"),
				ui.Div(ui.Class("r"), c("cell"), ui.Div(ui.Text("a block")), ui.Span(ui.Text("a span")), ui.Text(" and text "),
					c("cell")),
				ui.Div(ui.Class("r"), c("x"), c("y"), c("z")))),
		"what is no row in a table": styled(parts,
			ui.Div(ui.Class("t", "sp"),
				ui.Div(ui.Class("r"), c("a"), c("b")), ui.Div(ui.Text("a block between rows")),
				ui.Div(ui.Class("r"), c("c"), c("d")))),
		"a table of text alone": styled(parts,
			ui.Div(ui.Style("width: 200px"),
				ui.Div(ui.Class("t"), ui.Style("padding: 4px; border: 2px solid #000"), ui.Text("Just some text in it.")),
				ui.Div(ui.Class("t"), ui.Style("width: 100%"), ui.Text("As wide as its container.")),
				ui.Div(ui.Class("t"), ui.Style("margin: 0 auto"), ui.Text("Centred.")))),
		"an inline table in a line": styled(parts,
			ui.P(ui.Text("Before "), ui.Span(ui.Class("it"), ui.Span(ui.Class("r"), ui.Span(ui.Class("c"), ui.Text("a")),
				ui.Span(ui.Class("c"), ui.Text("b")))), ui.Text(" after."))),
		"a cell shown as a block": styled(parts+` table { border-spacing: 3px; } td { padding: 2px; }`,
			ui.Table(ui.Tbody(
				ui.Tr(ui.Td(ui.Text("one")), ui.Td(ui.Class("b"), ui.Text("a block")), ui.Td(ui.Class("b"), ui.Text("b")),
					ui.Td(ui.Text("four"))),
				ui.Tr(ui.Td(ui.Text("x")), ui.Td(ui.Text("y")), ui.Td(ui.Text("z")))))),
		"a row shown as a block": styled(parts+` td { padding: 2px; }`,
			ui.Table(ui.Tbody(
				ui.Tr(ui.Td(ui.Text("one")), ui.Td(ui.Text("two"))),
				ui.Tr(ui.Class("b"), ui.Td(ui.Text("three")), ui.Td(ui.Text("four"))),
				ui.Tr(ui.Td(ui.Text("five")), ui.Td(ui.Text("six")))))),
		"a table shown as a block": styled(parts+` td { padding: 2px; }`,
			ui.Table(ui.Class("b"), ui.Tbody(ui.Tr(ui.Td(ui.Text("one")), ui.Td(ui.Text("two")))))),
		"a table in a cell of divs": styled(parts,
			ui.Div(ui.Class("t"), ui.Div(ui.Class("r"), c("outer"),
				ui.Div(ui.Class("c"), ui.Div(ui.Class("g"), ui.Div(ui.Class("r"), c("in"), c("ner"))))))),
		"a sized table and sized cells": styled(parts,
			ui.Div(ui.Class("t"), ui.Style("width: 300px; height: 80px"),
				ui.Div(ui.Class("r"), c("fixed", ui.Style("width: 80px")), c("the rest")),
				ui.Div(ui.Class("r"), c("x", ui.Style("vertical-align: middle")), c("y", ui.Style("vertical-align: bottom"))))),
		"floats and absolutes are no parts": styled(parts,
			ui.Div(ui.Style("width: 300px; position: relative"),
				ui.Div(ui.Class("c"), ui.Style("float: left"), ui.Text("floated")),
				ui.Div(ui.Class("c"), ui.Style("position: absolute; right: 0; top: 30px"), ui.Text("absolute")),
				c("a cell"), c("another"))),
		"cells in a flex container are items": styled(parts,
			ui.Div(ui.Style("display: flex; width: 300px; gap: 4px"), c("one"), c("two"), c("three"))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) { chrome.Match(t, d, geom.V2(640, 400)) })
	}
}

func TestCellBaselinesMatchChrome(t *testing.T) {
	const sheet = `table { border-spacing: 3px; } td { padding: 2px; border: 1px solid #000; vertical-align: baseline; }
.big { font-size: 28px; } .pad { padding-top: 14px; } .box { width: 20px; height: 26px; background: #888; }
.top { vertical-align: top; } .mid { vertical-align: middle; } .bot { vertical-align: bottom; }
.it { display: inline-table; }`
	td := func(text string, class ...string) *ui.Node { return ui.Td(ui.Class(class...), ui.Span(ui.Text(text))) }
	cases := map[string]*ui.Document{
		"sizes and padding on one baseline": styled(sheet,
			ui.Table(ui.Tbody(
				ui.Tr(td("small"), td("Big", "big"), td("padded", "pad"),
					ui.Td(ui.Span(ui.Text("two")), ui.Br(), ui.Span(ui.Text("lines"))), td("top", "top")),
				ui.Tr(td("x"), td("y", "mid"), td("z", "bot"), td("w"), td("v", "big"))))),
		"a cell of a block alone": styled(sheet,
			ui.Table(ui.Tbody(ui.Tr(td("text"), ui.Td(ui.Div(ui.Class("box"))), td("Big", "big"))))),
		"an empty cell": styled(sheet, ui.Table(ui.Tbody(ui.Tr(td("text"), ui.Td(), td("Big", "big"))))),
		"a cell holding blocks": styled(sheet,
			ui.Table(ui.Tbody(ui.Tr(ui.Td(ui.Div(ui.Span(ui.Text("first"))), ui.Div(ui.Span(ui.Text("second")))),
				td("Big", "big"))))),
		"a cell spanning rows": styled(sheet,
			ui.Table(ui.Tbody(
				ui.Tr(ui.Td(ui.Attr("rowspan", "2"), ui.Span(ui.Text("spans"))), td("Big", "big"), td("small")),
				ui.Tr(td("a"), td("B", "big"))))),
		"a wrapped cell": styled(sheet,
			ui.Table(ui.Style("width: 160px"), ui.Tbody(ui.Tr(td("some words that wrap in their cell"), td("Big", "big"))))),
		"an inline table on its first row's baseline": styled(sheet,
			ui.Div(ui.Span(ui.Text("Before ")), ui.Table(ui.Class("it"), ui.Tbody(
				ui.Tr(td("one", "pad"), td("two")), ui.Tr(td("three"), td("four")))), ui.Span(ui.Text(" after.")))),
		"an inline table of middle cells": styled(sheet,
			ui.Div(ui.Span(ui.Text("Before ")), ui.Table(ui.Class("it"), ui.Tbody(
				ui.Tr(td("one", "mid", "pad"), td("Two", "mid", "big")), ui.Tr(td("three"), td("four")))),
				ui.Span(ui.Text(" after.")))),
		"an inline table of a block alone": styled(sheet,
			ui.Div(ui.Span(ui.Text("Before ")), ui.Table(ui.Class("it"), ui.Tbody(ui.Tr(ui.Td(ui.Div(ui.Class("box")))))),
				ui.Span(ui.Text(" after.")))),
		"an empty inline table": styled(sheet+` .e { width: 30px; height: 20px; }`,
			ui.P(ui.Span(ui.Text("Before ")), ui.Span(ui.Class("it", "e")), ui.Span(ui.Text(" after.")))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) { chrome.Match(t, d, geom.V2(640, 400)) })
	}
}

func TestGeneratedContentInTablePartsMatchesChrome(t *testing.T) {
	const sheet = `td { padding: 2px; }
.row tr::before { content: "no"; } .row tr::after { content: "end"; display: table-cell; }
.body tbody::before { content: "a row of its own"; } .table::before { content: "above"; }
.table::after { content: "below"; display: table-row; }
.cap::before { content: "a caption"; display: table-caption; }`
	grid := func(class string) *ui.Node {
		a, b := ui.Tr(), ui.Tr()
		a.Append(ui.Td(ui.Text("one")), ui.Td(ui.Text("two")))
		b.Append(ui.Td(ui.Text("three")), ui.Td(ui.Text("four")))
		return ui.Table(ui.Class(class), ui.Tbody(a, b))
	}
	for _, class := range []string{"row", "body", "table", "cap"} {
		t.Run(class, func(t *testing.T) { chrome.Match(t, styled(sheet, grid(class)), geom.V2(640, 400)) })
	}
}

func TestTableColumnsMatchChrome(t *testing.T) {
	const sheet = `table { border-spacing: 3px 5px; } td { padding: 2px; }
.w { width: 90px; } .pc { width: 40%; } .fixed { table-layout: fixed; width: 300px; }
.t { display: table; } .r { display: table-row; } .c { display: table-cell; padding: 2px; }
.col { display: table-column; } .cg { display: table-column-group; }`
	row := func(cells ...string) *ui.Node {
		tr := ui.Tr()
		for _, c := range cells {
			tr.Append(ui.Td(ui.Text(c)))
		}
		return tr
	}
	cases := map[string]*ui.Document{
		"columns over their tracks": styled(sheet,
			ui.Table(ui.Col(), ui.Col(ui.Class("w")), ui.Col(),
				ui.Tbody(row("one", "two", "three"), row("four", "five", "six")))),
		"a column spanning two, in a group": styled(sheet,
			ui.Table(ui.Colgroup(ui.Col(ui.Attr("span", "2")), ui.Col(ui.Class("w"))), ui.Colgroup(ui.Attr("span", "2")),
				ui.Tbody(row("one", "two", "three", "four", "five"), row("a", "b", "c", "d", "e")))),
		"a group's width is its columns'": styled(sheet,
			ui.Table(ui.Colgroup(ui.Class("w"), ui.Col(), ui.Col()), ui.Tbody(row("one", "two", "three")))),
		"a percentage column": styled(sheet,
			ui.Table(ui.Style("width: 400px"), ui.Col(ui.Class("pc")), ui.Col(), ui.Tbody(row("one", "two")))),
		"columns in a fixed table": styled(sheet,
			ui.Table(ui.Class("fixed"), ui.Col(ui.Class("w")), ui.Col(), ui.Col(),
				ui.Tbody(row("one", "two", "three")))),
		"a column's width attribute": styled(sheet,
			ui.Table(ui.Col(ui.Attr("width", "120")), ui.Col(ui.Attr("width", "30%")), ui.Tbody(row("one", "two", "three")))),
		"more columns than cells": styled(sheet,
			ui.Table(ui.Col(), ui.Col(ui.Class("w")), ui.Col(ui.Class("w")), ui.Tbody(row("one", "two")))),
		"columns of divs": styled(sheet,
			ui.Div(ui.Class("t"),
				ui.Div(ui.Class("cg"), ui.Div(ui.Class("col", "w")), ui.Div(ui.Class("col"))),
				ui.Div(ui.Class("r"), ui.Div(ui.Class("c"), ui.Text("one")), ui.Div(ui.Class("c"), ui.Text("two"))))),
		"columns under a caption, a head and a foot": styled(sheet,
			ui.Table(ui.Caption(ui.Text("A caption")), ui.Col(), ui.Col(ui.Class("w")),
				ui.Thead(row("h", "i")), ui.Tbody(row("one", "two")), ui.Tfoot(row("f", "g")))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) { chrome.Match(t, d, geom.V2(640, 400)) })
	}
}

func TestCollapsedBordersMatchChrome(t *testing.T) {
	const sheet = `table { border-collapse: collapse; } td { padding: 2px; border: 1px solid black; }
.b3 td { border-width: 3px; } .tb { border: 6px solid blue; } .thin { border: 1px solid blue; }
.wide { border: 5px solid red; } .none { border-style: none; } .hidden { border-style: hidden; }
.dashed { border: 3px dashed green; } .double { border: 4px double green; } .rb { border: 7px solid green; }
.pad { padding: 10px; border-spacing: 9px; } .w { width: 300px; } .h { height: 120px; }
.odd td { border-width: 1px 2px 3px 4px; }
.t { display: table; border-collapse: collapse; } .r { display: table-row; }
.c { display: table-cell; border: 3px solid black; padding: 2px; } .rtl { direction: rtl; }
.start { border-inline-start: 9px solid; } .tl { border-left: 2px solid; border-right: 8px solid; }`
	row := func(class string, cells ...*ui.Node) *ui.Node {
		tr := ui.Tr(ui.Class(class))
		for _, c := range cells {
			tr.Append(c)
		}
		return tr
	}
	td := func(text string, args ...ui.Arg) *ui.Node { return ui.Td(append([]ui.Arg{ui.Text(text)}, args...)...) }
	grid := func(class ...string) *ui.Node {
		return ui.Table(ui.Class(class...), ui.Tbody(row("", td("one"), td("two"), td("three")),
			row("", td("four"), td("five"), td("six"))))
	}
	cases := map[string]*ui.Document{
		"cells share their borders":   styled(sheet, grid()),
		"wide borders are halved":     styled(sheet, grid("b3")),
		"the table's own border":      styled(sheet, grid("tb")),
		"a table border thinner":      styled(sheet, grid("b3", "thin")),
		"each side its width":         styled(sheet, grid("odd")),
		"padding and spacing are not": styled(sheet, grid("pad", "b3")),
		"in a width and a height":     styled(sheet, grid("w", "h", "b3")),
		"a wide cell wins its edges": styled(sheet,
			ui.Table(ui.Tbody(row("", td("one"), td("two", ui.Class("wide")), td("three")),
				row("", td("four"), td("five"), td("six"))))),
		"a wide cell at the table's edge": styled(sheet,
			ui.Table(ui.Tbody(row("", td("one"), td("two"), td("three")),
				row("", td("four", ui.Class("wide")), td("five"), td("six"))))),
		"hidden wins, none loses": styled(sheet,
			ui.Table(ui.Class("b3"), ui.Tbody(row("", td("one", ui.Class("hidden")), td("two"), td("three", ui.Class("none"))),
				row("", td("four"), td("five", ui.Class("none")), td("six", ui.Class("hidden")))))),
		"a row's border": styled(sheet,
			ui.Table(ui.Tbody(row("rb", td("one"), td("two")), row("", td("three"), td("four"))))),
		"a row group's border": styled(sheet,
			ui.Table(ui.Thead(ui.Class("rb"), row("", td("h"), td("i"))), ui.Tbody(row("", td("one"), td("two")),
				row("", td("three"), td("four"))))),
		"a column's border": styled(sheet,
			ui.Table(ui.Colgroup(ui.Col(), ui.Col(ui.Class("rb"))), ui.Tbody(row("", td("one"), td("two"), td("three")),
				row("", td("four"), td("five"), td("six"))))),
		"a column group's border": styled(sheet,
			ui.Table(ui.Colgroup(ui.Class("rb"), ui.Col(), ui.Col()), ui.Colgroup(ui.Col()),
				ui.Tbody(row("", td("one"), td("two"), td("three")), row("", td("four"), td("five"), td("six"))))),
		"spans": styled(sheet,
			ui.Table(ui.Class("b3"), ui.Tbody(row("", td("one", ui.Attr("colspan", "2"), ui.Class("wide")), td("three")),
				row("", td("four", ui.Attr("rowspan", "2")), td("five"), td("six", ui.Class("dashed"))),
				row("", td("eight"), td("nine"))))),
		"a caption above": styled(sheet,
			ui.Table(ui.Class("tb"), ui.Caption(ui.Text("A caption")), ui.Tbody(row("", td("one"), td("two"))))),
		"divs": styled(sheet,
			ui.Div(ui.Class("t"), ui.Div(ui.Class("r"), ui.Div(ui.Class("c"), ui.Text("one")),
				ui.Div(ui.Class("c"), ui.Text("two"))), ui.Div(ui.Class("r"), ui.Div(ui.Class("c"), ui.Text("three")),
				ui.Div(ui.Class("c"), ui.Text("four"))))),
		"an empty table keeps its border":      styled(sheet, ui.Table(ui.Class("tb", "w", "h"))),
		"right to left, each side its width":   styled(sheet, grid("odd", "rtl")),
		"right to left, the table's own sides": styled(sheet, grid("tl", "rtl")),
		"right to left, a cell's start wins": styled(sheet,
			ui.Table(ui.Class("rtl"), ui.Tbody(row("", td("one", ui.Class("start")), td("two"), td("three", ui.Class("start"))),
				row("", td("four"), td("five", ui.Class("wide")), td("six"))))),
		"right to left, rows, columns and spans": styled(sheet,
			ui.Table(ui.Class("rtl", "b3"), ui.Colgroup(ui.Col(ui.Class("start")), ui.Col(ui.Class("rb"))),
				ui.Tbody(row("tl", td("one", ui.Attr("colspan", "2")), td("three")),
					row("", td("four", ui.Attr("rowspan", "2"), ui.Class("odd")), td("five"), td("six", ui.Class("tl"))),
					row("", td("eight"), td("nine"))))),
		"right to left, a row of direction ltr": styled(sheet,
			ui.Table(ui.Class("rtl"), ui.Tbody(row("", td("one", ui.Class("tl")), td("two", ui.Class("start"))),
				ui.Tr(ui.Style("direction: ltr"), td("three", ui.Class("start")), td("four"))))),
		"blocks in the cells": styled(sheet,
			ui.Table(ui.Class("b3", "tb"), ui.Tbody(row("", ui.Td(ui.Div(ui.Style("width: 50px; height: 30px"))),
				ui.Td(ui.Div(ui.Style("width: 20%; height: 10px"))))))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) { chrome.Match(t, d, geom.V2(640, 400)) })
	}
}

func TestCollapsedRowsAndColumnsMatchChrome(t *testing.T) {
	const sheet = `table { border-spacing: 4px 6px; } td { padding: 2px 6px; border: 1px solid #000; }
.x { visibility: collapse; } .w { width: 300px; }`
	td := func(text string, args ...ui.Arg) *ui.Node { return ui.Td(append([]ui.Arg{ui.Text(text)}, args...)...) }
	grid := func(table ...ui.Arg) *ui.Node {
		return ui.Table(append(table,
			ui.Tbody(ui.Tr(td("one"), td("two two"), td("three")), ui.Tr(td("four"), td("five"), td("six")),
				ui.Tr(td("seven", ui.Attr("rowspan", "2")), td("eight eight eight", ui.Attr("colspan", "2"))),
				ui.Tr(td("x"), td("y"))))...)
	}
	cols := func(classes ...string) *ui.Node {
		g := ui.Colgroup()
		for _, c := range classes {
			g.Append(ui.Col(ui.Class(c)))
		}
		return g
	}
	cases := map[string]*ui.Document{
		"a column":                   styled(sheet, grid(cols("", "x", ""))),
		"the first and last columns": styled(sheet, grid(cols("x", "", "x"))),
		"a column group": styled(sheet,
			grid(ui.Colgroup(ui.Col()), ui.Colgroup(ui.Class("x"), ui.Col(), ui.Col()))),
		"a column of a sized table":  styled(sheet, grid(ui.Class("w"), cols("", "x", ""))),
		"a column of a shrunk table": styled(sheet, ui.Div(ui.Style("float: left"), grid(cols("", "x", "")))),
		"a column of a table in a flex row": styled(sheet, ui.Div(ui.Style("display: flex; align-items: flex-start"),
			grid(cols("", "x", "")), grid(ui.Col(), ui.Col(ui.Class("x"))))),
		"rows": styled(sheet,
			ui.Table(ui.Tbody(ui.Tr(ui.Class("x"), td("one"), td("two")), ui.Tr(td("three"), td("four")),
				ui.Tr(ui.Class("x"), td("five"), td("six")), ui.Tr(td("seven"), td("eight")),
				ui.Tr(ui.Class("x"), td("nine"), td("ten"))))),
		"a row a cell spans": styled(sheet, ui.Table(ui.Tbody(
			ui.Tr(td("tall", ui.Attr("rowspan", "3")), td("a")), ui.Tr(ui.Class("x"), td("b")), ui.Tr(td("c"))))),
		"a row group": styled(sheet,
			ui.Table(ui.Thead(ui.Tr(td("head"), td("h"))), ui.Tbody(ui.Class("x"), ui.Tr(td("one"), td("two"))),
				ui.Tbody(ui.Tr(td("three"), td("four"))))),
		"a framed table's column group and row group": styled(sheet+` .f { border: 3px solid; padding: 5px; }`,
			ui.Table(ui.Class("f"), ui.Colgroup(ui.Col()), ui.Colgroup(ui.Class("x"), ui.Col(), ui.Col()),
				ui.Tbody(ui.Tr(td("one"), td("two two"), td("three")))),
			ui.Table(ui.Class("f"), ui.Col(), ui.Col(ui.Class("x")), ui.Tbody(ui.Class("x"), ui.Tr(td("a"), td("b"))),
				ui.Tbody(ui.Tr(ui.Class("x"), td("a"), td("b")), ui.Tr(ui.Class("x"), td("a"), td("b")),
					ui.Tr(td("a"), td("b"))))),
		"a row and a column of a sized table": styled(sheet,
			ui.Table(ui.Class("w"), ui.Style("height: 120px"), cols("", "x"),
				ui.Tbody(ui.Tr(td("one"), td("two")), ui.Tr(ui.Class("x"), td("three"), td("four")),
					ui.Tr(td("five"), td("six"))))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) { chrome.Match(t, d, geom.V2(640, 400)) })
	}
}

func TestAPercentageWidthOnACellSpanningColumnsMatchesChrome(t *testing.T) {
	const sheet = `table { border-spacing: 0; width: 400px; margin-bottom: 8px; } td { padding: 0; } .a { width: auto; }
.s { border-spacing: 4px; } .s td { padding: 2px 5px; border: 1px solid; }`
	td := func(text, width string, args ...ui.Arg) *ui.Node {
		args = append(args, ui.Text(text))
		if width != "" {
			args = append(args, ui.Style("width: "+width))
		}
		return ui.Td(args...)
	}
	span := ui.Attr("colspan", "2")
	d := styled(sheet,
		ui.Table(ui.Tbody(ui.Tr(td("aa", "50%", span), td("b", "")), ui.Tr(td("c", ""), td("dddd", ""), td("e", "")))),
		ui.Table(ui.Tbody(ui.Tr(td("aa", "75%", span), td("b", "10%")), ui.Tr(td("c", "20%"), td("dddd", ""), td("e", "")))),
		ui.Table(ui.Class("a"), ui.Tbody(ui.Tr(td("aa", "50%", span), td("a longer cell", "")),
			ui.Tr(td("c", ""), td("dd", ""), td("e", "")))),
		ui.Table(ui.Class("s"), ui.Tbody(ui.Tr(td("three wide", "60%", ui.Attr("colspan", "3")), td("x", "")),
			ui.Tr(td("a", ""), td("bb", ""), td("ccc", ""), td("d", "")))),
	)
	chrome.Match(t, d, geom.V2(640, 400))
}
