//go:build chrome

package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestColourFontsMeasureAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Layered; src: url(layered.ttf) tech(color-COLRv0); }
@font-face { font-family: Painted; src: url(painted.ttf) format("truetype") tech(color-COLRv1); }
@font-face { font-family: Struck; src: url(struck.ttf) tech(color-sbix); }
@font-face { font-family: Bitmapped; src: url(bitmapped.ttf) tech(color-CBDT); }
@font-face { font-family: Drawn; src: url(drawn.ttf) tech(color-SVG), url(heavy.ttf); }
@font-palette-values --blue { font-family: Layered; base-palette: 2; override-colors: 0 #00f; }
span { display: inline-block; font: 30px Layered, monospace; }
.painted { font-family: Painted, monospace; }
.struck { font-family: Struck, monospace; }
.bitmapped { font-family: Bitmapped, monospace; }
.drawn { font-family: Drawn, monospace; }
.blue { font-palette: --blue; }`))
	d.Body().Append(
		ui.Div(ui.Span(ui.Text("ABC xAx")), ui.Span(ui.Class("blue"), ui.Text("ABC"))),
		ui.Div(ui.Span(ui.Class("painted"), ui.Text("ABCDEFGHIJKLMN"))),
		ui.Div(ui.Span(ui.Class("struck"), ui.Text("ABC abc")), ui.Span(ui.Class("bitmapped"), ui.Text("ABC abc"))),
		ui.Div(ui.Span(ui.Class("drawn"), ui.Text("ABC, of the next source"))),
	)
	size := geom.V2(600, 400)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestACollectionDrawsItsFirstFontAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: First; src: url(bold-mono.ttc); }
@font-face { font-family: Second; src: url(bold-mono.ttc#1); }
span { display: inline-block; font: 24px First, monospace; }
.second { font-family: Second, monospace; }`))
	d.Body().Append(
		ui.Div(ui.Span(ui.ID("first"), ui.Text("AVATAR, the first font"))),
		ui.Div(ui.Span(ui.ID("second"), ui.Class("second"), ui.Text("AVATAR, the first font"))),
	)
	size := geom.V2(600, 200)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	assert.InDelta(t, d.ByID("first").Box().W, d.ByID("second").Box().W, 0.01, "a fragment picks no other font")
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestColourFontTechsAreSupportedAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
div { width: 10px; height: 10px; }
@supports font-tech(color-COLRv0) { .colrv0 { width: 20px; } }
@supports font-tech(color-COLRv1) { .colrv1 { width: 20px; } }
@supports font-tech(color-sbix) { .sbix { width: 20px; } }
@supports font-tech(color-CBDT) { .cbdt { width: 20px; } }
@supports font-tech(color-SVG) { .svg { width: 20px; } }
@supports not font-tech(color-SVG) { .not-svg { width: 30px; } }`), ui.BrowserSheets())
	for _, c := range []string{"colrv0", "colrv1", "sbix", "cbdt", "svg", "not-svg"} {
		d.Body().Append(ui.Div(ui.Class(c)))
	}
	size := geom.V2(200, 200)
	d.Layout(size)
	r := chrome.Match(t, d, size)
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
