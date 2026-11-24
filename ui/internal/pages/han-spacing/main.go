// text-spacing-trim and text-autospace against Chrome's, in a stand-in
// for a face of Chinese and Japanese with halt (a copy of
// internal/shape/testdata's go-han.ttf): full-width punctuation side by
// side set half wide, an opening bracket starting a wrapped line as each
// value has it, a closing bracket trimmed to fit a line's end, and the
// room between ideographs and letters, across elements, decorated and
// selected; and in sideways lines, spaced as horizontal ones.
package main

import (
	_ "embed"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

//go:embed go-han.ttf
var han []byte

const sheet = `
@font-face { font-family: Han; src: url(han.ttf); }
html { background-color: #fff; }
body { padding: 8px; font-family: Han; font-size: 22px; color: #2b2722; }
p { margin: 0 0 6px; white-space: nowrap; }
.wrap { white-space: normal; width: 160px; float: left; margin-right: 12px; }
.all { text-spacing-trim: space-all; }
.first { text-spacing-trim: space-first; }
.start { text-spacing-trim: trim-start; }
.auto { text-autospace: normal; }
.red { color: #c8473d; }
.box { border: 1px solid #3a6fd8; padding: 0 2px; }
.under { text-decoration: underline #3a6fd8; }
.clear { clear: left; }
.side { writing-mode: vertical-rl; text-orientation: sideways; height: 170px; float: left; margin: 0 6px 0 0; }
.rl { writing-mode: sideways-rl; }
.lr { writing-mode: sideways-lr; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"page.css": {Data: []byte(sheet)},
		"han.ttf":  {Data: han},
	}, "page.css"))
	const pairs = "漢」「字。「か、」字」」（「カ：「漢・「字」』"
	const wrapped = "漢字漢字漢」「字漢字（漢字」字字漢字字字」字"
	d.Body().Append(
		ui.P(ui.Text(pairs)),
		ui.P(ui.Class("all"), ui.Text(pairs)),
		ui.P(ui.Text("漢」"), ui.Span(ui.Class("red"), ui.Text("「字」")), ui.Text("「か"),
			ui.Span(ui.Class("box"), ui.Text("」「")), ui.Text("字")),
		ui.P(ui.Class("auto"), ui.Text("漢Ember字 1966か Kana カ "),
			ui.Span(ui.Class("red"), ui.Text("Red")), ui.Text("字"), ui.Span(ui.Class("box"), ui.Text("Box")),
			ui.Text("漢")),
		ui.P(ui.Class("auto", "under"), ui.Text("漢Line字 under 字1字")),
		ui.P(ui.Class("wrap"), ui.Text(wrapped)),
		ui.P(ui.Class("wrap", "first"), ui.Text(wrapped)),
		ui.P(ui.Class("wrap", "start", "auto"), ui.Text("字字漢字（字」字字字字Kanji字字")),
		ui.P(ui.Class("clear"), ui.Text("")),
		ui.P(ui.Class("side"), ui.Text(pairs)),
		ui.P(ui.Class("side", "rl", "auto"), ui.Text("漢Ember字」「か 1966")),
		ui.P(ui.Class("side", "lr"), ui.Text("漢」"), ui.Span(ui.Class("red"), ui.Text("「字」")), ui.Text("「か")),
		ui.P(ui.Class("side", "lr", "auto", "under"), ui.Text("漢Line字 字1字")),
	)
	pages.Run(d, 560, 480)
}
