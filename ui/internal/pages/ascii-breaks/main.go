// Line breaks between two ASCII characters against Chrome's, which takes
// them from a table of its own: URLs, code, punctuation and a minus sign
// in narrow boxes, under word-break: normal, break-all and keep-all, and
// across elements, decorated.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 8px; color: #2b2722; font-size: 14px; }
.row { display: flex; gap: 10px; margin-bottom: 8px; align-items: flex-start; }
p { margin: 0; width: 110px; background: #f1ece4; }
.narrow { width: 0; }
.red { color: #c8473d; }
.box { border: 1px solid #3a6fd8; padding: 0 2px; }
.under { text-decoration: underline #3a6fd8; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	texts := []string{
		"https://example.com/path/to/page?query=a&b=c#frag",
		"if(a<b&&c>d){return f(x[i],y[j]);}else{x+=1;}",
		"Wait...what?!No way!!It's (paren)[bracket]{brace}",
		"Pay $$100, 99%% off; ab-12 x-3 !-4 2025-11-23",
	}
	row := func(style string) *ui.Node {
		r := ui.Div(ui.Class("row"))
		for _, s := range texts {
			r.Append(ui.P(ui.Style(style), ui.Text(s)))
		}
		return r
	}
	d.Body().Append(
		row(""),
		row("word-break: break-all"),
		row("word-break: keep-all"),
		ui.Div(ui.Class("row"),
			ui.P(ui.Class("narrow"), ui.Text("ab!cd}ef|gh,(ij)")),
			ui.P(ui.Class("narrow", "under"), ui.Text("ab-"), ui.Span(ui.Class("red"), ui.Text("12")),
				ui.Text("?x"), ui.Span(ui.Class("box"), ui.Text("$$")), ui.Text("<y>")),
			ui.P(ui.Class("under"), ui.Text("see "), ui.Span(ui.Class("red"), ui.Text("www.example.org/a_b")),
				ui.Text(", then (or) [and] {more}!")),
			ui.Div(ui.Style("display: inline-block; width: min-content; background: #f1ece4"),
				ui.Text("x->y a=b+c f(x)")),
		),
	)
	pages.Run(d, 520, 440)
}
