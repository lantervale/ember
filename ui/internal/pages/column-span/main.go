// Blocks spanning a multi-column container's columns: a heading and a
// quote across three ruled columns, each row of columns balanced on its
// own and ruled only as high as it goes; a tinted spanner between two
// rows of two columns under a height, the last row filled in turn.
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
.three h2 { column-span: all; margin: 4px 0 8px; font-size: 16px; border-bottom: 2px solid #b0493c; }
.three blockquote { column-span: all; margin: 4px 40px 10px; padding: 4px 8px; background-color: #dfe9f6;
	font-style: italic; }
.two { columns: 2; column-gap: 20px; column-rule: 3px dashed #3c8a4f; column-fill: auto; height: 150px; }
.two .s { column-span: all; padding: 4px; margin: 6px 0; background-color: #e6dcc8; text-align: center; }
`

const sentence = "The quick brown fox jumps over the lazy dog, then naps in the warm afternoon sun."

func prose(n int) ui.Arg { return ui.Text(strings.TrimSpace(strings.Repeat(sentence+" ", n))) }

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"column-span.css": {Data: []byte(sheet)}}, "column-span.css"))
	d.Body().Append(
		ui.Div(ui.Class("m", "three"),
			ui.H2(ui.Text("Across the columns")),
			ui.P(prose(2)), ui.P(prose(1)),
			ui.Blockquote(ui.Text("A quote as wide as the container, between two rows of columns.")),
			ui.P(prose(3)),
		),
		ui.Div(ui.Class("m", "two"),
			ui.P(prose(1)),
			ui.Div(ui.Class("s"), ui.Text("Spanning both")),
			ui.P(prose(2)),
		),
	)
	pages.Run(d, 640, 420)
}
