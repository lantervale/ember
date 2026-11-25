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

// TestHanSpacingByAFacesChwsAsChrome compares text-spacing-trim in
// go-han-chws.ttf, the stand-in face with a chws of its own that spaces
// some pairs otherwise than Chrome's rules: Chrome hands it every pair,
// across elements, and keeps its rules for a line's start and end.
func TestHanSpacingByAFacesChwsAsChrome(t *testing.T) {
	face, err := os.ReadFile("../../internal/shape/testdata/go-han-chws.ttf")
	if err != nil {
		t.Fatal(err)
	}
	const sheet = `@font-face { font-family: Han; src: url(han.ttf); }
body { font-family: Han; font-size: 20px; margin: 8px; }
p { margin: 0 0 10px; width: 300px; }
.w { width: 100px; }`
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}, "han.ttf": {Data: face}},
		"page.css"))
	d.Body().Append(
		hanP("", ui.Text("漢」「漢。「漢、」漢」」漢「「漢・「漢」・漢")),
		hanP("text-spacing-trim: space-all", ui.Text("漢」「漢。「漢、」漢」」漢")),
		hanP("", ui.Text("」「漢「「漢」」")),
		hanP("", ui.Text("漢」"), hanSpan("", "「"), ui.Text("漢。"), hanSpan("", "「"), ui.Text("漢「"),
			hanSpan("", "「"), ui.Text("漢")),
		hanP("", ui.Text("漢"), hanSpan("", "」"), ui.Text("「漢"), hanSpan("text-spacing-trim: space-all", "」"),
			ui.Text("「漢")),
		hanP("", ui.Class("w"), ui.Text("漢字漢字」「漢字字字字」「字字字字（字」字")),
		hanP("text-spacing-trim: trim-start", ui.Class("w"), ui.Text("「字」漢字」「漢字字字」「字")),
		hanP("width: 95px", ui.Text("字字字字」字")),
		hanP("width: 100px", ui.Text("字字字字」「字")), hanP("width: 95px", ui.Text("字字字字」「字")),
		hanP("width: 100px", ui.Text("字字字字。」字")), hanP("width: 85px", ui.Text("字字字字」」字")),
	)
	fonts.Of(d).Await()
	matchHan(t, d)
}

// TestTrimStartAtABlocksStartAsChrome compares text-spacing-trim:
// trim-start where a block begins and after a forced break, in each
// Hiragino face, at the start of a text node and of an element: Chrome
// sets the opening bracket half wide in each, and full under normal and
// space-first.
func TestTrimStartAtABlocksStartAsChrome(t *testing.T) {
	const start = "text-spacing-trim: trim-start"
	matchHan(t, hanPage(t, `p { width: 100px; }
.mincho { font-family: "Hiragino Mincho ProN"; }`,
		hanP(start, ui.Text("「字」字字字字（字）")),
		hanP(start, hanSpan("", "「字」"), ui.Text("字字字字（字）")),
		hanP(start, ui.Class("gb"), ui.Text("「字」字字字字（字）")),
		hanP(start, ui.Class("gb"), ui.Text("“字”字字字字“字”")),
		hanP(start, ui.Class("mincho"), ui.Text("『字』字字字字【字】")),
		hanP(start, ui.Text(" 〔字〕字字")),
		hanP(start, ui.Text("字字"), ui.Br(), ui.Text("「字」字字字字（字）")),
		hanP(start, ui.Text("字字"), ui.Br(), hanSpan("", "「字」"), ui.Text("字字字字")),
		hanP(start, ui.Class("gb"), ui.Text("字字"), ui.Br(), ui.Text("「字」字字")),
		hanP("", ui.Text("「字」字字"), ui.Br(), ui.Text("「字」")),
		hanP("text-spacing-trim: space-first", ui.Text("「字」字字"), ui.Br(), ui.Text("「字」")),
	))
}

