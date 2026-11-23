// Controls' generated content as Chrome lays it out among their parts:
// progress bars and meters of appearance none in a line of text, with
// ::before and ::after on lines above and below their bars; ranges of
// either look with labels at the ends of their row; checkboxes and
// radios of appearance none drawn by a block ::before from their top.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #ffffff; }
body { padding: 8px; margin: 0; font: 14px "Go"; }
div { margin-bottom: 8px; }
.none { appearance: none; }
progress, meter { width: 100px; height: 12px; }
.p::-webkit-progress-bar { background-color: #ddd; border-radius: 6px; }
.p::-webkit-progress-value { background-color: #2a7; border-radius: 6px; }
.m::-webkit-meter-bar { background: #eee; border: 1px solid #999; }
.m::-webkit-meter-optimum-value { background: #3a3; }
.lo::before { content: "0%"; color: #555; } .hi::after { content: "100%"; color: #555; }
.label::before { content: "Loading"; display: block; font-size: 11px; color: #036; }
input[type="range"] { width: 160px; }
.r::-webkit-slider-runnable-track { height: 4px; background: #c00; border-radius: 2px; }
.r::-webkit-slider-thumb { appearance: none; width: 12px; height: 12px; border-radius: 50%; background: #00c;
  margin-top: -4px; }
.ends::before { content: "-"; padding: 0 4px; color: #900; }
.ends::after { content: "+"; padding: 0 4px; color: #090; }
.big::before { content: "Vol"; font-size: 22px; }
.box { width: 22px; height: 22px; border: 2px solid #345; border-radius: 4px; margin: 0 6px; }
.box::before { content: ""; display: block; height: 8px; background: #fc6; }
.box:checked::before { background: #345; }
.tick::after { content: "on"; display: block; font-size: 9px; color: #c03; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(
		ui.Div(ui.Text("Saved "), ui.Progress(ui.Value("0.4"), ui.Class("none", "p")), ui.Text(" then "),
			ui.Meter(ui.Value("0.7"), ui.Class("none", "m")), ui.Text(" and "), ui.Progress(ui.Value("0.4")),
			ui.Text(" end")),
		ui.Div(ui.Text("Bar "), ui.Progress(ui.Value("0.6"), ui.Class("none", "p", "lo", "hi")), ui.Text(" after")),
		ui.Div(ui.Text("Fetch "), ui.Progress(ui.Value("0.3"), ui.Class("none", "p", "label")),
			ui.Text(" and "), ui.Meter(ui.Value("0.5"), ui.Class("none", "m", "hi")), ui.Text(" end")),
		ui.Div(ui.Text("Level "), ui.Input(ui.Type("range"), ui.Value("30"), ui.Class("none", "r", "ends")),
			ui.Text(" after")),
		ui.Div(ui.Text("Mix "), ui.Input(ui.Type("range"), ui.Value("70"), ui.Class("none", "r", "big")),
			ui.Input(ui.Type("range"), ui.Value("50"), ui.Class("none", "r", "ends"), ui.Attr("dir", "rtl"))),
		ui.Div(ui.Text("Own "), ui.Input(ui.Type("range"), ui.Value("40"), ui.Class("ends")), ui.Text(" look")),
		ui.Div(ui.Text("Pick "), ui.Input(ui.Type("checkbox"), ui.Class("none", "box")),
			ui.Input(ui.Type("checkbox"), ui.Checked(true), ui.Class("none", "box", "tick")),
			ui.Input(ui.Type("radio"), ui.Checked(true), ui.Class("none", "box")), ui.Text(" end")),
	)
	pages.Run(d, 420, 280)
}
