//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestBrokenPicturesOnTheirLinesMatchChrome has a broken img with alt text stand on that
// text's line, as Chrome's does.
func TestBrokenPicturesOnTheirLinesMatchChrome(t *testing.T) {
	missing := ui.Attr("src", "missing.png")
	const page = `div { width: 300px; margin-bottom: 8px } .f img { width: 100%; height: auto }`
	cases := map[string]*ui.Document{
		"sized by their attributes' proportions": pictured(page,
			ui.Div(ui.Class("f"), ui.Img(missing, ui.Attr("width", "250"), ui.Attr("height", "167"),
				ui.Attr("alt", "a picture"))),
			ui.Div(ui.Class("f"), ui.Text("x"), ui.Img(missing, ui.Attr("width", "40"), ui.Attr("height", "80"),
				ui.Attr("alt", "a picture")))),
		"sized by their attributes": pictured(page,
			ui.Div(ui.Img(missing, ui.Attr("width", "100"), ui.Attr("height", "80"), ui.Attr("alt", "a picture"))),
			ui.Div(ui.Img(missing, ui.Attr("width", "100"), ui.Attr("height", "80")))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
