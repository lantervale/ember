//go:build chrome

package layout_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// labelled are divs of the classes given, each a word of text.
func labelled(classes ...string) []ui.Arg {
	var out []ui.Arg
	for _, c := range classes {
		out = append(out, ui.Div(ui.Class(c), ui.Text(c)))
	}
	return out
}

func gridOf(kids ...ui.Arg) *ui.Node {
	return ui.Div(append([]ui.Arg{ui.Class("g")}, kids...)...)
}

func TestGridShorthandLaysOutAsInChrome(t *testing.T) {
	const page = `.g { display: grid; width: 500px; gap: 4px; margin-bottom: 8px; grid-auto-rows: 7px;
	grid-auto-columns: 9px; grid-auto-flow: column }
.g > * { min-width: 0; background: #ddd }`
	cases := map[string]*ui.Document{
		"rows / columns": styled(page+`.g { grid: 30px 1fr / 100px repeat(2, 1fr) }`,
			gridOf(labelled("a", "b", "c", "d", "e", "f", "g", "h")...)),
		"areas": styled(page+`.g { grid: [top] 'h h' 30px [mid] 's m' auto / 120px 1fr }
.a { grid-area: h } .b { grid-row: mid }`,
			gridOf(labelled("a", "b", "c", "d")...)),
		"rows / auto-flow": styled(page+`.g { grid: 20px 40px / auto-flow 60px 1fr }`,
			gridOf(labelled("a", "b", "c", "d", "e", "f", "g")...)),
		"rows / auto-flow dense": styled(page+`.g { grid: repeat(3, 20px) / dense auto-flow 50px } .b { grid-row: span 2 }`,
			gridOf(labelled("a", "b", "c", "d", "e", "f")...)),
		"auto-flow / columns": styled(page+`.g { grid: auto-flow 25px 35px / 1fr 2fr 80px }`,
			gridOf(labelled("a", "b", "c", "d", "e", "f", "g")...)),
		"auto-flow dense / columns": styled(page+`.g { grid: auto-flow dense / repeat(4, 1fr) } .a { grid-column: span 3 }`,
			gridOf(labelled("a", "b", "c", "d", "e")...)),
		"none": styled(page+`.g { grid-template-columns: 1fr 1fr; grid: none }`,
			gridOf(labelled("a", "b", "c")...)),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			assert.Empty(t, d.Warnings())
		})
	}
}

