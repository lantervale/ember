// Controls drawn from their CSS in the states the pointer gives their
// parts, as Chrome's: a range's slider container, track and thumb hovered
// and pressed, a progress's and a meter's value hovered; ranges laid
// down their box by writing-mode and slider-vertical; an indeterminate
// progress's value at no width but its own. TestControlPartsFollowThePointerAsChromes
// moves the mouse over it.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #ffffff; }
body { margin: 0; font-family: "Go"; }
input, progress, meter { position: absolute; margin: 0; }
.none { appearance: none; }
.a { left: 20px; top: 20px; width: 200px; }
.a::-webkit-slider-container { background: #fe0; padding: 3px; }
.a::-webkit-slider-container:hover { background: #fc6; }
.a::-webkit-slider-runnable-track { height: 8px; background: #ccc; }
.a::-webkit-slider-runnable-track:hover { background: #0c0; }
.a::-webkit-slider-runnable-track:active { background: #c0c; }
.a::-webkit-slider-thumb { appearance: none; width: 20px; height: 20px; background: #00f; margin-top: -6px; }
.a::-webkit-slider-thumb:hover { background: #f00; }
.a::-webkit-slider-thumb:active { background: #000; }
.p { left: 260px; top: 20px; width: 150px; height: 14px; }
.p::-webkit-progress-bar { background: #ddd; }
.p::-webkit-progress-bar:hover { background: #bbb; }
.p::-webkit-progress-value { background: #2a7; }
.p::-webkit-progress-value:hover { background: #e80; }
.m { left: 260px; top: 50px; width: 150px; height: 14px; }
.m::-webkit-meter-bar { background: #eee; border: 1px solid #999; }
.m::-webkit-meter-optimum-value { background: #3a3; }
.m::-webkit-meter-optimum-value:hover { background: #36c; }
.v1 { left: 20px; top: 70px; height: 150px; writing-mode: vertical-lr; }
.v1::-webkit-slider-container { background: #fb8; padding: 2px; }
.v1::-webkit-slider-runnable-track { width: 8px; background: #ccc; }
.v1::-webkit-slider-runnable-track:active { background: #c9c; }
.v1::-webkit-slider-thumb { appearance: none; width: 20px; height: 10px; background: #00f; margin-left: -6px; }
.v1::-webkit-slider-thumb:hover { background: #f00; }
.v1::-webkit-slider-thumb:active { background: #000; }
.v2 { left: 70px; top: 70px; height: 150px; writing-mode: vertical-rl; direction: rtl; }
.v2::-webkit-slider-runnable-track { width: 4px; background: #888; }
.v3 { left: 110px; top: 70px; writing-mode: vertical-lr; accent-color: #e04; }
.v4 { left: 150px; top: 70px; width: 20px; height: 150px; appearance: slider-vertical; }
.v5 { left: 190px; top: 70px; height: 150px; writing-mode: sideways-lr; }
.v5::-webkit-slider-runnable-track { width: 6px; background: #9cf; border: 1px solid #036; padding: 2px 0; }
.v5::-webkit-slider-thumb { appearance: none; width: 6px; height: 12px; background: #036; }
.i1 { left: 260px; top: 90px; width: 150px; height: 14px; }
.i1::-webkit-progress-value { background: #f00; min-width: 30px; }
.i2 { left: 260px; top: 120px; width: 150px; height: 14px; }
.i2::-webkit-progress-bar { background: #ddd; padding: 2px; }
.i2::-webkit-progress-value { background: #f00; padding-left: 10px; border: 2px solid #00f; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	rng := func(value string, class ...string) *ui.Node {
		return ui.Input(ui.Type("range"), ui.Value(value), ui.Class(class...))
	}
	d.Body().Append(
		rng("20", "none", "a"),
		ui.Progress(ui.Value("0.5"), ui.Class("none", "p")),
		ui.Meter(ui.Value("0.5"), ui.Class("none", "m")),
		rng("30", "none", "v1"),
		rng("30", "none", "v2"),
		rng("30", "v3"),
		rng("30", "v4"),
		rng("30", "none", "v5"),
		ui.Progress(ui.Class("none", "i1")),
		ui.Progress(ui.Class("none", "i2")),
	)
	pages.Run(d, 440, 240)
}
