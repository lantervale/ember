//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// breakChars are the characters line-break's values tell apart: small
// kana and prolonged sound marks, iteration marks, CJK hyphens,
// inseparable characters, centred punctuation, postfixes and prefixes.
const breakChars = "ぁァゃッㇰｧーｰ々〻ゝゞヽヾ‐–〜゠…‥・：；‼⁇！？％°℃‰′″￠＄¥￥€£"

// strictnessPage is a paragraph per character of breakChars and context,
// each 100px wide so that the character comes at the end of its first
// line, before and after an ideograph, a letter and a digit and beside
// itself, in language lang under line-break: value.
func strictnessPage(t *testing.T, lang, value string) *ui.Document {
	ps := []ui.Arg{ui.Class("all"), ui.Attr("lang", lang), ui.Style("line-break: " + value)}
	for _, c := range breakChars {
		for _, x := range []string{"漢", "A", "1"} {
			ps = append(ps, hanP("", ui.Text("字字字字"+x+string(c)+"字")), hanP("", ui.Text("字字字字"+string(c)+x+"字")))
		}
		ps = append(ps, hanP("", ui.Text("字字字字"+string(c)+string(c)+"字")))
	}
	return hanPage(t, `.all { display: flex; flex-wrap: wrap; column-gap: 8px; }
.all p { width: 100px; }`, ui.Div(ps...))
}

// TestLineBreakStrictnessAsChrome compares line-break's auto, loose,
// normal and strict with Chrome's, each in Japanese, Chinese, Korean
// and English: where a line may start with a small kana, an iteration
// mark, a hyphen, an inseparable character, centred punctuation, a
// postfix and end with a prefix.
func TestLineBreakStrictnessAsChrome(t *testing.T) {
	for _, lang := range []string{"ja", "zh", "ko", "en"} {
		for _, value := range []string{"auto", "loose", "normal", "strict"} {
			t.Run(lang+" "+value, func(t *testing.T) {
				matchBreaks(t, strictnessPage(t, lang, value))
			})
		}
	}
}

// TestLineBreakStrictnessInContextAsChrome compares the rules in running
// text: the Document's language where no element names one, a value
// changing between elements, and mixed with word-break.
func TestLineBreakStrictnessInContextAsChrome(t *testing.T) {
	const text = "ちょっとジャーナリストのキャッシュ々々、ゝゞ〜。……‥‥ですね！？ラーメン・ソーメン１００％で＄５０"
	cases := map[string]func(t *testing.T) *ui.Document{
		"paragraphs": func(t *testing.T) *ui.Document {
			ps := []ui.Arg{ui.Class("all")}
			for _, v := range []string{"auto", "loose", "normal", "strict"} {
				for _, w := range []string{"100px", "130px", "170px"} {
					ps = append(ps, hanP("line-break: "+v+"; width: "+w, ui.Text(text)))
				}
			}
			return hanPage(t, ".all { display: flex; flex-wrap: wrap; column-gap: 8px; font-kerning: none; }",
				ui.Div(ps...))
		},
		"across elements": func(t *testing.T) *ui.Document {
			return hanPage(t, "p { width: 100px; }",
				hanP("line-break: strict", ui.Text("字字字字字"), hanSpan("line-break: loose", "ぁ字")),
				hanP("line-break: loose", ui.Text("字字字字字"), hanSpan("line-break: strict", "ぁ字")),
				hanP("line-break: strict", ui.Text("字字字字字"), hanSpan("line-break: normal", "々字")),
				hanP("line-break: loose", ui.Text("字字字字…"), hanSpan("", "…字")),
				hanP("", ui.Attr("lang", "en"), ui.Text("字字字字字"), hanSpan("line-break: loose", "〜字")),
				hanP("line-break: normal", ui.Attr("lang", "zh"), ui.Text("字字字字字"),
					ui.Span(ui.Attr("lang", "en"), ui.Text("〜字"))),
			)
		},
		"with word-break": func(t *testing.T) *ui.Document {
			return hanPage(t, "p { width: 100px; }",
				hanP("line-break: strict; word-break: break-all", ui.Text("字字字字字ぁ字")),
				hanP("line-break: loose; word-break: keep-all", ui.Text("字字字字字ぁ字 字字字字字々字")),
				hanP("line-break: strict; overflow-wrap: anywhere", ui.Text("ぁぁぁぁぁぁぁぁ")),
			)
		},
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) { matchBreaks(t, d(t)) })
	}
}

