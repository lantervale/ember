//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// el is a div of style holding kids: nodes, or text for a string.
func el(style string, kids ...any) *ui.Node {
	args := []ui.Arg{ui.Style(style)}
	for _, k := range kids {
		switch k := k.(type) {
		case string:
			args = append(args, ui.Text(k))
		case *ui.Node:
			args = append(args, k)
		}
	}
	return ui.Div(args...)
}

const alignSheet = `body { font: 12px/15px Go; margin: 8px }
body > div { margin-bottom: 10px; border: 1px solid; }
.a { line-height: 21px } .b { font: 32px/40px Go; padding: 3px 0 7px } .c { line-height: 30px; margin-top: 5px }
.w { width: 60px } .n { width: 40px; height: 30px; background: #ccc }`

func matchAll(t *testing.T, cases map[string]*ui.Document, size geom.Vec2) {
	t.Helper()
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, size)
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

func items() []any {
	return []any{
		ui.Div(ui.Class("a"), ui.Text("one")),
		ui.Div(ui.Class("b", "w"), ui.Text("two lines here")),
		ui.Div(ui.Class("c"), ui.Text("x")),
		ui.Div(ui.Class("n")),
		ui.Div(ui.Class("a", "w"), ui.Text("three lines of small text")),
	}
}

func TestLastBaselineFlexItemsMatchChrome(t *testing.T) {
	row := func(style string) *ui.Node { return el("display: flex; "+style, items()...) }
	cases := map[string]*ui.Document{
		"align-items: last baseline": styled(alignSheet, row("align-items: last baseline"),
			row("align-items: last baseline; height: 200px"), row("align-items: first baseline")),
		"first and last baselines in one line": styled(alignSheet+` .f > :nth-child(odd) { align-self: last baseline }`,
			el("display: flex; align-items: baseline", items()...),
			ui.Div(ui.Class("f"), ui.Style("display: flex; align-items: baseline; height: 150px"),
				el("", "a"), el("font: 32px/45px Go", "b"), el("width: 50px", "c d e f"), el("", "g"))),
		"wrapped lines": styled(alignSheet, row("align-items: last baseline; flex-wrap: wrap; width: 160px"),
			row("align-items: last baseline; flex-wrap: wrap-reverse; width: 160px")),
		"a column": styled(alignSheet, row("flex-direction: column; align-items: last baseline; width: 200px"),
			row("flex-direction: column; align-items: baseline; width: 200px")),
		"a column's items with margins": styled(alignSheet,
			el("display: flex; flex-direction: column; align-items: last baseline; width: 200px",
				el("margin-left: 10px; width: 40px", "a"), el("margin-right: 20px; width: 30px", "b"), el("width: 50px", "c")),
			el("display: flex; flex-direction: column; align-items: baseline; width: 200px",
				el("margin-left: 10px; width: 40px", "a"), el("margin-right: 20px; width: 30px", "b"), el("width: 50px", "c"))),
		"wrap-reverse with first baselines": styled(alignSheet,
			el("display: flex; flex-wrap: wrap-reverse; align-items: baseline; width: 160px", items()...)),
		"row-reverse": styled(alignSheet, row("flex-direction: row-reverse; align-items: last baseline")),
		"auto margins and stretch": styled(alignSheet,
			el("display: flex; align-items: last baseline; height: 120px",
				el("margin-top: auto", "auto"), el("font: 32px/40px Go", "Box"), el("align-self: stretch", "s"))),
		"nested containers": styled(alignSheet,
			el("display: flex; align-items: last baseline",
				el("display: flex; flex-direction: column", el("", "row 1"), el("line-height: 25px", "row 2")),
				el("display: grid; grid-template-columns: auto auto", el("", "g1"), el("line-height: 21px", "g2"),
					el("font: 32px/40px Go", "g3")),
				el("display: flex; flex-wrap: wrap; width: 50px", el("", "w1"), el("", "w2"), el("line-height: 30px", "w3")),
				el("", "four"))),
		"items without a baseline": styled(alignSheet,
			el("display: flex; align-items: last baseline; height: 100px",
				el("width: 30px; height: 40px; background: #ccc; padding: 2px; border: 3px solid; margin: 4px"),
				el("", "text"), el("overflow: hidden; height: 30px", "never shown text"),
				ui.Img(ui.Style("width: 20px; height: 25px; margin-bottom: 6px")))),
	}
	matchAll(t, cases, geom.V2(600, 1000))
}

