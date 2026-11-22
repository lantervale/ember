// Filters: each colour function on a striped card, blurs, drop shadows,
// lists of them, a group of overlapping squares made translucent as one
// picture, a blur cut by its parent's overflow, and a shadowed, blurred
// card cut by its clip-path after its filters.
package main

import (
	"strings"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 14px; display: grid; grid-template-columns: repeat(5, 96px); gap: 18px; }
.card { height: 72px; border: 3px solid #234; border-radius: 10px;
  background: linear-gradient(90deg, #e33 0 25%, #fc3 25% 50%, #3b5 50% 75%, #36d 75%); }
.card > div { margin: 18px auto; width: 34px; height: 34px; background-color: #fff; border-radius: 50%; }
.brightness { filter: brightness(1.6); }
.contrast { filter: contrast(0.4); }
.grayscale { filter: grayscale(1); }
.invert { filter: invert(1); }
.opacity { filter: opacity(0.4); }
.saturate { filter: saturate(3); }
.sepia { filter: sepia(0.8); }
.hue { filter: hue-rotate(120deg); }
.blur { filter: blur(3px); }
.shadow { filter: drop-shadow(5px 6px 4px rgba(10, 20, 40, 0.7)); }
.list { filter: sepia(1) hue-rotate(200deg) saturate(2); }
.after { filter: drop-shadow(0 0 6px #e33) blur(1px); }
.before { filter: invert(1) drop-shadow(4px 4px 0 #234); }
.group { position: relative; height: 78px; filter: opacity(0.5); }
.group span { position: absolute; width: 50px; height: 50px; }
.group span:first-child { left: 0; top: 0; background-color: #36d; }
.group span:last-child { left: 30px; top: 24px; background-color: #e33; }
.cut { height: 78px; overflow: hidden; border: 2px solid #234; }
.cut > div { margin: 20px; height: 34px; background-color: #3b5; filter: blur(5px); }
.half { filter: grayscale(0.5) brightness(0.8) contrast(1.4); }
.clipped { filter: drop-shadow(8px 6px 0 #234) blur(1px); clip-path: inset(4px 0 0 0 round 0 0 24px 24px); }
`

func main() {
	d := ui.NewDocument(ui.CSS(sheet))
	for _, class := range []string{
		"brightness", "contrast", "grayscale", "invert", "opacity", "saturate", "sepia", "hue", "blur", "shadow",
		"list", "after", "before", "half", "clipped",
	} {
		d.Body().Append(ui.Div(ui.Class(append([]string{"card"}, strings.Fields(class)...)...), ui.Div()))
	}
	d.Body().Append(ui.Div(ui.Class("group"), ui.Span(), ui.Span()), ui.Div(ui.Class("cut"), ui.Div()))
	pages.Run(d, 600, 420)
}