func TestSubgridsLayOutAsInChrome(t *testing.T) {
	const page = `.g { display: grid; grid-template-columns: auto auto auto; gap: 10px; margin-bottom: 10px;
	justify-content: start; position: relative }
.s { display: grid; grid-template-columns: subgrid; grid-column: span 2; background: #eee }
.x { width: 30px; height: 10px; background: #888 }`
	x := func(style string) *ui.Node { return ui.Div(ui.Class("x"), ui.Style(style)) }
	sub := func(style string, kids ...ui.Arg) *ui.Node {
		return ui.Div(append([]ui.Arg{ui.Class("s"), ui.Style(style)}, kids...)...)
	}
	grid := func(style string, kids ...ui.Arg) *ui.Node {
		return ui.Div(append([]ui.Arg{ui.Class("g"), ui.Style(style)}, kids...)...)
	}
	cases := map[string]*ui.Document{
		"columns": styled(page, grid("width: 400px; justify-content: normal",
			sub("", x("grid-column: 2"), x("")), x("")),
			grid("", sub("", x(""), x("width: 50px"), x("width: 70px")), x(""))),
		"padding on an empty edge track": styled(page,
			grid("width: 400px; justify-content: normal", sub("padding-left: 50px", x("grid-column: 2")), x("")),
			grid("", sub("padding-left: 50px", x("grid-column: 2")), x("")),
			grid("", sub("padding-left: 50px", x("grid-column: 1")), x("")),
			grid("", sub("padding-left: 50px; grid-column: span 3", x("grid-column: 2 / 4; width: 200px"))),
			grid("", sub("padding: 0 20px"), x(""))),
		"margin, border and padding at both edges": styled(page,
			grid("", sub("margin: 2px 6px; border: 3px solid; padding: 4px 5px", x(""), x("")), x("")),
			grid("", sub("margin-left: 12px; padding-right: 9px; grid-column: 2 / 4", x(""), x("")), x(""))),
		"its own gap": styled(page,
			grid("", sub("padding-left: 50px; column-gap: 30px", x(""), x("")), x("")),
			grid("", sub("column-gap: 30px", x("grid-column: 2")), x("")),
			grid("", sub("column-gap: 0; grid-column: span 3", x(""), x(""), x("")), x(""), x(""), x(""))),
		"nested": styled(page, grid("",
			sub("padding: 5px; border: 3px solid; margin: 2px 4px; grid-column: span 3",
				sub("padding: 0 7px; grid-column: 2 / 4", x("grid-column: 2"), x("grid-column: 1"))))),
		"rows": styled(page, grid("grid-template-rows: auto auto",
			sub("grid-template-rows: subgrid; grid-row: span 2; padding-top: 15px",
				x("height: 20px"), x("height: 40px"), x("")), x(""), x(""))),
		"rows only": styled(page+`.r { display: grid; grid-template-rows: subgrid; grid-row: span 3;
	grid-template-columns: 40px 40px; background: #eee }`, grid("grid-template-rows: 15px auto auto",
			ui.Div(ui.Class("r"), x("grid-row: 2; height: 35px"), x("grid-column: 2; grid-row: 3"), x("grid-row: 1")),
			x("grid-row: 3"), x("grid-row: 2"))),
		"items placed past the subgrid": styled(page, grid("",
			sub("", x("grid-column: 5"), x("grid-column: -5"), x(""), x(""), x("")), x(""))),
		"line names": styled(page+`.g { grid-template-columns: [a] auto [b] auto [c] auto [d] }
.n { grid-template-columns: subgrid [x] [y] [z]; grid-column: b / d }`, grid("",
			x("grid-column: a"), ui.Div(ui.Class("s", "n"), x("grid-column: c"), x("grid-column: y / z"),
				x("grid-column: x"), x("grid-column: c / d; width: 80px"))),
			grid("", ui.Div(ui.Class("s"),
				ui.Style("grid-template-columns: subgrid repeat(auto-fill, [r]); grid-column: span 3"),
				x("grid-column: r 2"), x("grid-column: r 3 / r 4; width: 60px")))),
		"areas": styled(page+`.g { grid-template-areas: 'l c r' }
.n { grid-column: 1 / 3; grid-template-areas: 'p q' }`, grid("",
			ui.Div(ui.Class("s", "n"), x("grid-column: c"), x("grid-area: p"), x("grid-column: q-start; width: 50px")),
			x("grid-area: r"))),
		"text sizing auto tracks": styled(page+`.s > div { background: #ccc }`, grid("",
			sub("", ui.Div(ui.Text("a longer label")), ui.Div(ui.Text("v"))),
			sub("", ui.Div(ui.Text("x")), ui.Div(ui.Text("another value here"))),
			ui.Div(ui.Text("end")))),
		"fr tracks": styled(page, grid("width: 500px; grid-template-columns: 1fr 2fr 100px",
			sub("padding: 0 10px", x("width: 200px"), x("")), x(""))),
		"both axes": styled(page+`.s { grid-template: subgrid / subgrid; grid-area: 1 / 1 / 3 / 3; padding: 6px }`,
			grid("grid-template-rows: auto auto auto", sub("", x("height: 30px"), x(""), x(""), x("height: 25px")),
				x(""), x("grid-row: 2"), x("grid-row: 3"))),
		"absolute in a subgrid": styled(page+`.s { position: relative } .a { position: absolute; inset: 0 }`, grid("",
			sub("", x(""), x("width: 60px"), ui.Div(ui.Class("x", "a"), ui.Style("grid-column: 2"))), x(""))),
		"items with margins, spanning": styled(page, grid("",
			sub("grid-column: span 3", x("margin-left: 25px"), x("grid-column: span 2; width: 90px; margin: 0 5px 0 15px")),
			x(""), x("margin-right: 40px"), x(""))),
		"content alignment of the grid": styled(page+`.g { width: 500px; height: 120px }`,
			grid("justify-content: space-between", sub("", x(""), x("")), x("")),
			grid("justify-content: center; align-content: space-around; grid-template-rows: auto auto",
				sub("grid-template-rows: subgrid; grid-row: span 2", x(""), x(""), x(""), x("")), x(""))),
		"self alignment and a size, set aside": styled(page, grid("",
			sub("justify-self: center; width: 20px; max-width: 10px", x(""), x("width: 60px")), x(""))),
		"past the explicit grid": styled(page+`.g { grid-template-columns: 50px; grid-auto-columns: auto }`, grid("",
			sub("grid-column: 2 / span 3", x(""), x("width: 45px"), x("width: 15px")), x("grid-column: 1"))),
		"column flow and a nested rows subgrid": styled(page+`.c { display: grid; grid-auto-flow: column;
	grid-template-rows: subgrid; grid-row: span 2; grid-template-columns: subgrid; grid-column: span 2; row-gap: 2px }`,
			grid("grid-template-rows: auto auto auto", ui.Div(ui.Class("c"), x("height: 20px"), x(""), x("height: 33px")),
				x("grid-row: 3"))),
		"overflow hidden, in a flex row": styled(page+`.f { display: flex } .s { overflow: hidden }`,
			ui.Div(ui.Class("f"), grid("", sub("padding: 0 3px", ui.Div(ui.Text("one two")), ui.Div(ui.Text("three"))),
				ui.Div(ui.Text("four five six"))))),
		"margin, border and padding wider than a track": styled(page+`.g { grid-template-columns: 40px 30px 50px }`,
			grid("", sub("padding: 0 60px 0 45px; grid-column: 1 / 3", x("width: 10px"), x("width: 10px")), x(""))),
		"percentages of the grid": styled(page+`.g { width: 400px; grid-template-columns: auto 1fr auto }`,
			grid("", sub("padding: 0 5% 0 10%; margin-left: 2%", x(""), x("")), x(""))),
		"not subgrids: a query container, out of flow, no grid round it": styled(page+`.q { container-type: inline-size }
.lone { display: grid; grid-template-columns: subgrid; width: 200px } .ab { position: absolute; top: 0; left: 0 }`,
			grid("", ui.Div(ui.Class("s", "q"), x(""), x("width: 50px")), x("")),
			grid("", ui.Div(ui.Class("s", "ab"), x(""), x("width: 50px")), x(""), x("")),
			ui.Div(ui.Class("lone"), x(""), x("")),
			ui.Div(ui.Style("display: flex"), ui.Div(ui.Class("lone"), x(""), x("")))),
		"rows only, text wrapping in its own columns": styled(page+`.r { display: grid; grid-template-rows: subgrid;
	grid-row: span 2; grid-template-columns: 50px 1fr; column-gap: 4px }`,
			grid("grid-template-columns: 220px 100px; grid-template-rows: auto auto",
				ui.Div(ui.Class("r"), ui.Div(ui.Text("several short words wrap here")), ui.Div(ui.Text("one line")),
					ui.Div(ui.Style("grid-column: 2"), ui.Text("a longer text that wraps in its column too"))),
				ui.Div(ui.Text("beside")), ui.Div(ui.Style("grid-column: 2"), ui.Text("below")))),
		"fr tracks of a grid as wide as its content": styled(page+`
.g { display: inline-grid; grid-template-columns: 1fr 2fr auto }`,
			grid("", sub("", ui.Div(ui.Text("alpha beta")), ui.Div(ui.Text("gamma"))), ui.Div(ui.Text("delta")))),
		"auto-fill tracks, the subgrid across them all": styled(page+`.g { width: 330px;
	grid-template-columns: [start] repeat(auto-fill, [c] 60px) [end] }`, grid("",
			sub("grid-column: start / end; grid-template-columns: subgrid repeat(auto-fill, [k])",
				x("grid-column: k 2"), x("grid-column: k 3 / span 2"), x(""), x(""), x(""), x(""), x("")))),
		"a rows subgrid inside a subgrid of both": styled(page+`.b { display: grid; grid-template: subgrid / subgrid;
	grid-area: span 2 / span 3; padding: 4px }
.r { display: grid; grid-template-rows: subgrid; grid-row: span 2; grid-template-columns: 40px auto;
	padding-bottom: 6px }`,
			grid("grid-template-rows: auto auto", ui.Div(ui.Class("b"), x(""), ui.Div(ui.Class("r"),
				ui.Div(ui.Text("one two three four")), x("height: 30px"), x("grid-column: 1 / 3")), x("grid-row: 2")),
				x("grid-row: 2"))),
		"no grid round it": styled(page+`.lone { display: grid; grid-template-columns: subgrid; width: 200px }`,
			ui.Div(ui.Class("lone"), x(""), x(""))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			assert.Empty(t, d.Warnings())
		})
	}
}

