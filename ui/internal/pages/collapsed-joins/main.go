// Where a table's collapsed borders meet: which edge goes through each
// corner, by width, style and the box it came from, as Chrome joins them.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #f4ede4; }
body { padding: 12px; margin: 0; }
.row { display: flex; gap: 14px; align-items: flex-start; margin-bottom: 14px; }
table { border-collapse: collapse; background-color: #ffffff; }
td { width: 22px; height: 16px; padding: 0; }
.a td { border-style: solid; border-width: 6px; border-color: #3421e6 #e03c31; }
.b td { border-style: solid; border-width: 4px 8px; border-color: #3421e6 #e03c31; }
.c td { border-style: solid; border-width: 8px 4px; border-color: #3421e6 #e03c31; }
.d td { border-width: 6px; border-style: solid double; border-color: #3421e6 #e03c31; }
.e td { border-width: 6px; border-style: double solid; border-color: #3421e6 #e03c31; }
.f td { border: 6px solid #e03c31; }
.f tr + tr td { border-color: #2a9d4a; }
.f td + td { border-color: #3421e6; }
.f tr + tr td + td { border-color: #e0a030; }
.g { border: 6px solid #e03c31; }
.g td { border: 6px solid #3421e6; }
.h td { border: 4px solid #3421e6; }
.h td + td { border-width: 6px; border-color: #e03c31; }
.i td { border-width: 6px; border-style: dashed solid; border-color: #3421e6 #e03c31; }
.j td { border-width: 6px; border-style: groove ridge; border-color: #3421e6 #e03c31; }
.k { border: 4px solid #2a9d4a; }
.k tr { border: 6px solid #3421e6; }
.k td { border: 2px solid #e03c31; }
.l td { border: 6px solid #3421e6; }
.l tr + tr td + td { border-left: 6px solid #e03c31; border-top: 6px solid #2a9d4a; }
`

func table(class string) *ui.Node {
	t := ui.Table(ui.Class(class))
	for range 3 {
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
	d.Body().Append(row("a", "b", "c", "d", "e", "f"), row("g", "h", "i", "j", "k", "l"))
	pages.Run(d, 600, 220)
}
