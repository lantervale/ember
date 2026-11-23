// Controls of appearance: none drawn from their CSS as Chrome draws them:
// checkboxes and radios empty boxes, those of their own look in their
// accent-color, ranges of a styled track and thumb
// or Chrome's own thumb, progress bars and meters of their default and
// styled parts.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #ffffff; }
body { padding: 8px; margin: 0; font-family: "Go"; }
.row { display: flex; gap: 12px; align-items: center; margin-bottom: 10px; }
.none { appearance: none; }
.box { width: 14px; height: 14px; border: 2px solid #c03; border-radius: 3px; }
.box:checked { background-color: #c03; }
input[type="range"] { width: 120px; }
.r::-webkit-slider-runnable-track { height: 4px; background: #c00; border-radius: 2px; }
.t::-webkit-slider-thumb { appearance: none; width: 16px; height: 16px; border-radius: 50%;
  background: #00c; margin-top: -6px; }
.tall::-webkit-slider-runnable-track { background: #9c6; }
.tall::-webkit-slider-thumb { appearance: none; width: 10px; height: 24px; background: #036; }
.padded::-webkit-slider-runnable-track { height: 12px; padding: 0 4px; background: #fc6; border: 1px solid #630; }
.padded::-webkit-slider-thumb { appearance: none; width: 8px; height: 12px; background: #630; }
progress, meter { width: 120px; height: 14px; }
.p::-webkit-progress-bar { background-color: #ddd; border-radius: 7px; padding: 2px; }
.p::-webkit-progress-value { background-color: #2a7; border-radius: 5px; }
.pi::-webkit-progress-inner-element { border: 2px solid #345; }
.m::-webkit-meter-bar { background: #eee; border: 1px solid #999; border-radius: 3px; }
.m::-webkit-meter-optimum-value { background: #3a3; }
.m::-webkit-meter-suboptimum-value { background: #ec3; }
.m::-webkit-meter-even-less-good-value { background: #d33; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	none := ui.Class("none")
	d.Body().Append(
		ui.Div(ui.Class("row"),
			ui.Input(ui.Type("checkbox"), ui.Checked(true)),
			ui.Input(ui.Type("checkbox"), ui.Checked(true), none),
			ui.Input(ui.Type("checkbox"), ui.Class("none", "box")),
			ui.Input(ui.Type("checkbox"), ui.Checked(true), ui.Class("none", "box")),
			ui.Input(ui.Type("radio"), ui.Checked(true), ui.Class("none", "box")),
		),
		ui.Div(ui.Class("row"),
			ui.Input(ui.Type("checkbox"), ui.Checked(true), ui.Style("accent-color: #e04")),
			ui.Input(ui.Type("checkbox"), ui.Checked(true), ui.Style("accent-color: #fe8")),
			ui.Input(ui.Type("radio"), ui.Checked(true), ui.Style("accent-color: #e04")),
			ui.Input(ui.Type("range"), ui.Value("40"), ui.Class("none"), ui.Style("accent-color: #e04")),
		),
		ui.Div(ui.Class("row"),
			ui.Input(ui.Type("range"), ui.Value("25"), ui.Class("none", "r", "t")),
			ui.Input(ui.Type("range"), ui.Value("80"), ui.Class("none", "r")),
			ui.Input(ui.Type("range"), ui.Value("100"), ui.Class("none", "tall")),
		),
		ui.Div(ui.Class("row"),
			ui.Input(ui.Type("range"), ui.Value("0"), ui.Class("none", "padded")),
			ui.Input(ui.Type("range"), ui.Value("50"), ui.Class("none", "padded"), ui.Attr("dir", "rtl")),
		),
		ui.Div(ui.Class("row"),
			ui.Progress(ui.Value("0.3"), none),
			ui.Progress(ui.Value("0.6"), ui.Class("none", "p")),
			ui.Progress(ui.Value("0.6"), ui.Class("none", "pi")),
		),
		ui.Div(ui.Class("row"),
			ui.Meter(ui.Value("0.5"), none),
			ui.Meter(ui.Value("0.9"), ui.Attr("low", "0.3"), ui.Attr("high", "0.6"), ui.Attr("optimum", "0.1"), none),
			ui.Meter(ui.Value("0.5"), ui.Attr("low", "0.3"), ui.Attr("high", "0.6"), ui.Attr("optimum", "0.1"), none),
		),
		ui.Div(ui.Class("row"),
			ui.Meter(ui.Value("0.7"), ui.Class("none", "m")),
			ui.Meter(ui.Value("0.5"), ui.Attr("low", "0.6"), ui.Attr("optimum", "0.9"), ui.Class("none", "m")),
			ui.Meter(ui.Value("0.1"), ui.Attr("low", "0.6"), ui.Attr("high", "0.8"), ui.Attr("optimum", "0.9"),
				ui.Class("none", "m")),
		),
	)
	pages.Run(d, 440, 280)
}
