// mix-blend-mode (decision 0130): each mode, a card of translucent and
// opaque stripes mixed with the page's stripes under it; the same card
// in an isolated group, which mixes only with what the group drew; a
// translucent group mixing as one picture; a turned card; and a filtered card faded
// once, as one picture.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 12px; display: grid; grid-template-columns: repeat(6, 84px); gap: 10px;
  background: linear-gradient(90deg, #f6d 0 20%, #2a8 20% 40%, #fe4 40% 60%, #136 60% 80%, #fff 80%); }
.cell { height: 64px; background: linear-gradient(#e33, #36d 50%, #888 50% 70%, #000 70% 85%, #fff 85%); }
.cell > div { margin: 10px; height: 44px; border-radius: 6px;
  background: linear-gradient(90deg, rgba(40, 200, 120, 0.9), #fc3 50%, rgba(80, 40, 200, 0.5)); }
.multiply > div { mix-blend-mode: multiply; }
.screen > div { mix-blend-mode: screen; }
.overlay > div { mix-blend-mode: overlay; }
.darken > div { mix-blend-mode: darken; }
.lighten > div { mix-blend-mode: lighten; }
.color-dodge > div { mix-blend-mode: color-dodge; }
.color-burn > div { mix-blend-mode: color-burn; }
.hard-light > div { mix-blend-mode: hard-light; }
.soft-light > div { mix-blend-mode: soft-light; }
.difference > div { mix-blend-mode: difference; }
.exclusion > div { mix-blend-mode: exclusion; }
.hue > div { mix-blend-mode: hue; }
.saturation > div { mix-blend-mode: saturation; }
.color > div { mix-blend-mode: color; }
.luminosity > div { mix-blend-mode: luminosity; }
.isolated { isolation: isolate; background: none; }
.isolated > div { mix-blend-mode: difference; background: #fc3; }
.isolated > div:first-child { margin: 4px; height: 30px; mix-blend-mode: normal; background: #36d; }
.isolated > div:last-child { margin: -20px 14px 0; height: 40px; }
.faded { opacity: 0.6; }
.faded > div { mix-blend-mode: multiply; }
.group { position: relative; opacity: 0.5; mix-blend-mode: hard-light; }
.group span { position: absolute; width: 40px; height: 40px; }
.group span:first-child { left: 6px; top: 6px; background-color: #36d; }
.group span:last-child { left: 30px; top: 18px; background-color: #e33; }
.turned > div { mix-blend-mode: difference; transform: rotate(12deg) scale(0.9); }
.both { opacity: 0.5; filter: grayscale(1) drop-shadow(4px 4px 0 #234); }
`

func main() {
	d := ui.NewDocument(ui.CSS(sheet))
	for _, class := range []string{
		"normal", "multiply", "screen", "overlay", "darken", "lighten", "color-dodge", "color-burn", "hard-light",
		"soft-light", "difference", "exclusion", "hue", "saturation", "color", "luminosity", "faded", "turned",
	} {
		d.Body().Append(ui.Div(ui.Class("cell", class), ui.Div()))
	}
	d.Body().Append(
		ui.Div(ui.Class("cell", "isolated"), ui.Div(), ui.Div()),
		ui.Div(ui.Class("cell", "group"), ui.Span(), ui.Span()),
		ui.Div(ui.Class("cell", "both"), ui.Div()),
	)
	pages.Run(d, 600, 330)
}
