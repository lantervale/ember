//go:build chrome

package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

const variableProse = "A variable font at any weight and width: its outlines and advances move as its axes say."

func TestAVariableFontIsLaidOutAtItsInstanceAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Auto; src: url(weighty.ttf); }
@font-face { font-family: Fixed; src: url(weighty.ttf); font-weight: 400; font-stretch: normal; }
@font-face { font-family: Ranged; src: url(weighty.ttf); font-weight: 300 700; font-stretch: 75% 125%; }
@font-face { font-family: Phantom; src: url(weighty-phantom.ttf); font-weight: 100 900; }
p { margin: 0 0 6px; width: 320px; font: 15px Auto, monospace; }
span { display: inline-block; font: 24px Auto, monospace; }
.fixed { font-family: Fixed, monospace; }
.ranged { font-family: Ranged, monospace; }
.phantom { font-family: Phantom, monospace; }`))
	for _, s := range []string{"font-weight: 100", "font-weight: 300", "font-weight: 650", "font-weight: 900",
		"font-stretch: 50%", "font-stretch: condensed", "font-stretch: 150%", "font-weight: 777; font-stretch: 66%",
		`font-variation-settings: "wght" 650, "wdth" 75`, `font-variation-settings: "wdth" 140, "wdth" 60`,
		`font-weight: 200; font-variation-settings: "wght" 800`, `font-variation-settings: "XXXX" 3, "wght" 500`} {
		d.Body().Append(ui.Div(ui.Span(ui.Style(s), ui.Text("AVOHé "+s))))
	}
	for _, s := range []string{"font-weight: 900", "font-weight: 200; font-stretch: 150%", "font-stretch: 60%"} {
		d.Body().Append(ui.Div(ui.Span(ui.Class("ranged"), ui.Style(s), ui.Text("Ranged "+s))))
	}
	d.Body().Append(
		ui.Div(ui.Span(ui.Class("fixed"), ui.Style("font-weight: 900"), ui.Text("Fixed at 400, made bold"))),
		ui.Div(ui.Span(ui.Class("phantom"), ui.Style("font-weight: 650; font-stretch: 125%"),
			ui.Text("Phantom points, no HVAR"))),
		ui.P(ui.Style("font-weight: 650"), ui.Text(variableProse)),
		ui.P(ui.Style(`font-variation-settings: "wdth" 70`), ui.Text(variableProse)),
	)
	size := geom.V2(600, 1000)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
