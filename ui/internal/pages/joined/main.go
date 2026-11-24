// Arabic shaping against Chrome's: a face with stand-ins for Arabic
// letters and marks (go-arabic.ttf, a copy of internal/shape/testdata's)
// joined, ligated, its marks on their bases and on each other, its
// letters attached cursively, right to left beside Latin, and its
// superiors and inferiors; a face of presentation forms alone
// (go-old-arabic.ttf) joined by HarfBuzz's fallback, and letter-spacing
// leaving Arabic unspaced.
package main

import (
	_ "embed"
	"strings"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const rtl = "rtl"

//go:embed go-arabic.ttf
var joined []byte

//go:embed go-old-arabic.ttf
var old []byte

const sheet = `
@font-face { font-family: Joined; src: url(joined.ttf); }
@font-face { font-family: Old; src: url(old.ttf); }
html { background-color: #fff; }
body { padding: 8px; font: 24px Joined; color: #000; }
div { white-space: nowrap; margin-bottom: 6px; }
.rtl { direction: rtl; }
.sup { font-variant-position: super; }
.sub { font-variant-position: sub; }
.old { font-family: Old; }
.spaced { letter-spacing: 5px; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"page.css":   {Data: []byte(sheet)},
		"joined.ttf": {Data: joined},
		"old.ttf":    {Data: old},
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
		{"rtl old", "بتب سلام لمحب لا بلا"},
		{"rtl old", "بَّت بَتِ بن هه ـبـ"},
		{"rtl spaced", "بتب سلام ab cd بمم. 12"},
	} {
		d.Body().Append(ui.Div(ui.Class(strings.Fields(line[0])...), ui.Text(line[1])))
	}
	pages.Run(d, 440, 520)
}
