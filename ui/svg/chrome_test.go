//go:build chrome

package svg_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// An img is sized as Chrome sizes it for each kind of file: one asking
// for a width and a height, one with a view box alone, one with neither,
// and one with a size and no view box, by itself and under each way a
// page sizes it.
func TestSVGImagesAreSizedAsChromeSizesThem(t *testing.T) {
	ways := [][]ui.Arg{
		nil,
		{ui.Attr("width", "100")},
		{ui.Attr("height", "30")},
		{ui.Attr("width", "100"), ui.Attr("height", "30")},
		{ui.Style("width: 120px")},
		{ui.Style("height: 40px")},
		{ui.Style("width: 90px; height: 70px")},
		{ui.Style("max-width: 50px")},
		{ui.Style("min-height: 200px")},
		{ui.Style("width: 50%")},
		{ui.Style("display: block")},
		{ui.Style("max-width: 100%")},
		{ui.Style("margin: 0 10px; padding: 5px; border: 2px solid")},
		{ui.Attr("width", "100"), ui.Style("width: 60px")},
	}
	for _, src := range []string{"sized.svg", "shaped.svg", "bare.svg", "plain.svg"} {
		t.Run(src, func(t *testing.T) {
			d := ui.NewDocument(ui.Stylesheet(samples(t, "body { margin: 0; width: 500px; }"), "page.css"))
			for _, way := range ways {
				d.Body().Append(ui.Div(ui.Img(append([]ui.Arg{ui.Attr("src", src)}, way...)...)))
			}
			onLine := ui.Img(ui.Attr("src", src), ui.Style("height: 20px"))
			d.Body().Append(ui.P(ui.Text("beside text "), onLine, ui.Text(" on a line")))
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

// Where its block shrinks to its content, a picture with a view box
// alone adds nothing to it and is as wide: nothing, in a float or an
// inline block.
func TestSVGImagesInShrunkBlocksAndFlexRowsAreSizedAsChromeSizesThem(t *testing.T) {
	for _, src := range []string{"sized.svg", "shaped.svg", "bare.svg"} {
		t.Run(src, func(t *testing.T) {
			d := ui.NewDocument(ui.Stylesheet(samples(t, "body { margin: 0; width: 500px; }"), "page.css"))
			img := func(args ...ui.Arg) *ui.Node { return ui.Img(append([]ui.Arg{ui.Attr("src", src)}, args...)...) }
			d.Body().Append(
				ui.Div(ui.Style("display: flex"), img(ui.Style("height: 30px")), img(ui.Style("width: 50px"))),
				ui.Div(ui.Style("display: flow-root"), ui.Div(ui.Style("float: left"), img())),
				ui.Div(ui.Style("display: inline-block"), img()),
			)
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
