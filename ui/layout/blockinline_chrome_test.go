//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestBlocksInsideInlinesMatchChrome(t *testing.T) {
	const page = `.wide { width: 300px; } .b { margin: 10px 0; background: #ccd; }`
	cases := map[string]*ui.Document{
		"a block cuts the inline holding it": styled(page,
			ui.Div(ui.Class("wide"),
				ui.Span(ui.Text("Before the block. "), ui.Div(ui.Class("b"), ui.Text("The block.")), ui.Text(" After it,")),
				ui.Text(" and the rest of the line.")),
		),
		"a framed inline keeps its start before the block and its end after": styled(
			page+` .f { border: 2px solid #a00; padding: 0 5px; margin: 0 3px; }`,
			ui.Div(ui.Class("wide"),
				ui.Text("Head "), ui.Span(ui.Class("f"), ui.Text("before"), ui.Div(ui.Class("b"), ui.Text("The block.")),
					ui.Text("after")), ui.Text(" tail.")),
		),
		"a block in nested inlines": styled(page,
			ui.Div(ui.Class("wide"),
				ui.Span(ui.Text("One "), ui.Em(ui.Text("two"), ui.Div(ui.Class("b"), ui.Text(sentence)), ui.Text("three")),
					ui.Text(" four"))),
		),
		"two blocks and the line between them": styled(page,
			ui.Div(ui.Class("wide"),
				ui.Span(ui.Div(ui.Class("b"), ui.Text("First.")), ui.Text("Between."),
					ui.Div(ui.Class("b"), ui.Text("Second.")))),
		),
		"a list in an inline nav": styled(`nav { display: inline; } ul { margin: 0; padding: 0; list-style: none; overflow: hidden; }
li { margin: 0 30% 0 15%; padding: 1em 0; border-top: 1px solid #999; } li a { display: block; margin-bottom: 4px; }`,
			ui.Div(ui.Class("wide"), ui.Nav(ui.Text("\n"), ui.Ul(
				ui.Li(ui.A(ui.Attr("href", "#"), ui.Text("Mid Century Modern")), ui.Text(" by "), ui.A(ui.Attr("href", "#"), ui.Text("Andrew"))),
				ui.Li(ui.A(ui.Attr("href", "#"), ui.Text("Garments")), ui.Text(" by "), ui.A(ui.Attr("href", "#"), ui.Text("Dan"))),
			), ui.Text("\n"))),
		),
		"a footer clears the floats of the inline holding them": styled(
			`main { display: inline; } main div { float: left; clear: left; width: 66%; padding: 8px 5% 8px 10%; }
footer { clear: both; padding: 20px 10%; text-align: center; background: #134; }`,
			ui.Div(ui.Style("width: 600px"),
				ui.Main(ui.Text("\n"),
					ui.Div(ui.Text(sentence)), ui.Text("\n"), ui.Div(ui.Text(sentence+" "+sentence)), ui.Text("\n"),
					ui.Footer(ui.A(ui.Attr("href", "#"), ui.Text("One")), ui.Text(" "), ui.A(ui.Attr("href", "#"), ui.Text("Two"))),
					ui.Text("\n")),
				ui.Aside(ui.Text("The aside."))),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