// Items aligned on their baseline in a subgrid sharing its grid's rows align with the grid's
// own in the same row, their subgrid's edges counted.
func TestSubgridBaselinesAsInChrome(t *testing.T) {
	const page = `.g { display: grid; grid-template-columns: auto auto auto; column-gap: 10px; margin-bottom: 10px;
	align-items: baseline; justify-content: start }
.s { display: grid; grid-column: span 2; background: #eee }
.big { font-size: 30px } .small { font-size: 12px }`
	text := func(class, s string) *ui.Node { return ui.Div(ui.Class(class), ui.Text(s)) }
	sub := func(style string, kids ...ui.Arg) *ui.Node {
		return ui.Div(append([]ui.Arg{ui.Class("s"), ui.Style(style)}, kids...)...)
	}
	grid := func(style string, kids ...ui.Arg) *ui.Node {
		return ui.Div(append([]ui.Arg{ui.Class("g"), ui.Style(style)}, kids...)...)
	}
	cases := map[string]*ui.Document{
		"rows and columns": styled(page, grid("",
			text("big", "Big"), sub("grid-template: subgrid / subgrid; padding-top: 7px",
				text("small", "small"), text("", "mid")))),
		"rows alone": styled(page, grid("grid-template-rows: auto auto",
			sub("grid-template-rows: subgrid; grid-row: span 2; grid-template-columns: auto auto; border-top: 5px solid",
				text("small", "one"), text("", "two"), text("big", "three")), text("big", "Big"))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			assert.Empty(t, d.Warnings())
		})
	}
}

