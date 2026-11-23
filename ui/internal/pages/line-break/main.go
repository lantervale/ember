// line-break against Chrome's, in a stand-in for a face of Chinese and
// Japanese (a copy of internal/shape/testdata's go-kana.ttf): small kana,
// prolonged sound marks, iteration marks, the wave dash, the CJK hyphen,
// ellipses, centred punctuation, postfixes and prefixes at the start and
// end of lines under each value, in Japanese and in English, and anywhere
// breaking between any two letters, across elements, decorated.
package main

import (
	_ "embed"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

//go:embed go-kana.ttf
var kana []byte

const sheet = `
@font-face { font-family: Kana; src: url(kana.ttf); }
html { background-color: #fff; }
body { padding: 8px; font-family: Kana; font-size: 16px; color: #2b2722; }
.row { display: flex; gap: 10px; margin-bottom: 8px; }
p { margin: 0; width: 96px; background: #f1ece4; }
.wide { width: 150px; }
.red { color: #c8473d; }
.box { border: 1px solid #3a6fd8; padding: 0 2px; }
.under { text-decoration: underline #3a6fd8; }
`

func main() {
	d := ui.NewDocument(ui.Language("ja"), ui.Stylesheet(fstest.MapFS{
		"page.css": {Data: []byte(sheet)},
		"kana.ttf": {Data: kana},
	}, "page.css"))
	const text = "漢字漢字漢ぁ字ーカァ々字ゝ漢字漢〜字‐漢字……漢字！字・漢字1％字A＄1漢字漢字。"
	row := func(lang string) *ui.Node {
		r := ui.Div(ui.Class("row"), ui.Attr("lang", lang))
		for _, v := range []string{"auto", "loose", "normal", "strict"} {
			r.Append(ui.P(ui.Style("line-break: "+v), ui.Text(text)))
		}
		return r
	}
	d.Body().Append(
		row("ja"),
		row("en"),
		ui.Div(ui.Class("row"),
			ui.P(ui.Class("wide"), ui.Style("line-break: anywhere"), ui.Text("Breaks anywhere, (even) here: 3.14!")),
			ui.P(ui.Class("wide"), ui.Text("plain"), ui.Span(ui.Class("red"), ui.Style("line-break: anywhere"),
				ui.Text("anywhereanywhere")), ui.Text("after")),
			ui.P(ui.Class("wide", "under"), ui.Style("line-break: anywhere"), ui.Text("abc"),
				ui.Span(ui.Class("box"), ui.Text("defghij")), ui.Text("klmnop漢字")),
		),
		ui.Div(ui.Class("row"),
			ui.P(ui.Style("line-break: strict"), ui.Text("漢字漢字漢"), ui.Span(ui.Class("red"),
				ui.Style("line-break: loose"), ui.Text("ぁ字々漢字"))),
			ui.P(ui.Style("line-break: loose"), ui.Text("漢字漢字漢"), ui.Span(ui.Class("red"),
				ui.Style("line-break: strict"), ui.Text("ぁ字々漢字"))),
			ui.P(ui.Style("line-break: strict; word-break: break-all"), ui.Text("漢字漢字漢ぁ字ABCDEFGHー")),
			ui.Div(ui.Style("display: inline-block; width: min-content; line-break: anywhere; background: #f1ece4"),
				ui.Text("min content")),
		),
	)
	pages.Run(d, 480, 420)
}
