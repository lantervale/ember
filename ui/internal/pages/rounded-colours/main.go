// Rounded borders whose sides differ in colour as Chrome draws them: the
// two colours meet on the line from the outer corner to the inner one,
// blended across it, round corners as wide as and wider than the border.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #f4ede4; }
body { padding: 12px; margin: 0; }
.row { display: flex; gap: 16px; align-items: flex-start; margin-bottom: 16px; }
.row > div { width: 48px; height: 32px; background-color: #ffffff; border-style: solid;
  border-color: #e03c31 #3421e6 #2a9d4a #e0a030; }
.a { border-width: 8px; border-radius: 6px; }
.b { border-width: 8px; border-radius: 16px; }
.c { border-width: 4px; border-radius: 20px 4px; }
.d { border-width: 10px; border-radius: 26px; }
.row > .e { border-width: 6px; border-radius: 12px; border-top-color: #2a9d4a; border-left-color: #2a9d4a; }
.row > .f { border-width: 8px; border-radius: 10px; border-color: transparent; border-top-color: #3421e6; }
`

func main() {
	d := ui.NewDocument(ui.CSS(sheet))
	row := ui.Div(ui.Class("row"))
	for _, c := range []string{"a", "b", "c", "d", "e", "f"} {
		row.Append(ui.Div(ui.Class(c)))
	}
	d.Body().Append(row)
	pages.Run(d, 480, 90)
}
