//go:build chrome

package layout_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

const shortSentence = "The quick brown fox jumps over the lazy dog."

func TestVerticalBlocksMatchChrome(t *testing.T) {
	cases := map[string]*ui.Document{
		"a vertical-rl block of text": styled(`.v { writing-mode: vertical-rl; background: #ccc; }`,
			ui.Div(ui.Class("v"), words("", shortSentence)),
		),
		"a vertical-rl block of a fixed height wraps its lines from the right": styled(
			`.v { writing-mode: vertical-rl; height: 120px; background: #ccc; }`,
			ui.Div(ui.Class("v"), words("", sentence)),
		),
		"a vertical-lr block wraps its lines from the left": styled(
			`.v { writing-mode: vertical-lr; height: 120px; padding: 3px 5px 7px 9px; border: 2px solid; }`,
			ui.Div(ui.Class("v"), words("", sentence)),
		),
		"blocks stack from the right": styled(`
.v { writing-mode: vertical-rl; height: 200px; border: 1px solid; padding: 4px 6px; }
.v > div { margin: 10px 4px 6px 8px; padding: 2px 3px; border: 2px solid; background: #ccc; }
.v > .sized { width: 30px; height: 50%; }
.v > .wide { inline-size: 80px; margin-inline: auto; }`,
			ui.Div(ui.Class("v"), ui.Div(words("", "first block")), ui.Div(ui.Class("sized"), words("", "sized")),
				ui.Div(ui.Class("wide"), words("", "centred")), ui.Div(words("", shortSentence))),
		),
		"a horizontal block in a vertical one": styled(`
.v { writing-mode: vertical-rl; height: 200px; }
.h { writing-mode: horizontal-tb; background: #ccc; margin: 4px; }
.h2 { writing-mode: horizontal-tb; width: 100px; }`,
			ui.Div(ui.Class("v"), ui.Div(words("", "before")), ui.Div(ui.Class("h"), words("", shortSentence)),
				ui.Div(ui.Class("h2"), words("", shortSentence)), ui.Div(words("", "after"))),
		),
		"a vertical block in a horizontal one of a fixed height": styled(`
.h { height: 150px; width: 300px; }
.v { writing-mode: vertical-lr; margin: 5px; background: #ccc; }`,
			ui.Div(ui.Class("h"), words("", "above"), ui.Div(ui.Class("v"), words("", sentence)), words("", "below")),
		),
		"sideways-lr lines run up from the left": styled(`.v { writing-mode: sideways-lr; height: 140px; }`,
			ui.Div(ui.Class("v"), words("", sentence)),
		),
		"sideways-rl as vertical-rl": styled(`.v { writing-mode: sideways-rl; height: 140px; }`,
			ui.Div(ui.Class("v"), words("", sentence)),
		),
		"right to left in vertical-rl starts lines at the bottom": styled(
			`.v { writing-mode: vertical-rl; direction: rtl; height: 140px; }
.v > div { inline-size: 60px; margin: 2px; background: #ccc; }`,
			ui.Div(ui.Class("v"), words("", sentence), ui.Div(words("", "short"))),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			assert.Empty(t, d.Warnings())
		})
	}
}

func TestVerticalFormattingContextsMatchChrome(t *testing.T) {
	cases := map[string]*ui.Document{
		"a flex row runs down a vertical-rl box": styled(`
.v { writing-mode: vertical-rl; height: 300px; display: flex; gap: 6px; padding: 4px; border: 1px solid; }
.v > div { padding: 2px; border: 1px solid; background: #ccc; }
.v > .grow { flex: 1; }
.v > .end { align-self: flex-end; inline-size: 40px; block-size: 30px; }`,
			ui.Div(ui.Class("v"), ui.Div(words("margin: 0", "one two")), ui.Div(ui.Class("grow"), words("margin: 0", "grows")),
				ui.Div(ui.Class("end"))),
		),
		"a flex column stacks across a vertical-lr box": styled(`
.v { writing-mode: vertical-lr; height: 200px; display: flex; flex-direction: column; align-items: center; }
.v > div { margin: 3px; background: #ccc; }`,
			ui.Div(ui.Class("v"), ui.Div(words("margin: 0", "first")), ui.Div(words("margin: 0", "second item")),
				ui.Div(words("margin: 0", "third"))),
		),
		"a grid's columns run down a vertical-rl box": styled(`
.v { writing-mode: vertical-rl; height: 240px; display: grid; grid-template-columns: 1fr 2fr 60px;
  grid-template-rows: 40px auto; gap: 4px 8px; }
.v > div { background: #ccc; }
.v > .span { grid-column: 1 / 3; }`,
			ui.Div(ui.Class("v"), ui.Div(ui.Text("a")), ui.Div(ui.Text("b")), ui.Div(ui.Text("c")),
				ui.Div(ui.Class("span"), ui.Text("spans two")), ui.Div(ui.Text("e"))),
		),
		"a vertical inline-block in a line": styled(`
.v { display: inline-block; writing-mode: vertical-rl; background: #ccc; margin: 2px 4px; padding: 3px; }`,
			ui.P(ui.Text("before "), ui.Span(ui.Class("v"), ui.Text("down")), ui.Text(" after")),
		),
		"percentages of a vertical box": styled(`
.v { writing-mode: vertical-rl; width: 400px; height: 200px; }
.v > div { inline-size: 50%; block-size: 25%; margin-block-start: 10px; background: #ccc; }`,
			ui.Div(ui.Class("v"), ui.Div(), ui.Div(ui.Style("width: 10%; height: 30%"))),
		),
		"floats in a vertical-rl box": styled(`
.v { writing-mode: vertical-rl; height: 200px; }
.f { float: left; inline-size: 50px; block-size: 40px; background: #888; }
.g { float: right; inline-size: 30px; block-size: 20px; background: #444; }`,
			ui.Div(ui.Class("v"), ui.Div(ui.Class("f")), ui.Div(ui.Class("g")), words("margin: 0", sentence)),
		),
		"absolutely positioned boxes in a vertical-rl box": styled(`
.v { writing-mode: vertical-rl; position: relative; width: 300px; height: 200px; }
.a { position: absolute; inset-block-start: 10px; inset-inline-start: 20px; inline-size: 40px; block-size: 30px;
  background: #888; }
.b { position: absolute; top: 5px; left: 7px; width: 20px; height: 20px; background: #444; }`,
			ui.Div(ui.Class("v"), ui.Div(ui.Class("a")), ui.Div(ui.Class("b")), words("margin: 0", "text")),
		),
		"vertical-rl inside vertical-lr": styled(`
.lr { writing-mode: vertical-lr; height: 160px; }
.rl { writing-mode: vertical-rl; margin: 4px; background: #ccc; }`,
			ui.Div(ui.Class("lr"), words("margin: 0", "left"), ui.Div(ui.Class("rl"), words("margin: 0", shortSentence)),
				words("margin: 0", "right")),
		),
		"inline elements of vertical lines": styled(`
.v { writing-mode: vertical-rl; height: 180px; }
.v b { padding: 2px 4px; border: 1px solid; margin-inline: 3px; }
.v .big { font-size: 24px; }`,
			ui.Div(ui.Class("v"), ui.P(ui.Text("Some "), ui.B(ui.Text("bold words")), ui.Text(" and "),
				ui.Span(ui.Class("big"), ui.Text("big")), ui.Text(" ones that wrap down the box."))),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			assert.Empty(t, d.Warnings())
		})
	}
}
