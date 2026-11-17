//go:build chrome

package ui_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// browserSite is a sheet as sites write them: every vendor's prefixes,
// @supports, @layer, and widths that are their content's. Chrome gets it
// as it is written here.
const browserSite = `
@layer reset, layout;
@layer layout {
  .card { width: 200px; padding: 12px; border: 3px solid #335; margin: 4px 0; }
  .row { display: -moz-box; display: -ms-flexbox; display: -webkit-flex; width: 300px; }
  .row > * { -webkit-flex: 1 1 0; -ms-flex: 2; -moz-box-flex: 1; height: 20px; }
}
@layer reset { #first.card { width: 50px; padding: 0; } * { margin: 0; } }
.border { -webkit-box-sizing: border-box; -moz-box-sizing: content-box; }
.calc { width: -webkit-calc(50% - 20px); width: -moz-calc(10px); width: -o-calc(10px); height: 10px; }
.min { width: -webkit-min-content; }
.fit { width: -webkit-fit-content; width: -moz-fit-content; }
@supports (display: grid) { .grid { display: grid; grid-template-columns: 1fr 2fr; width: 300px; } }
@supports not (display: grid) { .grid { width: 10px; } }
@supports (display: -ms-grid) { .grid { height: 77px; } }
@supports (-webkit-appearance: none) and (not (-moz-appearance: none)) { .card { height: 40px; } }
button::-moz-focus-inner, .card { height: 99px; }
.card:-ms-input-placeholder, .card { height: 98px; }
.card, ::-webkit-scrollbar { margin-left: 10px; }
.card, .card:-webkit-autofill { margin-right: 6px; }
input::-webkit-input-placeholder { color: #a00; }
@-moz-document url-prefix() { .card { width: 1px; } }
@-ms-viewport { width: 320px; }
@-khtml-foo { .card { width: 2px; } }
@media screen and (-webkit-min-device-pixel-ratio: 0) { .min { padding: 5px; } }
@media (-moz-min-device-pixel-ratio: 0), (-o-min-device-pixel-ratio: 0/1) { .min { padding: 50px; } }
.imp { width: 80px !important; height: 10px; -webkit-box-sizing: content-box; -khtml-box-sizing: border-box; }
@layer layout { .imp { width: 90px !important; padding: 3px; } }
button, input, textarea, select { width: 120px; height: 30px; padding: 6px; border: 2px solid #335; }
progress, meter { display: block; width: 100px; height: 10px; padding: 2px; }
`

func browserDoc() *ui.Document {
	d := ui.NewDocument(ui.BrowserSheets(), ui.CSS(browserSite))
	d.Body().Append(
		ui.Div(ui.ID("first"), ui.Class("card")),
		ui.Div(ui.Class("card", "border"), ui.Style("-webkit-box-shadow: none; -moz-opacity: 0.5; padding-left: 30px")),
		ui.Div(ui.Class("calc")),
		ui.Div(ui.Class("min"), ui.Text("the narrowest it can be")),
		ui.Div(ui.Class("fit"), ui.Text("as wide as its text")),
		ui.Div(ui.Class("grid"), ui.Div(ui.Text("one")), ui.Div(ui.Text("two"))),
		ui.Div(ui.Class("row"), ui.Div(), ui.Div(), ui.Div()),
		ui.Div(ui.Class("imp")),
		ui.Div(ui.Button(ui.Text("Go"))),
		ui.Div(ui.Input(ui.Attr("placeholder", "name"))),
		ui.Div(ui.Input(ui.Type("checkbox"))),
		ui.Div(ui.Input(ui.Type("search"))),
		ui.Div(ui.Textarea()),
		ui.Div(ui.Select(ui.Option(ui.Text("one")))),
		ui.Div(ui.Progress(ui.Attr("value", "0.5"))),
		ui.Div(ui.Meter(ui.Attr("value", "0.5"))),
	)
	return d
}

func TestABrowsersSheetLaysOutAsChromeReadsIt(t *testing.T) {
	d := browserDoc()
	r := chrome.Match(t, d, geom.V2(800, 900))
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// mediaSite changes with the viewport's width, height and shape; one of
// its queries asks for a feature neither Chrome nor ui knows.
const mediaSite = `
.side { width: 240px; height: 30px; }
.main { height: 40px; }
.note, .em, .nested { height: 5px; }
@media (max-width: 899px) { .side { width: 120px; } }
@media screen and (min-width: 600px) and (max-width: 899px) { .main { margin-left: 30px; } }
@media (width < 600px) { .side { width: auto; } .main { height: 60px; } }
@media not all and (min-width: 600px) { .note { height: 15px; } }
@media print, (orientation: portrait) { .note { width: 50%; } }
@media (min-height: 500px) and (min-aspect-ratio: 3/2) { .note { padding: 7px; } }
@media (min-width: 40em) { .em { width: 40em; } }
@media (hover: hover) and (pointer: fine) { .em { height: 12px; } }
@media (unknown-thing), (min-width: 1000px) { .em { margin-top: 9px; } }
@media (min-width: 600px) { @media (max-height: 650px) { .nested { height: 22px; } } }
@media (400px <= height <= 600px) { .nested { width: 300px; } }
`

func mediaDoc(opts ...ui.DocumentOption) *ui.Document {
	d := ui.NewDocument(append(opts, ui.CSS(mediaSite))...)
	d.Body().Append(ui.Div(ui.Class("side")), ui.Div(ui.Class("main")), ui.Div(ui.Class("note")),
		ui.Div(ui.Class("em")), ui.Div(ui.Class("nested")))
	return d
}

// TestMediaRulesFollowAResizeAsChromes lays one Document out at a
// window's sizes in turn, as a resize does, each crossing some rule's
// boundary, against a Chrome opened at that size.
func TestMediaRulesFollowAResizeAsChromes(t *testing.T) {
	for name, opts := range map[string][]ui.DocumentOption{"a game's sheet": nil,
		"a browser's sheet": {ui.BrowserSheets()}} {
		t.Run(name, func(t *testing.T) {
			d := mediaDoc(opts...)
			for _, size := range []geom.Vec2{{X: 1100, Y: 700}, {X: 800, Y: 600}, {X: 800, Y: 500}, {X: 600, Y: 900},
				{X: 560, Y: 400}, {X: 899, Y: 650}, {X: 900, Y: 651}, {X: 1100, Y: 700}} {
				t.Run(fmt.Sprintf("%gx%g", size.X, size.Y), func(t *testing.T) {
					r := chrome.Match(t, d, size)
					t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
				})
			}
			require.Len(t, d.Warnings(), 1)
			assert.ErrorContains(t, d.Warnings()[0], "the unknown-thing feature is not supported")
		})
	}
}
