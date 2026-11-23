// Arabic shaping against Chrome's: a face with stand-ins for Arabic
// letters and marks (go-arabic.ttf, a copy of internal/shape/testdata's)
// joined, ligated, its marks on their bases and on each other, its
// letters attached cursively, right to left beside Latin, and its
// superiors and inferiors.
package main

import (
	_ "embed"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const rtl = "rtl"

//go:embed go-arabic.ttf
var joined []byte

const sheet = `
@font-face { font-family: Joined; src: url(joined.ttf); }
html { background-color: #fff; }
body { padding: 8px; font: 24px Joined; color: #000; }
div { white-space: nowrap; margin-bottom: 6px; }
.rtl { direction: rtl; }
.sup { font-variant-position: super; }
.sub { font-variant-position: sub; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"page.css":   {Data: []byte(sheet)},
		"joined.ttf": {Data: joined},
	}, "page.css"))
	for _, line := range [][2]string{
		{rtl, "بتب بسم سلام دار"},
		{rtl, "ـبـ مممم بمب لا يلا"},
		{rtl, "سسس هههه سهس"},
		{rtl, "بَ بِ بَّ سِّ لَا لاِ"},
		{rtl, "بسم ab سلام (بتب) 12"},
		{"", "aé é́ office"},
		{"", "بتب مممم"},
		{"sup", "x2 E=mc2 10"},
		{"sub", "H2O CO2"},
	} {
		d.Body().Append(ui.Div(ui.Class(line[0]), ui.Text(line[1])))
	}
	pages.Run(d, 440, 400)
}
