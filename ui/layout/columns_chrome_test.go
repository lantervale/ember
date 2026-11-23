//go:build chrome

package layout_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// rows is a paragraph of n one-word lines, broken by newlines: Chrome reports a br that
// follows a column break from its paragraph's first column.
func rows(style string, n int) *ui.Node {
	args := []ui.Arg{ui.Style("white-space: pre; " + style)}
	for i := range n {
		if i > 0 {
			args = append(args, ui.Text("\n"))
		}
		args = append(args, ui.Span(ui.Text(fmt.Sprint("line ", i+1))))
	}
	return ui.P(args...)
}

// tableRows are n rows of one cell each, named by prefix and their number.
func tableRows(prefix string, n int) []ui.Arg {
	out := make([]ui.Arg, n)
	for i := range n {
		out[i] = ui.Tr(ui.Td(ui.Text(fmt.Sprint(prefix, i+1))))
	}
	return out
}

func prose(n int) string { return strings.TrimSpace(strings.Repeat(sentence+" ", n)) }

func TestColumnsMatchChrome(t *testing.T) {
	const base = `.m { column-count: 2; column-gap: 20px; width: 400px; background: #eee; margin-bottom: 10px; }
p { margin: 0 0 10px; background: #cdf; } .tall { height: 130px; background: #fcc; }
.box { width: 50px; height: 50px; background: #888; }`
	m := func(style string, kids ...ui.Arg) *ui.Node {
		return ui.Div(append([]ui.Arg{ui.Class("m"), ui.Style(style)}, kids...)...)
	}
	cases := map[string]*ui.Document{
		"blocks balanced over two columns": styled(base,
			m("", rows("", 3), rows("", 3), rows("", 5)),
			m("", rows("margin: 10px 0", 3), rows("margin: 10px 0", 3)),
			m("", rows("margin-bottom: 30px", 3), rows("margin: 0", 2)),
			m("", rows("margin: 0", 3), rows("margin: 0 0 40px", 2)),
		),
		"paragraphs break between lines": styled(base,
			m("", rows("", 7)),
			m("", words("", prose(3))),
			m("column-count: 3", words("text-align: justify", prose(4)), words("text-align: right", prose(2))),
			m("padding: 5px; border: 3px solid", rows("padding: 4px; border: 2px solid red", 6)),
			m("", rows("margin: 0", 3), ui.Div(ui.Style("padding: 10px; border: 1px solid"), rows("margin: 0", 3))),
		),
		"text of the container itself": styled(base,
			m("", ui.Text(prose(3))),
			m("column-count: 3; text-align: center", ui.Text(prose(3))),
		),
		"empty room is cut anywhere": styled(base,
			m("", ui.Div(ui.Class("tall"))),
			m("", rows("", 3), ui.Div(ui.Class("tall"), ui.Style("height: 50px")), rows("", 2)),
			m("column-count: 3", ui.Div(ui.Class("tall"), ui.Style("height: 100px; padding: 6px; border: 2px solid")),
				rows("margin: 0", 2)),
		),
		"orphans and widows": styled(base+` p { margin: 0; }`,
			m("", rows("", 5)), m("", rows("", 3)), m("", rows("", 2), rows("", 3)),
			m("", rows("", 1), rows("", 4), rows("", 1)),
			m("column-count: 3", rows("", 7)), m("column-count: 3", rows("", 4)),
			m("column-count: 4", rows("", 5)), m("", rows("orphans: 1; widows: 1", 3)),
			m("height: 80px; column-fill: auto", rows("", 1), rows("orphans: 4", 6)),
			m("column-count: 3", rows("orphans: 3; widows: 3", 8)),
		),
		"monolithic boxes and break-inside": styled(base+` p { margin: 0; }`,
			m("", ui.Img(ui.Style("display: block; width: 50px; height: 70px")), rows("", 2)),
			m("", rows("", 2), ui.Div(ui.Style("overflow: hidden; height: 70px")), rows("", 1)),
			m("", rows("", 2), rows("break-inside: avoid", 4), rows("", 2)),
			m("", rows("", 2), rows("page-break-inside: avoid", 4), rows("", 2)),
			m("", rows("", 3), ui.Div(ui.Style("display: inline-block; width: 30px; height: 45px; background: #888")),
				rows("", 2)),
		),
		"forced and avoided breaks": styled(base+` p { margin: 0; }`,
			m("", rows("", 1), rows("break-before: column; margin-top: 10px", 3), rows("", 1)),
			m("column-count: 3", rows("break-after: column", 1), rows("", 4)),
			m("", rows("", 4), rows("break-after: avoid", 1), rows("", 5)),
		),
		"breaks Chrome honours": styled(base+` p { margin: 0; } .f { display: flex; flex-direction: column; }`,
			m("", rows("break-before: column", 2), rows("", 3)),
			m("", rows("", 3), ui.Div(rows("break-before: column", 2), rows("", 1))),
			m("", rows("", 3), ui.Div(rows("", 2), rows("break-after: column", 1)), rows("", 2)),
			m("", rows("", 4), ui.Div(rows("break-before: avoid", 2)), rows("", 3)),
			m("", rows("", 4), ui.Div(rows("", 1), rows("break-after: avoid", 1)), rows("", 3)),
			m("", rows("", 2), rows("break-inside: avoid-column", 4), rows("", 2)),
			m("", rows("", 2), rows("break-before: avoid-column", 3), rows("", 4)),
			m("", rows("", 2), rows("break-before: page", 2), rows("", 2)),
			m("", rows("", 2), rows("break-inside: avoid", 9)),
			m("", ui.Div(ui.Class("f"), rows("", 2), rows("break-before: column", 2), rows("", 1))),
			m("column-count: 3", rows("", 1), ui.Div(ui.Style("break-inside: avoid"), rows("", 2),
				rows("break-before: column", 2))),
		),
		"a height, column-fill and columns that overflow": styled(base+` p { margin: 0; }`,
			m("height: 100px", rows("", 6)),
			m("height: 50px", rows("", 7)),
			m("height: 100px; column-fill: auto", rows("", 6)),
			m("column-fill: auto", rows("", 6)),
			m("max-height: 50px", rows("", 7)),
			m("max-height: 50px; column-fill: auto", rows("", 7)),
			m("min-height: 100px", rows("", 6)),
			m("height: 45px; overflow: auto", rows("", 7)),
			m("height: 60px; padding: 4px; border: 2px solid; box-sizing: content-box", rows("", 9)),
		),
		"column-width and the columns shorthand": styled(base+` p { margin: 0; }`,
			m("column-count: auto; column-width: 90px", rows("", 7)),
			m("columns: 3 150px", rows("", 7)),
			m("columns: 3 100px; column-gap: normal", rows("", 7)),
			m("columns: 500px", rows("", 3)),
			m("columns: 2; column-gap: 10%", rows("", 4)),
			m("columns: 2; column-gap: 2em; font-size: 12px", rows("", 4)),
		),
		"a container that shrinks to its columns": styled(base+` p { margin: 0; } .m { width: auto; }`,
			m("display: inline-block", ui.P(ui.Text("hello world")), ui.P(ui.Text("hello"))),
			ui.Div(ui.Style("overflow: hidden"),
				m("float: left; columns: 100px", ui.P(ui.Text("hello world")), ui.P(ui.Text("hello"))),
				m("float: left; clear: left; columns: 2 100px", ui.P(ui.Text("hello world")), ui.P(ui.Text("hello")))),
			ui.Div(ui.Style("position: relative; height: 40px"),
				m("position: absolute; column-count: 3", ui.P(ui.Text("hello world")), ui.P(ui.Text("hello")),
					ui.P(ui.Text("you")))),
		),
		"positioned boxes": styled(base+` p { margin: 0; } .m { position: relative; }
.abs { position: absolute; width: 10px; height: 10px; background: #f00; }`,
			m("", ui.Div(ui.Class("abs"), ui.Style("top: 3px; right: 4px")), rows("", 3),
				ui.P(ui.Style("white-space: pre"), ui.Span(ui.Text("one"), ui.Span(ui.Class("abs"))), ui.Text("\ntwo\nthree"))),
			m("", rows("", 3), rows("position: relative; top: 5px; left: 5px", 3)),
			m("", rows("", 3), ui.Div(ui.Class("abs"), ui.Style("position: absolute")), rows("", 3)),
			m("", rows("", 2), ui.Div(ui.Style("position: relative"), rows("", 2),
				ui.Div(ui.Class("abs"), ui.Style("bottom: 0; right: 0")))),
		),
		"floats in columns": styled(base+` p { margin: 0; }`,
			m("", ui.Div(ui.Class("box"), ui.Style("float: left")), rows("", 6)),
			m("", rows("", 3), ui.Div(ui.Class("box"), ui.Style("float: right")), rows("", 3)),
			m("", ui.Div(ui.Class("box"), ui.Style("float: left; margin-right: 6px")), words("", prose(3))),
		),
		"a table cut between rows": styled(base+` p { margin: 0; } table { border-spacing: 0; background: #cdf; }
td { padding: 0; }`,
			m("", ui.Table(tableRows("r", 6)...)),
			m("", rows("", 3), ui.Table(tableRows("r", 4)...), rows("", 2)),
			m("", ui.Table(ui.Thead(ui.Tr(ui.Td(ui.Text("head")))), ui.Tbody(tableRows("r", 5)...))),
			m("", ui.Table(ui.Style("border: 2px solid; padding: 3px"), ui.Tbody(tableRows("r", 4)...),
				ui.Tbody(tableRows("s", 3)...))),
			m("", ui.Table(ui.Style("break-inside: avoid"), ui.Tbody(tableRows("r", 4)...)), rows("", 2)),
		),
		"a flex container cut between its items and lines": styled(base+` p { margin: 0; }
.f { display: flex; background: #cdf; } .w { width: 45px; height: 30px; background: #888; }`,
			m("", ui.Div(ui.Class("f"), ui.Style("flex-direction: column"), rows("", 2), rows("", 2), rows("", 2))),
			m("", ui.Div(ui.Class("f"), ui.Style("flex-wrap: wrap; width: 100px"), ui.Div(ui.Class("w")),
				ui.Div(ui.Class("w")), ui.Div(ui.Class("w")), ui.Div(ui.Class("w")), ui.Div(ui.Class("w")),
				ui.Div(ui.Class("w")))),
			m("", ui.Div(ui.Class("f"), rows("", 4), rows("", 2))),
			m("", rows("", 3), ui.Div(ui.Class("f"), ui.Style("padding: 4px; border: 2px solid; gap: 6px"),
				rows("", 3), ui.Div(ui.Class("w"))), rows("", 1)),
			m("", ui.Div(ui.Class("f"), ui.Style("break-inside: avoid"), rows("", 4)), rows("", 2)),
		),
		"a positioned box cut": styled(base+` p { margin: 0; } .m { position: relative; }
.abs { position: absolute; top: 10px; left: 0; width: 60px; background: #f00; }`,
			m("", rows("position: relative; top: 4px", 6)),
			m("", ui.Div(ui.Style("position: relative"), rows("", 2), ui.Div(ui.Class("abs"), rows("", 4)))),
			m("height: 60px; column-fill: auto", ui.Div(ui.Class("abs"), rows("", 6))),
		),
		"a nested multi-column container": styled(base+` p { margin: 0; }`,
			m("", rows("", 2), m("width: auto; margin: 0", rows("", 4)), rows("", 2)),
		),
		"blocks spanning the columns": styled(base+` p { margin: 10px 0; } .s { column-span: all; margin: 15px 0;
background: #cfc; } .m { padding: 5px; border: 2px solid; }`,
			m("", rows("", 3), ui.H3(ui.Class("s"), ui.Text("Across")), rows("", 4), rows("", 2)),
			m("", ui.Div(ui.Class("s"), ui.Text("first")), rows("", 3),
				ui.Div(ui.Class("s"), ui.Style("width: 50%; margin: 5px auto"), ui.Text("narrow")),
				ui.Div(ui.Class("s"), ui.Text("last"))),
			m("height: 200px", rows("", 4), ui.Div(ui.Class("s"), ui.Text("middle")), rows("", 5)),
			m("height: 200px; column-fill: auto", rows("", 4), ui.Div(ui.Class("s"), ui.Text("middle")), rows("", 5)),
			m("column-count: 3; column-rule: 2px solid", words("", prose(2)), ui.H2(ui.Class("s"), ui.Text("Heading")),
				words("", prose(2)), ui.Div(ui.Class("s"), ui.Style("display: flex"), ui.Span(ui.Text("a flex")),
					ui.Span(ui.Text("row")))),
			m("", ui.Div(ui.Class("s"), ui.Style("float: left; width: 40px; margin: 0"), ui.Text("float")), rows("", 4)),
		),
		"spanners and positioned boxes": styled(base+` p { margin: 0; } .m { position: relative; }
.s { column-span: all; background: #cfc; } .abs { position: absolute; width: 10px; height: 10px; background: #f00; }`,
			m("", rows("", 4), ui.Div(ui.Class("s"), ui.Span(ui.Text("span"), ui.Span(ui.Class("abs")))), rows("", 2),
				ui.Div(ui.Class("abs"), ui.Style("bottom: 0; left: 0"))),
			m("display: inline-block; width: auto", rows("", 2), ui.Div(ui.Class("s"), ui.Text("a wider spanner"))),
		),
		"lists and headings": styled(base+` p { margin: 0; }`,
			m("", ui.Ul(ui.Li(ui.Text("one")), ui.Li(ui.Text("two")), ui.Li(ui.Text(sentence)), ui.Li(ui.Text("four")),
				ui.Li(ui.Text("five")))),
			m("column-count: 3", ui.H3(ui.Text("Title")), ui.P(ui.Text(prose(2))), ui.H3(ui.Text("Another")),
				ui.P(ui.Text(prose(2)))),
		),
		"columns right to left": styled(base+` p { margin: 0; } .m { position: relative; }
.s { column-span: all; background: #cfc; } .abs { position: absolute; width: 10px; height: 10px; background: #f00; }`,
			m("direction: rtl", rows("", 3), rows("", 4)),
			m("direction: rtl; column-count: 3", ui.Text(prose(3))),
			m("direction: rtl; column-count: 3; padding: 4px 9px 4px 2px; border-left: 5px solid", words("", prose(3))),
			m("direction: rtl; columns: 3 100px; column-gap: 7px", rows("", 3), ui.Div(ui.Class("s"), ui.Text("across")),
				rows("", 5)),
			m("direction: rtl; height: 60px; column-fill: auto", rows("", 7)),
			m("direction: rtl", ui.Div(ui.Class("box"), ui.Style("float: right")), rows("", 6)),
			m("direction: rtl", rows("", 3), ui.P(ui.Style("white-space: pre"), ui.Text("one\n"),
				ui.Span(ui.Text("two"), ui.Span(ui.Class("abs"))), ui.Text("\nthree"))),
			m("direction: rtl; display: inline-block; width: auto", rows("", 2), rows("", 2)),
			m("direction: rtl; column-count: auto", rows("", 3), ui.P(ui.Style("white-space: pre"), ui.Text("one\n"),
				ui.Span(ui.Text("two"), ui.Span(ui.Class("abs"))), ui.Text("\nthree"))),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(900, 1400))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

// TestTheRequirementsSectionMatchesChrome is CSS Zen Garden 221's Requirements
// section: an absolute container a percentage wide, two columns 2em apart, paragraphs
// with side margins and an absolute heading, less its rotation: Chrome's boxes are the
// turned ones, and the pixel page multicol has it turned.
func TestTheRequirementsSectionMatchesChrome(t *testing.T) {
	const sheet = `body { font-size: 14px; line-height: 1.6; margin: 0; }
.main { position: relative; height: 900px; }
.main .requirements { position: absolute; column-count: 2; column-gap: 2em; right: 0; z-index: 2;
	margin: 6em 0 0 0; width: 60%; padding: 1em 0 0 0; box-sizing: border-box; }
@media (max-width: 70em) { .main .requirements { width: 75%; column-count: 1; column-gap: 1em; } }
.main .requirements h3 { position: absolute; margin: 0; top: 4em; left: -6em; }
.main .requirements p { margin: 0 1em 1em 1em; }`
	page := func() *ui.Document {
		ps := []ui.Arg{ui.Class("requirements"), ui.H3(ui.Text("Requirements"))}
		for i := range 5 {
			ps = append(ps, words("", prose(2+i%3)))
		}
		return styled(sheet, ui.Div(ui.Class("main"), ui.Div(ps...)))
	}
	for _, w := range []float32{700, 1000, 1200, 1500} {
		t.Run(fmt.Sprint(w), func(t *testing.T) {
			r := chrome.Match(t, page(), geom.V2(w, 900))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

func TestColumnsMatchChromeAtSeveralWidths(t *testing.T) {
	const sheet = `
.m { columns: 3 160px; column-gap: 24px; column-rule: 2px solid #888; padding: 8px; border: 1px solid #000; }
p { margin: 0 0 8px; } h3 { margin: 0 0 6px; } .note { padding: 6px; border: 2px solid #c00; background: #fee; }
.pic { float: right; width: 60px; height: 44px; margin: 0 0 4px 6px; background: #888; }`
	page := func() *ui.Document {
		return styled(sheet, ui.Div(ui.Class("m"),
			ui.Div(ui.Class("pic")), words("", prose(3)),
			ui.H3(ui.Text("Columns")), words("", prose(3)),
			ui.Div(ui.Class("note"), words("margin: 0", prose(2))),
			ui.Ul(ui.Li(ui.Text("one")), ui.Li(ui.Text(sentence)), ui.Li(ui.Text("three"))),
			words("text-align: center", prose(2))))
	}
	for _, w := range []float32{320, 480, 640, 900} {
		t.Run(fmt.Sprint(w), func(t *testing.T) {
			r := chrome.Match(t, page(), geom.V2(w, 1200))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
