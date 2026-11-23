// Devanagari shaping against Chrome's, in a subset of Noto Sans
// Devanagari (noto-devanagari.ttf, a copy of internal/shape/testdata's):
// pre-base matras before their consonants, reph above the end of its
// syllable, half and below forms, conjuncts, nukta forms, joiners, and
// a dotted circle under a vowel sign with no base.
package main

import (
	_ "embed"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

//go:embed noto-devanagari.ttf
var devanagari []byte

const sheet = `
@font-face { font-family: Deva; src: url(deva.ttf); }
html { background-color: #fff; }
body { padding: 8px; font: 26px Deva; color: #000; }
div { white-space: nowrap; margin-bottom: 4px; }
.nohalf { font-feature-settings: "half" 0; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"page.css": {Data: []byte(sheet)},
		"deva.ttf": {Data: devanagari},
	}, "page.css"))
	for _, line := range [][2]string{
		{"", "नमस्ते दुनिया हिन्दी"},
		{"", "कि र्कि किं धर्म"},
		{"", "क्षत्रिय प्रकाश श्री"},
		{"", "स्त्री द्ध ज्ञान पृथ्वी"},
		{"", "क्\u200dष क्\u200cष क़ ड़ ॐ"},
		{"", "ि अा १२३"},
		{"", "नमस्ते। उद्ध्व द्य ह्म"},
		{"nohalf", "क्षत्रिय स्त्री"},
	} {
		d.Body().Append(ui.Div(ui.Class(line[0]), ui.Text(line[1])))
	}
	pages.Run(d, 420, 400)
}
