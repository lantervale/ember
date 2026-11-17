//go:build chrome

package layout_test

import (
	"fmt"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// squeezed is build's paragraph at each width from 20px to 150px, so every break it can
// take is taken in one of them.
func squeezed(build func(width string) *ui.Node) []*ui.Node {
	var out []*ui.Node
	for w := 20; w <= 150; w += 3 {
		out = append(out, build(fmt.Sprintf("margin: 2px 0; width: %dpx", w)))
	}
	return out
}

func TestSpacesAtLineEdgesMatchChrome(t *testing.T) {
	const padded = "padding: 0 5px; border: 1px solid #888; margin: 0 3px"
	// Chrome gives a span that paints nothing of its own a rectangle per child on a line,
	// not one per line; a background makes it a box.
	const boxed = "background: #ddd"
	cases := map[string]*ui.Document{
		"a leading space at the line's start": page(
			ui.P(ui.Span(ui.Text("  lead")), ui.Text(" after")),
			ui.P(ui.Span(ui.Style(padded), ui.Text("  lead")), ui.Text(" after")),
		),
		"a trailing space at the line's end": page(
			ui.P(ui.Text("before "), ui.Span(ui.Text("trail   "))),
			ui.P(ui.Text("before "), ui.Span(ui.Style(padded), ui.Text("trail   "))),
			ui.P(ui.Text("before "), ui.Span(ui.Text("trail")), ui.Text(" ")),
			ui.P(ui.Text("before "), ui.Span(ui.Text("trail ")), ui.Text(" "), ui.Span(ui.Text(" "))),
		),
		"both": page(
			ui.P(ui.Span(ui.Text(" both "))),
			ui.P(ui.Span(ui.Style(padded), ui.Text(" both "))),
			ui.Div(ui.Span(ui.Text(" one ")), ui.Span(ui.Text(" two "))),
		),
		"in the middle of a line": page(
			ui.P(ui.Text("a"), ui.Span(ui.Text(" mid ")), ui.Text("b")),
			ui.P(ui.Text("a "), ui.Span(ui.Text(" mid ")), ui.Text(" b")),
			ui.P(ui.Text("a"), ui.Span(ui.Style(padded), ui.Text(" mid ")), ui.Text("b")),
			ui.P(ui.Text("a"), ui.Span(ui.Text("mid ")), ui.Span(ui.Text(" next ")), ui.Text("b")),
		),
		"nested spans": page(
			ui.P(ui.Span(ui.Text(" "), ui.Span(ui.Text(" in ")), ui.Text(" "))),
			ui.P(ui.Span(ui.Span(ui.Span(ui.Text(" deep "))))),
			ui.P(ui.Span(ui.Style(padded), ui.Text(" out "), ui.Span(ui.Style(padded), ui.Text(" in ")))),
			ui.P(ui.Text("a "), ui.Span(ui.Style(boxed), ui.Text("out "), ui.Span(ui.Text("in "))), ui.Text("b"),
				ui.Span(ui.Style(boxed), ui.Text(" c "), ui.Span(ui.Text(" d ")))),
		),
		"a trailing space at a soft wrap": page(squeezed(func(width string) *ui.Node {
			return ui.P(ui.Style(width), ui.Text("aaa "), ui.Span(ui.Text(" bbb ccc ")), ui.Text("ddd eee"))
		})...),
		"a leading space at a soft wrap": page(squeezed(func(width string) *ui.Node {
			return ui.P(ui.Style(width), ui.Text("aaa"), ui.Span(ui.Text(" bbb ccc")), ui.Text(" ddd"),
				ui.Span(ui.Style("padding: 0 4px"), ui.Text(" eee ")), ui.Text("fff"))
		})...),
		"only white space": page(
			ui.P(ui.Text("a"), ui.Span(ui.Text(" ")), ui.Text("b")),
			ui.P(ui.Span(ui.Text(" ")), ui.Text("b")),
			ui.P(ui.Text("a"), ui.Span(ui.Text("  "))),
			ui.P(ui.Text("a"), ui.Span(ui.Style(padded), ui.Text("  "))),
			ui.P(ui.Text("a "), ui.Span(ui.Text(" ")), ui.Text(" b")),
			ui.P(ui.Text("a"), ui.Span(ui.Span(ui.Text(" "))), ui.Span(ui.Text(" "))),
		),
		"pre": page(
			ui.Div(ui.Style("white-space: pre"), ui.Span(ui.Text("  pre  "))),
			ui.Div(ui.Style("white-space: pre"), ui.Text("a"), ui.Span(ui.Text("  pre  ")), ui.Text("b")),
			ui.Div(ui.Span(ui.Style("white-space: pre"), ui.Text("  pre  "))),
		),
		"pre-wrap": page(
			ui.Div(ui.Style("white-space: pre-wrap"), ui.Span(ui.Style(boxed), ui.Text("  kept  "))),
			ui.Div(ui.Style("white-space: pre-wrap"), ui.Text("a"), ui.Span(ui.Text("  kept  ")), ui.Text("b")),
			ui.Div(ui.Span(ui.Style("white-space: pre-wrap; "+boxed), ui.Text("  kept  "))),
		),
		"pre-line and nowrap": page(
			ui.Div(ui.Style("white-space: pre-line"), ui.Span(ui.Style(boxed), ui.Text("  one  \n  two  "))),
			ui.Div(ui.Style("white-space: nowrap"), ui.Span(ui.Text("  one  ")), ui.Span(ui.Text("  two  "))),
		),
		"an inline-block shrinks to fit": page(
			ui.Div(ui.Div(ui.Style("display: inline-block; border: 1px solid"), ui.Span(ui.Text(" shrink ")))),
			ui.Div(ui.Div(ui.Style("display: inline-block; border: 1px solid"),
				ui.Text("a "), ui.Span(ui.Text(" shrink to fit ")), ui.Text(" "))),
			ui.Div(ui.Style("display: flex"), ui.Div(ui.Span(ui.Text(" item "))), ui.Div(ui.Span(ui.Text(" item ")))),
			ui.Div(ui.Style("width: min-content; border: 1px solid"), ui.Span(ui.Text(" two words "))),
			ui.Div(ui.Style("float: left; border: 1px solid"), ui.Span(ui.Style(padded), ui.Text(" floated "))),
			ui.Div(ui.Style("clear: left; width: min-content"),
				ui.Text("aaa"), ui.Span(ui.Style("padding-left: 20px"), ui.Text(" b"))),
			ui.Div(ui.Style("width: min-content"),
				ui.Text("a "), ui.Span(ui.Style("padding-right: 30px"), ui.Text("b ")), ui.Text("c")),
			ui.Div(ui.Style("width: min-content"),
				ui.Text("a "), ui.Span(ui.Style("padding: 0 30px"), ui.Text(" b ")), ui.Text(" c")),
		),
		"text-align": page(
			ui.P(ui.Style("text-align: right"), ui.Span(ui.Text(" right "))),
			ui.P(ui.Style("text-align: center"), ui.Span(ui.Text(" center "))),
			ui.P(ui.Style("text-align: right; width: 90px"), ui.Text("aaa "), ui.Span(ui.Text(" bbb ccc ")), ui.Text("ddd eee")),
			ui.P(ui.Style("text-align: center; width: 90px"), ui.Text("aaa "), ui.Span(ui.Text(" bbb ccc ")), ui.Text("ddd eee")),
			ui.P(ui.Style("text-align: justify; width: 90px"), ui.Text("aaa "), ui.Span(ui.Text(" bbb ccc ")),
				ui.Text("ddd eee "), ui.Span(ui.Text(" fff ggg hhh ")), ui.Text("iii")),
			ui.P(ui.Style("text-align: justify; width: 120px"), ui.Span(ui.Text(" "+sentence+" "))),
		),
		"a link alone in a list item": page(
			ui.Div(ui.Style("width: 200px"),
				ui.Div(ui.Span(ui.Text("\n\t\t\t\t\tView All Designs\t\t\t\t"))),
				ui.Div(ui.Span(ui.Text("\n\t\t\t\t\tCSS Resources\t\t\t\t")), ui.Text("\n\t\t\t")),
				ui.Div(ui.Text("\n\t\t\t"), ui.Span(ui.Text("\n\t\t\t\t\tFAQ\t\t\t\t")), ui.Text("\n\t\t\t"))),
		),
		"beside a float": styled(".l { float: left; width: 60px; height: 50px; }",
			ui.Div(ui.Style("width: 200px"), ui.Div(ui.Class("l")),
				ui.P(ui.Style("margin: 0"), ui.Span(ui.Text(" aaa bbb ")), ui.Span(ui.Text(" ccc ddd eee fff ")),
					ui.Text("ggg hhh "), ui.Span(ui.Text(" iii jjj kkk lll mmm nnn ")))),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			matchText(t, d, geom.V2(800, 600), 0.5)
			matchSpanFragments(t, d, geom.V2(800, 600), 0.5)
		})
	}
}
