// OpenType features against Chrome's: a face with ligatures, small
// capitals, figures, fractions and class kerning (go-features.ttf, a
// copy of internal/shape/testdata's) shaped as each line asks, beside
// Go's small capitals, made up for a face without them.
package main

import (
	_ "embed"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

//go:embed go-features.ttf
var featured []byte

const sheet = `
@font-face { font-family: Featured; src: url(featured.ttf); }
html { background-color: #fff; }
body { padding: 8px; font: 22px Featured; color: #000; }
div { white-space: nowrap; margin-bottom: 2px; }
.small { font-size: 14px; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"page.css":     {Data: []byte(sheet)},
		"featured.ttf": {Data: featured},
	}, "page.css"))
	for _, line := range [][2]string{
		{"", "office affinity, AVATAR LTd To"},
		{"font-variant-ligatures: none; font-kerning: none", "office affinity, AVATAR LTd To"},
		{"font-variant-ligatures: discretionary-ligatures", "act fact, the office"},
		{"font-variant-caps: small-caps", "Small Capitals, 1984"},
		{"font-variant-caps: all-small-caps", "All Small Capitals"},
		{"font-variant-caps: small-caps; font-family: Go", "Made up in Go"},
		{"font-variant-numeric: oldstyle-nums", "0123456789"},
		{"font-variant-numeric: tabular-nums slashed-zero", "1010 2020"},
		{"font-variant-numeric: diagonal-fractions", "1/2 3/4 7/8"},
		{`font-feature-settings: "smcp", "tnum"; font-size: 14px`, "settings: office 1024"},
	} {
		d.Body().Append(ui.Div(ui.Style(line[0]), ui.Text(line[1])))
	}
	pages.Run(d, 440, 360)
}
