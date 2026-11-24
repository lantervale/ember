//go:build chrome

package layout_test

import (
	"strconv"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

// WPT css/css-grid/subgrid/auto-track-sizing-002.html: text in subgrids of rows nested in
// subgrids of rows, in one or both axes, sizing the grid's auto row; in Go Mono for
// monospace, the face ui and Chrome share on every system.
func TestWPTSubgridAutoTrackSizing002MatchesChrome(t *testing.T) {
	const sheet = `html, body { color: black; background-color: white; font: 16px/1 "Go Mono"; padding: 0; margin: 0 }
.grid { display: inline-grid; max-width: 260px; padding: 1px 5px; border: 1px solid; grid-template-columns: 100px }
.subgrid { display: grid; grid-template-rows: subgrid; grid-template-columns: 100%; margin: 0px 5px 0px 10px }
.inner { margin: 0px 5px 0px 0px; background: grey }`
	const text = "The cat can not be separated from milk"
	nested := func(outer, inner string) *ui.Node {
		return ui.Article(ui.Class("grid"), ui.Div(ui.Class("subgrid"), ui.Style(outer),
			ui.Div(ui.Class("subgrid", "inner"), ui.Style(inner), ui.Text(text))))
	}
	const cols = "grid-template-columns: subgrid"
	matchPages(t, geom.V2(800, 600), map[string]*ui.Document{
		"test": wptDoc(sheet,
			ui.Article(ui.Class("grid"), ui.Div(ui.Class("subgrid"),
				ui.Style("margin: 0px 10px 0px 10px; background: grey"), ui.Text(text))),
			nested("", ""), nested(cols, ""), nested("", cols), nested(cols, cols)),
	})
}

// WPT css/css-grid/subgrid/writing-directions-002.html: a left to right subgrid in a right
// to left one, the margin, border and padding of each on its own side. Its n and x elements
// are spans, its monospace sans-serif: Chrome stands the generated numbers of its monospace
// a point lower.
func TestWPTSubgridWritingDirections002MatchesChrome(t *testing.T) {
	const sheet = `html, body { font: 12px/1 sans-serif }
.grid { display: grid; grid: 0.2em 1.4em / repeat(10, auto); border: 1px solid; padding: 0 0 0 0 }
div > div { display: grid; grid-column: 1 / span 3; grid: auto / subgrid; border: 1px solid; background: grey }
.n { grid-row: 1; counter-increment: n }
.n::before { content: counter(n, decimal) }
.x { background: silver }
.rtl { direction: rtl }
.ltr { direction: ltr }`
	grid := func(outer, inner string) *ui.Node {
		args := []ui.Arg{ui.Class("grid")}
		for range 10 {
			args = append(args, ui.Span(ui.Class("n")))
		}
		return ui.Div(append(args, ui.Div(ui.Class("rtl"), ui.Style("grid-template-columns: subgrid; "+outer),
			ui.Div(ui.Class("ltr"), ui.Style("grid-template-columns: subgrid; "+inner),
				ui.Span(ui.Class("x"), ui.Style("grid-column: 1"), ui.Text("x")))))...)
	}
	matchPages(t, geom.V2(800, 600), map[string]*ui.Document{
		"test": wptDoc(sheet,
			grid("padding: 0 0 0 10px", ""),
			grid("", "padding: 0 0 0 10px"),
			grid("border-right: 20px solid; border-left: 10px solid", ""),
			grid("", "border-right: 20px solid; border-left: 10px solid"),
			grid("margin: 0 20px 0 0", ""),
			grid("", "margin: 0 20px 0 0")),
	})
}

// WPT css/css-grid/subgrid/orthogonal-writing-mode-001.html: subgrids of a vertical writing
// mode in a horizontal grid, their subgridded axes the grid's other ones. Its x elements are
// spans, in Go Mono for monospace, the face ui and Chrome share on every system.
func TestWPTSubgridOrthogonalWritingMode001MatchesChrome(t *testing.T) {
	const sheet = `html, body { color: black; background-color: white; font: 12px/1 "Go Mono" }
.grid { display: grid; grid: repeat(4, auto) / repeat(5, auto); place-content: start; border: 1px solid }
.subgrid { display: grid; grid: subgrid / auto; background: lightgrey; grid-column: 2 / span 5;
grid-row: 2 / span 5; min-width: 10px; min-height: 0; border: 0 solid lightblue; border-bottom-width: 40px }
.vlr { writing-mode: vertical-lr; grid: auto / subgrid; grid-column: 2 / span 5; grid-row: 3 / span 5;
background: blue }
.x { min-width: 10px; min-height: 0px }
.x:nth-child(2n+1) { background: silver }
.x:nth-child(2n+2) { background: grey }
.x:nth-child(2n+3) { background: pink }
.x:nth-child(2n+4) { background: black }
.hl { writing-mode: horizontal-tb; direction: ltr }
.hr { writing-mode: horizontal-tb; direction: rtl }
.vl { writing-mode: vertical-lr }
.vr { writing-mode: vertical-rl }
.vlr { writing-mode: vertical-lr; direction: rtl }
.vrl { writing-mode: vertical-rl; direction: ltr }`
	grid := func(sub *ui.Node) *ui.Node {
		args := []ui.Arg{ui.Class("grid")}
		for r := 1; r <= 6; r++ {
			args = append(args, ui.Span(ui.Class("x"), ui.Style("grid-row: "+strconv.Itoa(r))))
		}
		return ui.Div(append(args, sub)...)
	}
	matchPages(t, geom.V2(800, 600), map[string]*ui.Document{
		"test": wptDoc(sheet,
			grid(ui.Div(ui.Class("subgrid", "vrl"), ui.Style("text-indent: 10px"), ui.Text("A B"))),
			grid(ui.Div(ui.Class("subgrid", "vr"), ui.Div(ui.Class("subgrid", "hr"), ui.Text("A B")))),
			grid(ui.Div(ui.Class("subgrid", "vlr"), ui.Div(ui.Class("subgrid", "hl"),
				ui.Span(ui.Class("x"), ui.Style("grid-row: 2 / span 2; height: 100px"), ui.Text("A B C D E F")))))),
	})
}

// WPT css/css-grid/subgrid/orthogonal-writing-mode-006.html: an item of each writing mode in
// a subgrid of both axes of each writing mode, in a grid of each, in sans-serif for Ahem.
func TestWPTSubgridOrthogonalWritingMode006MatchesChrome(t *testing.T) {
	const sheet = `body { background: white; color: black; display: grid; font: 20px/1 sans-serif; place-items: start;
padding: 0; margin: 0 }
.grid { background: cyan; display: grid; padding: 5px }
.grid > .grid { background: pink; grid-template: subgrid / subgrid }
.h { writing-mode: horizontal-tb }
.v { writing-mode: vertical-rl }`
	var nodes []*ui.Node
	for _, outer := range []string{"h", "v"} {
		for _, sub := range []string{"h", "v"} {
			for _, item := range []string{"h", "v"} {
				nodes = append(nodes, ui.Div(ui.Class("grid", outer),
					ui.Div(ui.Class("grid", sub), ui.Div(ui.Class(item), ui.Text("1234")))))
			}
		}
	}
	matchPages(t, geom.V2(800, 600), map[string]*ui.Document{"test": wptDoc(sheet, nodes...)})
}