// TestHanSpacingInVerticalLinesAsChrome compares both properties in
// vertical lines with Chrome's, each character in a span of its own:
// the punctuation standing upright trimmed by the face's vhal by the
// rules of horizontal lines, the room text-autospace adds after an
// ideograph or a letter turned with the line, and none between upright
// letters.
func TestHanSpacingInVerticalLinesAsChrome(t *testing.T) {
	chars := func(s string) []ui.Arg {
		var out []ui.Arg
		for _, r := range s {
			out = append(out, hanSpan("", string(r)))
		}
		return out
	}
	d := hanPage(t, `p { writing-mode: vertical-rl; width: auto; height: 300px; display: inline-block; }
p { margin: 0 10px 0 0; }
.auto { text-autospace: normal; }`,
		hanP("", chars("漢」「漢。「漢、」漢」」漢「「漢")...),
		hanP("", chars("漢・「漢」・漢：「漢“漢”「漢")...),
		hanP("text-spacing-trim: space-all", chars("漢」「漢。「漢")...),
		hanP("", append([]ui.Arg{ui.Class("gb")}, chars("漢」「漢。「漢、」漢，「漢“漢”「漢")...)...),
		hanP("", ui.Class("auto"), hanSpan("", "漢A漢"), ui.Text(" "), hanSpan("", "漢1漢"), ui.Text(" "),
			hanSpan("", "かAB字"), ui.Text("字Kana字")),
		hanP("text-orientation: upright", append([]ui.Arg{ui.Class("auto")}, chars("漢A漢 」「漢")...)...),
		hanP("height: 100px", ui.Text("漢字漢字」"), hanSpan("", "「"), ui.Text("漢字字字字」「字")),
		hanP("height: 100px; text-spacing-trim: trim-start", hanSpan("", "「"), ui.Text("字」漢字字")),
	)
	r := chrome.Match(t, d, geom.V2(800, 1000))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
	matchSpanFragments(t, d, geom.V2(800, 1000))
}

// TestHanSpacingInSidewaysLinesAsChrome compares both properties in
// sideways lines (text-orientation: sideways, writing-mode: sideways-*)
// with Chrome's, each character in a span of its own: every character
// turned with the line and spaced as in a horizontal line.
func TestHanSpacingInSidewaysLinesAsChrome(t *testing.T) {
	chars := func(s string) []ui.Arg {
		var out []ui.Arg
		for _, r := range s {
			out = append(out, hanSpan("", string(r)))
		}
		return out
	}
	d := hanPage(t, `p { writing-mode: vertical-rl; text-orientation: sideways; width: auto; height: 300px;
  display: inline-block; margin: 0 10px 0 0; }
.rl { writing-mode: sideways-rl; text-orientation: mixed; }
.lr { writing-mode: sideways-lr; text-orientation: mixed; }
.auto { text-autospace: normal; }`,
		hanP("", chars("漢」「漢。「漢、」漢」」漢「「漢")...),
		hanP("", chars("漢・「漢」・漢：「漢“漢”「漢")...),
		hanP("text-spacing-trim: space-all", chars("漢」「漢。「漢")...),
		hanP("", append([]ui.Arg{ui.Class("gb")}, chars("漢」「漢。「漢、」漢，「漢“漢”「漢")...)...),
		hanP("", append([]ui.Arg{ui.Class("rl")}, chars("漢」「漢。「漢、」漢」」漢")...)...),
		hanP("", append([]ui.Arg{ui.Class("lr")}, chars("漢」「漢。「漢、」漢」」漢")...)...),
		hanP("", ui.Class("auto"), hanSpan("", "漢A漢"), ui.Text(" "), hanSpan("", "漢1漢"), ui.Text(" "),
			hanSpan("", "かAB字"), ui.Text("字Kana字")),
		hanP("", ui.Class("auto", "rl"), hanSpan("", "漢A漢"), ui.Text(" "), hanSpan("", "かAB字")),
		hanP("", ui.Class("auto", "lr"), hanSpan("", "漢A漢"), ui.Text(" "), hanSpan("", "かAB字")),
		hanP("height: 100px", ui.Text("漢字漢字」"), hanSpan("", "「"), ui.Text("漢字字字字」「字")),
		hanP("height: 100px; text-spacing-trim: trim-start", hanSpan("", "「"), ui.Text("字」漢字字")),
		hanP("height: 100px", ui.Class("lr"), ui.Text("漢字漢字」"), hanSpan("", "「"), ui.Text("漢字字字字」「字")),
	)
	r := chrome.Match(t, d, geom.V2(800, 1000))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
	matchSpanFragments(t, d, geom.V2(800, 1000))
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
