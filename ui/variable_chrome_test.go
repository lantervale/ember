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

// TestAVariableFontsLayoutIsItsInstancesAsInChrome lays out a font whose
// kerning and mark anchors vary by GDEF's deltas, and whose GSUB and GPOS
// swap features by axis ranges (varfonttest.KernAV).
func TestAVariableFontsLayoutIsItsInstancesAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Varied; src: url(weighty-layout.ttf); font-weight: 100 900; font-stretch: 50% 150%; }
p { margin: 0 0 6px; width: 300px; font: 15px Varied, monospace; }
span { display: inline-block; font: 24px Varied, monospace; }`))
	for _, s := range []string{"", "font-weight: 100", "font-weight: 250", "font-weight: 260", "font-weight: 525",
		"font-weight: 550", "font-weight: 650", "font-weight: 900", "font-stretch: 120%", "font-stretch: 125%",
		"font-stretch: 150%", "font-weight: 700; font-stretch: 150%", "font-weight: 380; font-stretch: 140%",
		`font-variation-settings: "wght" 800, "wdth" 60`, `font-weight: 900; font-kerning: none`} {
		d.Body().Append(ui.Div(ui.Span(ui.Style(s), ui.Text("AV To LY x^x^ $z "+s))))
	}
	d.Body().Append(
		ui.P(ui.Style("font-weight: 777"), ui.Text(variedProse)),
		ui.P(ui.Style("font-weight: 150; font-stretch: 130%"), ui.Text(variedProse)),
	)
	size := geom.V2(760, 1000)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

const variedProse = "AVAST, To LYNX: zealous $ AVOWALS. Today LYRICAL Toyota VANS $z, x^ AVAIL To all."

func TestFontSynthesisWithoutSmallCapsDrawsTheLettersAsTheyAreAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Plain; src: url(go-regular-latin.woff2); }
span { display: inline-block; font: 20px Plain, monospace; font-variant-caps: small-caps; }
.none { font-synthesis: none; }
.weight { font-synthesis: weight style; }
.caps { font-synthesis: small-caps; }
.all { font-variant-caps: all-small-caps; font-synthesis: weight; }`))
	for _, class := range []string{"", "none", "weight", "caps", "all"} {
		d.Body().Append(ui.Div(ui.Span(ui.Class(class), ui.Text("Small Capitals, made up or not"))))
	}
	size := geom.V2(600, 300)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
