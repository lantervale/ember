// Boxes columns cut each where its own content allows: a framed note
// round a heading that spans the columns, its border and padding on the
// parts above and below it; a table row whose two cells break apart; and
// a flex row of two items whose lines are of different heights.
package main

import (
	"fmt"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; font-size: 12px; line-height: 16px; color: #2b2722; }
p { margin: 0; }
.m { columns: 2; column-gap: 24px; column-rule: 2px solid #b0493c; margin-bottom: 12px; padding: 6px;
	background-color: #f4f1ec; }
.note { margin: 4px 10px; padding: 4px 8px; border: 2px solid #3c5a8a; background-color: #dfe9f6; }
.note h3 { column-span: all; margin: 0; padding: 2px 6px; font-size: 13px; background-color: #e6dcc8; }
table { border-collapse: separate; border-spacing: 4px; width: 100%; background-color: #dfe9f6;
	border: 2px solid #3c5a8a; }
td { padding: 2px 6px; vertical-align: top; background-color: #c8d8ee; }
.f { display: flex; gap: 8px; }
.f > div { flex: 1; padding: 4px; border: 2px solid #3c8a4f; background-color: #e4f1e6; }
.f > div + div { line-height: 23px; }
`

func lines(prefix string, n int) []ui.Arg {
	var out []ui.Arg
	for i := range n {
		out = append(out, ui.P(ui.Text(fmt.Sprintf("%s line %d", prefix, i+1))))
	}
	return out
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"column-cuts-more.css": {Data: []byte(sheet)}},
		"column-cuts-more.css"))
	d.Body().Append(
		ui.Div(ui.Class("m"), ui.Div(append(append([]ui.Arg{ui.Class("note")}, lines("before", 3)...),
			append([]ui.Arg{ui.H3(ui.Text("A heading across the columns"))}, lines("after", 5)...)...)...)),
		ui.Div(ui.Class("m"), ui.Table(ui.Tr(ui.Td(lines("long", 8)...), ui.Td(lines("short", 3)...)))),
		ui.Div(ui.Class("m"), ui.Div(ui.Class("f"), ui.Div(lines("left", 9)...), ui.Div(lines("right", 5)...))),
	)
	pages.Run(d, 560, 420)
}
