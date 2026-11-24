// Columns cut lanes past their lines: a flex row whose first item forces a
// column break, its other item whole in the first column and stretched on;
// a grid row whose item does the same; and a note positioned beside the
// lines of its block, cut by its own lines, which don't line up with them.
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
.br { break-before: column; color: #b0493c; }
.f { display: flex; gap: 8px; }
.f > div { flex: 1; padding: 4px; border: 2px solid #3c8a4f; background-color: #e4f1e6; }
.g { display: grid; grid-template-columns: 1fr 1fr; gap: 4px; padding: 4px; background-color: #dfe9f6; }
.g > div { padding: 2px 6px; background-color: #c8d8ee; }
.cb { position: relative; padding-right: 150px; }
.note { position: absolute; top: 10px; right: 0; width: 130px; padding: 0 6px; line-height: 21px;
	border-left: 3px solid #3c5a8a; background-color: #dfe9f6; }
`

func lines(prefix string, n int) []ui.Arg {
	var out []ui.Arg
	for i := range n {
		out = append(out, ui.P(ui.Text(fmt.Sprintf("%s line %d", prefix, i+1))))
	}
	return out
}

func broken(before, after int) []ui.Arg {
	out := lines("before", before)
	out = append(out, ui.P(ui.Class("br"), ui.Text("after the break")))
	return append(out, lines("after", after)...)
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"column-lanes-more.css": {Data: []byte(sheet)}},
		"column-lanes-more.css"))
	d.Body().Append(
		ui.Div(ui.Class("m"), ui.Div(ui.Class("f"), ui.Div(broken(2, 2)...), ui.Div(lines("whole", 6)...))),
		ui.Div(ui.Class("m"), ui.Div(ui.Class("g"), ui.Div(broken(1, 3)...), ui.Div(lines("whole", 5)...))),
		ui.Div(ui.Class("m"), ui.Div(append([]ui.Arg{ui.Class("cb")}, append(lines("flow", 9),
			ui.Div(append([]ui.Arg{ui.Class("note")}, lines("note", 6)...)...))...)...)),
	)
	pages.Run(d, 560, 420)
}
