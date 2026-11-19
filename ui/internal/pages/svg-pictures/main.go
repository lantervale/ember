// SVG pictures against Chrome's: as img elements at the size their file
// asks for and at sizes the page gives, and as backgrounds under each
// form of background-size and each repeat, for the three kinds of file:
// one asking for a width and a height, one with a view box alone, which
// has proportions and no size, and one with neither. The pictures are
// svg's own samples (svg/testdata, its README has their licences) and
// bare.svg, ember's own.
package main

import (
	"embed"
	"strings"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
	_ "github.com/lantervale/ember/ui/svg"
)

//go:embed *.svg page.css
var files embed.FS

func main() {
	d := ui.NewDocument(ui.Stylesheet(files, "page.css"))
	img := func(src string, args ...ui.Arg) *ui.Node {
		return ui.Img(append([]ui.Arg{ui.Attr("src", src)}, args...)...)
	}
	d.Body().Append(ui.Div(ui.Class("imgs"),
		img("own-use.svg"), img("feather-settings.svg"), img("material-home.svg", ui.Attr("width", "40")),
		img("feather-settings.svg", ui.Style("width: 64px")), img("own-no-viewbox.svg"),
		img("own-no-viewbox.svg", ui.Style("width: 90px; height: 40px")),
		img("own-gradients.svg", ui.Style("width: 80px")), img("heroicons-solid-cog.svg", ui.Attr("height", "56")),
		img("own-par.svg", ui.Style("width: 60px; height: 60px")),
		img("own-strokes.svg", ui.Style("width: 110px; height: 60px; object-fit: cover")),
		img("own-gradients.svg", ui.Style("width: 60px; height: 90px; object-fit: contain; object-position: left bottom")),
		img("bare.svg", ui.Style("width: 120px; height: 70px")), img("bare.svg", ui.Attr("width", "50")),
	))
	grid := ui.Div(ui.Class("grid"))
	for _, kind := range []string{"sized", "shaped", "bare"} {
		for _, size := range []string{
			"auto", "length", "one", "high", "percent", "cover", "contain", "over", "tiled", "across", "down",
		} {
			grid.Append(ui.Div(ui.Class(strings.Fields(kind + " " + size)...)))
		}
	}
	d.Body().Append(grid, ui.Div(ui.Class("fill"), img("own-gradients.svg")))
	pages.Run(d, 860, 640)
}
