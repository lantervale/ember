//go:build chrome

package svg_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// Pictures and svg elements positioned out of a vertical box into the
// page's horizontal one, or standing across their vertical line, take their
// natural size in their own writing mode, as in Chrome.
func TestPicturesAcrossWritingModesAreSizedAsChromeSizesThem(t *testing.T) {
	const markup = `<svg xmlns="http://www.w3.org/2000/svg" width="44" height="12"><rect width="44" height="12"/></svg>`
	for name, sheet := range map[string]string{
		"positioned":            `.a { position: absolute; }`,
		"positioned of a width": `.a { position: absolute; width: 22px; }`,
		"horizontal":            `.a { writing-mode: horizontal-tb; }`,
		"positioned in vertical-lr": `.v { writing-mode: vertical-lr; } .a { position: absolute;
			max-height: 20px; }`,
	} {
		t.Run(name, func(t *testing.T) {
			d := ui.NewDocument(ui.Stylesheet(samples(t, `.v { writing-mode: vertical-rl; width: 120px;
				height: 100px; border: 1px solid; margin: 10px; } .v > p { margin: 0; inline-size: 20px;
				block-size: 14px; }`+sheet), "page.css"))
			d.Body().Append(ui.Div(ui.Class("v"), ui.P(), ui.Img(ui.Class("a"), ui.Attr("src", "sized.svg")),
				ui.Svg(markup, ui.Class("a")), ui.P()))
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

// A picture with a view box alone has a ratio and no size: an unsized
// flex item and an absolutely positioned one take the width their
// container offers, as in Chrome.
func TestAPictureWithARatioAloneIsSizedAsChromeSizesIt(t *testing.T) {
	for _, src := range []string{"shaped.svg", "sized.svg", "bare.svg"} {
		t.Run(src, func(t *testing.T) {
			d := ui.NewDocument(ui.Stylesheet(samples(t, "body { margin: 0; width: 500px; }"), "page.css"))
			img := func(args ...ui.Arg) *ui.Node { return ui.Img(append([]ui.Arg{ui.Attr("src", src)}, args...)...) }
			d.Body().Append(
				ui.Div(ui.Style("display: flex"), img()),
				ui.Div(ui.Style("display: flex; width: 300px"), img(), img()),
				ui.Div(ui.Style("display: flex; flex-direction: column; width: 200px"), img()),
				ui.Div(ui.Style("display: flex; height: 50px"), img()),
				ui.Div(ui.Style("position: relative; width: 240px; height: 300px"),
					img(ui.Style("position: absolute")),
					img(ui.Style("position: absolute; left: 10px; right: 30px")),
					img(ui.Style("position: absolute; top: 0; bottom: 100px; left: 0"))),
			)
			r := chrome.Match(t, d, geom.V2(800, 900))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
