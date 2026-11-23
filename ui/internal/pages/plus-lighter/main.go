// mix-blend-mode: plus-lighter: cards of translucent and opaque stripes
// added to the page's stripes under them, up to white; the same card in
// an isolated group, which adds only to what the group drew; and a
// translucent card.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 12px; display: grid; grid-template-columns: repeat(4, 84px); gap: 10px;
  background: linear-gradient(90deg, #f6d 0 20%, #2a8 20% 40%, #fe4 40% 60%, #136 60% 80%, #fff 80%); }
.cell { height: 64px; background: linear-gradient(#e33, #36d 50%, #888 50% 70%, #000 70% 85%, #fff 85%); }
.cell > div { margin: 10px; height: 44px; border-radius: 6px; mix-blend-mode: plus-lighter;
  background: linear-gradient(90deg, rgba(40, 200, 120, 0.9), #fc3 50%, rgba(80, 40, 200, 0.5)); }
.dark > div { background: linear-gradient(90deg, #102030, rgba(60, 20, 10, 0.6)); }
.isolated { isolation: isolate; background: none; }
.isolated > div:first-child { margin: 4px; height: 30px; mix-blend-mode: normal; background: #36d; }
.isolated > div:last-child { margin: -20px 14px 0; height: 40px; }
.faded > div { opacity: 0.5; }
.empty { background: none; }
`

func main() {
	d := ui.NewDocument(ui.CSS(sheet))
	for _, class := range []string{"plain", "dark", "faded", "empty"} {
		d.Body().Append(ui.Div(ui.Class("cell", class), ui.Div()))
	}
	d.Body().Append(ui.Div(ui.Class("cell", "isolated"), ui.Div(), ui.Div()))
	pages.Run(d, 420, 180)
}