// TestLineBreakAnywhereAsChrome compares line-break: anywhere with
// Chrome's: a break between any two typographic letter units, before
// and after punctuation and spaces, across elements, and the
// min-content width it gives.
func TestLineBreakAnywhereAsChrome(t *testing.T) {
	cases := map[string]func(t *testing.T) *ui.Document{
		"letters and punctuation": func(*testing.T) *ui.Document {
			return page(
				ui.P(ui.Style("width: 80px; line-break: anywhere"), ui.Text("Hello, world! (a) b-c and/or 3.14")),
				ui.P(ui.Style("width: 53px; line-break: anywhere"), ui.Text("“Quoted,” he said… (really?) [yes]; ok.")),
				ui.P(ui.Style("width: 61px; line-break: anywhere"), ui.Text("https://example.com/a/b?c=d&e=f")),
				ui.P(ui.Style("width: 70px; line-break: anywhere"), ui.Text("été ﬁnal näive cafés")),
				ui.P(ui.Style("width: 70px; line-break: anywhere; text-align: right"), ui.Text(sentence)),
				ui.P(ui.Style("width: 90px; line-break: anywhere; letter-spacing: 3px"), ui.Text(sentence)),
				ui.P(ui.Style("width: 90px; line-break: anywhere; word-break: keep-all"), ui.Text(sentence)),
				ui.P(ui.Style("width: 90px; line-break: anywhere; white-space: nowrap"), ui.Text("no wrap at all here")),
			)
		},
		"spaces": func(*testing.T) *ui.Document {
			return page(
				ui.P(ui.Style("width: 50px; line-break: anywhere"), ui.Text("aaa bbbbbb c dd")),
				ui.P(ui.Style("width: 50px; line-break: anywhere; white-space: pre-wrap"), ui.Text("aaa  bbbbbb    c dd")),
				ui.P(ui.Style("width: 40px; line-break: anywhere; white-space: pre-wrap"), ui.Text("ab      cd\n  efgh")),
				ui.P(ui.Style("width: 50px; line-break: anywhere; white-space: pre-line"), ui.Text("aaa bbbbbb\nc dd")),
			)
		},
		"across elements": func(*testing.T) *ui.Document {
			return page(
				ui.P(ui.Style("width: 40px"), ui.Text("abcdef"), ui.Span(ui.Style("line-break: anywhere"), ui.Text("ghijkl")),
					ui.Text("mnop")),
				ui.P(ui.Style("width: 40px"), ui.Span(ui.Style("line-break: anywhere"), ui.Text("abcdef")),
					ui.Text("ghijkl")),
				ui.P(ui.Style("width: 40px; line-break: anywhere"), ui.Text("abc"),
					ui.Span(ui.Style("font-weight: bold"), ui.Text("defgh")), ui.Text("ijk")),
				ui.P(ui.Style("width: 40px; line-break: anywhere"), ui.Text("abc"),
					ui.Span(ui.Style("padding: 0 4px; border: 1px solid"), ui.Text("defgh")), ui.Text("ijk")),
				ui.P(ui.Style("width: 40px; line-break: anywhere"), ui.Text("abc"),
					ui.Span(ui.Style("line-break: auto"), ui.Text("defghij")), ui.Text("klm")),
			)
		},
		"min-content": func(*testing.T) *ui.Document {
			cell := func(style, text string) *ui.Node {
				return ui.Div(ui.Style("display: inline-block; width: min-content; border: 1px solid; vertical-align: top; "+
					"margin: 2px; line-break: anywhere; "+style), ui.Text(text))
			}
			return page(
				cell("", "min-content"), cell("", "a b c"), cell("", "(x)"), cell("", "ab, cd!"),
				cell("", "éé"), cell("", "x—y"), cell("letter-spacing: 4px", "wide"),
				cell("white-space: pre-wrap", "a  b"), cell("white-space: nowrap", "no wrap"),
				cell("word-break: keep-all", "keep all"), cell("font-size: 30px", "Wi"),
				ui.Div(ui.Style("display: flex; width: 30px"),
					ui.Div(ui.Style("line-break: anywhere"), ui.Text("flex item"))),
				ui.Div(ui.Style("display: flex; width: 30px"),
					ui.Div(ui.Style("line-break: anywhere"), ui.Text("ab"), ui.Span(ui.Style("font-size: 30px"), ui.Text("W")))),
			)
		},
		"ideographs": func(t *testing.T) *ui.Document {
			return hanPage(t, "p { width: 100px; line-break: anywhere; }",
				hanP("", ui.Text("字字字字字。字")), hanP("", ui.Text("字字字字「字」字")), hanP("", ui.Text("字字字字字ぁー")),
				hanP("", ui.Text("字字字字A字……")),
			)
		},
		"ideographs at the content's end": func(t *testing.T) *ui.Document {
			return hanPage(t, "p { width: 100px; line-break: anywhere; }",
				hanP("width: auto; display: inline-block; width: min-content", ui.Text("漢字。」")))
		},
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) { matchBreaks(t, d(t)) })
	}
}

func matchBreaks(t *testing.T, d *ui.Document) {
	t.Helper()
	size := geom.V2(800, 1000)
	r := chrome.Match(t, d, size)
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
	matchText(t, d, size)
	for _, w := range d.Warnings() {
		t.Error(w)
	}
}
