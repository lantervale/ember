// A table's collapsed borders in the styles Chrome draws apart from solid:
// double, inset and outset, groove and ridge, on the table and on its
// cells, as wide and wider than their neighbours.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #f4ede4; }
body { padding: 12px; margin: 0; }
.row { display: flex; gap: 16px; align-items: flex-start; margin-bottom: 16px; }
table { border-collapse: collapse; background-color: #ffffff; }
td { width: 24px; height: 16px; padding: 0; }
.dbl { border: 6px double #2a2530; }
.dbl td { border: 1px solid #3421e6; }
.cells td { border: 6px double #e03c31; }
.in { border: 6px inset #808080; }
.in td { border: 2px solid #3421e6; }
.out { border: 6px outset #808080; }
.cin td { border: 4px inset #e03c31; }
.cout td { border: 4px outset #2a9d4a; }
.gr td { border: 6px groove #808080; }
.ri td { border: 6px ridge #808080; }
`

func table(class string) *ui.Node {
	t := ui.Table(ui.Class(class))
	for range 2 {
		tr := ui.Tr()
		for range 3 {
			tr.Append(ui.Td())
		}
		t.Append(tr)
	}
	return t
}

func main() {
	d := ui.NewDocument(ui.CSS(sheet))
	row := func(classes ...string) *ui.Node {
		r := ui.Div(ui.Class("row"))
		for _, c := range classes {
			r.Append(table(c))
		}
		return r
	}
	d.Body().Append(row("dbl", "cells", "in", "out"), row("cin", "cout", "gr", "ri"))
	pages.Run(d, 520, 160)
}
