// Math functions where only paint reads them: gradient stops of min(),
// round() and clamp() over the gradient line's percentages, conic stops
// of angles and percentages added, gradient angles of atan2() and
// asin(), boxes turned by trigonometry, and basic shapes of math over
// the reference box.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 12px; font-size: 10px; }
.row { display: flex; gap: 12px; margin-bottom: 14px; }
.g { width: 80px; height: 56px; }
.c { width: 64px; height: 64px; }
.t { width: 40px; height: 24px; margin: 12px; background-color: #3a6fd8; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"math.css": {Data: []byte(sheet)}}, "math.css"))
	box := func(class, style string) *ui.Node { return ui.Div(ui.Class(class), ui.Style(style)) }
	d.Body().Append(
		ui.Div(ui.Class("row"),
			box("g", "background-image: linear-gradient(to right, #e63946 min(50%, 30px), #1d3557 min(50%, 30px))"),
			box("g", "background-image: linear-gradient(to right, #222 round(30%, 7px), #ddd max(60%, 4em))"),
			box("g", "background-image: linear-gradient(#f1c40f clamp(10px, 20%, none), #8e44ad mod(90%, 40px))"),
			box("g", "background-image: linear-gradient(atan2(1em, 20px), #16a085, #f39c12 abs(-80%))"),
			box("g", "background-image: linear-gradient(asin(0.5), #2980b9 hypot(20%, 9px), #ecf0f1)"),
		),
		ui.Div(ui.Class("row"),
			box("c", "background-image: conic-gradient(#e63946 calc(10% + 36deg), #1d3557 0)"),
			box("c", "background-image: conic-gradient(from atan2(1, 1), #f1c40f round(33%, 10%), #8e44ad 0)"),
			box("c", "background-image: conic-gradient(#16a085 0 min(25%, 1rad), #f39c12 0 max(50%, 2rad), #222 0)"),
		),
		ui.Div(ui.Class("row"),
			box("t", "transform: rotate(atan2(1em, 20px))"),
			box("t", "transform: rotate(calc(acos(-1) / 4)) scale(sign(1px), abs(-0.5))"),
			box("t", "transform: skew(atan(0.5), 0) translateX(round(1.3em, 4px))"),
			box("t", "opacity: calc(1em / 20px)"),
		),
		ui.Div(ui.Class("row"),
			box("g", "background-color: #e63946; clip-path: inset(min(10%, 4px) round(up, 20%, 7px) 0 "+
				"clamp(1px, 30%, 2em) round 4px)"),
			box("g", "background-color: #1d3557; clip-path: xywh(min(10%, 5px) 4px max(50%, 30px) calc(100% - 8px))"),
			box("g", "background-color: #16a085; clip-path: circle(hypot(30%, 10px) at round(50%, 8px) 50%)"),
			box("g", "background-color: #8e44ad; clip-path: polygon(min(50%, 20px) 0, 100% abs(-50%), "+
				"0 calc(100% * sign(50% - 10px)))"),
			box("g", "background-color: #f39c12; clip-path: ellipse(mod(70%, 30px) rem(80%, 20px))"),
		),
	)
	pages.Run(d, 520, 330)
}
