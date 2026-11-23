//go:build chrome

package layout_test

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

const clampSheet = `.c { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; width: 200px;
margin-bottom: 20px; background: #eef } .h { overflow: hidden } .one { -webkit-line-clamp: 1 }
.three { -webkit-line-clamp: 3 } .framed { padding: 5px; border: 2px solid #a00 } .pad { padding: 3px }`

const long = "one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen"

func TestWebkitLineClampMatchesChrome(t *testing.T) {
	clamp := func(classes string, kids ...ui.Arg) *ui.Node {
		return ui.Div(append([]ui.Arg{ui.Class(append([]string{"c"}, strings.Fields(classes)...)...)}, kids...)...)
	}
	cases := map[string]*ui.Document{
		"text clamped to two lines": styled(clampSheet, clamp(""), clamp("", ui.Text(long)),
			clamp("h", ui.Text(long))),
		"text that ends before the clamp": styled(clampSheet, clamp("", ui.Text("one two")),
			clamp("three", ui.Text("one two three four five six seven eight nine ten"))),
		"paragraphs and their margins": styled(clampSheet+` p { margin: 4px 0 } .b { margin: 6px 0 9px }`,
			clamp("framed", ui.P(ui.Text("first line")), ui.P(ui.Class("b"), ui.Text("second line")),
				ui.P(ui.Text("third"))),
			clamp("three", ui.P(ui.Text("one two three four five")),
				ui.P(ui.Text("six seven eight nine ten eleven twelve thirteen fourteen")), ui.P(ui.Text("after")))),
		"a framed block holding the clamp": styled(clampSheet,
			clamp("", ui.Div(ui.Class("framed"), ui.Text(long)), ui.Div(ui.Text("after")))),
		"a padded inline over the clamp": styled(clampSheet,
			clamp("", ui.Span(ui.Class("pad"), ui.Text(long)), ui.Text(" tail"))),
		"a line break and centred text": styled(clampSheet+` .m { text-align: center }`,
			clamp("h", ui.Text("one two"), ui.Br(), ui.Text(long)),
			clamp("m", ui.Text("one two"), ui.Br(), ui.Text("three four five six seven eight")),
			clamp("m", ui.Text(long))),
		"blocks after the clamp's line": styled(clampSheet+` .e { height: 30px }`,
			clamp("", ui.Div(ui.Text("first line")), ui.Div(ui.Text("second line")), ui.Div(ui.Class("e"))),
			clamp("", ui.Div(ui.Text("first line")), ui.Div(ui.Text("second line")))),
		"a block of its own formatting context counts no line": styled(clampSheet+` .r { display: flow-root }`,
			clamp("", ui.Div(ui.Class("r", "framed"), ui.Text("first"), ui.Br(), ui.Text("second"), ui.Br(),
				ui.Text("third")), ui.Div(ui.Text("after")))),
		"a tall inline block on a line": styled(clampSheet+` .ib { display: inline-block }`,
			clamp("", ui.Text("first "), ui.Span(ui.Class("ib"), ui.Text("inline"), ui.Br(), ui.Text("block")),
				ui.Text(" line second line third"))),
		"sizes of its own": styled(clampSheet+` .s { padding: 5px; height: 10px } .m { max-height: 20px }`,
			clamp("s", ui.Text(long)), clamp("m", ui.Text(long)),
			clamp("one", ui.Text("first line second line third line "), ui.B(ui.Text("bold")))),
		"an inline box clamped": styled(clampSheet+` .ib { display: -webkit-inline-box; width: 100px }`,
			ui.Div(ui.Class("ib", "one"), ui.Style("-webkit-box-orient: vertical"), ui.Text(long)), ui.Span(ui.Text(" after"))),
		"no clamp without a vertical box": styled(`div { width: 100px; -webkit-line-clamp: 1 }
.v { -webkit-box-orient: vertical } .w { display: -webkit-box }`,
			ui.Div(ui.Class("v"), ui.Text("one two three four five six")),
			ui.Div(ui.Class("w"), ui.Text("one two three four five six"))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

// TestLineClampMatchesChrome compares line-clamp with Chrome's, which ships it
// behind its CSSLineClamp flag.
func TestLineClampMatchesChrome(t *testing.T) {
	const sheet = `.c { line-clamp: 2; width: 200px; margin-bottom: 20px } .framed { padding: 5px; border: 2px solid }
p { margin: 4px 0 } .three { line-clamp: 3 } .s { height: 10px; padding: 5px }`
	cases := map[string]*ui.Document{
		"text clamped to two lines": styled(sheet, ui.Div(ui.Class("c"), ui.Text(long)),
			ui.Div(ui.Class("c"), ui.Text("one two"))),
		"blocks clamped": styled(sheet,
			ui.Div(ui.Class("c", "three"), ui.P(ui.Text("one two three four five")),
				ui.Div(ui.Class("framed"), ui.Text(long)), ui.P(ui.Text("after"))),
			ui.Div(ui.Class("c"), ui.P(ui.Text("first")), ui.P(ui.Text("second")))),
		"sizes of its own": styled(sheet, ui.Div(ui.Class("c", "s"), ui.Text(long)),
			ui.Div(ui.Class("c"), ui.Style("display: inline-block"), ui.Text(long))),
		"no ellipsis": styled(sheet+` .n { line-clamp: 2 no-ellipsis } .k { line-clamp: no-ellipsis 2 }`,
			ui.Div(ui.Class("c", "n"), ui.Span(ui.Text(long))), ui.Div(ui.Class("c", "k"), ui.Text(long))),
		"no ellipsis ending inside a block": styled(sheet+` .n { line-clamp: 2 no-ellipsis } .b { margin: 6px 0 9px }`,
			ui.Div(ui.Class("c", "n"), ui.P(ui.Text("one")), ui.P(ui.Text("two")), ui.P(ui.Text("three"))),
			ui.Div(ui.Class("c", "n"), ui.P(ui.Text("one")), ui.Div(ui.Class("framed", "b"), ui.Text("two")),
				ui.P(ui.Text("three"))),
			ui.Div(ui.Class("c", "n"), ui.Div(ui.Class("framed", "b"), ui.Text(long)), ui.P(ui.Text("after"))),
			ui.Div(ui.Class("c", "n"), ui.P(ui.Text("one")), ui.P(ui.Text("two")))),
		"auto, at the last line the height holds": styled(sheet+` .a { line-clamp: auto } .m { max-height: 45px }
.h { height: 45px } .n { line-clamp: no-ellipsis }`,
			ui.Div(ui.Class("c", "a", "m"), ui.Text(long+" "+long)),
			ui.Div(ui.Class("c", "a", "h"), ui.Text(long+" "+long)),
			ui.Div(ui.Class("c", "a"), ui.Text(long)),
			ui.Div(ui.Class("c", "a", "m"), ui.Text("one two")),
			ui.Div(ui.Class("c", "n", "m"), ui.Text(long+" "+long)),
			ui.Div(ui.Class("c", "a", "framed"), ui.Style("max-height: 80px"), ui.P(ui.Text("one")),
				ui.Div(ui.Class("framed"), ui.Text(long)), ui.P(ui.Text("after")))),
		"-webkit-legacy clamps a vertical -webkit-box alone": styled(sheet+` .l { line-clamp: 2 -webkit-legacy }
.v { display: -webkit-box; -webkit-box-orient: vertical }`,
			ui.Div(ui.Class("c", "l"), ui.Text(long)),
			ui.Div(ui.Class("c", "l", "v"), ui.Text(long), ui.Div(ui.Text("after")))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.MatchWith(t, d, geom.V2(800, 600), "CSSLineClamp")
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
