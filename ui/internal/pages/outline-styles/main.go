// Outlines in every style Chrome draws: dashed and dotted, inset and
// outset, groove and ridge, square and round, and auto, its focus ring.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #f4ede4; }
body { padding: 20px; margin: 0; }
.row { display: flex; gap: 28px; align-items: flex-start; margin-bottom: 28px; }
.row > div { width: 48px; height: 32px; background-color: #ffffff; outline-color: #3421e6; }
.a { outline: 2px dashed; }
.b { outline: 3px dotted; }
.c { outline: 4px inset; }
.d { outline: 4px outset; }
.e { outline: 6px groove; }
.f { outline: 6px ridge; }
.g { outline: 4px dashed; outline-offset: 3px; border-radius: 10px; }
.h { outline: 4px dotted; border-radius: 16px; }
.i { outline: 6px groove; border-radius: 12px; }
.j { outline: auto; }
.row > .k { outline: 3px auto #e03c31; outline-offset: 2px; }
.l { outline: auto; border-radius: 12px; }
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
	pages.Run(d, 520, 160)
}
