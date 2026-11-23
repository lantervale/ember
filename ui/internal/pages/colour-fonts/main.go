// Colour fonts against Chrome's: COLR version 0 layers in each palette
// font-palette picks, version 1's gradients, transforms and composites,
// a stroke that leaves them as they are, sbix bitmaps, CBDT ones in a font
// without outlines, and an SVG table Chrome leaves to the outlines.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/internal/colourfont/colourtest"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
@font-face { font-family: Layered; src: url(layered.ttf); }
@font-face { font-family: Painted; src: url(painted.ttf); }
@font-face { font-family: Struck; src: url(struck.ttf); }
@font-face { font-family: Bitmapped; src: url(bitmapped.ttf); }
@font-face { font-family: Drawn; src: url(drawn.ttf); }
@font-palette-values --blue { font-family: Layered; base-palette: 2; override-colors: 0 #00f; }
@font-palette-values --second { font-family: Painted; base-palette: 1; }
html { background-color: #fff; }
body { padding: 8px; color: #333; font-size: 32px; }
div { white-space: nowrap; height: 44px; }
.layered { font-family: Layered; }
.painted { font-family: Painted; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"faces.css":     {Data: []byte(sheet)},
		"layered.ttf":   {Data: colourtest.COLRv0()},
		"painted.ttf":   {Data: colourtest.COLRv1()},
		"struck.ttf":    {Data: colourtest.Sbix(true)},
		"bitmapped.ttf": {Data: colourtest.CBDT(false)},
		"drawn.ttf":     {Data: colourtest.SVG(false)},
	}, "faces.css"))
	d.Body().Append(
		ui.Div(ui.Class("layered"), ui.Text("ABC xAx "),
			ui.Span(ui.Style("font-palette: light"), ui.Text("ABC ")),
			ui.Span(ui.Style("font-palette: dark"), ui.Text("ABC ")),
			ui.Span(ui.Style("font-palette: --blue"), ui.Text("ABC"))),
		ui.Div(ui.Class("painted"), ui.Text("ABCDEFG")),
		ui.Div(ui.Class("painted"), ui.Text("HIJKLMN")),
		ui.Div(ui.Class("painted"), ui.Style("font-palette: --second; color: #c0c"), ui.Text("ABCHJK")),
		ui.Div(ui.Class("painted"), ui.Style("-webkit-text-stroke: 2px #000"), ui.Text("ABDx")),
		ui.Div(ui.Style("font-family: Struck"), ui.Text("ABC "),
			ui.Span(ui.Style("font-size: 20px"), ui.Text("ABC "))),
		ui.Div(ui.Style("font-family: Bitmapped"), ui.Text("ABC "),
			ui.Span(ui.Style("font-family: Drawn; color: #a00"), ui.Text("ABC"))),
	)
	pages.Run(d, 520, 340)
}
