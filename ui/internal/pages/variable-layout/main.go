// A variable font's layout at its instances against Chrome's: kerning
// and a mark's anchor that GDEF's deltas move with the weight and width,
// and GSUB and GPOS features swapped by axis ranges (rvrn draws $ as S
// or s, liga z as Z, a wide kern of A V), from 100 to 900 and 50% to
// 150%.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/internal/varfont/varfonttest"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
@font-face { font-family: Varied; src: url(varied.ttf); font-weight: 100 900; font-stretch: 50% 150%; }
html { background-color: #fff; }
body { padding: 8px; font: 22px Varied; color: #000; }
div { white-space: nowrap; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"faces.css":  {Data: []byte(sheet)},
		"varied.ttf": {Data: varfonttest.Build(varfonttest.Options{Layout: true})},
	}, "faces.css"))
	for _, s := range []string{"font-weight: 100", "font-weight: 400", "font-weight: 650", "font-weight: 900",
		"font-stretch: 150%", "font-weight: 900; font-stretch: 150%", "font-weight: 300; font-stretch: 70%",
		`font-variation-settings: "wght" 250, "wdth" 140`} {
		d.Body().Append(ui.Div(ui.Style(s), ui.Text("AVAV To LY x^x^ $z AVOW")))
	}
	pages.Run(d, 480, 280)
}
