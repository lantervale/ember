//go:build chrome

package layout_test

import (
	"os"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/fonts"
)

// hanPage is a Document in Japanese drawn in the machine's Hiragino
// faces, sheet after the faces and the size.
func hanPage(t *testing.T, sheet string, nodes ...*ui.Node) *ui.Document {
	t.Helper()
	for _, f := range []string{"/System/Library/Fonts/ヒラギノ角ゴシック W3.ttc", "/System/Library/Fonts/Hiragino Sans GB.ttc"} {
		if _, err := os.Stat(f); err != nil {
			t.Skip("Hiragino is not installed")
		}
	}
	sheet = `body { font-family: "Hiragino Sans"; font-size: 20px; margin: 8px; }
p { margin: 0 0 10px; width: 300px; }
.gb { font-family: "Hiragino Sans GB"; }
` + sheet
	d := ui.NewDocument(ui.SystemFonts(), ui.Language("ja"),
		ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(nodes...)
	fonts.Of(d).Await()
	return d
}

func hanP(style string, args ...ui.Arg) *ui.Node {
	return ui.P(append([]ui.Arg{ui.Style(style)}, args...)...)
}

func hanSpan(style, text string) *ui.Node { return ui.Span(ui.Style(style), ui.Text(text)) }

// TestHanPunctuationTrimsAsChrome compares text-spacing-trim with
// Chrome's: full-width punctuation side by side set half its width by
// the face's halt, at the start of lines as each value has it, and a
// closing bracket a line ends at trimmed when that makes it fit.
func TestHanPunctuationTrimsAsChrome(t *testing.T) {
	const pairs = "漢」「漢。「字、」字」」字「「字（「字：「字・「字」・字」：字」，字“漢”字”“字‘’字」！字　「字"
	cases := map[string]func(t *testing.T) *ui.Document{
		"pairs": func(t *testing.T) *ui.Document {
			return hanPage(t, "", hanP("", ui.Text(pairs)), hanP("text-spacing-trim: space-all", ui.Text(pairs)),
				hanP("", ui.Class("gb"), ui.Text(pairs)), hanP("", ui.Text("〚「〛「」〚〛〛字｢「」｣字〝「〞〟字")))
		},
		"pairs across elements": func(t *testing.T) *ui.Document {
			return hanPage(t, "",
				hanP("", ui.Text("漢」"), hanSpan("", "」"), ui.Text("漢」"), hanSpan("font-weight: bold", "。"), ui.Text("漢」"),
					hanSpan("", "「"), ui.Text("漢")),
				hanP("", ui.Text("漢」"), hanSpan("text-spacing-trim: space-all", "「"), ui.Text("漢"),
					hanSpan("text-spacing-trim: space-all", "」"), ui.Text("「漢"), hanSpan("text-spacing-trim: space-all", "」"),
					ui.Text("」漢")),
				hanP("text-spacing-trim: space-all", ui.Text("漢"), hanSpan("text-spacing-trim: normal", "」"), ui.Text("」漢」"),
					hanSpan("text-spacing-trim: normal", "」"), ui.Text("漢")),
				hanP("", ui.Text("漢」"), hanSpan("padding: 0 4px; border: 1px solid", "「字」"), ui.Text("「漢")),
			)
		},
		"line starts": func(t *testing.T) *ui.Document {
			const text = "漢字漢字「漢字」漢「字」漢字漢字」「漢字字字字」「字"
			return hanPage(t, "p { width: 100px; }",
				hanP("", ui.Text(text)), hanP("text-spacing-trim: space-first", ui.Text(text)),
				hanP("text-spacing-trim: trim-start", ui.Text(text)), hanP("text-spacing-trim: space-all", ui.Text(text)),
				hanP("text-spacing-trim: trim-start", hanSpan("", "「字」"), ui.Text("字字字字（字）")),
				hanP("text-spacing-trim: trim-start", ui.Class("gb"), ui.Text("“字”字字字字“字”")),
				hanP("text-spacing-trim: space-first", ui.Text("字字"), hanSpan("", "「字」"), ui.Text("「字」字字字「字」")),
				hanP("", ui.Text("」「」「」「」「」「」「")),
			)
		},
		"line ends": func(t *testing.T) *ui.Document {
			return hanPage(t, "p { width: 95px; }",
				hanP("", ui.Text("字字字字」字")), hanP("", ui.Text("字字字字）字")), hanP("", ui.Text("字字字字』字")),
				hanP("", ui.Text("字字字字”字")), hanP("", ui.Class("gb"), ui.Text("字字字字”字")),
				hanP("", ui.Class("gb"), ui.Text("字字字字，字")), hanP("", ui.Text("字字字字。字")),
				hanP("width: 85px", ui.Text("字字字字」字")), hanP("text-spacing-trim: space-all", ui.Text("字字字字」字")),
				hanP("text-spacing-trim: trim-start", ui.Text("字字字字」字")),
				hanP("width: 91px", ui.Text("字字字字」」字")), hanP("text-align: right", ui.Text("字字字字」字")),
				hanP("", ui.Text("字字字字」"), ui.Br(), ui.Text("字")),
			)
		},
		"the content's end": func(t *testing.T) *ui.Document {
			chrome.Since(t, 158, "a closing bracket the content's last line would overflow by is trimmed")
			return hanPage(t, "p { width: 95px; }",
				hanP("", ui.Text("字字字字」")), hanP("", ui.Text("字字字字）")), hanP("", ui.Text("字字字字」 ")),
				hanP("text-align: right", ui.Text("字字字字」")), hanP("width: 100px", ui.Text("字字字字」")),
				hanP("text-spacing-trim: space-all", ui.Text("字字字字」")),
			)
		},
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) { matchHan(t, d(t)) })
	}
}

