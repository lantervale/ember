// Solid borders whose sides differ in colour as Chrome draws them: each
// side a trapezoid mitered into its neighbours, so a box of no size with
// one coloured side is a triangle, the arrow of a menu or a tooltip.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #f4ede4; }
body { padding: 12px; margin: 0; }
.row { display: flex; gap: 16px; align-items: center; }
.down { width: 0; height: 0; border: 20px solid transparent; border-top-color: #3421e6; }
.right { width: 0; height: 0; border: 16px solid transparent; border-left-color: #e03c31; border-right-width: 0; }
.up { width: 0; height: 0; border-left: 12px solid transparent; border-right: 24px solid transparent;
  border-bottom: 30px solid #2a2530; }
.four { width: 0; height: 0; border: 24px solid; border-color: #e03c31 #3421e6 #2a9d4a #e0a030; }
.frame { width: 40px; height: 24px; background-color: #ffffff; border: 10px solid;
  border-color: #e03c31 #3421e6 #2a9d4a #e0a030; }
.uneven { width: 40px; height: 24px; background-color: #ffffff; border-style: solid; border-width: 4px 16px 12px 8px;
  border-color: #e03c31 #3421e6 #2a9d4a #e0a030; }
.faded { width: 40px; height: 24px; border: 10px solid rgba(52, 33, 230, 0.5);
  border-top-color: rgba(224, 60, 49, 0.5); }
.same { width: 40px; height: 24px; border: 10px solid #3421e6; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	row := ui.Div(ui.Class("row"))
	for _, c := range []string{"down", "right", "up", "four", "frame", "uneven", "faded", "same"} {
		row.Append(ui.Div(ui.Class(c)))
	}
	d.Body().Append(row)
	pages.Run(d, 560, 90)
}
