// A page whose root is in vertical-rl, wider than its window, its
// viewport scrolled from the right part of the way left: columns of
// turned text, a sticky head held at the viewport's right edge and a
// fixed badge in its corner. The viewport shows no bar, as tall.go's.
package main

import (
	"fmt"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { writing-mode: vertical-rl; background-color: #f4efe6; scrollbar-width: none; }
body { margin: 12px; }
#head { position: sticky; right: 0; width: 26px; background-color: #ad9773; color: #fff; }
#badge { position: fixed; left: 10px; top: 10px; width: 60px; height: 30px; background-color: #4f7cff; }
.col { width: 40px; margin-left: 6px; background-color: #e0d4c0; border-right: 3px solid #3a332c; }
`

func main() {
	d := ui.NewDocument(ui.ScrollingViewport(), ui.CSS(sheet))
	d.Body().Append(ui.Div(ui.ID("head"), ui.Text("Head")), ui.Div(ui.ID("badge")))
	for i := range 24 {
		d.Body().Append(ui.Div(ui.Class("col"), ui.Text(fmt.Sprint("column ", i))))
	}
	dom.Data(d.HTML()).Scroll = geom.V2(-300, 0)
	pages.Run(d, 400, 240)
}
