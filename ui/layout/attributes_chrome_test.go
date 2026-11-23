//go:build chrome

package layout_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

// tableTags are the elements whose presentational attributes Chrome reads under its own
// user-agent sheet: left in ember's, exported as the page's, its rules would win over them.
const tableTags = textTags + "|table|td|th|caption"

func cells(row []ui.Arg, cells ...*ui.Node) *ui.Node {
	tr := ui.Tr(row...)
	tr.Append(cells...)
	return ui.Tbody(tr)
}

func block(w, h string) *ui.Node {
	return ui.Div(ui.Style("width: " + w + "; height: " + h + "; background: #888"))
}

func TestPresentationalAttributesLayOutAsInChrome(t *testing.T) {
	field := func() *ui.Node {
		return ui.Div(ui.Style("display: inline-block; width: 300px; height: 30px; background: #888"))
	}
	search := func(width string) *ui.Node {
		return ui.Div(ui.Style("width: "+width),
			ui.Table(ui.Attr("cellpadding", "0"), ui.Attr("cellspacing", "0"), cells([]ui.Arg{ui.Attr("valign", "top")},
				ui.Td(ui.Attr("width", "25%"), ui.Text(" ")),
				ui.Td(ui.Attr("align", "center"), ui.Attr("nowrap", ""), field(), ui.Br(), ui.Text("two buttons")),
				ui.Td(ui.Attr("width", "25%"), ui.Attr("align", "left"), ui.Attr("nowrap", ""), ui.Text("Advanced")))))
	}
	cases := map[string]*ui.Document{
		"a hidden field": page(
			ui.Form(ui.Input(ui.Type("hidden"), ui.Attr("value", "fr")),
				ui.Input(ui.Attr("value", "q"), ui.Style("width: 120px")), ui.Input(ui.Type("hidden")), ui.Text(" after"))),
		"absolute units": styled(`div { height: 10pt; margin: 2mm 0; background: #888 }
.pt { width: 72pt } .pc { width: 6pc } .in { width: 1.5in } .cm { width: 2.54cm } .mm { width: 30mm } .q { width: 120Q }
p { font-size: 10pt; line-height: 14pt; padding: 0 1pc }`,
			ui.Div(ui.Class("pt")), ui.Div(ui.Class("pc")), ui.Div(ui.Class("in")), ui.Div(ui.Class("cm")),
			ui.Div(ui.Class("mm")), ui.Div(ui.Class("q")), ui.P(ui.Text(sentence))),
		"sized cells": page(
			ui.Table(ui.Attr("width", "400"), cells(nil,
				ui.Td(ui.Attr("width", "100"), ui.Text("a")), ui.Td(ui.Text("b")),
				ui.Td(ui.Attr("width", "25%"), ui.Attr("height", "40"), ui.Text("c")))),
			ui.Table(ui.Attr("width", "80%"), ui.Attr("cellpadding", "6"), ui.Attr("cellspacing", "0"), cells(nil,
				ui.Td(ui.Text("padded")), ui.Td(ui.Text("cells")))),
			ui.Table(ui.Attr("cellspacing", "9"), ui.Attr("height", "60"), cells(nil,
				ui.Td(ui.Text("spaced")), ui.Td(ui.Text("cells")))),
			ui.Div(ui.Style("width: 120px"), ui.Table(cells(nil,
				ui.Td(ui.Attr("nowrap", ""), ui.Text("a cell that does not wrap its text")), ui.Td(ui.Text(sentence))))),
		),
		"percentage columns": page(
			search("560px"), search("700px"), search("340px"),
			ui.Table(cells(nil, ui.Td(ui.Attr("width", "20%"), ui.Text("one")), ui.Td(ui.Attr("width", "30%"), ui.Text("two")))),
			ui.Table(cells(nil, ui.Td(ui.Attr("width", "60%"), ui.Text("one")), ui.Td(ui.Attr("width", "60%"), ui.Text("two")),
				ui.Td(ui.Text("three")))),
			ui.Table(ui.Attr("width", "500"), cells(nil, ui.Td(ui.Attr("width", "30%"), ui.Text("one")),
				ui.Td(ui.Attr("width", "120"), ui.Text("two")), ui.Td(ui.Text("three")), ui.Td(ui.Text("and four")))),
			ui.Table(ui.Attr("width", "500"), cells(nil, ui.Td(ui.Attr("width", "30%"), ui.Text("one")),
				ui.Td(ui.Attr("width", "20%"), ui.Text("two")))),
			ui.Table(ui.Tbody(
				ui.Tr(ui.Td(ui.Attr("width", "50%"), ui.Text("one")), ui.Td(ui.Text("two"))),
				ui.Tr(ui.Td(ui.Attr("colspan", "2"), ui.Text("a cell across both columns of the table"))))),
		),
		"aligned blocks": page(
			ui.Div(ui.Attr("align", "center"), block("100px", "20px"), ui.Text("centred"),
				ui.Table(cells(nil, ui.Td(ui.Text("a table")))), ui.P(ui.Text("and a paragraph")),
				ui.Div(ui.Style("margin-left: 30px; width: 50px; height: 10px"))),
			ui.Div(ui.Attr("align", "right"), block("100px", "20px"), ui.Text("to the right"),
				ui.Table(cells(nil, ui.Td(ui.Text("a table"))))),
			ui.P(ui.Attr("align", "center"), ui.Text("a paragraph")),
			ui.H2(ui.Attr("align", "right"), ui.Text("a heading")),
			ui.Table(ui.Attr("width", "100%"), cells(nil,
				ui.Td(ui.Attr("align", "center"), block("80px", "20px"), ui.Text("centred")),
				ui.Td(ui.Attr("align", "right"), block("80px", "20px"), ui.Text("right")),
				ui.Td(block("80px", "20px"), ui.Text("left")))),
			ui.Table(ui.Attr("align", "center"), cells(nil, ui.Td(ui.Text("a centred table")))),
			ui.Center(ui.Table(cells(nil, ui.Td(ui.Text("in a center")))), block("120px", "20px"), ui.Text("text"),
				ui.Div(ui.Div(ui.Style("width: 60px; height: 10px")))),
		),
		"valign": page(
			ui.Table(cells([]ui.Arg{ui.Attr("valign", "top")},
				ui.Td(block("40px", "80px")), ui.Td(ui.Text("top")), ui.Td(ui.Attr("valign", "bottom"), ui.Text("bottom")),
				ui.Td(ui.Attr("valign", "middle"), ui.Text("middle")), ui.Td(block("20px", "20px")),
				ui.Td(ui.Attr("valign", "bottom"), block("20px", "20px")))),
			ui.Table(ui.Tbody(ui.Attr("valign", "bottom"),
				ui.Tr(ui.Td(block("40px", "60px")), ui.Td(ui.Text("bottom"))),
				ui.Tr(ui.Attr("valign", "top"), ui.Td(block("40px", "60px")), ui.Td(ui.Text("top"))))),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			matchesChromesOwn(t, d, geom.V2(800, 900), tableTags)
			assert.Empty(t, d.Warnings())
		})
	}
}

