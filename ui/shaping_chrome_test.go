//go:build chrome

package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// joined lays out spans of arabic.ttf, Go with stand-ins for Arabic
// letters and marks, their joined forms, lam-alef, marks on bases and
// cursive attachment, each with its declarations, against Chrome.
func joined(t *testing.T, sheet string, spans ...[2]string) {
	t.Helper()
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Joined; src: url(arabic.ttf); }
div { margin: 0 0 4px; }
span { display: inline-block; font: 24px Joined, monospace; }
`+sheet))
	for _, s := range spans {
		d.Body().Append(ui.Div(ui.Span(ui.Style(s[0]), ui.Text(s[1]))))
	}
	size := geom.V2(700, 900)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestArabicLettersJoinAsInChrome(t *testing.T) {
	joined(t, "",
		[2]string{"", "بتب"},
		[2]string{"", "بسم سلام"},
		[2]string{"", "دار اب"},
		[2]string{"", "ه ه هه"},
		[2]string{"", "ـبـ ـلا"},
		[2]string{"", "مممم بمب ببم"},
		[2]string{"", "لا يلا بلا"},
		[2]string{"", "سسس هههه سهس"},
		[2]string{"direction: rtl", "بتب دار"},
		[2]string{`font-feature-settings: "init" 0, "medi" 0`, "بتب"},
		[2]string{`font-feature-settings: "rlig" 0`, "يلا"},
		[2]string{"font-variant-ligatures: no-contextual", "بم ببم"},
		[2]string{`font-feature-settings: "curs" 0`, "سسس"},
	)
}

func TestMarksSitOnTheirBasesAsInChrome(t *testing.T) {
	joined(t, "",
		[2]string{"", "بَ بِ بَّ سِّ"},
		[2]string{"", "لَا لاِ بلَا لِاَ"},
		[2]string{"", "aé é́"},
		[2]string{`font-feature-settings: "mark" 0, "mkmk" 0`, "بَّ"},
	)
}

func TestSuperiorsAndInferiorsComeFromTheFaceAsInChrome(t *testing.T) {
	joined(t, ".go { font-family: Go, monospace; }",
		[2]string{"font-variant-position: super", "x2 23"},
		[2]string{"font-variant-position: sub", "H2O 10"},
		[2]string{"font-variant: sub oldstyle-nums", "CO2"},
		[2]string{"font-variant-position: super; font-family: Go, monospace", "x2 left as it is"},
		[2]string{`font-variant-position: super; font-feature-settings: "sups" 0`, "x2"},
	)
}

func TestTextRightToLeftMixesArabicAndLatinAsInChrome(t *testing.T) {
	joined(t, "div { direction: rtl; } span { display: inline; }",
		[2]string{"", "بسم ab سلام"},
		[2]string{"", "سلام (بتب) 12"},
		[2]string{"word-spacing: 6px", "بسم سلام بتب"},
	)
}

// syllabic lays out spans of devanagari.ttf, a subset of Noto Sans
// Devanagari, each with its declarations, against Chrome.
func syllabic(t *testing.T, sheet string, spans ...[2]string) {
	t.Helper()
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Deva; src: url(devanagari.ttf); }
div { margin: 0 0 4px; }
span { display: inline-block; font: 24px Deva, monospace; }
`+sheet))
	for _, s := range spans {
		d.Body().Append(ui.Div(ui.Span(ui.Style(s[0]), ui.Text(s[1]))))
	}
	size := geom.V2(700, 900)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestDevanagariSyllablesAreReorderedAndFormedAsInChrome(t *testing.T) {
	syllabic(t, "",
		[2]string{"", "नमस्ते दुनिया"},
		[2]string{"", "हिन्दी कि र्कि"},
		[2]string{"", "क्षत्रिय प्रकाश श्री स्त्री"},
		[2]string{"", "धर्म द्ध ज्ञान पृथ्वी"},
		[2]string{"", "क्\u200dष क्\u200cष"},
		[2]string{"", "क़ ड़ ॐ १२३"},
		[2]string{"", "Hello नमस्ते, world"},
		[2]string{`font-feature-settings: "half" 0`, "क्षत्रिय स्त्री"},
		[2]string{`font-feature-settings: "rphf" 0`, "धर्म र्कि"},
	)
}

func TestVowelSignsWithoutABaseTakeADottedCircleAsInChrome(t *testing.T) {
	syllabic(t, "",
		[2]string{"", "ि ा"},
		[2]string{"", "अा कि"},
	)
}