func TestLastBaselineGridItemsMatchChrome(t *testing.T) {
	grid := func(style string) *ui.Node {
		return el("display: grid; grid-template-columns: repeat(3, 70px); "+style, items()...)
	}
	cases := map[string]*ui.Document{
		"align-items": styled(alignSheet, grid("align-items: last baseline"), grid("align-items: baseline"),
			grid("align-items: last baseline; grid-auto-rows: 120px")),
		"mixed in a row": styled(alignSheet+` .g > :nth-child(2n) { align-self: last baseline }`,
			ui.Div(ui.Class("g"), ui.Style("display: grid; grid-template-columns: repeat(4, 60px); align-items: baseline"),
				el("", "a"), el("font: 32px/45px Go", "b"), el("", "c d e f g h"), el("line-height: 21px", "i j k l m"))),
		"spanning rows": styled(alignSheet,
			el("display: grid; grid-template-columns: 60px 60px; align-items: last baseline",
				el("grid-row: span 2", "two rows of text"), el("", "x"), el("line-height: 25px", "y"))),
		"justify-self: baseline": styled(alignSheet,
			el("display: grid; grid-template-columns: 120px; justify-items: baseline",
				el("width: 50px", "a"), el("width: 30px; padding-left: 9px", "b")),
			el("display: grid; grid-template-columns: 120px; justify-items: last baseline",
				el("width: 50px", "a"), el("width: 30px; margin-right: 7px", "b")),
			el("display: grid; grid-template-columns: 120px; justify-items: last baseline",
				el("writing-mode: vertical-lr", "a b c"), el("writing-mode: vertical-lr; font: 32px/40px Go", "d"))),
	}
	matchAll(t, cases, geom.V2(600, 1000))
}

func TestSafeAlignmentMatchesChrome(t *testing.T) {
	wide := func() []any {
		return []any{el("width: 120px; height: 20px; background: #ccc", ""), el("width: 90px; height: 50px; flex: none")}
	}
	cases := map[string]*ui.Document{
		"flex justify-content": styled(alignSheet,
			el("display: flex; width: 150px; justify-content: safe center", wide()...),
			el("display: flex; width: 150px; justify-content: center", wide()...),
			el("display: flex; width: 150px; justify-content: safe end", wide()...),
			el("display: flex; width: 150px; justify-content: unsafe flex-end", wide()...),
			el("display: flex; width: 150px; flex-direction: row-reverse; justify-content: safe center", wide()...),
			el("display: flex; width: 300px; justify-content: safe center", wide()...)),
		"flex align-items and align-self": styled(alignSheet,
			el("display: flex; height: 30px; align-items: safe center", wide()...),
			el("display: flex; height: 30px; align-items: safe end", wide()...),
			el("display: flex; height: 30px; align-items: unsafe end", wide()...),
			el("display: flex; height: 30px", el("align-self: safe flex-end; height: 50px", "s"),
				el("align-self: center; height: 50px", "c"))),
		"flex align-content": styled(alignSheet,
			el("display: flex; flex-wrap: wrap; width: 100px; height: 50px; align-content: safe center", wide()...),
			el("display: flex; flex-wrap: wrap; width: 100px; height: 50px; align-content: center", wide()...),
			el("display: flex; flex-wrap: wrap; width: 100px; height: 50px; align-content: safe end", wide()...)),
		"grid content": styled(alignSheet,
			el("display: grid; width: 100px; grid-template-columns: 80px 80px; justify-content: safe center", wide()...),
			el("display: grid; width: 100px; grid-template-columns: 80px 80px; justify-content: center", wide()...),
			el("display: grid; height: 40px; grid-template-rows: 30px 30px; align-content: safe end", wide()...),
			el("display: grid; height: 40px; grid-template-rows: 30px 30px; align-content: end", wide()...)),
		"grid items": styled(alignSheet,
			el("display: grid; grid-template-columns: 80px 80px; justify-items: safe center", wide()...),
			el("display: grid; grid-template-columns: 80px 80px; justify-items: safe end", wide()...),
			el("display: grid; grid-template-columns: 80px 80px; justify-items: end", wide()...),
			el("display: grid; grid-template-rows: 30px; align-items: safe end", wide()...),
			el("display: grid; grid-template-rows: 30px; align-items: unsafe center", wide()...)),
		"a block's content": styled(alignSheet,
			el("height: 30px; align-content: safe center", el("height: 60px", "tall")),
			el("height: 30px; align-content: center", el("height: 60px", "tall")),
			el("height: 30px; align-content: safe end", el("height: 60px", "tall"))),
		"a block's justify-self": styled(alignSheet,
			el("width: 100px", el("width: 140px; justify-self: safe center", "wide"),
				el("width: 140px; justify-self: center", "wide"), el("width: 60px; justify-self: safe end", "fits"),
				el("width: 60px; justify-self: last baseline", "end"))),
		"absolutely positioned boxes": styled(alignSheet,
			el("position: relative; width: 100px; height: 60px",
				el("position: absolute; inset: 0; width: 140px; height: 20px; justify-self: safe center"),
				el("position: absolute; inset: 0; width: 140px; height: 20px; margin: auto 0; justify-self: center"),
				el("position: absolute; inset: 0; width: 20px; height: 90px; align-self: safe end"),
				el("position: absolute; inset: 0; width: 20px; height: 90px; align-self: unsafe end")),
			el("position: relative; width: 100px; height: 60px",
				el("position: absolute; inset: 0; width: 20px; height: 20px; align-self: last baseline"),
				el("position: absolute; inset: 0; width: 20px; height: 20px; justify-self: last baseline"),
				el("position: absolute; width: 20px; height: 20px; align-self: last baseline"))),
		"static positions in flex and grid": styled(alignSheet,
			el("display: flex; position: relative; height: 60px; align-items: last baseline",
				el("position: absolute; width: 20px; height: 20px"), el("", "text")),
			el("display: grid; position: relative; height: 60px; justify-items: last baseline",
				el("position: absolute; width: 20px; height: 20px"), el("", "text"))),
	}
	matchAll(t, cases, geom.V2(600, 1200))
}

