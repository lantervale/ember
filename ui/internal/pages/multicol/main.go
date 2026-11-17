// Multi-column containers against Chrome's: text balanced over columns
// with rules of several styles, a bordered block and a tinted paragraph
// cut between columns, a rule under a height taller than the balanced
// columns, columns that overflow a fixed height, and a section modelled
// on CSS Zen Garden 221's Requirements, its heading turned on its side.
package main

import (
	"strings"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; font-size: 12px; line-height: 16px; color: #2b2722; }
p { margin: 0 0 8px; }
.m { margin-bottom: 12px; padding: 6px; background-color: #f4f1ec; }
.three { column-count: 3; column-gap: 24px; column-rule: 2px solid #b0493c; }
.note { padding: 4px 6px; border: 2px solid #3c6fb0; background-color: #dfe9f6; }
.tint { background-color: #e6dcc8; }
.pair { display: flex; gap: 12px; }
.pair > div { flex: 1; }
.dashed { columns: 2; column-gap: 20px; column-rule: 3px dashed #3c8a4f; height: 76px; }
.dotted { columns: 3 60px; column-gap: 16px; column-rule: 4px dotted #7a4fb0; height: 40px; }
.main { position: relative; height: 150px; background-color: #ece7df; }
.requirements { position: absolute; column-count: 2; column-gap: 2em; right: 0; margin: 1em 0 0 0; width: 60%;
	padding: 1em 0 0 0; box-sizing: border-box; column-rule: 1px solid #8a8378; }
.requirements h3 { position: absolute; margin: 0; top: 3em; left: -6em; font-size: 14px; color: #b0493c;
	transform: rotate(-90deg); }
.requirements p { margin: 0 1em 1em 1em; }
`

const sentence = "The quick brown fox jumps over the lazy dog, then naps in the warm afternoon sun."

func prose(n int) ui.Arg { return ui.Text(strings.TrimSpace(strings.Repeat(sentence+" ", n))) }

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"multicol.css": {Data: []byte(sheet)}}, "multicol.css"))
	d.Body().Append(
		ui.Div(ui.Class("m", "three"),
			ui.P(prose(2)),
			ui.Div(ui.Class("note"), ui.P(ui.Style("margin: 0"), prose(3))),
			ui.P(ui.Class("tint"), prose(3)),
		),
		ui.Div(ui.Class("pair"),
			ui.Div(ui.Class("m", "dashed"), ui.P(prose(2))),
			ui.Div(ui.Class("m", "dotted"), ui.P(prose(2))),
		),
		ui.Div(ui.Class("main"),
			ui.Div(ui.Class("requirements"),
				ui.H3(ui.Text("Requirements")),
				ui.P(prose(1)), ui.P(prose(2)), ui.P(prose(1)), ui.P(prose(1)),
			),
		),
	)
	pages.Run(d, 640, 440)
}
