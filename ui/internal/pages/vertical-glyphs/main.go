// Glyphs standing in vertical lines against Chrome: the symbols UAX #50 stands (©, §,
// ±, ×, ÷, ¼) among turned letters, a face's vertical advances and top side bearings
// (vhea, vmtx), its vertical alternates (vert: × as +, § as ¶), VORG's origins, inline
// elements of several font sizes on the central baseline, and inline-blocks in lines
// taller than them in vertical-rl and vertical-lr.
package main

import (
	"testing/fstest"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
@font-face { font-family: Vertical; src: url(vertical.ttf); }
@font-face { font-family: Origins; src: url(origins.ttf); }
html { background-color: #fff; }
body { padding: 8px; color: #2b2722; font-size: 16px; display: flex; gap: 10px; align-items: flex-start; }
p { margin: 0; }
.box { height: 200px; border: 1px solid #c8473d; padding: 4px; writing-mode: vertical-rl; }
.lr { writing-mode: vertical-lr; line-height: 34px; }
.vertical { font-family: Vertical; font-size: 20px; }
.origins { font-family: Origins; font-size: 20px; }
.big { font-size: 26px; color: #3a6fd8; }
.small { font-size: 11px; }
.ib { display: inline-block; inline-size: 24px; block-size: 10px; background-color: #3a6fd8; }
.top { vertical-align: top; background-color: #c8473d; }
.pad { background-color: #f0c040; }
`

func main() {
	vertical, err := pages.VerticalFont(goregular.TTF, map[rune]rune{'×': '+', '§': '¶'}, 0)
	if err != nil {
		panic(err)
	}
	origins, err := pages.VerticalFont(goregular.TTF, map[rune]rune{'×': '+', '§': '¶'}, 1500)
	if err != nil {
		panic(err)
	}
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"glyphs.css":   {Data: []byte(sheet)},
		"vertical.ttf": {Data: vertical},
		"origins.ttf":  {Data: origins},
	}, "glyphs.css"))
	d.Body().Append(
		ui.Div(ui.Class("box"), ui.P(ui.Text("Signs © § ± × ÷ ¼ stand up."))),
		ui.Div(ui.Class("box", "vertical"), ui.P(ui.Span(ui.Class("pad"), ui.Text("©§±×÷¼")), ui.Text(" Abc ©"))),
		ui.Div(ui.Class("box", "origins"), ui.P(ui.Span(ui.Class("pad"), ui.Text("©§±×÷¼")), ui.Text(" Abc ©"))),
		ui.Div(ui.Class("box"), ui.P(ui.Text("Of "), ui.Span(ui.Class("big"), ui.Text("big ©")),
			ui.Text(" and "), ui.Span(ui.Class("small", "pad"), ui.Text("small §")), ui.Text(" sizes"))),
		ui.Div(ui.Class("box", "lr"), ui.P(ui.Text("Left "), ui.Span(ui.Class("ib")), ui.Text(" to "),
			ui.Span(ui.Class("ib", "top")), ui.Text(" right "), ui.Span(ui.Class("ib")), ui.Text(" blocks"))),
		ui.Div(ui.Class("box"), ui.P(ui.Text("Right "), ui.Span(ui.Class("ib")), ui.Text(" to "),
			ui.Span(ui.Class("ib", "top")), ui.Text(" left"))),
	)
	pages.Run(d, 420, 240)
}
