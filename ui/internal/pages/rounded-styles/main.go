// Rounded borders Chrome draws apart from one ring: sides of other widths,
// of other styles, inset, outset, groove and ridge, and three or four
// colours meeting round the corners.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #f4ede4; }
body { padding: 12px; margin: 0; }
.row { display: flex; gap: 16px; align-items: flex-start; margin-bottom: 16px; }
.row > div { width: 48px; height: 32px; background-color: #ffffff; border-style: solid; border-color: #3421e6; }
.a { border-width: 2px 8px 4px 12px; border-radius: 16px; }
.row > .b { border-width: 10px 4px; border-radius: 12px 24px; border-color: #e03c31 #3421e6 #2a9d4a #e0a030; }
.row > .c { border-width: 6px; border-radius: 14px; border-color: #e03c31 #3421e6 #2a9d4a; }
.row > .d { border-width: 8px; border-radius: 18px; border-color: #e03c31 #3421e6 #e03c31 #2a9d4a; }
.row > .e { border: 6px inset #808080; border-radius: 12px; }
.row > .f { border: 6px outset #e03c31; border-radius: 16px; }
.row > .g { border: 8px groove #808080; border-radius: 14px; }
.row > .h { border: 8px ridge #3421e6; border-radius: 20px; }
.row > .i { border-width: 6px; border-style: solid double; border-color: #e03c31; border-radius: 14px; }
.row > .j { border-width: 4px 10px; border-style: inset; border-color: #2a9d4a; border-radius: 16px; }
.row > .k { border-width: 9px; border-style: double; border-color: #3421e6; border-width: 9px 3px; border-radius: 14px; }
.row > .l { border-width: 6px; border-style: groove solid; border-color: #e0a030; border-radius: 10px 20px; }
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
	d.Body().Append(row("a", "b", "c", "d", "e", "f"), row("g", "h", "i", "j", "k", "l"))
	pages.Run(d, 520, 150)
}