// TestHanAutospaceAsChrome compares text-autospace: normal with
// Chrome's: an eighth of the font size between an ideograph and a
// letter or a digit, after the first of the two, which a line ending
// between them drops.
func TestHanAutospaceAsChrome(t *testing.T) {
	cases := map[string]func(t *testing.T) *ui.Document{
		"between ideographs and letters": func(t *testing.T) *ui.Document {
			return hanPage(t, "p { text-autospace: normal; }",
				hanP("", ui.Text("漢A A漢 漢1 1漢 かA カA 漢Ａ 漢。A 漢é 漢α 漢% 漢 A")),
				hanP("", ui.Text("ーA ヽA 〆A 〻A ｦA ｰA ㇰA ゛A ゝA 〃A ゠A 〇A 々A")),
				hanP("", ui.Text("AÅ漢 漢ʰ 漢Ⅻ 漢² 漢ａ 漢ｱ ｶA 漢_ 漢- 漢( 漢@ 漢$")),
				hanP("text-autospace: no-autospace", ui.Text("漢A A漢 漢1")),
				hanP("letter-spacing: 3px", ui.Text("漢A漢 漢1漢")),
			)
		},
		"across elements": func(t *testing.T) *ui.Document {
			return hanPage(t, "p { text-autospace: normal; }",
				hanP("", ui.Text("漢"), hanSpan("font-weight: bold", "A"), ui.Text("漢")),
				hanP("", ui.Text("漢"), hanSpan("padding: 0 3px; border: 1px solid", "A"), ui.Text("漢")),
				hanP("", ui.Text("漢"), hanSpan("text-autospace: no-autospace", "A"), ui.Text("漢")),
				hanP("", ui.Text("漢"), hanSpan("text-autospace: no-autospace", "A"), hanSpan("", "漢")),
				hanP("font-size: 16px", ui.Text("漢A"), hanSpan("font-size: 32px", "A漢")),
				hanP("", ui.Text("漢"), ui.Span(), ui.Text("A")),
			)
		},
		"wrapped": func(t *testing.T) *ui.Document {
			return hanPage(t, "p { text-autospace: normal; width: 100px; }",
				hanP("", ui.Text("漢字漢字AB漢字")), hanP("", ui.Text("漢字漢字漢AB字")),
				hanP("text-align: right", ui.Text("漢字漢字漢AB字")), hanP("text-align: center", ui.Text("字字字AB漢字字")),
				hanP("", ui.Text("漢字漢字漢字漢字漢A漢字字字字字字字字字字字字字字A字字字")),
				hanP("width: 41px", ui.Text("漢漢A")), hanP("width: 41px", ui.Text("AB漢")),
				hanP("width: 42px", ui.Text("漢AA漢")),
				hanP("", ui.Text("字字字字"), hanSpan("", "字"), ui.Text("A漢")),
			)
		},
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) { matchHan(t, d(t)) })
	}
}

// TestHanSpacingInAStandInFaceAsChrome compares both properties in
// go-han.ttf, a face of stand-ins with halt that every machine draws
// alike: the brackets trimmed by its feature, the dots and the colon
// classed by where their ink stands.
func TestHanSpacingInAStandInFaceAsChrome(t *testing.T) {
	face, err := os.ReadFile("../../internal/shape/testdata/go-han.ttf")
	if err != nil {
		t.Fatal(err)
	}
	const sheet = `@font-face { font-family: Han; src: url(han.ttf); }
body { font-family: Han; font-size: 20px; margin: 8px; }
p { margin: 0 0 10px; width: 300px; }
.w { width: 100px; }
.auto { text-autospace: normal; }`
	const pairs = "漢」「字。「か、」字」」（「カ：「漢・「字」』漢。。字：：字　「字"
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}, "han.ttf": {Data: face}},
		"page.css"))
	d.Body().Append(
		hanP("", ui.Text(pairs)), hanP("text-spacing-trim: space-all", ui.Text(pairs)),
		hanP("", ui.Class("w"), ui.Text("漢字漢字」「漢字字字字」「字字字字（字」字")),
		hanP("text-spacing-trim: space-first", ui.Class("w"), ui.Text("漢字漢字」「漢字字字字」「字字字字（字」字")),
		hanP("width: 95px", ui.Text("字字字字」字")), hanP("width: 95px", ui.Text("字字字字。字")),
		hanP("", ui.Class("auto"), ui.Text("漢Ember字 1966か Kana カ"), hanSpan("font-size: 30px", "A"),
			ui.Text("字")),
		hanP("", ui.Class("auto", "w"), ui.Text("字字字字AB漢字字字字漢A")),
	)
	fonts.Of(d).Await()
	matchHan(t, d)
}

func matchHan(t *testing.T, d *ui.Document) {
	t.Helper()
	size := geom.V2(800, 1000)
	r := chrome.Match(t, d, size)
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
	matchText(t, d, size)
	matchSpanFragments(t, d, size)
	for _, w := range d.Warnings() {
		t.Error(w)
	}
}
