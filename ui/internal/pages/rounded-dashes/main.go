// Rounded dashed and dotted borders as Chrome draws them: the dashes and
// the dots laid round the corners along the middle of the border.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #f4ede4; }
body { padding: 12px; margin: 0; }
.row { display: flex; gap: 16px; align-items: flex-start; margin-bottom: 16px; }
.row > div { width: 56px; height: 36px; background-color: #ffffff; border-color: #3421e6; }
.a { border: 2px dashed; border-radius: 10px; }
.b { border: 4px dashed; border-radius: 18px; }
.c { border: 1px dashed; border-radius: 8px; }
.d { border: 4px dotted; border-radius: 16px; }
.e { border: 6px dotted; border-radius: 24px; }
.f { border: 2px dotted; border-radius: 12px; }
.row > .g { border: 3px dashed #e03c31; border-radius: 22px; width: 80px; }
.row > .h { border-style: dashed; border-width: 2px 6px; border-radius: 14px; }
.row > .i { border: 4px dashed; border-color: #e03c31 #3421e6; border-radius: 16px; }
.row > .j { border: 5px dotted #2a9d4a; border-radius: 50%; width: 40px; height: 40px; }
.row > .k { border-width: 4px; border-style: dashed solid; border-radius: 14px; }
`

func main() {
	d := ui.NewDocument(ui.CSS(sheet))
	row := func(classes ...string) *ui.Node {
		r := ui.Div(ui.Class("row"))
		for _, c := range classes {
			r.Append(ui.Div(ui.Class(c)))
		}
		return r
	}
	d.Body().Append(row("a", "b", "c", "d", "e", "f"), row("g", "h", "i", "j", "k"))
	pages.Run(d, 560, 160)
}
