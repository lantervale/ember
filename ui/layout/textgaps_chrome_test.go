//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestTextSpacingAndShiftsMatchChrome(t *testing.T) {
	slab := func(h string) ui.Arg { return ui.Label(ui.Style("height: " + h)) }
	shifted := func(outer, inner string, held ...ui.Arg) *ui.Node {
		return ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: "+outer), ui.Text("y"),
			ui.Span(append([]ui.Arg{ui.Style(inner), ui.Text("z")}, held...)...)))
	}
	const slabs = "p { font-size: 20px; line-height: 30px; margin: 0 } " +
		"p label { display: inline-block; width: 10px; background: #888; }"
	cases := map[string]*ui.Document{
		"spaces Go lacks": page(
			ui.P(ui.Span(ui.Text("a\u2003b\u2009c\u2002d\u2004e\u2005f\u2006g"))),
			ui.P(ui.Span(ui.Text("h\u2000i\u2001j\u2007k\u2008l\u200am\u202fn\u205fo"))),
			ui.P(ui.Style("font-size: 24px"), ui.Span(ui.Text("1\u20032\u20093"))),
			ui.P(ui.Style("font-weight: bold"), ui.Span(ui.Text("q\u2003r\u2009s"))),
			ui.P(ui.Style("font-family: Go Mono; font-style: italic"), ui.Span(ui.Text("t\u2003u\u2009v"))),
		),
		"word-spacing on a block's first space": page(
			ui.Div(ui.Style("word-spacing: 10px"), ui.Span(ui.Style("white-space: pre"), ui.Text(" a b"))),
			ui.Div(ui.Style("word-spacing: 10px"), ui.Span(ui.Style("white-space: pre"), ui.Text("  a"))),
			ui.Div(ui.Style("word-spacing: 10px"), ui.Text("x"), ui.Span(ui.Style("white-space: pre"), ui.Text(" a"))),
			ui.Div(ui.Style("word-spacing: 10px; white-space: pre"), ui.Text(" a b ")),
			ui.Div(ui.Style("word-spacing: 10px"), ui.Text(" "), ui.Span(ui.Style("white-space: pre"), ui.Text(" a"))),
			ui.Div(ui.Style("word-spacing: 10px; white-space: pre-wrap"), ui.Text(" a b")),
			ui.Div(ui.Style("word-spacing: 10px; white-space: pre"), ui.Span(ui.Text(" a b"))),
			ui.Div(ui.Style("word-spacing: 10px"),
				ui.Span(ui.Style("white-space: pre-wrap; background: #ddd"), ui.Text(" a b"))),
			ui.Div(ui.Style("word-spacing: 10px; white-space: pre"),
				ui.Span(ui.Style("white-space: normal"), ui.Text("x")), ui.Text(" a")),
			ui.Div(ui.Style("word-spacing: 10px; white-space: pre"), ui.Text("a\n b")),
		),
		"a top or bottom box holding a shifted one": aligning(slabs,
			shifted("top", "vertical-align: 12px", slab("50px")),
			shifted("top", "vertical-align: 12px", slab("10px")),
			shifted("top", "vertical-align: 12px"),
			shifted("top", "vertical-align: 30px"),
			shifted("top", "", slab("50px")),
			shifted("top", "vertical-align: -12px", slab("50px")),
			shifted("bottom", "vertical-align: -12px", slab("50px")),
			shifted("bottom", "vertical-align: 12px", slab("50px")),
			shifted("top", "vertical-align: 12px", ui.Span(ui.Text("w"), slab("10px"))),
			shifted("top", "vertical-align: 12px", ui.Span(ui.Style("vertical-align: 5px"), ui.Text("w"), slab("10px"))),
			shifted("top", "vertical-align: 12px; padding: 2px; border: 1px solid"),
			shifted("top", "vertical-align: 12px; background: #ddd"),
			shifted("top", "vertical-align: 12px; font-size: 30px", slab("10px")),
			shifted("top", "vertical-align: sub", slab("10px")),
			shifted("bottom", "vertical-align: super; margin: 0 3px"),
			ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: top"), ui.Text("y"),
				ui.Span(ui.Text("v"), ui.Span(ui.Style("vertical-align: 12px"), ui.Text("z"), slab("10px"))))),
			ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: top"), ui.Text("y"),
				ui.Label(ui.Style("height: 60px; vertical-align: 15px")))),
			ui.P(ui.Text("x"), ui.Span(ui.Style("vertical-align: bottom"), ui.Text("y"),
				ui.Span(ui.Style("vertical-align: sub; font-size: 40px"), ui.Text("z")))),
		),
		"centred lines ending in spans": page(
			ui.P(ui.Style("width: 300px; text-align: center"), ui.Text("Centred text "), ui.Span(ui.Text("ends here"))),
			ui.P(ui.Style("width: 300px; text-align: center"), ui.Text("Centred "),
				ui.Span(ui.Style("font-weight: bold"), ui.Text("bold end "))),
			ui.P(ui.Style("width: 140px; text-align: center"), ui.Text("one two three "),
				ui.Span(ui.Style("background: #ddd"), ui.Text("four five six seven ")), ui.Text("eight")),
			ui.P(ui.Style("width: 140px; text-align: center"), ui.Span(ui.Text("one two three four five six seven "))),
			ui.P(ui.Style("width: 140px; text-align: center; letter-spacing: 2px"), ui.Text("one two "),
				ui.Span(ui.Text("three four five"))),
			ui.P(ui.Style("width: 140px; text-align: center"), ui.Text("one two "),
				ui.Span(ui.Style("padding: 0 4px; border: 1px solid"), ui.Text("three four five six"))),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 1000))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			matchText(t, d, geom.V2(800, 1000), 0.5)
			if name != "a top or bottom box holding a shifted one" {
				// Chrome gives a span holding one that makes a box of its own a
				// rectangle per child, not a box.
				matchSpanFragments(t, d, geom.V2(800, 1000), 0.5)
			}
		})
	}
}