func TestBaselineAlignmentInVerticalModesMatchesChrome(t *testing.T) {
	cases := map[string]*ui.Document{
		"a flex row in vertical-rl": styled(alignSheet,
			el("writing-mode: vertical-rl; display: flex; align-items: last baseline; height: 300px", items()...),
			el("writing-mode: vertical-lr; display: flex; align-items: last baseline; height: 300px", items()...),
			el("writing-mode: vertical-rl; display: flex; align-items: baseline; height: 300px", items()...)),
		"a grid in vertical-lr": styled(alignSheet,
			el("writing-mode: vertical-lr; display: grid; grid-template-columns: repeat(3, 70px); align-items: last baseline",
				items()...),
			el("writing-mode: vertical-rl; display: grid; grid-template-columns: repeat(3, 70px); align-items: baseline",
				items()...)),
		"orthogonal items": styled(alignSheet,
			el("display: flex; align-items: last baseline",
				el("", "text"), el("writing-mode: vertical-rl; height: 60px", "vertical"),
				el("writing-mode: vertical-lr; line-height: 25px; height: 60px", "lr")),
			el("display: grid; grid-template-columns: repeat(3, auto); align-items: last baseline",
				el("", "text"), el("writing-mode: vertical-rl; height: 60px", "vertical"),
				el("writing-mode: vertical-lr; line-height: 25px; height: 60px", "lr"))),
		"inline-blocks on vertical-lr lines": styled(alignSheet+` .i { display: inline-block }`,
			el("writing-mode: vertical-lr; height: 300px",
				ui.Text("x "), el("display: inline-block; font: 32px/40px Go", "A"),
				ui.Text(" y "), el("display: inline-block; height: 60px; line-height: 21px", "two lines here"),
				ui.Text(" z "), el("display: inline-block; writing-mode: vertical-rl; line-height: 25px; height: 20px", "rl")),
			el("writing-mode: vertical-lr; font: 32px Go",
				el("display: inline-block", "V")),
			el("writing-mode: vertical-rl; height: 300px",
				ui.Text("x "), el("display: inline-block; writing-mode: vertical-lr; height: 60px; line-height: 21px",
					"two lines here"))),
		"safe in vertical-rl": styled(alignSheet,
			el("writing-mode: vertical-rl; display: flex; height: 100px; justify-content: safe center",
				el("height: 150px", "a"), el("", "b")),
			el("writing-mode: vertical-rl; display: grid; width: 40px; align-items: safe end; grid-template-columns: 30px",
				el("width: 60px", "a"))),
	}
	matchAll(t, cases, geom.V2(600, 1400))
}