// hintTags are tableTags and the other elements whose attributes Chrome reads under its
// own sheet.
const hintTags = tableTags + "|thead|tbody|tfoot|tr|col|colgroup|hr|font|img"

func TestTheOldAttributesOfTablesRulesAndPicturesLayOutAsInChrome(t *testing.T) {
	a := ui.Attr
	src := a("src", "tile.png")
	grid := func(args ...ui.Arg) *ui.Node {
		return ui.Table(append(args, ui.Tbody(
			ui.Tr(ui.Td(ui.Text("one")), ui.Td(ui.Text("two"))),
			ui.Tr(ui.Td(ui.Text("three")), ui.Td(ui.Text("four")))))...)
	}
	cases := map[string]*ui.Document{
		"table borders": aligning("",
			grid(a("border", "1")), grid(a("border", "4"), a("cellspacing", "0")), grid(a("border", "")),
			grid(a("border", "2"), a("bordercolor", "red")), grid(a("rules", "all")),
			grid(a("rules", "rows"), a("border", "3")), grid(a("rules", "cols")), grid(a("frame", "box")),
			grid(a("frame", "hsides"), a("border", "5")), grid(a("border", "5"), a("frame", "vsides")),
			grid(a("frame", "void"), a("rules", "all")),
			ui.Table(a("rules", "groups"), ui.Colgroup(ui.Col()), ui.Colgroup(ui.Col()),
				ui.Thead(ui.Tr(ui.Td(ui.Text("a")), ui.Td(ui.Text("b")))),
				ui.Tbody(ui.Tr(ui.Td(ui.Text("c")), ui.Td(ui.Text("d"))))),
		),
		"floated tables and pictures": aligning("",
			ui.Div(ui.Style("width: 500px"), grid(a("align", "left")), grid(a("align", "right")), ui.Text(sentence),
				ui.Br(a("clear", "all")), ui.Text("below both")),
			ui.Div(ui.Style("width: 500px"), ui.Img(src, a("align", "left"), a("hspace", "10"), a("vspace", "5")),
				ui.Img(src, a("align", "right"), a("border", "2")), ui.Text(sentence+" "+sentence),
				ui.Br(a("clear", "left")), ui.Text("after")),
			ui.P(ui.Text("Hxg "), ui.Img(src, a("align", "top")), ui.Text(" "), ui.Img(src, a("align", "absmiddle")),
				ui.Text(" "), ui.Img(src, a("align", "middle")), ui.Text(" "), ui.Img(src, a("align", "texttop")),
				ui.Text(" "), ui.Img(src, a("align", "absbottom")), ui.Text(" "), ui.Img(src, a("align", "bottom")),
				ui.Text(" "), ui.Img(src, a("hspace", "5%"))),
		),
		"rules": aligning("",
			ui.Hr(a("size", "1")), ui.Hr(a("size", "8")), ui.Hr(a("noshade", ""), a("size", "3")),
			ui.Hr(a("width", "50%")), ui.Hr(a("width", "120"), a("align", "left")), ui.Hr(a("width", "0")),
			ui.Hr(a("width", "120"), a("align", "right")), ui.Hr(a("width", "120"), a("align", "center")),
			ui.Hr(a("color", "red"), a("size", "5")),
		),
		"font": aligning("",
			ui.P(ui.Font(a("size", "1"), ui.Text("one ")), ui.Font(a("size", "2"), ui.Text("two ")),
				ui.Font(a("size", "4"), ui.Text("four ")), ui.Font(a("size", "+3"), ui.Text("six ")),
				ui.Font(a("size", "7"), ui.Text("seven")), ui.Font(a("size", "-1"), ui.Text(" less"))),
			ui.P(ui.Font(a("face", "Go Mono"), a("color", "green"), ui.Text(sentence))),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			matchesChromesOwn(t, d, geom.V2(800, 900), hintTags)
			assert.Empty(t, d.Warnings())
		})
	}
}