// In a subgrid of columns alone, an item's width that follows from its height takes that
// height from the subgrid's own rows when it sizes the grid's columns.
func TestSubgridItemsSizedByTheirRowsAsInChrome(t *testing.T) {
	const page = `.g { display: grid; grid-template-columns: auto auto 40px; column-gap: 10px; margin-bottom: 10px;
	justify-content: start }
.s { display: grid; grid-template-columns: subgrid; grid-column: span 2; grid-template-rows: 60px 30px;
	background: #eee }
.x { background: #888 }`
	x := func(style string) *ui.Node { return ui.Div(ui.Class("x"), ui.Style(style)) }
	cases := map[string]*ui.Document{
		"aspect ratio, a grid alone": styled(page, ui.Div(ui.Class("g"), ui.Style("grid-template-rows: 60px 30px"),
			x("aspect-ratio: 1; justify-self: start"), x("grid-row: 2; aspect-ratio: 2; justify-self: start"))),
		"aspect ratio": styled(page, ui.Div(ui.Class("g"),
			ui.Div(ui.Class("s"), x("aspect-ratio: 1; justify-self: start"),
				x("grid-row: 2; aspect-ratio: 2; justify-self: start")),
			x("height: 10px"))),
		"percentage height": styled(page, ui.Div(ui.Class("g"),
			ui.Div(ui.Class("s"), x("height: 50%; aspect-ratio: 2; justify-self: start"), x("")), x("height: 10px"))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
