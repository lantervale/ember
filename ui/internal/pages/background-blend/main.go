// background-blend-mode (decision 0130): two gradients and a colour
// mixed in each mode, isolated from the page under them; a list of modes
// repeated over more layers, a rounded box, and a body's own background
// mixed on the canvas.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 12px; display: grid; grid-template-columns: repeat(6, 84px); gap: 10px;
  background: linear-gradient(45deg, rgba(255, 255, 255, 0.6), transparent), #8ac;
  background-blend-mode: overlay; }
.cell { height: 64px; background-color: #c63;
  background-image: linear-gradient(90deg, rgba(40, 200, 120, 0.9), #fc3 50%, rgba(80, 40, 200, 0.3)),
    linear-gradient(#e33, #36d 50%, #888 50% 70%, #000 70% 85%, transparent 85%); }
.multiply { background-blend-mode: multiply; }
.screen { background-blend-mode: screen; }
.overlay { background-blend-mode: overlay; }
.darken { background-blend-mode: darken; }
.lighten { background-blend-mode: lighten; }
.color-dodge { background-blend-mode: color-dodge; }
.color-burn { background-blend-mode: color-burn; }
.hard-light { background-blend-mode: hard-light; }
.soft-light { background-blend-mode: soft-light; }
.difference { background-blend-mode: difference; }
.exclusion { background-blend-mode: exclusion; }
.hue { background-blend-mode: hue; }
.saturation { background-blend-mode: saturation; }
.color { background-blend-mode: color; }
.luminosity { background-blend-mode: luminosity; }
.list { background-blend-mode: normal, difference; border-radius: 20px 6px; }
.repeat { background-image: linear-gradient(90deg, #fc3, transparent), linear-gradient(#36d, #e33),
  radial-gradient(#fff, #000); background-blend-mode: multiply, screen; }
`

func main() {
	d := ui.NewDocument(ui.CSS(sheet))
	for _, class := range []string{
		"normal", "multiply", "screen", "overlay", "darken", "lighten", "color-dodge", "color-burn", "hard-light",
		"soft-light", "difference", "exclusion", "hue", "saturation", "color", "luminosity", "list", "repeat",
	} {
		d.Body().Append(ui.Div(ui.Class("cell", class)))
	}
	pages.Run(d, 600, 260)
}
