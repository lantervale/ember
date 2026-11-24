//go:build chrome

package ui_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// scripted lays out spans in the fonts of internal/shape/testdata, each
// with its declarations, against Chrome: Noto Sans Thai, Lao, Hebrew and
// KR subsets (Thai, Lao, Hebrew, Hangul), and Go's stand-ins for old
// Thai, Hebrew and Arabic faces with no layout tables (OldThai,
// OldHebrew, OldArabic).
func scripted(t *testing.T, spans ...[2]string) {
	t.Helper()
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Thai; src: url(thai.ttf); }
@font-face { font-family: Lao; src: url(lao.ttf); }
@font-face { font-family: Hebrew; src: url(hebrew.ttf); }
@font-face { font-family: Hangul; src: url(hangul.ttf); }
@font-face { font-family: OldThai; src: url(old-thai.ttf); }
@font-face { font-family: OldHebrew; src: url(old-hebrew.ttf); }
@font-face { font-family: OldArabic; src: url(old-arabic.ttf); }
div { margin: 0 0 4px; }
span { display: inline-block; font-size: 24px; }
`))
	for _, s := range spans {
		d.Body().Append(ui.Div(ui.Span(ui.Style("font-family: "+s[0]), ui.Text(s[1]))))
	}
	size := geom.V2(700, 900)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestThaiAndLaoAreShapedAsInChrome(t *testing.T) {
	scripted(t,
		[2]string{"Thai", "ภาษาไทย น้ำ ทำ คำ"},
		[2]string{"Thai", "ปั่น ญู ฐุ ที่ ก้ำ ฝั่ง"},
		[2]string{"Lao", "ພາສາລາວ ນ້ຳ ຄຳ"},
		[2]string{"OldThai", "ปั่น ญู ฐุ ฎุ ที่ ก้ำ ป่ ปิ่"},
	)
}

func TestHangulIsComposedAsInChrome(t *testing.T) {
	scripted(t,
		[2]string{"Hangul", "한국어"},
		[2]string{"Hangul", "\u1112\u1161\u11AB\u1100\u1173\u11AF\u1100\u1161\u11A8"},
		[2]string{"Hangul", "\u1100\u119E"},
		[2]string{"Hangul", "\u1100\u119E\u11AB"},
		[2]string{"Hangul", "\uA960\u1161\u1100\uD7B0"},
		[2]string{"Hangul", "\uAC00\u11A8\uAC00\u11FF"},
		[2]string{"Hangul", "\u1100\u1161\u302E \uAC00\u302F \u1100\u119E\u302E"},
	)
}

func TestHebrewPointsAreShapedAsInChrome(t *testing.T) {
	scripted(t,
		[2]string{"Hebrew", "בְּרֵאשִׁית בָּרָא"},
		[2]string{"Hebrew", "שָׁלוֹם תּוֹרָה"},
		[2]string{"OldHebrew", "שׁ שּׁ בּ יִ אַ וֹ ײַ כֿ"},
		[2]string{"OldHebrew", "שָׁלוֹם תּוֹרָה"},
		[2]string{"Hebrew", "בְַֽ"},
	)
}

// TestArabicInAFaceOfPresentationFormsJoinsAsInChrome lays out Arabic in
// a face whose only joined forms are the presentation forms of its cmap
// against Chrome, whose HarfBuzz joins the letters with them, forms the
// ligatures it knows and places the marks by their combining classes.
func TestArabicInAFaceOfPresentationFormsJoinsAsInChrome(t *testing.T) {
	scripted(t,
		[2]string{"OldArabic", "بتب سلام يمين"},
		[2]string{"OldArabic", "لا بلا لأ لأب"},
		[2]string{"OldArabic", "بن نبن هه ههه"},
		[2]string{"OldArabic", "بم لمح لمحب ـبـ"},
		[2]string{"OldArabic", "بَّت بَتِ بْ بَّ لَا"},
		[2]string{"OldArabic", "أب دب ء ب"},
	)
}

func TestAVariationSequenceTakesItsGlyphAsInChrome(t *testing.T) {
	scripted(t,
		[2]string{"OldThai", "a︀b︁a︁"},
		[2]string{"OldThai", "aaa a︀a"},
	)
}

// TestAnArabicWordCutByElementsJoinsAsInChrome lays out Arabic words
// whose letters stand in elements of their own, of another size or
// colour, against Chrome, which shapes each with its paragraph's text
// round it: the letters keep their joined forms.
func TestAnArabicWordCutByElementsJoinsAsInChrome(t *testing.T) {
	d := ui.NewDocument(ui.FetchFonts(foundry(t)), ui.CSS(`
@font-face { font-family: Joined; src: url(arabic.ttf); }
div { margin: 0 0 4px; font: 24px Joined, monospace; }
.big { font-size: 30px; }
.red { color: #c00; }
`))
	for _, line := range [][]string{
		{"ب", "big:ت", "ب"},
		{"بب", "big:ت"},
		{"big:ب", "تب"},
		{"بس", "red:ت", "م"},
		{"ب", "big:ت", " ب"},
		{"ممم ", "big:ب", " ممم"},
		{"ب", "big:تَ", "ب"},
	} {
		var parts []ui.Arg
		for _, part := range line {
			if class, text, ok := strings.Cut(part, ":"); ok {
				parts = append(parts, ui.Span(ui.Class(class), ui.Text(text)))
			} else {
				parts = append(parts, ui.Text(part))
			}
		}
		d.Body().Append(ui.Div(parts...))
	}
	size := geom.V2(500, 400)
	landed(t, d, size)
	r := chrome.Match(t, d, size)
	assert.Empty(t, d.Warnings())
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
