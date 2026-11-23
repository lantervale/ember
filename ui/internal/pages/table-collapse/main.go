// Tables with rows and columns taken out: empty-cells: hide leaving its
// empty cells undrawn over the layers behind them, the cell holding white
// space with them and the one holding a no-break space drawn; and
// visibility: collapse taking a row, a column and a row group out with
// their room, their cells unpainted; the table round the collapsed column
// is bare, as Chrome paints its background as wide as before.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 12px; display: flex; flex-wrap: wrap; gap: 16px; align-items: flex-start; font-size: 13px; }
table { border-spacing: 5px; background-color: #e4ecf7; }
td { padding: 4px 8px; border: 2px solid #345; background-color: #fde9c8; }
.hide { empty-cells: hide; }
.hide tr.lit { background-color: #7c7; }
.hide col.lit { background-color: #f6c; }
.gone { visibility: collapse; }
col.tint { background-color: #9cf; }
.bare { background-color: transparent; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"table-collapse.css": {Data: []byte(sheet)}}, "table-collapse.css"))
	td := func(text string, args ...ui.Arg) *ui.Node { return ui.Td(append([]ui.Arg{ui.Text(text)}, args...)...) }
	d.Body().Append(
		ui.Table(ui.Class("hide"), ui.Colgroup(ui.Col(), ui.Col(ui.Class("lit")), ui.Col()),
			ui.Tbody(
				ui.Tr(td("one"), ui.Td(), td("three")),
				ui.Tr(ui.Class("lit"), ui.Td(), td(" "), td(" ")),
				ui.Tr(td("seven"), td("eight"), ui.Td(ui.Span())))),
		ui.Table(ui.Class("bare"), ui.Colgroup(ui.Col(), ui.Col(ui.Class("gone")), ui.Col(ui.Class("tint"))),
			ui.Tbody(
				ui.Tr(td("one"), td("two"), td("three")),
				ui.Tr(ui.Class("gone"), td("four"), td("five"), td("six")),
				ui.Tr(td("seven", ui.Attr("rowspan", "2")), td("eight", ui.Attr("colspan", "2"))),
				ui.Tr(td("ten"), td("eleven")))),
		ui.Table(ui.Thead(ui.Tr(td("head"), td("of it"))), ui.Tbody(ui.Class("gone"), ui.Tr(td("gone"), td("too"))),
			ui.Tbody(ui.Tr(td("body"), td("stays")))),
	)
	pages.Run(d, 520, 200)
}
