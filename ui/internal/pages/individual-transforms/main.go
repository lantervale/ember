// translate, rotate and scale: each alone, about an origin, together and
// with transform after them, nested, mirrored, a scroller their child
// overflows, and a -webkit-box moved down over a red square
// (css-flexbox/flexbox-safe-overflow-position-006).
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 20px; display: grid; grid-template-columns: repeat(4, 110px); gap: 30px; }
.cell { height: 80px; background-color: #fde9c8; }
.cell > div { width: 70px; height: 40px; border: 4px solid #345;
  background-color: #36c; }
.moved > div { translate: 20px 50%; }
.turned > div { rotate: 30deg; }
.scaled > div { scale: 1.4 0.6; transform-origin: 0 0; }
.axis > div { rotate: 0 0 -1 0.1turn; transform-origin: right bottom; }
.all > div { translate: -10px 15px; rotate: -20deg; scale: 0.8; transform: skewX(10deg) translate(5px); }
.nested { rotate: 15deg; }
.nested > div { translate: 15px; scale: 1.2 0.8; rotate: 45deg; }
.mirror > div { scale: -1 1; translate: 0 20px; }
.scroller { overflow: auto; }
.scroller > div { translate: 60px 70px; }
.flexbox { position: relative; height: 100px; background: none; }
.flexbox > .red { position: absolute; width: 100px; height: 100px; background: red; border: 0;
  z-index: -1; }
.flexbox > .box { display: -webkit-box; -webkit-box-align: end; width: 90px; height: 90px; translate: 0 10px;
  background: none; border: 0; }
.flexbox .item { min-width: 100px; min-height: 100px; background: green; align-self: safe center; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"t.css": {Data: []byte(sheet)}}, "t.css"))
	for _, class := range []string{"moved", "turned", "scaled", "axis", "all", "nested", "mirror", "scroller"} {
		d.Body().Append(ui.Div(ui.Class("cell", class), ui.Div()))
	}
	d.Body().Append(ui.Div(ui.Class("cell", "flexbox"), ui.Div(ui.Class("red")),
		ui.Div(ui.Class("box"), ui.Div(ui.Class("item")))))
	pages.Run(d, 600, 420)
}
