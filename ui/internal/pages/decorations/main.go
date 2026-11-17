// Text decorations: each line and style, coloured and thick ones, and
// those a box draws across the text of what it holds, against Chrome's.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 8px; color: #2b2722; font-size: 16px; }
p { margin: 0 0 6px; }
.big { font-size: 24px; }
.small { font-size: 11px; }
.red { color: #c8473d; }
.over { text-decoration: overline #3a6fd8; }
.through { text-decoration: line-through; }
.block { display: inline-block; }
.float { float: right; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"decorations.css": {Data: []byte(sheet)}}, "decorations.css"))
	for _, decoration := range []string{
		"underline",
		"overline",
		"line-through",
		"underline overline line-through",
		"underline dotted",
		"underline dashed",
		"underline double",
		"overline double #3a6fd8",
		"line-through double #c8473d",
		"underline #c8473d 3px",
		"line-through dashed #3a6fd8 2px",
		"underline dotted #3a6fd8 4px",
	} {
		d.Body().Append(ui.P(ui.Style("text-decoration: "+decoration), ui.Text("Handgloves "+decoration)))
	}
	d.Body().Append(
		ui.P(ui.Style("text-decoration: underline"),
			ui.Text("A paragraph's line runs "),
			ui.Span(ui.Class("red"), ui.Text("under a red span")),
			ui.Text(", "),
			ui.Span(ui.Class("big"), ui.Text("a big one")),
			ui.Text(" and "),
			ui.Span(ui.Class("small"), ui.Text("a small one")),
			ui.Text("."),
		),
		ui.P(ui.Class("big"), ui.Style("text-decoration: underline line-through"),
			ui.Text("Big, with "),
			ui.Span(ui.Class("small", "over"), ui.Text("a small span's own overline")),
			ui.Text(" inside"),
		),
		ui.P(ui.Style("text-decoration: overline"),
			ui.Text("Over "),
			ui.Span(ui.Class("big"), ui.Text("a big one")),
			ui.Text(" and "),
			ui.Span(ui.Class("small"), ui.Text("a small one")),
		),
		ui.Div(ui.Style("text-decoration: underline #c8473d"),
			ui.P(ui.Class("big"), ui.Text("A box's line under its paragraphs, "),
				ui.Span(ui.Class("through"), ui.Text("struck")), ui.Text(",")),
			ui.P(
				ui.Span(ui.Class("float"), ui.Text("not a float")),
				ui.Text("but "),
				ui.Span(ui.Class("block"), ui.Text("not an inline-block")),
				ui.Text(" either."),
			),
		),
	)
	pages.Run(d, 420, 560)
}
