//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestWebkitBoxMatchesChrome(t *testing.T) {
	const sheet = `.b { display: -webkit-box; width: 300px; height: 100px; margin-bottom: 4px; background: #eee }
.b > div { width: 40px; height: 30px; background: #888 } .b > .a { width: auto; height: auto }
.v { -webkit-box-orient: vertical } .n { width: 100px }`
	box := func(style string, kids ...ui.Arg) *ui.Node {
		return ui.Div(append([]ui.Arg{ui.Class("b"), ui.Style(style)}, kids...)...)
	}
	item := func(style string, kids ...ui.Arg) *ui.Node {
		return ui.Div(append([]ui.Arg{ui.Style(style)}, kids...)...)
	}
	auto := func(style, text string) *ui.Node {
		return ui.Div(ui.Class("a"), ui.Style(style), ui.Text(text))
	}
	cases := map[string]*ui.Document{
		"-webkit-box-pack": styled(sheet,
			box("-webkit-box-pack: center", item(""), item("")),
			box("-webkit-box-pack: end", item(""), item("")),
			box("-webkit-box-pack: justify", item(""), item("")),
			box("-webkit-box-pack: start; -webkit-box-pack: center", item(""), item("")),
			box("-webkit-box-pack: center", ui.Class("n"), item("width: 80px"), item("width: 80px")),
			box("-webkit-box-pack: justify", ui.Class("n"), item("width: 80px"), item("width: 80px"))),
		"-webkit-box-align": styled(sheet,
			box("", item(""), auto("", "x")),
			box("-webkit-box-align: center", item(""), auto("", "x")),
			box("-webkit-box-align: end", item(""), auto("", "x")),
			box("-webkit-box-align: start", item(""), auto("", "x")),
			box("-webkit-box-align: baseline", auto("font-size: 30px; line-height: 40px", "a"), auto("", "x")),
			box("-webkit-box-align: center", item("margin-top: auto"), item("margin: auto"))),
		"-webkit-box-direction and the orient": styled(sheet,
			box("-webkit-box-direction: reverse", item(""), auto("", "x")),
			box("-webkit-box-direction: reverse; -webkit-box-pack: end", item(""), auto("", "x")),
			box("-webkit-box-direction: reverse", ui.Class("v"), item(""), auto("", "x")),
			box("-webkit-box-orient: block-axis; -webkit-box-align: end", item(""), auto("", "x")),
			box("-webkit-box-orient: inline-axis; -webkit-box-align: end", item(""), auto("", "x")),
			box("-webkit-box-pack: center; -webkit-box-align: center", ui.Class("v"), item(""), auto("", "xyz"))),
		"-webkit-box-flex grows and shrinks": styled(sheet,
			box("", item("-webkit-box-flex: 1"), item("-webkit-box-flex: 2")),
			box("", auto("-webkit-box-flex: 1", "aaa"), auto("-webkit-box-flex: 1", "bbbbbbbbb")),
			box("", ui.Class("n"), item("width: 80px"), item("width: 80px")),
			box("", ui.Class("n"), item("width: 80px; -webkit-box-flex: 1"), item("width: 80px")),
			box("", ui.Class("n"), auto("-webkit-box-flex: 1", "longwordhere"), item("width: 80px")),
			box("", ui.Class("n"), auto("-webkit-box-flex: -1", "longwordhere"),
				item("width: 80px; -webkit-box-flex: 0.5")),
			box("height: 50px", ui.Class("v"), auto("", "a"), auto("height: 40px; -webkit-box-flex: 1", "b"),
				item("height: auto", ui.Text("c"))),
			box("height: auto", ui.Class("v"), auto("", "a"), auto("height: 40px; -webkit-box-flex: 1", "b"))),
		"an item as wide as the box at most": styled(sheet+` .w { width: 120px }`,
			box("", ui.Class("w"), auto("", "some long words here and more"), auto("", "x")),
			box("", ui.Class("w"), auto("-webkit-box-flex: 1", "some long words here and more"), auto("", "x")),
			box("", ui.Class("w"), auto("-webkit-box-flex: 1", "some long words here"),
				auto("-webkit-box-flex: 1", "x y z w v u t s"))),
		"-webkit-box-ordinal-group": styled(sheet,
			box("", item("-webkit-box-ordinal-group: 2"), auto("-webkit-box-ordinal-group: 1", "x"), item("")),
			box("", item("-webkit-box-ordinal-group: 1"), auto("-webkit-box-ordinal-group: 0", "x"))),
		"flex's own properties": styled(sheet,
			box("justify-content: center; align-items: center", item("flex-grow: 1"),
				auto("order: -1; align-self: end", "x")),
			box("", ui.Class("n"), auto("-webkit-box-flex: 1; flex: 0 0 auto", "longwordhere"),
				item("width: 80px; flex: 1"))),
		"inline content as one item": styled(sheet+` .i { display: inline-block; width: 30px; height: 10px }`,
			box("", ui.Class("n"), ui.Text("some text "), ui.Span(ui.Text("inline span")), ui.Text(" "),
				ui.B(ui.Text("bold"))),
			box("", ui.Class("n"), ui.Text("ab "), ui.Span(ui.Class("i")), ui.Text(" cd"), item("")),
			box("width: 200px", ui.I(ui.Text("some")), ui.Text(" "), ui.I(ui.Text("text")), ui.Text(" "),
				ui.Span(ui.I(ui.Text("inline")), ui.Text(" "), ui.I(ui.Text("span"))), ui.Text(" "), ui.B(ui.Text("bold"))),
			box("width: 200px", ui.Text("some text inline span bold"), auto("", "blk"), ui.Text("more text here")),
			ui.Span(ui.Class("b"), ui.Style("display: -webkit-inline-box; width: auto; height: auto"),
				item("-webkit-box-flex: 1", ui.Text("a")), item("", ui.Text("b")))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 1200))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
