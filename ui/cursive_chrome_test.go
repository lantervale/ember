//go:build chrome

package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestLetterSpacingLeavesCursiveScriptsUnspacedAsInChrome lays out text
// with letter-spacing against Chrome, which spaces no letter of a script
// that joins them (Arabic in a face with its joined forms and in one of
// presentation forms), nor the digits, punctuation and marks of its
// runs, but spaces their spaces and no-break spaces, Hebrew's letters and
// Latin's, once a grapheme cluster.
func TestLetterSpacingLeavesCursiveScriptsUnspacedAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Joined; src: url(arabic.ttf); }
@font-face { font-family: OldArabic; src: url(old-arabic.ttf); }
@font-face { font-family: Hebrew; src: url(hebrew.ttf); }
div { margin: 0 0 4px; font: 24px Joined; letter-spacing: 6px; }
span { display: inline-block; }
.old { font-family: OldArabic; }
.hebrew { font-family: Hebrew; }
.em { letter-spacing: 0.3em; }
p { width: 260px; margin: 0 0 4px; font: 20px Joined; letter-spacing: 4px; }
`))
	for _, line := range [][2]string{
		{"", "بتب بمم لا"},
		{"", "ب.ب ب1ب (ب)"},
		{"", "ب ب ب  ب"},
		{"", "بَتِب بّم"},
		{"", "ab بتب cd"},
		{"", "aَb áb"},
		{"", "a.ب ب.a 1ب"},
		{"old", "بتب سلام لمحب"},
		{"old", "بَّت لا بن"},
		{"hebrew", "שלום עולם"},
		{"em", "بتب abc"},
	} {
		d.Body().Append(ui.Div(ui.Class(line[0]), ui.Span(ui.Text(line[1]))))
	}
	d.Body().Append(ui.P(ui.Text("السلام بين الناس سليم. يسير لمن يريد and some Latin words after it.")))
	d.Body().Append(ui.P(ui.Attr("dir", "rtl"), ui.Text("بسم الله abc السلام بين يدي 123 بتب بمم. ودين")))
	size := geom.V2(500, 700)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
