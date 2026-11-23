//go:build chrome

package ui_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestAnIframeWithOnlyARatioFillsItsLineAsInChrome(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(ui.Div(ui.Style("width: 400px"),
		ui.Iframe(nil, ui.Style("aspect-ratio: 2 / 1")),
		ui.Iframe(nil, ui.Style("aspect-ratio: 2 / 1; border: 0")),
		ui.Iframe(nil, ui.Style("aspect-ratio: 3 / 1; border: 10px solid red; padding: 5px; box-sizing: content-box")),
		ui.Iframe(nil, ui.Style("aspect-ratio: 2 / 1; max-width: 120px")),
		ui.Iframe(nil, ui.Style("aspect-ratio: 2 / 1; margin: 0 30px")),
		ui.Span(ui.Text("after"))),
		ui.Div(ui.Style("width: 400px"), ui.Iframe(nil, ui.Style("display: block; aspect-ratio: 4 / 1"))))
	chrome.Match(t, d, geom.V2(640, 400))
}

func TestAPercentHeightOfAnInlineReplacedBoxIsItsContainersShareAsInChrome(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(
		ui.Div(ui.Style("width: 400px; height: 300px"),
			ui.Iframe(nil, ui.Style("height: 40%")),
			ui.Iframe(nil, ui.Style("height: 50%; border: 0")),
			ui.Canvas(func(*twod.Canvas) {}, ui.Style("height: 40%")),
			ui.Img(ui.Style("height: 40%; width: 20px"))),
		ui.Div(ui.Style("width: 400px"),
			ui.Iframe(nil, ui.Style("height: 40%")),
			ui.Canvas(func(*twod.Canvas) {}, ui.Style("height: 40%"))))
	chrome.Match(t, d, geom.V2(640, 800))
}
