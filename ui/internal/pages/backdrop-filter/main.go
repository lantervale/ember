// backdrop-filter (decision 0130): frosted panels over striped cards,
// blurred, recoloured and both, rounded, translucent, transformed, one
// over a mixed card, and one under a backdrop root that stops what it
// reads.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 12px; display: grid; grid-template-columns: repeat(4, 120px); gap: 14px; }
.cell { position: relative; height: 96px;
  background: linear-gradient(90deg, #e33 0 20%, #fc3 20% 40%, #3b5 40% 60%, #36d 60% 80%, #222 80%); }
.cell::before { content: ""; position: absolute; left: 14px; top: 10px; width: 30px; height: 30px;
  border-radius: 50%; background-color: #fff; }
.cell > div { position: absolute; left: 20px; top: 24px; width: 80px; height: 50px; }
.blur > div { backdrop-filter: blur(4px); }
.frost > div { backdrop-filter: blur(6px) saturate(1.8); background-color: rgba(255, 255, 255, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.7); border-radius: 12px; }
.invert > div { backdrop-filter: invert(1); }
.gray > div { backdrop-filter: grayscale(1) brightness(1.4); border-radius: 50% 10px; }
.sepia > div { backdrop-filter: sepia(1) hue-rotate(90deg); }
.faded > div { backdrop-filter: invert(1); opacity: 0.5; }
.turned > div { backdrop-filter: grayscale(1); transform: rotate(15deg); }
.text > div { backdrop-filter: contrast(0.3); font: 14px sans-serif; color: #000; }
.rooted { opacity: 0.99; }
.rooted > div { backdrop-filter: invert(1); }
.wide > div { left: -6px; width: 132px; backdrop-filter: blur(3px); }
.zero > div { backdrop-filter: blur(0); background-color: rgba(0, 0, 0, 0.2); }
.none > div { backdrop-filter: none; background-color: rgba(0, 0, 0, 0.2); }
`

func main() {
	d := ui.NewDocument(ui.CSS(sheet))
	for _, class := range []string{"blur", "frost", "invert", "gray", "sepia", "faded", "turned", "rooted",
		"wide", "zero", "none"} {
		d.Body().Append(ui.Div(ui.Class("cell", class), ui.Div()))
	}
	d.Body().Append(ui.Div(ui.Class("cell", "text"), ui.Div(ui.Text("frosted"))))
	pages.Run(d, 560, 360)
}
