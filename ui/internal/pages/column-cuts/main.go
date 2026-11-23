// What columns cut besides blocks: a striped table cut between its rows,
// a flex row whose bordered items are cut where each column ends, and an
// absolutely positioned note cut with the block holding it.
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
table { border-collapse: separate; border-spacing: 0; width: 100%; background-color: #dfe9f6; border: 2px solid #3c5a8a; }
td { padding: 2px 6px; }
tr:nth-child(even) td { background-color: #c8d8ee; }
.f { display: flex; gap: 8px; }
.f > div { flex: 1; padding: 4px; border: 2px solid #3c8a4f; background-color: #e4f1e6; }
.r { position: relative; }
.note { position: absolute; top: 16px; right: 0; width: 90px; padding: 0 4px; background-color: #f6dfdf;
	border-left: 3px solid #b0493c; }
`

func lines(prefix string, n int) []ui.Arg {
	var out []ui.Arg
	for i := range n {
		out = append(out, ui.P(ui.Text(fmt.Sprintf("%s line %d", prefix, i+1))))
	}
	return out
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"column-cuts.css": {Data: []byte(sheet)}}, "column-cuts.css"))
	var rows []ui.Arg
	for i := range 9 {
		rows = append(rows, ui.Tr(ui.Td(ui.Text(fmt.Sprint("Row ", i+1))), ui.Td(ui.Text("a cell"))))
	}
	d.Body().Append(
		ui.Div(ui.Class("m"), ui.Table(ui.Tbody(rows...))),
		ui.Div(ui.Class("m"), ui.Div(ui.Class("f"), ui.Div(lines("left", 7)...), ui.Div(lines("right", 4)...))),
		ui.Div(ui.Class("m"), ui.Div(append([]ui.Arg{ui.Class("r")},
			append(lines("held", 4), ui.Div(append([]ui.Arg{ui.Class("note")}, lines("note", 5)...)...))...)...)),
	)
	pages.Run(d, 560, 420)
}
